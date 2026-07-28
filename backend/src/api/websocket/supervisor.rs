use std::{
    collections::HashMap,
    sync::{
        Arc,
        atomic::{AtomicBool, AtomicU64, AtomicUsize, Ordering},
    },
    time::Duration,
};

use axum::extract::ws::WebSocket;
use serde_json::json;
use tokio::{
    sync::mpsc,
    task::{JoinError, JoinSet},
    time::{Instant, MissedTickBehavior, interval_at, timeout, timeout_at},
};
use tokio_util::sync::CancellationToken;

use super::{
    connection::{Connection, OutboundFrame},
    ports::{AgentStreamProvider, WorkerStatusProvider},
    transport::{
        AxumWebSocketTransport, InboundFrame, SocketError, SocketReader, SocketWriter,
        WebSocketTransport,
    },
    types::{
        CHANNEL_WORKER_STATUS, SubscribeMessage, WorkerStatusMessage, WorkerStatusSnapshot,
        WorkerStatusUpdate,
    },
};

const DEFAULT_MAX_CONNECTIONS: usize = 1000;
const DEFAULT_PING_PERIOD: Duration = Duration::from_secs(30);
const DEFAULT_PONG_WAIT: Duration = Duration::from_secs(60);
const DEFAULT_WRITE_WAIT: Duration = Duration::from_secs(10);
const DEFAULT_SHUTDOWN_WAIT: Duration = Duration::from_secs(10);

#[derive(Debug, Clone)]
pub struct WebSocketConfig {
    pub max_connections: usize,
    pub ping_period: Duration,
    pub pong_wait: Duration,
    pub write_wait: Duration,
    pub shutdown_wait: Duration,
}

impl Default for WebSocketConfig {
    fn default() -> Self {
        Self {
            max_connections: DEFAULT_MAX_CONNECTIONS,
            ping_period: DEFAULT_PING_PERIOD,
            pong_wait: DEFAULT_PONG_WAIT,
            write_wait: DEFAULT_WRITE_WAIT,
            shutdown_wait: DEFAULT_SHUTDOWN_WAIT,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, thiserror::Error)]
pub enum WebSocketConfigError {
    #[error("WebSocket max_connections must be greater than zero")]
    MaxConnectionsZero,
    #[error("WebSocket max_connections exceeds the Tokio channel capacity limit")]
    MaxConnectionsTooLarge,
    #[error("WebSocket ping_period must be greater than zero")]
    PingPeriodZero,
    #[error("WebSocket ping_period cannot form a representable deadline")]
    PingPeriodUnrepresentable,
    #[error("WebSocket pong_wait must be greater than zero")]
    PongWaitZero,
    #[error("WebSocket pong_wait cannot form a representable deadline")]
    PongWaitUnrepresentable,
    #[error("WebSocket write_wait must be greater than zero")]
    WriteWaitZero,
    #[error("WebSocket write_wait cannot form a representable deadline")]
    WriteWaitUnrepresentable,
    #[error("WebSocket shutdown_wait must be greater than zero")]
    ShutdownWaitZero,
    #[error("WebSocket shutdown_wait cannot form a representable deadline")]
    ShutdownWaitUnrepresentable,
}

fn validate_config(config: &WebSocketConfig) -> Result<(), WebSocketConfigError> {
    if config.max_connections == 0 {
        return Err(WebSocketConfigError::MaxConnectionsZero);
    }
    if config.max_connections > tokio::sync::Semaphore::MAX_PERMITS / 2 {
        return Err(WebSocketConfigError::MaxConnectionsTooLarge);
    }
    if config.ping_period.is_zero() {
        return Err(WebSocketConfigError::PingPeriodZero);
    }
    if config.pong_wait.is_zero() {
        return Err(WebSocketConfigError::PongWaitZero);
    }
    if config.write_wait.is_zero() {
        return Err(WebSocketConfigError::WriteWaitZero);
    }
    if config.shutdown_wait.is_zero() {
        return Err(WebSocketConfigError::ShutdownWaitZero);
    }
    let now = Instant::now();
    if now.checked_add(config.ping_period).is_none() {
        return Err(WebSocketConfigError::PingPeriodUnrepresentable);
    }
    if now.checked_add(config.pong_wait).is_none() {
        return Err(WebSocketConfigError::PongWaitUnrepresentable);
    }
    if now.checked_add(config.write_wait).is_none() {
        return Err(WebSocketConfigError::WriteWaitUnrepresentable);
    }
    if now.checked_add(config.shutdown_wait).is_none() {
        return Err(WebSocketConfigError::ShutdownWaitUnrepresentable);
    }
    Ok(())
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, thiserror::Error)]
pub enum AdmissionError {
    #[error("WebSocket admission queue is full")]
    Full,
    #[error("WebSocket admission is closed")]
    Closed,
}

struct Admission {
    transport: Box<dyn WebSocketTransport>,
}

#[derive(Clone)]
pub struct WebSocketSupervisorHandle {
    admissions: mpsc::Sender<Admission>,
    accepting: Arc<AtomicBool>,
    active_connections: Arc<AtomicUsize>,
}

impl WebSocketSupervisorHandle {
    pub fn try_admit<T>(&self, transport: T) -> Result<(), AdmissionError>
    where
        T: WebSocketTransport,
    {
        if !self.accepting.load(Ordering::Acquire) {
            return Err(AdmissionError::Closed);
        }

        match self.admissions.try_send(Admission {
            transport: Box::new(transport),
        }) {
            Ok(()) => Ok(()),
            Err(mpsc::error::TrySendError::Full(_)) => Err(AdmissionError::Full),
            Err(mpsc::error::TrySendError::Closed(_)) => Err(AdmissionError::Closed),
        }
    }

    pub fn try_admit_axum(&self, socket: WebSocket) -> Result<(), AdmissionError> {
        self.try_admit(AxumWebSocketTransport::new(socket))
    }

    pub fn active_connections(&self) -> usize {
        self.active_connections.load(Ordering::Acquire)
    }

    pub fn is_accepting(&self) -> bool {
        self.accepting.load(Ordering::Acquire)
    }
}

/// The entire `on_upgrade` continuation: transfer the socket to the
/// application-owned actor and return. Dropping a rejected socket closes it.
pub async fn hand_off_upgrade(socket: WebSocket, supervisor: WebSocketSupervisorHandle) {
    let _ = supervisor.try_admit_axum(socket);
}

pub struct WebSocketSupervisor {
    config: WebSocketConfig,
    admissions: mpsc::Receiver<Admission>,
    control_tx: mpsc::Sender<Control>,
    control_rx: mpsc::Receiver<Control>,
    accepting: Arc<AtomicBool>,
    active_connections: Arc<AtomicUsize>,
    next_connection_id: AtomicU64,
    next_task_id: AtomicU64,
    agent_streams: Arc<dyn AgentStreamProvider>,
    worker_statuses: Arc<dyn WorkerStatusProvider>,
}

impl WebSocketSupervisor {
    pub fn new(
        config: WebSocketConfig,
        agent_streams: Arc<dyn AgentStreamProvider>,
        worker_statuses: Arc<dyn WorkerStatusProvider>,
    ) -> Result<(WebSocketSupervisorHandle, Self), WebSocketConfigError> {
        validate_config(&config)?;
        let queue_capacity = config.max_connections;
        let (admissions_tx, admissions) = mpsc::channel(queue_capacity);
        let (control_tx, control_rx) = mpsc::channel(queue_capacity * 2);
        let accepting = Arc::new(AtomicBool::new(true));
        let active_connections = Arc::new(AtomicUsize::new(0));
        let handle = WebSocketSupervisorHandle {
            admissions: admissions_tx,
            accepting: accepting.clone(),
            active_connections: active_connections.clone(),
        };
        Ok((
            handle,
            Self {
                config,
                admissions,
                control_tx,
                control_rx,
                accepting,
                active_connections,
                next_connection_id: AtomicU64::new(1),
                next_task_id: AtomicU64::new(1),
                agent_streams,
                worker_statuses,
            },
        ))
    }

    pub async fn run(
        mut self,
        root_cancellation: CancellationToken,
    ) -> Result<(), WebSocketSupervisorError> {
        let all_connections = root_cancellation.child_token();
        let mut tasks = JoinSet::new();
        let mut streams = HashMap::<StreamKey, StreamRegistration>::new();
        let mut task_failures = Vec::new();

        loop {
            tokio::select! {
                biased;
                () = root_cancellation.cancelled() => break,
                completed = tasks.join_next(), if !tasks.is_empty() => {
                    if observe_task(completed, &mut streams, &mut task_failures) {
                        break;
                    }
                }
                control = self.control_rx.recv() => {
                    if let Some(control) = control {
                        self.apply_control(
                            control,
                            &mut tasks,
                            &mut streams,
                        );
                    }
                }
                admission = self.admissions.recv() => {
                    let Some(admission) = admission else {
                        break;
                    };
                    self.admit(
                        admission,
                        &all_connections,
                        &mut tasks,
                    );
                }
            }
        }

        self.accepting.store(false, Ordering::Release);
        self.admissions.close();
        all_connections.cancel();
        while self.admissions.try_recv().is_ok() {}
        for registration in streams.values() {
            registration.cancellation.cancel();
        }

        let timed_out = drain_tasks(
            &mut tasks,
            self.config.shutdown_wait,
            &mut streams,
            &mut task_failures,
        )
        .await;
        self.control_rx.close();
        while self.control_rx.try_recv().is_ok() {}

        if timed_out || !task_failures.is_empty() {
            Err(WebSocketSupervisorError {
                timed_out,
                task_failures,
            })
        } else {
            Ok(())
        }
    }

    fn admit(
        &self,
        admission: Admission,
        all_connections: &CancellationToken,
        tasks: &mut JoinSet<OwnedTaskOutcome>,
    ) {
        if self.active_connections.load(Ordering::Acquire) >= self.config.max_connections {
            return;
        }

        let connection_id = self.next_connection_id.fetch_add(1, Ordering::Relaxed);
        let (reader, writer) = admission.transport.split();
        let connection_cancellation = all_connections.child_token();
        let (connection, outbound_rx) = Connection::new(&connection_cancellation);
        let lifetime = Arc::new(ConnectionLifetime::new(
            connection_cancellation,
            self.active_connections.clone(),
        ));

        let writer_lifetime = lifetime.clone();
        let writer_config = self.config.clone();
        tasks.spawn(async move {
            let result = run_writer(
                writer,
                outbound_rx,
                writer_lifetime.cancellation.clone(),
                &writer_config,
            )
            .await;
            writer_lifetime.cancellation.cancel();
            OwnedTaskOutcome::Writer {
                connection_id,
                result,
            }
        });

        let session_lifetime = lifetime.clone();
        let agent_streams = self.agent_streams.clone();
        let worker_statuses = self.worker_statuses.clone();
        let control = self.control_tx.clone();
        let pong_wait = self.config.pong_wait;
        tasks.spawn(async move {
            let result = run_session(
                reader,
                SessionContext {
                    connection_id,
                    connection,
                    lifetime: session_lifetime.clone(),
                    agent_streams,
                    worker_statuses,
                    control,
                    pong_wait,
                },
            )
            .await;
            session_lifetime.cancellation.cancel();
            OwnedTaskOutcome::Session {
                connection_id,
                result,
            }
        });
    }

    fn apply_control(
        &self,
        control: Control,
        tasks: &mut JoinSet<OwnedTaskOutcome>,
        streams: &mut HashMap<StreamKey, StreamRegistration>,
    ) {
        match control {
            Control::StartAgent {
                connection_id,
                request_id,
                receiver,
                connection,
                lifetime,
            } => {
                let key = StreamKey::Agent(connection_id);
                cancel_stream(streams, &key);
                if lifetime.cancellation.is_cancelled() {
                    return;
                }
                let cancellation = lifetime.cancellation.child_token();
                let task_id = self.next_task_id.fetch_add(1, Ordering::Relaxed);
                streams.insert(
                    key.clone(),
                    StreamRegistration {
                        task_id,
                        cancellation: cancellation.clone(),
                        request_id: Some(request_id.clone()),
                    },
                );
                tasks.spawn(async move {
                    run_agent_stream(request_id, receiver, connection, cancellation).await;
                    drop(lifetime);
                    OwnedTaskOutcome::Stream { key, task_id }
                });
            }
            Control::StopAgent {
                connection_id,
                request_id,
            } => {
                let key = StreamKey::Agent(connection_id);
                if streams
                    .get(&key)
                    .is_some_and(|entry| entry.request_id.as_deref() == Some(&request_id))
                {
                    cancel_stream(streams, &key);
                }
            }
            Control::StartWorker {
                connection_id,
                snapshot,
                receiver,
                connection,
                lifetime,
            } => {
                let key = StreamKey::Worker(connection_id);
                cancel_stream(streams, &key);
                if lifetime.cancellation.is_cancelled() {
                    return;
                }
                let cancellation = lifetime.cancellation.child_token();
                let task_id = self.next_task_id.fetch_add(1, Ordering::Relaxed);
                streams.insert(
                    key.clone(),
                    StreamRegistration {
                        task_id,
                        cancellation: cancellation.clone(),
                        request_id: None,
                    },
                );
                tasks.spawn(async move {
                    run_worker_stream(snapshot, receiver, connection, cancellation).await;
                    drop(lifetime);
                    OwnedTaskOutcome::Stream { key, task_id }
                });
            }
            Control::StopWorker { connection_id } => {
                cancel_stream(streams, &StreamKey::Worker(connection_id));
            }
        }
    }
}

struct ConnectionLifetime {
    cancellation: CancellationToken,
    active_connections: Arc<AtomicUsize>,
}

impl ConnectionLifetime {
    fn new(cancellation: CancellationToken, active_connections: Arc<AtomicUsize>) -> Self {
        active_connections.fetch_add(1, Ordering::AcqRel);
        Self {
            cancellation,
            active_connections,
        }
    }
}

impl Drop for ConnectionLifetime {
    fn drop(&mut self) {
        self.active_connections.fetch_sub(1, Ordering::AcqRel);
    }
}

enum Control {
    StartAgent {
        connection_id: u64,
        request_id: String,
        receiver: mpsc::Receiver<super::types::AgentStreamEvent>,
        connection: Connection,
        lifetime: Arc<ConnectionLifetime>,
    },
    StopAgent {
        connection_id: u64,
        request_id: String,
    },
    StartWorker {
        connection_id: u64,
        snapshot: Vec<WorkerStatusSnapshot>,
        receiver: mpsc::Receiver<WorkerStatusUpdate>,
        connection: Connection,
        lifetime: Arc<ConnectionLifetime>,
    },
    StopWorker {
        connection_id: u64,
    },
}

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
enum StreamKey {
    Agent(u64),
    Worker(u64),
}

struct StreamRegistration {
    task_id: u64,
    cancellation: CancellationToken,
    request_id: Option<String>,
}

enum OwnedTaskOutcome {
    Session {
        connection_id: u64,
        result: Result<(), SocketError>,
    },
    Writer {
        connection_id: u64,
        result: Result<(), SocketError>,
    },
    Stream {
        key: StreamKey,
        task_id: u64,
    },
}

struct SessionContext {
    connection_id: u64,
    connection: Connection,
    lifetime: Arc<ConnectionLifetime>,
    agent_streams: Arc<dyn AgentStreamProvider>,
    worker_statuses: Arc<dyn WorkerStatusProvider>,
    control: mpsc::Sender<Control>,
    pong_wait: Duration,
}

async fn run_session(
    mut reader: Box<dyn SocketReader>,
    context: SessionContext,
) -> Result<(), SocketError> {
    let cancellation = context.lifetime.cancellation.clone();
    let mut pong_deadline = checked_deadline(
        context.pong_wait,
        "WebSocket pong deadline is unrepresentable",
    )?;
    let mut worker_subscribed = false;

    loop {
        tokio::select! {
            biased;
            () = cancellation.cancelled() => return Ok(()),
            () = tokio::time::sleep_until(pong_deadline) => return Ok(()),
            incoming = reader.receive() => {
                let Some(incoming) = incoming? else {
                    return Ok(());
                };
                match incoming {
                    InboundFrame::Pong => {
                        pong_deadline = checked_deadline(
                            context.pong_wait,
                            "WebSocket pong deadline is unrepresentable",
                        )?;
                    }
                    InboundFrame::Ping => {
                        context.connection.send(OutboundFrame::Pong);
                    }
                    InboundFrame::Close => return Ok(()),
                    InboundFrame::Text(text) => {
                        let Ok(message) =
                            serde_json::from_str::<SubscribeMessage>(&text)
                        else {
                            continue;
                        };
                        if message.channel == CHANNEL_WORKER_STATUS {
                            if message.action == "subscribe"
                                && !worker_subscribed
                            {
                                worker_subscribed = true;
                                context.connection.send(OutboundFrame::Text(
                                    json!({
                                        "type": "subscribed",
                                        "channel": CHANNEL_WORKER_STATUS,
                                    })
                                    .to_string(),
                                ));
                                // Subscribe before reading the snapshot so an
                                // update concurrent with snapshot collection is
                                // queued rather than lost.
                                let receiver =
                                    context.worker_statuses.subscribe();
                                let snapshot =
                                    context.worker_statuses.snapshot();
                                let command = Control::StartWorker {
                                    connection_id: context.connection_id,
                                    snapshot,
                                    receiver,
                                    connection: context.connection.clone(),
                                    lifetime: context.lifetime.clone(),
                                };
                                if send_control(
                                    &context.control,
                                    command,
                                    &cancellation,
                                )
                                .await
                                .is_err()
                                {
                                    return Ok(());
                                }
                            } else if message.action == "unsubscribe"
                                && worker_subscribed
                            {
                                worker_subscribed = false;
                                let command =
                                    Control::StopWorker {
                                        connection_id: context.connection_id,
                                    };
                                if send_control(
                                    &context.control,
                                    command,
                                    &cancellation,
                                )
                                .await
                                .is_err()
                                {
                                    return Ok(());
                                }
                            }
                            continue;
                        }

                        if message.channel.is_empty()
                            && message.action == "subscribe"
                            && !message.request_id.is_empty()
                        {
                            match context.agent_streams
                                .take_response_stream(&message.request_id)
                            {
                                Some(receiver) => {
                                    let command = Control::StartAgent {
                                        connection_id: context.connection_id,
                                        request_id: message.request_id,
                                        receiver,
                                        connection: context.connection.clone(),
                                        lifetime: context.lifetime.clone(),
                                    };
                                    if send_control(
                                        &context.control,
                                        command,
                                        &cancellation,
                                    )
                                    .await
                                    .is_err()
                                    {
                                        return Ok(());
                                    }
                                }
                                None => {
                                    context.connection.send(OutboundFrame::Text(
                                        json!({
                                            "requestId": message.request_id,
                                            "type": "error",
                                            "error": "Request not found",
                                            "done": true,
                                        })
                                        .to_string(),
                                    ));
                                }
                            }
                        } else if message.channel.is_empty()
                            && message.action == "unsubscribe"
                            && !message.request_id.is_empty()
                        {
                            let command = Control::StopAgent {
                                connection_id: context.connection_id,
                                request_id: message.request_id,
                            };
                            if send_control(
                                &context.control,
                                command,
                                &cancellation,
                            )
                            .await
                            .is_err()
                            {
                                return Ok(());
                            }
                        }
                    }
                    InboundFrame::Binary => {}
                }
            }
        }
    }
}

async fn send_control(
    control: &mpsc::Sender<Control>,
    command: Control,
    cancellation: &CancellationToken,
) -> Result<(), ()> {
    tokio::select! {
        biased;
        () = cancellation.cancelled() => Err(()),
        result = control.send(command) => result.map_err(|_| ()),
    }
}

async fn run_writer(
    mut writer: Box<dyn SocketWriter>,
    mut receiver: mpsc::Receiver<OutboundFrame>,
    cancellation: CancellationToken,
    config: &WebSocketConfig,
) -> Result<(), SocketError> {
    let ping_deadline = checked_deadline(
        config.ping_period,
        "WebSocket ping deadline is unrepresentable",
    )?;
    let mut ping = interval_at(ping_deadline, config.ping_period);
    ping.set_missed_tick_behavior(MissedTickBehavior::Delay);
    let mut prefer_ping = false;

    loop {
        let event = tokio::select! {
            biased;
            () = cancellation.cancelled() => {
                let _ = timeout(
                    config.write_wait,
                    writer.send(OutboundFrame::Close),
                )
                .await;
                return Ok(());
            }
            event = next_writer_event(
                &mut ping,
                &mut receiver,
                prefer_ping,
            ) => event,
        };
        let frame = match event {
            WriterEvent::Ping => {
                prefer_ping = false;
                OutboundFrame::Ping
            }
            WriterEvent::Frame(Some(frame)) => {
                prefer_ping = true;
                frame
            }
            WriterEvent::Frame(None) => return Ok(()),
        };

        timeout(config.write_wait, writer.send(frame))
            .await
            .map_err(|_| SocketError::new("WebSocket write timed out"))??;
    }
}

enum WriterEvent {
    Ping,
    Frame(Option<OutboundFrame>),
}

async fn next_writer_event(
    ping: &mut tokio::time::Interval,
    receiver: &mut mpsc::Receiver<OutboundFrame>,
    prefer_ping: bool,
) -> WriterEvent {
    if prefer_ping {
        tokio::select! {
            biased;
            _ = ping.tick() => WriterEvent::Ping,
            frame = receiver.recv() => WriterEvent::Frame(frame),
        }
    } else {
        tokio::select! {
            biased;
            frame = receiver.recv() => WriterEvent::Frame(frame),
            _ = ping.tick() => WriterEvent::Ping,
        }
    }
}

fn checked_deadline(duration: Duration, message: &'static str) -> Result<Instant, SocketError> {
    Instant::now()
        .checked_add(duration)
        .ok_or_else(|| SocketError::new(message))
}

async fn run_agent_stream(
    request_id: String,
    mut receiver: mpsc::Receiver<super::types::AgentStreamEvent>,
    connection: Connection,
    cancellation: CancellationToken,
) {
    loop {
        let event = tokio::select! {
            biased;
            () = cancellation.cancelled() => return,
            event = receiver.recv() => {
                let Some(event) = event else {
                    return;
                };
                event
            }
        };
        let (text, terminal) = event.into_wire_message(&request_id);
        connection.send(OutboundFrame::Text(text));
        if terminal {
            return;
        }
    }
}

async fn run_worker_stream(
    snapshot: Vec<WorkerStatusSnapshot>,
    mut receiver: mpsc::Receiver<WorkerStatusUpdate>,
    connection: Connection,
    cancellation: CancellationToken,
) {
    for status in &snapshot {
        if cancellation.is_cancelled() {
            return;
        }
        if let Ok(text) = serde_json::to_string(&WorkerStatusMessage::initial(status)) {
            connection.send(OutboundFrame::Text(text));
        }
    }

    loop {
        let update = tokio::select! {
            biased;
            () = cancellation.cancelled() => return,
            update = receiver.recv() => {
                let Some(update) = update else {
                    return;
                };
                update
            }
        };
        if let Ok(text) = serde_json::to_string(&WorkerStatusMessage::update(&update)) {
            connection.send(OutboundFrame::Text(text));
        }
    }
}

fn cancel_stream(streams: &mut HashMap<StreamKey, StreamRegistration>, key: &StreamKey) {
    if let Some(registration) = streams.remove(key) {
        registration.cancellation.cancel();
    }
}

fn observe_task(
    completed: Option<Result<OwnedTaskOutcome, JoinError>>,
    streams: &mut HashMap<StreamKey, StreamRegistration>,
    failures: &mut Vec<String>,
) -> bool {
    match completed {
        Some(Ok(OwnedTaskOutcome::Stream { key, task_id })) => {
            if streams
                .get(&key)
                .is_some_and(|registration| registration.task_id == task_id)
            {
                streams.remove(&key);
            }
            false
        }
        Some(Ok(OwnedTaskOutcome::Session {
            connection_id,
            result,
        }))
        | Some(Ok(OwnedTaskOutcome::Writer {
            connection_id,
            result,
        })) => {
            if let Err(error) = result {
                tracing::debug!(
                    connection_id,
                    %error,
                    "WebSocket connection task ended"
                );
            }
            false
        }
        Some(Err(error)) if error.is_cancelled() => false,
        Some(Err(error)) => {
            failures.push(format!("WebSocket task failed: {error}"));
            true
        }
        None => false,
    }
}

async fn drain_tasks(
    tasks: &mut JoinSet<OwnedTaskOutcome>,
    shutdown_wait: Duration,
    streams: &mut HashMap<StreamKey, StreamRegistration>,
    failures: &mut Vec<String>,
) -> bool {
    let Some(deadline) = Instant::now().checked_add(shutdown_wait) else {
        tasks.abort_all();
        while let Some(completed) = tasks.join_next().await {
            observe_task(Some(completed), streams, failures);
        }
        streams.clear();
        return true;
    };
    while !tasks.is_empty() {
        match timeout_at(deadline, tasks.join_next()).await {
            Ok(completed) => {
                observe_task(completed, streams, failures);
            }
            Err(_) => {
                tasks.abort_all();
                while let Some(completed) = tasks.join_next().await {
                    observe_task(Some(completed), streams, failures);
                }
                streams.clear();
                return true;
            }
        }
    }
    streams.clear();
    false
}

#[derive(Debug)]
pub struct WebSocketSupervisorError {
    timed_out: bool,
    task_failures: Vec<String>,
}

impl std::fmt::Display for WebSocketSupervisorError {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let mut parts = self.task_failures.clone();
        if self.timed_out {
            parts.push("WebSocket tasks exceeded the shutdown deadline".into());
        }
        formatter.write_str(&parts.join("; "))
    }
}

impl std::error::Error for WebSocketSupervisorError {}
