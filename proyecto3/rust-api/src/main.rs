use axum::{extract::State, http::StatusCode, routing::{get, post}, Json, Router};
use serde::{Deserialize, Serialize};
use std::{env, sync::Arc};

#[derive(Deserialize, Serialize, Debug)]
struct WarReport {
    country: String,
    warplanes_in_air: i32,
    warships_in_water: i32,
    timestamp: String,
}

struct AppState {
    client: reqwest::Client,
    go_url: String,
}

const VALID: [&str; 5] = ["USA", "RUS", "CHN", "ESP", "GTM"];

async fn health() -> &'static str {
    "ok"
}

async fn report(
    State(st): State<Arc<AppState>>,
    Json(r): Json<WarReport>,
) -> Result<Json<serde_json::Value>, (StatusCode, String)> {
    if !VALID.contains(&r.country.to_uppercase().as_str()) {
        return Err((StatusCode::BAD_REQUEST, "pais invalido".into()));
    }
    let resp = st
        .client
        .post(&st.go_url)
        .json(&r)
        .send()
        .await
        .map_err(|e| (StatusCode::BAD_GATEWAY, e.to_string()))?;
    let code = resp.status();
    if !code.is_success() {
        return Err((StatusCode::BAD_GATEWAY, format!("go-client respondio {}", code)));
    }
    Ok(Json(serde_json::json!({"status": "ok"})))
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt::init();
    let go_url = env::var("GO_CLIENT_URL")
        .unwrap_or_else(|_| "http://go-client:8081/report".into());
    let st = Arc::new(AppState { client: reqwest::Client::new(), go_url });

    let app = Router::new()
        .route("/health", get(health))
        .route("/report", post(report))
        .with_state(st);

    let port = env::var("PORT").unwrap_or_else(|_| "8080".into());
    let listener = tokio::net::TcpListener::bind(format!("0.0.0.0:{}", port)).await.unwrap();
    tracing::info!("rust-api escuchando en :{}", port);
    axum::serve(listener, app).await.unwrap();
}
