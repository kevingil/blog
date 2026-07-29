use std::{net::SocketAddr, panic::AssertUnwindSafe, sync::Arc};

use diesel_async::RunQueryDsl;
use futures_util::FutureExt;
use secrecy::ExposeSecret;
use tokio::{
    net::TcpListener,
    task::JoinSet,
    time::{Duration, timeout},
};
use tokio_util::sync::CancellationToken;

use crate::{
    api::{
        agent::AgentState,
        article::ArticleState,
        auth::AuthState,
        datasource::DataSourceState,
        image::ImageState,
        insight::InsightState,
        organization::OrganizationState,
        page::PageState,
        profile::ProfileState,
        project::ProjectState,
        source::SourceState,
        storage::StorageState,
        taskrun::TaskRunState,
        websocket::{WebSocketConfig, WebSocketSupervisor},
        worker::{WorkerState, WorkerStatusAdapter},
    },
    app::{self, AppDependencies, AppState},
    config::Config,
    constants::BACKGROUND_SHUTDOWN_TIMEOUT,
    core::{
        article::{ArticleRepository, ArticleService},
        auth::AuthService,
        chat::ChatMessageService,
        copilot::{ArticleDraftAdapter, CopilotConfig, CopilotManager},
        datasource::{DataSourceService, RecommendationService},
        image::ImageService,
        insight::InsightService,
        ml::{
            TextGenerationService,
            llm::{
                Agent, AskQuestionTool, GenerateImagePromptTool, GetRelevantSourcesTool,
                InMemorySessionStore, Model, ModelProvider, ReadDocumentTool, ReplaceLinesTool,
                SearchWebSourcesTool, SelectSourcesForEditTool, SessionStore, Tool,
            },
        },
        organization::OrganizationService,
        page::PageService,
        profile::ProfileService,
        project::ProjectService,
        source::SourceService,
        storage::StorageService,
        taskrun::TaskRunService,
        worker::{
            ContentCrawler, CrawlWorker, DiscoveryWorker, InsightWorker, ManagerConfig,
            PipelineWorker, StatusService, SystemClock, WorkerManager,
        },
    },
    database::pool::create_pool,
    database::repository::{
        account::DieselAccountRepository, article::DieselArticleRepository,
        chat_message::DieselChatMessageRepository,
        content_topic_match::DieselContentTopicMatchRepository,
        crawled_content::DieselCrawledContentRepository, data_source::DieselDataSourceRepository,
        image::DieselImageRepository, insight::DieselInsightRepository,
        insight_topic::DieselInsightTopicRepository, organization::DieselOrganizationRepository,
        page::DieselPageRepository, project::DieselProjectRepository,
        site_settings::DieselSiteSettingsRepository, source::DieselSourceRepository,
        tag::DieselTagRepository, task_run::DieselTaskRunRepository,
        user_insight_status::DieselUserInsightStatusRepository,
    },
    integrations::{
        exa::ExaClient, fetch::HttpFetchExtract, llm::GroqClient, openai::OpenAiClient,
        s3::S3ObjectStore,
    },
    runtime::{
        AgentQueueWorker, CombinedAgentStreamProvider, CopilotRuntime, INSIGHT_INSTRUCTIONS,
        ImageQueueWorker, RuntimeAgentQueue, RuntimeImageQueue, RuntimeInsightGenerator,
    },
    server,
};

pub struct Application {
    address: SocketAddr,
    router: axum::Router,
    cancellation: CancellationToken,
    tasks: JoinSet<anyhow::Result<()>>,
    articles: Arc<DieselArticleRepository>,
    agent_worker: Option<AgentQueueWorker>,
    image_worker: Option<ImageQueueWorker>,
    copilot: Arc<CopilotRuntime>,
    worker_manager: Arc<WorkerManager>,
    websocket: WebSocketSupervisor,
}

pub async fn build(config: Config) -> anyhow::Result<Application> {
    let pool = create_pool(&config.database_url)?;
    let mut connection = pool
        .get()
        .await
        .map_err(|error| anyhow::anyhow!("failed to connect to PostgreSQL: {error}"))?;
    diesel::sql_query("SELECT 1")
        .execute(&mut connection)
        .await
        .map_err(|error| anyhow::anyhow!("PostgreSQL startup check failed: {error}"))?;
    drop(connection);
    let cancellation = CancellationToken::new();
    let accounts = Arc::new(DieselAccountRepository::new(pool.clone()));
    let auth = AuthState::new(Arc::new(AuthService::new(
        accounts.clone(),
        config.jwt_secret.expose_secret(),
    )?));
    let articles = Arc::new(DieselArticleRepository::new(pool.clone()));
    let tags = Arc::new(DieselTagRepository::new(pool.clone()));
    let openai = Arc::new(OpenAiClient::with_base_url(
        config.openai_api_key.expose_secret(),
        config.openai_base_url.clone(),
    )?);
    let groq = Arc::new(GroqClient::with_base_url(
        config.groq_api_key.expose_secret(),
        config.groq_base_url.clone(),
        Model::new(
            "openai/gpt-oss-120b",
            ModelProvider::GROQ,
            "openai/gpt-oss-120b",
            2_000,
            true,
            true,
        ),
        INSIGHT_INSTRUCTIONS,
        Some("medium".to_owned()),
    )?);
    let exa = Arc::new(ExaClient::with_base_url(
        config.exa_api_key.expose_secret(),
        config.exa_base_url.clone(),
    )?);
    let object_store = Arc::new(
        S3ObjectStore::for_s3_compatible(
            config.s3_endpoint.clone(),
            config.s3_access_key_id.expose_secret(),
            config.s3_secret_access_key.expose_secret(),
            config.s3_bucket.clone(),
        )
        .await?,
    );

    let chat = Arc::new(ChatMessageService::new(
        Arc::new(DieselChatMessageRepository::new(pool.clone())),
        cancellation.child_token(),
    ));
    let task_runs = Arc::new(TaskRunService::new(
        Arc::new(DieselTaskRunRepository::new(pool.clone())),
        cancellation.child_token(),
    ));
    let article_service = Arc::new(
        ArticleService::new(articles.clone(), accounts.clone(), tags.clone())
            .with_embedding_provider(openai.clone())
            .with_context_writer(openai.clone()),
    );
    let (article_generation_queue, agent_worker) =
        RuntimeAgentQueue::new(chat.clone(), article_service.clone(), openai.clone());
    let article = ArticleState::new(article_service.clone(), article_generation_queue.clone());

    let data_sources_repository = Arc::new(DieselDataSourceRepository::new(pool.clone()));
    let crawled_content = Arc::new(DieselCrawledContentRepository::new(pool.clone()));
    let fetch = Arc::new(HttpFetchExtract::new()?);
    let data_sources = Arc::new(DataSourceService::new(
        data_sources_repository.clone(),
        crawled_content.clone(),
    ));
    let recommendations = Arc::new(RecommendationService::new(
        data_sources_repository.clone(),
        exa.clone(),
    ));
    let datasource = DataSourceState::new(data_sources.clone(), recommendations, accounts.clone());

    let image_service = Arc::new(ImageService::new(Arc::new(DieselImageRepository::new(
        pool.clone(),
    ))));
    let (image_queue, image_worker) = RuntimeImageQueue::new(
        image_service.clone(),
        article_service,
        openai.clone(),
        object_store.clone(),
        config.s3_url_prefix.clone(),
    );
    let image = ImageState::new(image_service, image_queue);

    let topics = Arc::new(DieselInsightTopicRepository::new(pool.clone()));
    let topic_matches = Arc::new(DieselContentTopicMatchRepository::new(pool.clone()));
    let insight_service = Arc::new(InsightService::new(
        Arc::new(DieselInsightRepository::new(pool.clone())),
        topics.clone(),
        Arc::new(DieselUserInsightStatusRepository::new(pool.clone())),
        crawled_content.clone(),
        topic_matches.clone(),
        openai.clone(),
    ));
    let insight = InsightState::new(insight_service.clone(), accounts.clone());
    let organizations = Arc::new(DieselOrganizationRepository::new(pool.clone()));
    let organization = OrganizationState::new(Arc::new(OrganizationService::new(
        organizations.clone(),
        organizations.clone(),
    )));
    let page = PageState::new(Arc::new(PageService::new(Arc::new(
        DieselPageRepository::new(pool.clone()),
    ))));
    let site_settings = Arc::new(DieselSiteSettingsRepository::new(pool.clone()));
    let profile = ProfileState::new(Arc::new(ProfileService::new(
        site_settings.clone(),
        site_settings.clone(),
        site_settings,
        organizations,
    )));
    let project = ProjectState::new(Arc::new(ProjectService::new(
        Arc::new(DieselProjectRepository::new(pool.clone())),
        tags,
    )));
    let source_service = Arc::new(SourceService::new(
        Arc::new(DieselSourceRepository::new(pool.clone())),
        articles.clone(),
        openai.clone(),
        fetch.clone(),
    ));
    let source = SourceState::new(source_service.clone());
    let storage = StorageState::new(Arc::new(StorageService::new(
        object_store,
        config.s3_url_prefix,
        cancellation.child_token(),
    )));
    let taskrun = TaskRunState::new(task_runs.clone());

    let clock = Arc::new(SystemClock);
    let worker_status = Arc::new(StatusService::new(clock.clone()));
    let worker_manager = WorkerManager::new(
        worker_status.clone(),
        Some(task_runs),
        cancellation.child_token(),
        ManagerConfig::default(),
    )
    .map_err(|error| anyhow::anyhow!("invalid worker manager configuration: {error}"))?;
    worker_manager.register(Arc::new(CrawlWorker::new(
        worker_status.clone(),
        data_sources.clone(),
        Some(Arc::new(ContentCrawler::new(
            exa.clone(),
            fetch,
            data_sources_repository,
            crawled_content,
            openai.clone(),
            insight_service.clone(),
        ))),
    )));
    worker_manager.register(Arc::new(DiscoveryWorker::new(
        worker_status.clone(),
        data_sources,
        Some(exa.clone()),
    )));
    let insight_generator = groq.is_configured().then(|| {
        Arc::new(RuntimeInsightGenerator::new(
            topics,
            topic_matches,
            Arc::new(DieselCrawledContentRepository::new(pool.clone())),
            groq,
            insight_service.clone(),
            clock,
        )) as Arc<dyn crate::core::worker::InsightGenerationPort>
    });
    worker_manager.register(Arc::new(
        InsightWorker::new(worker_status.clone(), insight_generator)
            .map_err(|error| anyhow::anyhow!("invalid insight worker: {error}"))?,
    ));
    worker_manager.register(Arc::new(PipelineWorker::new(
        WorkerManager::downgrade(&worker_manager),
        worker_status.clone(),
    )));
    worker_manager
        .start()
        .map_err(|error| anyhow::anyhow!("failed to start worker manager: {error}"))?;
    let worker = WorkerState::new(worker_manager.clone(), worker_status.clone());

    let drafts = Arc::new(ArticleDraftAdapter::new(articles.clone()));
    let session_store: Arc<dyn SessionStore> = Arc::new(InMemorySessionStore::default());
    let text_generation = Arc::new(TextGenerationService::new(openai.clone()));
    let tools: Vec<Arc<dyn Tool>> = vec![
        Arc::new(ReadDocumentTool),
        Arc::new(ReplaceLinesTool::new(Some(drafts.clone()))),
        Arc::new(GenerateImagePromptTool::new(text_generation)),
        Arc::new(AskQuestionTool::new(exa.clone())),
        Arc::new(SearchWebSourcesTool::new(
            exa.clone(),
            source_service.clone(),
        )),
        Arc::new(GetRelevantSourcesTool::new(source_service.clone())),
        Arc::new(SelectSourcesForEditTool::new(source_service.clone())),
    ];
    let copilot_manager = CopilotManager::new(
        Agent::new(openai, session_store.clone(), tools),
        session_store,
        chat.clone(),
        Some(source_service),
        Some(drafts),
        CopilotConfig::from_env()
            .map_err(|error| anyhow::anyhow!("invalid copilot configuration: {error}"))?,
        cancellation.child_token(),
    );
    let copilot = CopilotRuntime::new(copilot_manager);
    let agent_streams =
        CombinedAgentStreamProvider::new(copilot.clone(), article_generation_queue.clone());
    let websocket_config = WebSocketConfig {
        shutdown_wait: BACKGROUND_SHUTDOWN_TIMEOUT.saturating_sub(Duration::from_secs(1)),
        ..WebSocketConfig::default()
    };
    let (websocket_handle, websocket) = WebSocketSupervisor::new(
        websocket_config,
        agent_streams,
        Arc::new(WorkerStatusAdapter::new(worker_status)),
    )
    .map_err(|error| anyhow::anyhow!("invalid WebSocket configuration: {error}"))?;
    let state = AppState::new(
        pool,
        auth,
        websocket_handle,
        AppDependencies {
            agent: AgentState::new(chat, copilot.clone()),
            article,
            datasource,
            image,
            insight,
            organization,
            page,
            profile,
            project,
            source,
            storage,
            taskrun,
            worker,
        },
    );

    Ok(Application {
        address: SocketAddr::new(config.host, config.port),
        router: app::router(state, &config.cors_origins)?,
        cancellation,
        tasks: JoinSet::new(),
        articles,
        agent_worker: Some(agent_worker),
        image_worker: Some(image_worker),
        copilot,
        worker_manager,
        websocket,
    })
}

impl Application {
    pub async fn serve(mut self) -> anyhow::Result<()> {
        let listener = TcpListener::bind(self.address).await?;
        let cancellation = self.cancellation.clone();
        tracing::info!(address = %self.address, "server listening");

        let signal_cancellation = cancellation.clone();
        self.tasks.spawn(async move {
            shutdown_signal(signal_cancellation).await;
            Ok(())
        });
        let agent_worker = self
            .agent_worker
            .take()
            .ok_or_else(|| anyhow::anyhow!("agent queue worker missing"))?;
        let agent_cancellation = cancellation.child_token();
        self.tasks
            .spawn(async move { agent_worker.run(agent_cancellation).await });
        let image_worker = self
            .image_worker
            .take()
            .ok_or_else(|| anyhow::anyhow!("image queue worker missing"))?;
        let image_cancellation = cancellation.child_token();
        self.tasks
            .spawn(async move { image_worker.run(image_cancellation).await });
        let websocket_cancellation = cancellation.child_token();
        let websocket_failure_cancellation = cancellation.clone();
        let websocket = self.websocket;
        self.tasks.spawn(async move {
            match AssertUnwindSafe(websocket.run(websocket_cancellation))
                .catch_unwind()
                .await
            {
                Ok(Ok(())) if websocket_failure_cancellation.is_cancelled() => Ok(()),
                Ok(Ok(())) => {
                    websocket_failure_cancellation.cancel();
                    Err(anyhow::anyhow!("WebSocket supervisor stopped unexpectedly"))
                }
                Ok(Err(error)) => {
                    websocket_failure_cancellation.cancel();
                    Err(anyhow::anyhow!("WebSocket supervisor failed: {error}"))
                }
                Err(_) => {
                    websocket_failure_cancellation.cancel();
                    Err(anyhow::anyhow!("WebSocket supervisor panicked"))
                }
            }
        });
        let server_result = server::serve(
            listener,
            self.router,
            cancellation.clone(),
            BACKGROUND_SHUTDOWN_TIMEOUT,
        )
        .await;

        cancellation.cancel();
        let copilot_result = self.copilot.shutdown(BACKGROUND_SHUTDOWN_TIMEOUT).await;
        let worker_result = self.worker_manager.shutdown().await;
        let task_errors =
            drain_application_tasks(&mut self.tasks, BACKGROUND_SHUTDOWN_TIMEOUT).await;
        let article_result = self
            .articles
            .shutdown_background_tasks(BACKGROUND_SHUTDOWN_TIMEOUT)
            .await;

        let mut errors = Vec::new();
        if let Err(error) = server_result {
            errors.push(error.to_string());
        }
        if let Err(error) = article_result {
            errors.push(format!("article repository shutdown failed: {error}"));
        }
        if let Err(error) = worker_result {
            errors.push(format!(
                "worker manager shutdown failed: timed_out={}, failures={}",
                error.timed_out,
                error.task_failures.join(", ")
            ));
        }
        if let Err(error) = copilot_result {
            errors.push(format!("copilot shutdown failed: {error}"));
        }
        errors.extend(task_errors);
        if errors.is_empty() {
            Ok(())
        } else {
            Err(anyhow::anyhow!(errors.join("; ")))
        }
    }
}

async fn drain_application_tasks(
    tasks: &mut JoinSet<anyhow::Result<()>>,
    shutdown_timeout: Duration,
) -> Vec<String> {
    let drain = async {
        let mut errors = Vec::new();
        while let Some(result) = tasks.join_next().await {
            match result {
                Ok(Ok(())) => {}
                Ok(Err(error)) => errors.push(format!("application task failed: {error}")),
                Err(error) => errors.push(format!("application task join failed: {error}")),
            }
        }
        errors
    };

    match timeout(shutdown_timeout, drain).await {
        Ok(errors) => errors,
        Err(_) => {
            tasks.abort_all();
            let abort_wait = async { while tasks.join_next().await.is_some() {} };
            let mut errors = vec![format!(
                "application tasks exceeded the {} second shutdown deadline",
                shutdown_timeout.as_secs()
            )];
            if timeout(Duration::from_secs(1), abort_wait).await.is_err() {
                errors.push("application tasks did not stop after cancellation".to_owned());
            }
            errors
        }
    }
}

async fn shutdown_signal(cancellation: CancellationToken) {
    #[cfg(unix)]
    {
        use tokio::signal::unix::{SignalKind, signal};

        let mut terminate = match signal(SignalKind::terminate()) {
            Ok(signal) => signal,
            Err(error) => {
                tracing::error!(%error, "failed to register SIGTERM handler");
                cancellation.cancel();
                return;
            }
        };
        tokio::select! {
            result = tokio::signal::ctrl_c() => {
                if let Err(error) = result {
                    tracing::error!(%error, "failed to receive ctrl-c");
                }
            }
            _ = terminate.recv() => {}
            _ = cancellation.cancelled() => {}
        }
    }

    #[cfg(not(unix))]
    {
        let _ = tokio::signal::ctrl_c().await;
    }

    cancellation.cancel();
}
