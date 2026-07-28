use std::sync::Arc;

use async_trait::async_trait;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

use crate::{
    api::image::{ImageGenerationJob, ImageGenerationQueue},
    core::{article::ArticleService, image::ImageService, storage::ObjectStore},
    error::AppError,
    integrations::openai::{GeneratedImage, OpenAiClient},
};

const QUEUE_CAPACITY: usize = 32;
const PROVIDER: &str = "openai";
const MODEL: &str = "gpt-image-1";
const IMAGE_PROMPT_INSTRUCTIONS: &str = "Turn the supplied article or image idea into a vivid, concise image-generation prompt. Include subject, setting, composition, style, lighting, mood, and colors. Return only the prompt.";

#[derive(Clone)]
pub struct RuntimeImageQueue {
    jobs: mpsc::Sender<ImageGenerationJob>,
}

pub struct ImageQueueWorker {
    jobs: mpsc::Receiver<ImageGenerationJob>,
    service: Arc<ImageService>,
    articles: Arc<ArticleService>,
    openai: Arc<OpenAiClient>,
    object_store: Arc<dyn ObjectStore>,
    url_prefix: Arc<str>,
}

impl RuntimeImageQueue {
    pub fn new(
        service: Arc<ImageService>,
        articles: Arc<ArticleService>,
        openai: Arc<OpenAiClient>,
        object_store: Arc<dyn ObjectStore>,
        url_prefix: impl Into<Arc<str>>,
    ) -> (Arc<Self>, ImageQueueWorker) {
        let (jobs, receiver) = mpsc::channel(QUEUE_CAPACITY);
        (
            Arc::new(Self { jobs }),
            ImageQueueWorker {
                jobs: receiver,
                service,
                articles,
                openai,
                object_store,
                url_prefix: url_prefix.into(),
            },
        )
    }
}

#[async_trait]
impl ImageGenerationQueue for RuntimeImageQueue {
    fn provider(&self) -> &str {
        PROVIDER
    }

    fn model_name(&self) -> &str {
        MODEL
    }

    async fn enqueue(&self, job: ImageGenerationJob) -> Result<(), AppError> {
        self.jobs.send(job).await.map_err(|_| AppError::External)
    }
}

impl ImageQueueWorker {
    pub async fn run(mut self, cancellation: CancellationToken) -> anyhow::Result<()> {
        loop {
            tokio::select! {
                biased;
                () = cancellation.cancelled() => break,
                job = self.jobs.recv() => {
                    let Some(job) = job else { break };
                    self.process(job).await;
                }
            }
        }
        self.jobs.close();
        while let Some(job) = self.jobs.recv().await {
            let _ = self
                .service
                .mark_failed(job.image_id, "application is shutting down".to_owned())
                .await;
        }
        Ok(())
    }

    async fn process(&self, job: ImageGenerationJob) {
        let result = self.generate(&job).await;
        match result {
            Ok(output_url) => {
                if let Err(error) = self
                    .service
                    .mark_completed(job.image_id, output_url.clone(), None)
                    .await
                {
                    tracing::error!(image_id = %job.image_id, %error, "failed to persist completed image generation");
                    return;
                }
                if let Err(error) = self
                    .articles
                    .apply_generated_image(job.article_id, job.image_id, &output_url)
                    .await
                {
                    tracing::error!(
                        image_id = %job.image_id,
                        article_id = %job.article_id,
                        %error,
                        "failed to attach generated image to article draft"
                    );
                }
            }
            Err(error) => {
                let message = error.to_string();
                if let Err(persist_error) = self.service.mark_failed(job.image_id, message).await {
                    tracing::error!(image_id = %job.image_id, %persist_error, "failed to persist failed image generation");
                }
            }
        }
    }

    async fn generate(&self, job: &ImageGenerationJob) -> Result<String, AppError> {
        let prompt = if job.generate_prompt {
            self.openai
                .generate_text(IMAGE_PROMPT_INSTRUCTIONS, &job.prompt)
                .await?
        } else {
            job.prompt.clone()
        };
        match self.openai.generate_image(&prompt).await? {
            GeneratedImage::Url(url) => Ok(url),
            GeneratedImage::Bytes(bytes) => {
                let key = format!("images/{}.png", job.request_id);
                self.object_store.put(&key, bytes).await?;
                Ok(format!("{}/{key}", self.url_prefix.trim_end_matches('/')))
            }
        }
    }
}
