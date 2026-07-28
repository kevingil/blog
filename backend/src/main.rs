use blog_backend::{bootstrap, config::Config, telemetry};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    dotenvy::dotenv().ok();
    telemetry::init()?;
    bootstrap::build(Config::from_env()?).await?.serve().await
}
