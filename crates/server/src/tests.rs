use axum::{body::Body, http::Request};
use http_body_util::BodyExt;
use tower::ServiceExt;

#[tokio::test]
async fn system_info_over_connect_json() -> Result<(), Box<dyn std::error::Error>> {
    let request = Request::post("/heph.system.v1.SystemService/GetSystemInfo")
        .header("content-type", "application/json")
        .header("connect-protocol-version", "1")
        .body(Body::from("{}"))?;
    let response = super::app().oneshot(request).await?;
    assert_eq!(response.status(), 200);
    let body = response.into_body().collect().await?.to_bytes();
    let info: serde_json::Value = serde_json::from_slice(&body)?;
    assert_eq!(info["name"], "heph-v2");
    assert_eq!(info["version"], env!("CARGO_PKG_VERSION"));
    Ok(())
}

#[tokio::test]
async fn unknown_rpc_returns_not_found() -> Result<(), Box<dyn std::error::Error>> {
    let response = super::app()
        .oneshot(
            Request::post("/heph.system.v1.SystemService/Missing")
                .header("content-type", "application/json")
                .header("connect-protocol-version", "1")
                .body(Body::from("{}"))?,
        )
        .await?;
    assert_eq!(response.status(), 404);
    Ok(())
}
