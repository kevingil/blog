use std::{
    collections::HashMap,
    future::pending,
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, Ordering},
    },
    time::Duration,
};

use async_trait::async_trait;
use blog_backend::api::websocket::{
    AdmissionError, AgentStreamEvent, AgentStreamProvider, EmptyWorkerStatusProvider, InboundFrame,
    SocketError, SocketReader, SocketWriter, UnavailableAgentStreamProvider, WebSocketConfig,
    WebSocketConfigError, WebSocketSupervisor, WebSocketSupervisorHandle, WebSocketTransport,
    WorkerStatus, WorkerStatusProvider, WorkerStatusSnapshot, WorkerStatusUpdate,
    connection::{Connection, OutboundFrame, QueueResult},
};
use chrono::{TimeZone, Utc};
use serde_json::{Value, json};
use tokio::{
    sync::mpsc,
    task::JoinHandle,
    time::{Instant, sleep, timeout},
};
use tokio_util::sync::CancellationToken;

struct TestTransport {
    reader: TestReader,
    writer: TestWriter,
}

impl WebSocketTransport for TestTransport {
    fn split(self: Box<Self>) -> (Box<dyn SocketReader>, Box<dyn SocketWriter>) {
        let Self { reader, writer } = *self;
        (Box::new(reader), Box::new(writer))
    }
}

struct TestReader {
    incoming: mpsc::UnboundedReceiver<InboundFrame>,
    dropped: Arc<AtomicBool>,
}

impl Drop for TestReader {
    fn drop(&mut self) {
        self.dropped.store(true, Ordering::SeqCst);
    }
}

#[async_trait]
impl SocketReader for TestReader {
    async fn receive(&mut self) -> Result<Option<InboundFrame>, SocketError> {
        Ok(self.incoming.recv().await)
    }
}

enum WriterBehavior {
    Record(mpsc::UnboundedSender<OutboundFrame>),
    SlowRecord {
        outgoing: mpsc::UnboundedSender<OutboundFrame>,
        delay: Duration,
    },
    Hang,
}

struct TestWriter {
    behavior: WriterBehavior,
    dropped: Arc<AtomicBool>,
}

impl Drop for TestWriter {
    fn drop(&mut self) {
        self.dropped.store(true, Ordering::SeqCst);
    }
}

#[async_trait]
impl SocketWriter for TestWriter {
    async fn send(&mut self, frame: OutboundFrame) -> Result<(), SocketError> {
        match &self.behavior {
            WriterBehavior::Record(outgoing) => outgoing
                .send(frame)
                .map_err(|_| SocketError::new("test client disconnected")),
            WriterBehavior::SlowRecord { outgoing, delay } => {
                sleep(*delay).await;
                outgoing
                    .send(frame)
                    .map_err(|_| SocketError::new("test client disconnected"))
            }
            WriterBehavior::Hang => pending::<Result<(), SocketError>>().await,
        }
    }
}

struct TestSocket {
    transport: TestTransport,
    incoming: mpsc::UnboundedSender<InboundFrame>,
    outgoing: mpsc::UnboundedReceiver<OutboundFrame>,
    reader_dropped: Arc<AtomicBool>,
    writer_dropped: Arc<AtomicBool>,
}

impl TestSocket {
    fn recording() -> Self {
        let (incoming_tx, incoming) = mpsc::unbounded_channel();
        let (outgoing, outgoing_rx) = mpsc::unbounded_channel();
        let reader_dropped = Arc::new(AtomicBool::new(false));
        let writer_dropped = Arc::new(AtomicBool::new(false));
        Self {
            transport: TestTransport {
                reader: TestReader {
                    incoming,
                    dropped: reader_dropped.clone(),
                },
                writer: TestWriter {
                    behavior: WriterBehavior::Record(outgoing),
                    dropped: writer_dropped.clone(),
                },
            },
            incoming: incoming_tx,
            outgoing: outgoing_rx,
            reader_dropped,
            writer_dropped,
        }
    }

    fn hanging_writer() -> Self {
        let (incoming_tx, incoming) = mpsc::unbounded_channel();
        let (_outgoing, outgoing_rx) = mpsc::unbounded_channel();
        let reader_dropped = Arc::new(AtomicBool::new(false));
        let writer_dropped = Arc::new(AtomicBool::new(false));
        Self {
            transport: TestTransport {
                reader: TestReader {
                    incoming,
                    dropped: reader_dropped.clone(),
                },
                writer: TestWriter {
                    behavior: WriterBehavior::Hang,
                    dropped: writer_dropped.clone(),
                },
            },
            incoming: incoming_tx,
            outgoing: outgoing_rx,
            reader_dropped,
            writer_dropped,
        }
    }

    fn slow_recording(delay: Duration) -> Self {
        let (incoming_tx, incoming) = mpsc::unbounded_channel();
        let (outgoing, outgoing_rx) = mpsc::unbounded_channel();
        let reader_dropped = Arc::new(AtomicBool::new(false));
        let writer_dropped = Arc::new(AtomicBool::new(false));
        Self {
            transport: TestTransport {
                reader: TestReader {
                    incoming,
                    dropped: reader_dropped.clone(),
                },
                writer: TestWriter {
                    behavior: WriterBehavior::SlowRecord { outgoing, delay },
                    dropped: writer_dropped.clone(),
                },
            },
            incoming: incoming_tx,
            outgoing: outgoing_rx,
            reader_dropped,
            writer_dropped,
        }
    }
}

#[derive(Default)]
struct TestAgentStreams {
    streams: Mutex<HashMap<String, mpsc::Receiver<AgentStreamEvent>>>,
}

impl TestAgentStreams {
    fn insert(
        &self,
        request_id: &str,
        stream: mpsc::Receiver<AgentStreamEvent>,
    ) -> anyhow::Result<()> {
        let mut streams = self
            .streams
            .lock()
            .map_err(|_| anyhow::anyhow!("agent stream mutex poisoned"))?;
        streams.insert(request_id.to_owned(), stream);
        Ok(())
    }
}

impl AgentStreamProvider for TestAgentStreams {
    fn take_response_stream(&self, request_id: &str) -> Option<mpsc::Receiver<AgentStreamEvent>> {
        self.streams.lock().ok()?.remove(request_id)
    }
}

struct TestWorkerStatuses {
    snapshot: Vec<WorkerStatusSnapshot>,
    updates: Mutex<Option<mpsc::Receiver<WorkerStatusUpdate>>>,
}

impl WorkerStatusProvider for TestWorkerStatuses {
    fn snapshot(&self) -> Vec<WorkerStatusSnapshot> {
        self.snapshot.clone()
    }

    fn subscribe(&self) -> mpsc::Receiver<WorkerStatusUpdate> {
        if let Ok(mut updates) = self.updates.lock()
            && let Some(receiver) = updates.take()
        {
            return receiver;
        }
        let (_sender, receiver) = mpsc::channel(1);
        receiver
    }
}

struct SubscribeBeforeSnapshotStatuses {
    calls: Mutex<Vec<&'static str>>,
    updates: Mutex<Option<mpsc::Sender<WorkerStatusUpdate>>>,
    timestamp: chrono::DateTime<Utc>,
}

impl WorkerStatusProvider for SubscribeBeforeSnapshotStatuses {
    fn snapshot(&self) -> Vec<WorkerStatusSnapshot> {
        if let Ok(mut calls) = self.calls.lock() {
            calls.push("snapshot");
        }
        if let Ok(updates) = self.updates.lock()
            && let Some(updates) = updates.as_ref()
        {
            let _ = updates.try_send(WorkerStatusUpdate {
                worker_name: "crawler".into(),
                status: worker_status(None),
                timestamp: self.timestamp,
            });
        }
        vec![WorkerStatusSnapshot {
            worker_name: "crawler".into(),
            status: worker_status(None),
        }]
    }

    fn subscribe(&self) -> mpsc::Receiver<WorkerStatusUpdate> {
        if let Ok(mut calls) = self.calls.lock() {
            calls.push("subscribe");
        }
        let (updates, receiver) = mpsc::channel(4);
        if let Ok(mut registered) = self.updates.lock() {
            *registered = Some(updates);
        }
        receiver
    }
}

struct RunningSupervisor {
    handle: WebSocketSupervisorHandle,
    cancellation: CancellationToken,
    task: JoinHandle<Result<(), blog_backend::api::websocket::WebSocketSupervisorError>>,
}

impl RunningSupervisor {
    fn start(
        config: WebSocketConfig,
        agent: Arc<dyn AgentStreamProvider>,
        workers: Arc<dyn WorkerStatusProvider>,
    ) -> anyhow::Result<Self> {
        let (handle, supervisor) = WebSocketSupervisor::new(config, agent, workers)?;
        let cancellation = CancellationToken::new();
        let run_cancellation = cancellation.clone();
        let task = tokio::spawn(supervisor.run(run_cancellation));
        Ok(Self {
            handle,
            cancellation,
            task,
        })
    }

    async fn stop(self) -> anyhow::Result<()> {
        self.cancellation.cancel();
        self.task.await??;
        Ok(())
    }
}

fn fast_config() -> WebSocketConfig {
    WebSocketConfig {
        max_connections: 1000,
        ping_period: Duration::from_secs(5),
        pong_wait: Duration::from_secs(10),
        write_wait: Duration::from_millis(50),
        shutdown_wait: Duration::from_millis(100),
    }
}

async fn wait_for_active(
    handle: &WebSocketSupervisorHandle,
    expected: usize,
) -> anyhow::Result<()> {
    timeout(Duration::from_secs(1), async {
        while handle.active_connections() != expected {
            tokio::task::yield_now().await;
        }
    })
    .await?;
    Ok(())
}

async fn receive_frame(
    outgoing: &mut mpsc::UnboundedReceiver<OutboundFrame>,
) -> anyhow::Result<OutboundFrame> {
    timeout(Duration::from_secs(1), outgoing.recv())
        .await?
        .ok_or_else(|| anyhow::anyhow!("test WebSocket output closed"))
}

async fn receive_json(
    outgoing: &mut mpsc::UnboundedReceiver<OutboundFrame>,
) -> anyhow::Result<Value> {
    loop {
        if let OutboundFrame::Text(text) = receive_frame(outgoing).await? {
            return Ok(serde_json::from_str(&text)?);
        }
    }
}

fn send_text(incoming: &mpsc::UnboundedSender<InboundFrame>, value: Value) -> anyhow::Result<()> {
    incoming
        .send(InboundFrame::Text(value.to_string()))
        .map_err(|_| anyhow::anyhow!("test WebSocket input closed"))
}

fn event(value: Value) -> anyhow::Result<AgentStreamEvent> {
    AgentStreamEvent::from_value(value)
        .ok_or_else(|| anyhow::anyhow!("agent test event must be an object"))
}

fn worker_status(task_run_id: Option<&str>) -> WorkerStatus {
    WorkerStatus {
        name: "crawler".into(),
        state: "running".into(),
        task_run_id: task_run_id.map(str::to_owned),
        progress: 25,
        message: "crawling".into(),
        started_at: Some("2026-07-28T00:00:00Z".into()),
        completed_at: None,
        error: String::new(),
        items_total: 20,
        items_done: 5,
    }
}

#[tokio::test]
async fn outbound_queue_is_exactly_256_and_drops_newest_in_order() -> anyhow::Result<()> {
    let cancellation = CancellationToken::new();
    let (connection, mut receiver) = Connection::new(&cancellation);

    for index in 0..256 {
        let result = connection.send(OutboundFrame::Text(index.to_string()));
        anyhow::ensure!(result == QueueResult::Queued);
    }
    anyhow::ensure!(
        connection.send(OutboundFrame::Text("dropped".into())) == QueueResult::DroppedNewest
    );
    for index in 0..256 {
        let frame = receiver
            .recv()
            .await
            .ok_or_else(|| anyhow::anyhow!("queue closed early"))?;
        anyhow::ensure!(
            frame == OutboundFrame::Text(index.to_string()),
            "outbound ordering changed at {index}"
        );
    }
    anyhow::ensure!(receiver.try_recv().is_err());

    connection.close();
    anyhow::ensure!(connection.send(OutboundFrame::Close) == QueueResult::Closed);
    Ok(())
}

#[tokio::test]
async fn missing_and_malformed_request_subscriptions_match_contract() -> anyhow::Result<()> {
    let running = RunningSupervisor::start(
        fast_config(),
        Arc::new(UnavailableAgentStreamProvider),
        Arc::new(EmptyWorkerStatusProvider),
    )?;
    let mut socket = TestSocket::recording();
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;

    socket
        .incoming
        .send(InboundFrame::Text("{not-json".into()))
        .map_err(|_| anyhow::anyhow!("test WebSocket input closed"))?;
    send_text(
        &socket.incoming,
        json!({"action": "subscribe", "requestId": ""}),
    )?;
    anyhow::ensure!(
        timeout(Duration::from_millis(20), socket.outgoing.recv())
            .await
            .is_err(),
        "malformed or empty request unexpectedly produced a frame"
    );

    send_text(
        &socket.incoming,
        json!({"action": "subscribe", "requestId": "missing"}),
    )?;
    anyhow::ensure!(
        receive_json(&mut socket.outgoing).await?
            == json!({
                "requestId": "missing",
                "type": "error",
                "error": "Request not found",
                "done": true,
            })
    );

    running.stop().await?;
    Ok(())
}

#[tokio::test]
async fn agent_stream_overwrites_request_id_preserves_order_and_stops_on_done() -> anyhow::Result<()>
{
    let provider = Arc::new(TestAgentStreams::default());
    let (events_tx, events) = mpsc::channel(4);
    provider.insert("request-1", events)?;
    let running =
        RunningSupervisor::start(fast_config(), provider, Arc::new(EmptyWorkerStatusProvider))?;
    let mut socket = TestSocket::recording();
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;

    send_text(
        &socket.incoming,
        json!({"action": "subscribe", "requestId": "request-1"}),
    )?;
    events_tx
        .send(event(json!({
            "requestId": "wrong",
            "type": "content_delta",
            "content": "one",
        }))?)
        .await?;
    events_tx
        .send(event(json!({
            "type": "done",
            "done": true,
        }))?)
        .await?;
    events_tx
        .send(event(json!({
            "type": "content_delta",
            "content": "too late",
        }))?)
        .await?;

    let first = receive_json(&mut socket.outgoing).await?;
    let terminal = receive_json(&mut socket.outgoing).await?;
    anyhow::ensure!(first["requestId"] == "request-1");
    anyhow::ensure!(first["content"] == "one");
    anyhow::ensure!(terminal["requestId"] == "request-1");
    anyhow::ensure!(terminal["done"] == true);
    anyhow::ensure!(
        timeout(Duration::from_millis(20), socket.outgoing.recv())
            .await
            .is_err(),
        "stream continued after done"
    );

    running.stop().await?;
    Ok(())
}

#[tokio::test]
async fn agent_stream_stops_on_nonempty_error() -> anyhow::Result<()> {
    let provider = Arc::new(TestAgentStreams::default());
    let (events_tx, events) = mpsc::channel(3);
    provider.insert("request-error", events)?;
    let running =
        RunningSupervisor::start(fast_config(), provider, Arc::new(EmptyWorkerStatusProvider))?;
    let mut socket = TestSocket::recording();
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;
    send_text(
        &socket.incoming,
        json!({
            "action": "subscribe",
            "requestId": "request-error",
        }),
    )?;
    events_tx
        .send(event(json!({
            "type": "error",
            "error": "provider failed",
        }))?)
        .await?;
    events_tx
        .send(event(json!({"type": "done", "done": true}))?)
        .await?;

    let terminal = receive_json(&mut socket.outgoing).await?;
    anyhow::ensure!(terminal["requestId"] == "request-error");
    anyhow::ensure!(terminal["error"] == "provider failed");
    anyhow::ensure!(
        timeout(Duration::from_millis(20), socket.outgoing.recv())
            .await
            .is_err()
    );
    running.stop().await?;
    Ok(())
}

#[tokio::test]
async fn worker_subscribe_ack_snapshot_update_and_unsubscribe_match_contract() -> anyhow::Result<()>
{
    let (updates_tx, updates) = mpsc::channel(4);
    let workers = Arc::new(TestWorkerStatuses {
        snapshot: vec![WorkerStatusSnapshot {
            worker_name: "crawler".into(),
            status: worker_status(Some("initial-task-is-omitted")),
        }],
        updates: Mutex::new(Some(updates)),
    });
    let running = RunningSupervisor::start(
        fast_config(),
        Arc::new(UnavailableAgentStreamProvider),
        workers,
    )?;
    let mut socket = TestSocket::recording();
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;

    send_text(
        &socket.incoming,
        json!({"action": "subscribe", "channel": "worker-status"}),
    )?;
    anyhow::ensure!(
        receive_json(&mut socket.outgoing).await?
            == json!({
                "type": "subscribed",
                "channel": "worker-status",
            })
    );
    let initial = receive_json(&mut socket.outgoing).await?;
    anyhow::ensure!(initial["type"] == "worker-status");
    anyhow::ensure!(initial.get("timestamp").is_none());
    anyhow::ensure!(initial["status"].get("task_run_id").is_none());
    anyhow::ensure!(initial["status"]["error"] == "");

    let timestamp = Utc
        .with_ymd_and_hms(2026, 7, 28, 1, 2, 3)
        .single()
        .ok_or_else(|| anyhow::anyhow!("invalid test timestamp"))?;
    updates_tx
        .send(WorkerStatusUpdate {
            worker_name: "crawler".into(),
            status: worker_status(Some("run-123")),
            timestamp,
        })
        .await?;
    let update = receive_json(&mut socket.outgoing).await?;
    anyhow::ensure!(update["timestamp"] == "2026-07-28T01:02:03Z");
    anyhow::ensure!(update["status"].get("task_run_id").is_none());
    anyhow::ensure!(update["status"]["error"] == "");

    send_text(
        &socket.incoming,
        json!({"action": "subscribe", "channel": "worker-status"}),
    )?;
    anyhow::ensure!(
        timeout(Duration::from_millis(20), socket.outgoing.recv())
            .await
            .is_err(),
        "duplicate worker subscription emitted a second ack"
    );

    send_text(
        &socket.incoming,
        json!({"action": "unsubscribe", "channel": "worker-status"}),
    )?;
    tokio::task::yield_now().await;
    tokio::task::yield_now().await;
    let _ = updates_tx
        .send(WorkerStatusUpdate {
            worker_name: "crawler".into(),
            status: worker_status(Some("late")),
            timestamp,
        })
        .await;
    anyhow::ensure!(
        timeout(Duration::from_millis(20), socket.outgoing.recv())
            .await
            .is_err(),
        "unsubscribe emitted an ack or a later update"
    );

    running.stop().await?;
    Ok(())
}

#[tokio::test]
async fn worker_receiver_is_registered_before_snapshot_without_update_gap() -> anyhow::Result<()> {
    let timestamp = Utc
        .with_ymd_and_hms(2026, 7, 28, 4, 5, 6)
        .single()
        .ok_or_else(|| anyhow::anyhow!("invalid test timestamp"))?;
    let workers = Arc::new(SubscribeBeforeSnapshotStatuses {
        calls: Mutex::new(Vec::new()),
        updates: Mutex::new(None),
        timestamp,
    });
    let running = RunningSupervisor::start(
        fast_config(),
        Arc::new(UnavailableAgentStreamProvider),
        workers.clone(),
    )?;
    let mut socket = TestSocket::recording();
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;

    send_text(
        &socket.incoming,
        json!({"action": "subscribe", "channel": "worker-status"}),
    )?;
    anyhow::ensure!(receive_json(&mut socket.outgoing).await?["type"] == "subscribed");
    let initial = receive_json(&mut socket.outgoing).await?;
    anyhow::ensure!(initial.get("timestamp").is_none());
    let concurrent_update = receive_json(&mut socket.outgoing).await?;
    anyhow::ensure!(concurrent_update["timestamp"] == "2026-07-28T04:05:06Z");
    let calls = workers
        .calls
        .lock()
        .map_err(|_| anyhow::anyhow!("worker call mutex poisoned"))?
        .clone();
    anyhow::ensure!(calls == vec!["subscribe", "snapshot"]);

    running.stop().await?;
    Ok(())
}

#[tokio::test]
async fn ping_and_pong_liveness_use_configured_periods() -> anyhow::Result<()> {
    let config = WebSocketConfig {
        ping_period: Duration::from_millis(10),
        pong_wait: Duration::from_millis(45),
        ..fast_config()
    };
    let running = RunningSupervisor::start(
        config,
        Arc::new(UnavailableAgentStreamProvider),
        Arc::new(EmptyWorkerStatusProvider),
    )?;
    let mut socket = TestSocket::recording();
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;

    anyhow::ensure!(receive_frame(&mut socket.outgoing).await? == OutboundFrame::Ping);
    socket
        .incoming
        .send(InboundFrame::Pong)
        .map_err(|_| anyhow::anyhow!("test WebSocket input closed"))?;
    sleep(Duration::from_millis(30)).await;
    anyhow::ensure!(running.handle.active_connections() == 1);

    wait_for_active(&running.handle, 0).await?;
    running.stop().await?;
    Ok(())
}

#[tokio::test]
async fn saturated_outbound_queue_cannot_starve_ready_ping_ticks() -> anyhow::Result<()> {
    let provider = Arc::new(TestAgentStreams::default());
    let (events_tx, events) = mpsc::channel(400);
    for index in 0..300 {
        events_tx
            .send(event(json!({
                "type": "content_delta",
                "content": index.to_string(),
            }))?)
            .await?;
    }
    provider.insert("busy-request", events)?;
    let config = WebSocketConfig {
        ping_period: Duration::from_millis(5),
        pong_wait: Duration::from_secs(1),
        write_wait: Duration::from_millis(50),
        ..fast_config()
    };
    let running = RunningSupervisor::start(config, provider, Arc::new(EmptyWorkerStatusProvider))?;
    let mut socket = TestSocket::slow_recording(Duration::from_millis(6));
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;
    send_text(
        &socket.incoming,
        json!({"action": "subscribe", "requestId": "busy-request"}),
    )?;

    let mut saw_ping = false;
    let mut saw_text = false;
    for _ in 0..8 {
        match receive_frame(&mut socket.outgoing).await? {
            OutboundFrame::Ping => saw_ping = true,
            OutboundFrame::Text(_) => saw_text = true,
            OutboundFrame::Pong | OutboundFrame::Close => {}
        }
        if saw_ping && saw_text {
            break;
        }
    }
    anyhow::ensure!(
        saw_ping && saw_text,
        "fair writer scheduling did not advance both ping and text"
    );
    running.stop().await?;
    Ok(())
}

#[tokio::test]
async fn disconnect_cleans_up_and_root_closes_admission() -> anyhow::Result<()> {
    let agent = Arc::new(TestAgentStreams::default());
    let (events_tx, events) = mpsc::channel(1);
    agent.insert("active-request", events)?;
    let running =
        RunningSupervisor::start(fast_config(), agent, Arc::new(EmptyWorkerStatusProvider))?;
    let mut socket = TestSocket::recording();
    let reader_dropped = socket.reader_dropped.clone();
    let writer_dropped = socket.writer_dropped.clone();
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;
    send_text(
        &socket.incoming,
        json!({
            "action": "subscribe",
            "requestId": "active-request",
        }),
    )?;
    events_tx
        .send(event(json!({
            "type": "content_delta",
            "content": "active",
        }))?)
        .await?;
    let streamed = receive_json(&mut socket.outgoing).await?;
    anyhow::ensure!(streamed["content"] == "active");
    socket
        .incoming
        .send(InboundFrame::Close)
        .map_err(|_| anyhow::anyhow!("test WebSocket input closed"))?;
    wait_for_active(&running.handle, 0).await?;
    timeout(Duration::from_secs(1), events_tx.closed()).await?;
    anyhow::ensure!(reader_dropped.load(Ordering::SeqCst));
    anyhow::ensure!(writer_dropped.load(Ordering::SeqCst));

    let handle = running.handle.clone();
    running.stop().await?;
    anyhow::ensure!(!handle.is_accepting());
    let late = TestSocket::recording();
    anyhow::ensure!(handle.try_admit(late.transport) == Err(AdmissionError::Closed));
    Ok(())
}

#[tokio::test]
async fn request_unsubscribe_cancels_only_the_matching_stream_without_ack() -> anyhow::Result<()> {
    let provider = Arc::new(TestAgentStreams::default());
    let (events_tx, events) = mpsc::channel(1);
    provider.insert("request-unsubscribe", events)?;
    let running =
        RunningSupervisor::start(fast_config(), provider, Arc::new(EmptyWorkerStatusProvider))?;
    let mut socket = TestSocket::recording();
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;
    send_text(
        &socket.incoming,
        json!({
            "action": "subscribe",
            "requestId": "request-unsubscribe",
        }),
    )?;
    events_tx
        .send(event(json!({
            "type": "content_delta",
            "content": "before unsubscribe",
        }))?)
        .await?;
    anyhow::ensure!(receive_json(&mut socket.outgoing).await?["content"] == "before unsubscribe");

    send_text(
        &socket.incoming,
        json!({
            "action": "unsubscribe",
            "requestId": "different-request",
        }),
    )?;
    events_tx
        .send(event(json!({
            "type": "content_delta",
            "content": "still subscribed",
        }))?)
        .await?;
    anyhow::ensure!(receive_json(&mut socket.outgoing).await?["content"] == "still subscribed");

    send_text(
        &socket.incoming,
        json!({
            "action": "unsubscribe",
            "requestId": "request-unsubscribe",
        }),
    )?;
    timeout(Duration::from_secs(1), events_tx.closed()).await?;
    anyhow::ensure!(
        timeout(Duration::from_millis(20), socket.outgoing.recv())
            .await
            .is_err(),
        "request unsubscribe emitted an ack"
    );
    running.stop().await?;
    Ok(())
}

#[tokio::test]
async fn max_connections_rejects_excess_socket() -> anyhow::Result<()> {
    let config = WebSocketConfig {
        max_connections: 1,
        ..fast_config()
    };
    let running = RunningSupervisor::start(
        config,
        Arc::new(UnavailableAgentStreamProvider),
        Arc::new(EmptyWorkerStatusProvider),
    )?;
    let first = TestSocket::recording();
    running.handle.try_admit(first.transport)?;
    wait_for_active(&running.handle, 1).await?;

    let excess = TestSocket::recording();
    let excess_reader_dropped = excess.reader_dropped.clone();
    let excess_writer_dropped = excess.writer_dropped.clone();
    running.handle.try_admit(excess.transport)?;
    timeout(Duration::from_secs(1), async {
        while !excess_reader_dropped.load(Ordering::SeqCst)
            || !excess_writer_dropped.load(Ordering::SeqCst)
        {
            tokio::task::yield_now().await;
        }
    })
    .await?;
    anyhow::ensure!(running.handle.active_connections() == 1);

    running.stop().await?;
    Ok(())
}

#[tokio::test]
async fn shutdown_deadline_force_aborts_and_reaps_owned_tasks() -> anyhow::Result<()> {
    let config = WebSocketConfig {
        ping_period: Duration::from_millis(1),
        write_wait: Duration::from_secs(10),
        shutdown_wait: Duration::from_millis(20),
        ..fast_config()
    };
    let running = RunningSupervisor::start(
        config,
        Arc::new(UnavailableAgentStreamProvider),
        Arc::new(EmptyWorkerStatusProvider),
    )?;
    let socket = TestSocket::hanging_writer();
    let reader_dropped = socket.reader_dropped.clone();
    let writer_dropped = socket.writer_dropped.clone();
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;
    sleep(Duration::from_millis(5)).await;

    let handle = running.handle.clone();
    running.cancellation.cancel();
    let error = running
        .task
        .await?
        .err()
        .ok_or_else(|| anyhow::anyhow!("forced shutdown unexpectedly passed"))?;
    anyhow::ensure!(error.to_string().contains("shutdown deadline"));
    anyhow::ensure!(handle.active_connections() == 0);
    anyhow::ensure!(reader_dropped.load(Ordering::SeqCst));
    anyhow::ensure!(writer_dropped.load(Ordering::SeqCst));
    Ok(())
}

#[tokio::test]
async fn default_config_preserves_go_limits_and_timing() -> anyhow::Result<()> {
    let config = WebSocketConfig::default();
    anyhow::ensure!(config.max_connections == 1000);
    anyhow::ensure!(config.ping_period == Duration::from_secs(30));
    anyhow::ensure!(config.write_wait == Duration::from_secs(10));
    anyhow::ensure!(config.pong_wait == Duration::from_secs(60));
    Ok(())
}

fn constructor_error(config: WebSocketConfig) -> anyhow::Result<WebSocketConfigError> {
    WebSocketSupervisor::new(
        config,
        Arc::new(UnavailableAgentStreamProvider),
        Arc::new(EmptyWorkerStatusProvider),
    )
    .err()
    .ok_or_else(|| anyhow::anyhow!("invalid WebSocket config was accepted"))
}

#[test]
fn constructor_rejects_every_panicking_or_zero_lifecycle_value() -> anyhow::Result<()> {
    anyhow::ensure!(
        constructor_error(WebSocketConfig {
            max_connections: 0,
            ..fast_config()
        })? == WebSocketConfigError::MaxConnectionsZero
    );
    anyhow::ensure!(
        constructor_error(WebSocketConfig {
            max_connections: tokio::sync::Semaphore::MAX_PERMITS / 2 + 1,
            ..fast_config()
        })? == WebSocketConfigError::MaxConnectionsTooLarge
    );
    anyhow::ensure!(
        constructor_error(WebSocketConfig {
            ping_period: Duration::ZERO,
            ..fast_config()
        })? == WebSocketConfigError::PingPeriodZero
    );
    anyhow::ensure!(
        constructor_error(WebSocketConfig {
            pong_wait: Duration::ZERO,
            ..fast_config()
        })? == WebSocketConfigError::PongWaitZero
    );
    anyhow::ensure!(
        constructor_error(WebSocketConfig {
            write_wait: Duration::ZERO,
            ..fast_config()
        })? == WebSocketConfigError::WriteWaitZero
    );
    anyhow::ensure!(
        constructor_error(WebSocketConfig {
            shutdown_wait: Duration::ZERO,
            ..fast_config()
        })? == WebSocketConfigError::ShutdownWaitZero
    );
    anyhow::ensure!(
        constructor_error(WebSocketConfig {
            ping_period: Duration::MAX,
            ..fast_config()
        })? == WebSocketConfigError::PingPeriodUnrepresentable
    );
    anyhow::ensure!(
        constructor_error(WebSocketConfig {
            pong_wait: Duration::MAX,
            ..fast_config()
        })? == WebSocketConfigError::PongWaitUnrepresentable
    );
    anyhow::ensure!(
        constructor_error(WebSocketConfig {
            write_wait: Duration::MAX,
            ..fast_config()
        })? == WebSocketConfigError::WriteWaitUnrepresentable
    );
    anyhow::ensure!(
        constructor_error(WebSocketConfig {
            shutdown_wait: Duration::MAX,
            ..fast_config()
        })? == WebSocketConfigError::ShutdownWaitUnrepresentable
    );
    Ok(())
}

#[test]
fn websocket_router_records_stable_openapi_handshake() -> anyhow::Result<()> {
    let (_, document) =
        blog_backend::api::websocket::router::<WebSocketSupervisorHandle>().split_for_parts();
    let document = serde_json::to_value(document)?;
    let operation = &document["paths"]["/websocket"]["get"];
    anyhow::ensure!(operation["operationId"] == "connectWebSocket");
    anyhow::ensure!(operation["tags"] == json!(["agent"]));
    anyhow::ensure!(operation["responses"].get("101").is_some());
    anyhow::ensure!(operation["responses"].get("400").is_some());
    anyhow::ensure!(operation["responses"].get("426").is_some());
    Ok(())
}

#[tokio::test]
async fn root_cancellation_finishes_before_a_bounded_deadline() -> anyhow::Result<()> {
    let running = RunningSupervisor::start(
        fast_config(),
        Arc::new(UnavailableAgentStreamProvider),
        Arc::new(EmptyWorkerStatusProvider),
    )?;
    let socket = TestSocket::recording();
    running.handle.try_admit(socket.transport)?;
    wait_for_active(&running.handle, 1).await?;
    let started = Instant::now();
    running.stop().await?;
    anyhow::ensure!(started.elapsed() < Duration::from_secs(1));
    Ok(())
}
