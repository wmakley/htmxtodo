use std::env;
use std::net::{IpAddr, SocketAddr};

use anyhow::{anyhow, Context};
use axum::extract::Path;
use axum::http::StatusCode;
use axum::response::{IntoResponse, Redirect, Response};
use axum::Form;
use axum::{
    extract::{FromRef, State},
    routing::{delete, get},
    Router,
};
use axum_template::{engine::Engine, RenderHtml};
use chrono;
use dotenvy::dotenv;
use handlebars::Handlebars;
use serde::{Deserialize, Serialize};
use serde_json::json;
use sqlx::pool::Pool;
use sqlx::postgres::{PgPoolOptions, Postgres};
use tower_http::trace::TraceLayer;
use tracing::debug;

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
    let host: IpAddr = match env::var("HOST") {
        Ok(host) => {
            let parts = host.split(".");
            let mut octets: [u8; 4] = [0, 0, 0, 0];
            for (i, part) in parts.enumerate() {
                if i > 3 {
                    return Err(anyhow!("could not parse HOST"));
                }
                octets[i] = part.parse::<u8>().context("could not parse HOST")?;
            }
            IpAddr::from(octets)
        }
        Err(_) => IpAddr::from([0, 0, 0, 0]),
    };
    let port: u16 = match env::var("PORT") {
        Ok(port) => port.parse::<u16>().context("could not parse PORT")?,
        Err(_) => 8080,
    };

    tracing_subscriber::fmt::init();

    let pool = PgPoolOptions::new()
        .max_connections(5)
        .connect(&database_url)
        .await
        .context("could not connect to database_url")?;

    let mut hbs = Handlebars::new();
    hbs.register_template_file("layouts/main", "./views/layouts/main.hbs")
        .context("could not register layouts/main template")?;
    // hbs.register_template_file("shared/_head", "./views/shared/_head.hbs")
    //     .context("could not register shared/head template")?;
    hbs.register_template_file("lists/index", "./views/lists/index.hbs")
        .context("could not register lists/index template")?;
    hbs.register_template_file("lists/_form", "./views/lists/_form.hbs")
        .context("could not register lists/_form template")?;
    hbs.register_template_file("lists/_card", "./views/lists/_card.hbs")
        .context("could not register lists/_card template")?;

    let shared_state = AppState {
        template_engine: Engine::from(hbs),
        database_pool: pool,
    };

    let app = Router::new()
        .route("/", get(|| async { Redirect::temporary("/lists") }))
        .route("/lists", get(lists_index).post(create_list))
        .route("/lists/:id", delete(delete_list).patch(update_list))
        .route("/lists/:id/edit", get(edit_list))
        .with_state(shared_state)
        .layer(TraceLayer::new_for_http());

    let addr = SocketAddr::from((host, port));
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
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

async fn lists_index(State(state): State<AppState>) -> impl IntoResponse {
    let sql = "SELECT * FROM list ORDER BY id".to_string();

    let lists = sqlx::query_as::<_, List>(&sql)
        .fetch_all(&state.database_pool)
        .await
        .unwrap();

    let new_list = List {
        id: 0,
        name: "".to_string(),
        created_at: chrono::Utc::now(),
        updated_at: chrono::Utc::now(),
    };

    RenderHtml(
        "lists/index".to_string(),
        state.template_engine,
        json!({
            "lists": lists,
            "new_list": new_list,
        }),
    )
}

#[derive(Debug, Deserialize)]
pub struct CreateListForm {
    pub name: String,
}

async fn create_list(
    State(state): State<AppState>,
    Form(form): Form<CreateListForm>,
) -> impl IntoResponse {
    debug!("create_list form: {:?}", form);

    let name = form.name.trim();
    if name.is_empty() {
        // TODO
        return (StatusCode::BAD_REQUEST, "name cannot be empty").into_response();
    }

    let sql = "INSERT INTO list (name) VALUES ($1) RETURNING *".to_string();

    let list = sqlx::query_as::<_, List>(&sql)
        .bind(name)
        .fetch_one(&state.database_pool)
        .await
        .unwrap();

    RenderHtml(
        "lists/_card".to_string(),
        state.template_engine,
        json!({
            "list": list,
            "editing_name": false,
        }),
    )
    .into_response()
}

async fn delete_list(State(state): State<AppState>, Path(id): Path<i64>) -> impl IntoResponse {
    debug!("delete_list id: {:?}", id);

    let sql = "DELETE FROM list WHERE id = $1".to_string();

    sqlx::query(&sql)
        .bind(id)
        .execute(&state.database_pool)
        .await
        .unwrap();

    (StatusCode::NO_CONTENT, "")
}

// Make our own error that wraps `anyhow::Error`.
struct AppError(anyhow::Error);

// Tell axum how to convert `AppError` into a response.
impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        (
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("Something went wrong: {}", self.0),
        )
            .into_response()
    }
}

// This enables using `?` on functions that return `Result<_, anyhow::Error>` to turn them into
// `Result<_, AppError>`. That way you don't need to do that manually.
impl<E> From<E> for AppError
where
    E: Into<anyhow::Error>,
{
    fn from(err: E) -> Self {
        Self(err.into())
    }
}

#[axum::debug_handler]
async fn edit_list(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<Response, AppError> {
    debug!("edit_list id: {:?}", id);

    let sql = "SELECT * FROM list WHERE id = $1".to_string();

    let list = sqlx::query_as::<_, List>(&sql)
        .bind(id)
        .fetch_one(&state.database_pool)
        .await?;

    Ok(RenderHtml(
        "lists/_card".to_string(),
        state.template_engine,
        json!({
            "list": list,
            "editing_name": true,
        }),
    )
    .into_response())
}

#[derive(Debug, Deserialize)]
pub struct UpdateListForm {
    pub name: String,
}

async fn update_list(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    Form(form): Form<UpdateListForm>,
) -> impl IntoResponse {
    debug!("update_list id: {:?}", id);

    let name = form.name.trim();
    if name.is_empty() {
        // TODO
        return (StatusCode::BAD_REQUEST, "name cannot be empty").into_response();
    }

    let mut transaction = state.database_pool.begin().await.unwrap();

    let sql = "SELECT * FROM list WHERE id = $1".to_string();

    let list = sqlx::query_as::<_, List>(&sql)
        .bind(id)
        .fetch_one(&mut *transaction)
        .await
        .unwrap();

    let sql = "UPDATE list SET name = $1, updated_at = NOW() WHERE id = $2 RETURNING *".to_string();

    let list = sqlx::query_as::<_, List>(&sql)
        .bind(name)
        .bind(id)
        .fetch_one(&mut *transaction)
        .await
        .unwrap();

    transaction.commit().await.unwrap();

    RenderHtml(
        "lists/_card".to_string(),
        state.template_engine,
        json!({
            "list": list,
            "editing_name": false,
        }),
    )
    .into_response()
}
