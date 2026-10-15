use heph_server::config::Config;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    tracing_subscriber::fmt::init();
    let config = Config::from_env()?;
    let listener = tokio::net::TcpListener::bind(config.listen_addr).await?;
    tracing::info!(address = %listener.local_addr()?, "heph server listening");
    axum::serve(listener, heph_server::app())
        .with_graceful_shutdown(async {
            if let Err(error) = tokio::signal::ctrl_c().await {
                tracing::error!(%error, "failed to listen for shutdown signal");
            }
        })
        .await?;
    Ok(())
}
