use std::{
    collections::BTreeMap,
    sync::{
        Arc, Mutex, MutexGuard,
        atomic::{AtomicU64, Ordering},
    },
};

use chrono::{DateTime, Utc};
use tokio::sync::mpsc;
use uuid::Uuid;

use super::{WorkerState, WorkerStatus, WorkerStatusUpdate};

const SUBSCRIBER_CAPACITY: usize = 100;

pub trait Clock: Send + Sync {
    fn now(&self) -> DateTime<Utc>;
}

#[derive(Debug, Default)]
pub struct SystemClock;

impl Clock for SystemClock {
    fn now(&self) -> DateTime<Utc> {
        Utc::now()
    }
}

type SendUpdate = dyn Fn(&WorkerStatusUpdate) -> SubscriberState + Send + Sync;

struct Subscriber {
    send: Box<SendUpdate>,
    is_closed: Box<dyn Fn() -> bool + Send + Sync>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SubscriberState {
    Active,
    Closed,
}

#[derive(Default)]
struct StatusState {
    statuses: BTreeMap<String, WorkerStatus>,
    subscribers: BTreeMap<u64, Subscriber>,
}

pub struct StatusService {
    state: Mutex<StatusState>,
    clock: Arc<dyn Clock>,
    next_subscriber_id: AtomicU64,
}

impl StatusService {
    pub fn new(clock: Arc<dyn Clock>) -> Self {
        Self {
            state: Mutex::new(StatusState::default()),
            clock,
            next_subscriber_id: AtomicU64::new(1),
        }
    }

    pub fn register_worker(&self, name: &str) {
        let mut state = self.state();
        state
            .statuses
            .entry(name.to_owned())
            .or_insert_with(|| WorkerStatus::idle(name));
    }

    pub fn update_status(
        &self,
        name: &str,
        worker_state: WorkerState,
        progress: i32,
        message: impl Into<String>,
    ) {
        let now = self.clock.now();
        let mut state = self.state();
        let status = state
            .statuses
            .entry(name.to_owned())
            .or_insert_with(|| WorkerStatus::idle(name));
        status.state = worker_state;
        status.progress = progress;
        status.message = message.into();
        if worker_state == WorkerState::Running && status.started_at.is_none() {
            status.started_at = Some(now);
            status.completed_at = None;
            status.error.clear();
        }
        if matches!(worker_state, WorkerState::Completed | WorkerState::Failed) {
            status.completed_at = Some(now);
        }
        let status = status.clone();
        Self::broadcast(&mut state, now, name, status);
    }

    pub fn set_progress(
        &self,
        name: &str,
        items_done: i32,
        items_total: i32,
        message: impl Into<String>,
    ) {
        let now = self.clock.now();
        let mut state = self.state();
        let Some(status) = state.statuses.get_mut(name) else {
            return;
        };
        status.items_done = items_done;
        status.items_total = items_total;
        status.message = message.into();
        if items_total > 0 {
            status.progress = items_done.saturating_mul(100) / items_total;
        }
        let status = status.clone();
        Self::broadcast(&mut state, now, name, status);
    }

    pub fn set_error(&self, name: &str, error: impl Into<String>) {
        let now = self.clock.now();
        let mut state = self.state();
        let Some(status) = state.statuses.get_mut(name) else {
            return;
        };
        status.error = error.into();
        status.state = WorkerState::Failed;
        status.completed_at = Some(now);
        let status = status.clone();
        Self::broadcast(&mut state, now, name, status);
    }

    pub fn start_worker(&self, name: &str, task_run_id: Option<Uuid>) {
        let now = self.clock.now();
        let mut state = self.state();
        let status = state
            .statuses
            .entry(name.to_owned())
            .or_insert_with(|| WorkerStatus::idle(name));
        status.state = WorkerState::Running;
        status.task_run_id = task_run_id;
        status.started_at = Some(now);
        status.completed_at = None;
        status.progress = 0;
        status.message = "Starting...".to_owned();
        status.error.clear();
        status.items_done = 0;
        status.items_total = 0;
        let status = status.clone();
        Self::broadcast(&mut state, now, name, status);
    }

    pub fn complete_worker(&self, name: &str, message: impl Into<String>) {
        let now = self.clock.now();
        let mut state = self.state();
        let Some(status) = state.statuses.get_mut(name) else {
            return;
        };
        status.state = WorkerState::Completed;
        status.completed_at = Some(now);
        status.progress = 100;
        status.message = message.into();
        let status = status.clone();
        Self::broadcast(&mut state, now, name, status);
    }

    pub fn reset_worker(&self, name: &str) {
        let now = self.clock.now();
        let mut state = self.state();
        let Some(status) = state.statuses.get_mut(name) else {
            return;
        };
        status.state = WorkerState::Idle;
        status.progress = 0;
        status.message.clear();
        status.items_done = 0;
        status.items_total = 0;
        let status = status.clone();
        Self::broadcast(&mut state, now, name, status);
    }

    pub fn status(&self, name: &str) -> Option<WorkerStatus> {
        self.state().statuses.get(name).cloned()
    }

    pub fn snapshot(&self) -> Vec<(String, WorkerStatus)> {
        self.state()
            .statuses
            .iter()
            .map(|(name, status)| (name.clone(), status.clone()))
            .collect()
    }

    pub fn subscribe(&self) -> mpsc::Receiver<WorkerStatusUpdate> {
        self.subscribe_mapped(Clone::clone)
    }

    pub fn subscribe_mapped<T, F>(&self, map: F) -> mpsc::Receiver<T>
    where
        T: Send + 'static,
        F: Fn(&WorkerStatusUpdate) -> T + Send + Sync + 'static,
    {
        let (sender, receiver) = mpsc::channel(SUBSCRIBER_CAPACITY);
        let closed_sender = sender.clone();
        let id = self.next_subscriber_id.fetch_add(1, Ordering::Relaxed);
        let mut state = self.state();
        state
            .subscribers
            .retain(|_, subscriber| !(subscriber.is_closed)());
        state.subscribers.insert(
            id,
            Subscriber {
                send: Box::new(move |update| match sender.try_send(map(update)) {
                    Ok(()) | Err(mpsc::error::TrySendError::Full(_)) => SubscriberState::Active,
                    Err(mpsc::error::TrySendError::Closed(_)) => SubscriberState::Closed,
                }),
                is_closed: Box::new(move || closed_sender.is_closed()),
            },
        );
        receiver
    }

    pub fn subscriber_count(&self) -> usize {
        let mut state = self.state();
        state
            .subscribers
            .retain(|_, subscriber| !(subscriber.is_closed)());
        state.subscribers.len()
    }

    fn broadcast(
        state: &mut StatusState,
        timestamp: DateTime<Utc>,
        name: &str,
        status: WorkerStatus,
    ) {
        let update = WorkerStatusUpdate {
            worker_name: name.to_owned(),
            status,
            timestamp,
        };
        state
            .subscribers
            .retain(|_, subscriber| (subscriber.send)(&update) == SubscriberState::Active);
    }

    fn state(&self) -> MutexGuard<'_, StatusState> {
        self.state
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }
}
