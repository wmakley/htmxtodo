use std::net::SocketAddr;

use anyhow::Context;
use axum::http::StatusCode;
use axum::response::{IntoResponse, Redirect};
use axum::{
    extract::{FromRef, State},
    routing::get,
    Json, Router,
};
use axum_template::engine::Engine;
use dotenvy::dotenv;
use handlebars::Handlebars;
use serde::{Deserialize, Serialize};
use sqlx::pool::Pool;
use sqlx::postgres::{PgPoolOptions, Postgres};
use std::env;
use tower_http::trace::TraceLayer;

type TemplateEngine = Engine<Handlebars<'static>>;
type DatabasePool = Pool<Postgres>;

#[derive(Clone)]
struct AppState {
    template_engine: TemplateEngine,
    database_pool: DatabasePool,
}

impl FromRef<AppState> for DatabasePool {
    fn from_ref(app_state: &AppState) -> DatabasePool {
        app_state.database_pool.clone()
    }
}

impl FromRef<AppState> for TemplateEngine {
    fn from_ref(app_state: &AppState) -> TemplateEngine {
        app_state.template_engine.clone()
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    match dotenv() {
        Ok(_) => println!("Found .env file"),
        Err(_) => println!("No .env file found"),
    }
    let database_url = env::var("DATABASE_URL").expect("DATABASE_URL not found in environment");
    let port: u16 = match env::var("PORT") {
        Ok(port) => port.parse::<u16>().context("could not parse PORT")?,
        Err(_) => 8000,
    };

    tracing_subscriber::fmt::init();

    let pool = PgPoolOptions::new()
        .max_connections(5)
        .connect(&database_url)
        .await
        .context("could not connect to database_url")?;

    let mut hbs = Handlebars::new();
    hbs.register_template_file("lists/index", "./views/lists/index.hbs")
        .context("could not register lists/index template")?;

    let shared_state = AppState {
        template_engine: Engine::from(hbs),
        database_pool: pool,
    };

    let app = Router::new()
        .route("/", get(|| async { Redirect::temporary("/lists") }))
        .route("/lists", get(lists_index))
        .with_state(shared_state)
        .layer(TraceLayer::new_for_http());

    let addr = SocketAddr::from(([0, 0, 0, 0], port));
    println!("listening on {}", addr);

    axum::Server::bind(&addr)
        .serve(app.into_make_service())
        .await?;
    Ok(())
}

#[derive(sqlx::FromRow, Deserialize, Serialize)]
pub struct List {
    pub id: i64,
    pub name: String,
}

async fn lists_index(State(database_pool): State<DatabasePool>) -> impl IntoResponse {
    let sql = "SELECT id, name FROM list ORDER BY id".to_string();

    let task = sqlx::query_as::<_, List>(&sql)
        .fetch_all(&database_pool)
        .await
        .unwrap();

    (StatusCode::OK, Json(task))
}
