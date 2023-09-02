use std::env;
use std::net::{IpAddr, SocketAddr};

use anyhow::{anyhow, Context};
use axum::extract::Path;
use axum::http::StatusCode;
use axum::response::{IntoResponse, Redirect, Response};
use axum::Form;
use axum::{
    error_handling::HandleErrorLayer,
    extract::{FromRef, State},
    routing::{delete, get},
    Router,
};
use axum_template::{engine::Engine, RenderHtml};
use chrono;
use dotenvy::dotenv;
use handlebars::Handlebars;
use serde::Deserialize;
use serde_json::{json, Value};
use sqlx::postgres::PgPoolOptions;
use std::time::Duration;
use tower::ServiceBuilder;
use tower_http::trace::TraceLayer;
use tracing::debug;

mod db;
mod error;

use error::AppError;

type TemplateEngine = Engine<Handlebars<'static>>;
type Page = RenderHtml<String, TemplateEngine, Value>;

#[derive(Clone)]
struct AppState {
    template_engine: TemplateEngine,
    repo: db::Repo,
}

impl FromRef<AppState> for db::Repo {
    fn from_ref(app_state: &AppState) -> db::Repo {
        app_state.repo.clone()
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
        repo: db::Repo::new(pool),
    };

    let app = Router::new()
        .route("/", get(|| async { Redirect::temporary("/lists") }))
        .route("/lists", get(lists_index).post(create_list))
        .route("/lists/:id", delete(delete_list).patch(update_list))
        .route("/lists/:id/edit", get(edit_list))
        .with_state(shared_state)
        .layer(TraceLayer::new_for_http());
    // .layer(
    //     ServiceBuilder::new()
    //         // `timeout` will produce an error if the handler takes
    //         // too long so we must handle those
    //         .layer(HandleErrorLayer::new(handle_timeout_error))
    //         .timeout(Duration::from_secs(30)),
    // );

    let addr = SocketAddr::from((host, port));
    println!("listening on {}", addr);

    axum::Server::bind(&addr)
        .serve(app.into_make_service())
        .await?;
    Ok(())
}

async fn handle_timeout_error(err: axum::BoxError) -> (StatusCode, String) {
    if err.is::<tower::timeout::error::Elapsed>() {
        (
            StatusCode::REQUEST_TIMEOUT,
            "Request took too long".to_string(),
        )
    } else {
        (
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("Unhandled internal error: {}", err),
        )
    }
}

async fn lists_index(State(state): State<AppState>) -> Result<Page, AppError> {
    let lists = state.repo.filter_lists().await?;

    let new_list = db::List {
        id: 0,
        name: "".to_string(),
        created_at: chrono::Utc::now(),
        updated_at: chrono::Utc::now(),
    };

    Ok(RenderHtml(
        "lists/index".to_string(),
        state.template_engine,
        json!({
            "lists": lists,
            "new_list": new_list,
        }),
    ))
}

#[derive(Debug, Deserialize)]
pub struct CreateListForm {
    pub name: String,
}

async fn create_list(
    State(state): State<AppState>,
    Form(form): Form<CreateListForm>,
) -> Result<(StatusCode, Page), AppError> {
    debug!("create_list form: {:?}", form);

    let name = form.name.trim();
    if name.is_empty() {
        return Ok((
            StatusCode::BAD_REQUEST,
            RenderHtml(
                "lists/_form".to_string(),
                state.template_engine,
                json!({
                    "list": db::List {
                        id: 0,
                        name: name.to_string(),
                        created_at: chrono::Utc::now(),
                        updated_at: chrono::Utc::now(),
                    },
                    "errors": vec!["name cannot be empty".to_string()],
                }),
            ),
        ));
    }

    let list = state.repo.create_list(name).await?;

    Ok((
        StatusCode::OK,
        RenderHtml(
            "lists/_card".to_string(),
            state.template_engine,
            json!({
                "list": list,
                "editing_name": false,
            }),
        ),
    ))
}

async fn delete_list(
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<StatusCode, AppError> {
    debug!("delete_list id: {:?}", id);

    state.repo.delete_list(id).await?;

    Ok(StatusCode::NO_CONTENT)
}

async fn edit_list(State(state): State<AppState>, Path(id): Path<i64>) -> Result<Page, AppError> {
    debug!("edit_list id: {:?}", id);

    let list = state.repo.get_list(id).await?;

    Ok(RenderHtml(
        "lists/_card".to_string(),
        state.template_engine,
        json!({
            "list": list,
            "editing_name": true,
        }),
    ))
}

#[derive(Debug, Deserialize)]
pub struct UpdateListForm {
    pub name: String,
}

async fn update_list(
    State(state): State<AppState>,
    Path(id): Path<i64>,
    Form(form): Form<UpdateListForm>,
) -> Result<Response, AppError> {
    debug!("update_list id: {:?}", id);

    let name = form.name.trim();
    if name.is_empty() {
        // TODO
        return Ok((StatusCode::BAD_REQUEST, "name cannot be empty").into_response());
    }

    let list = state.repo.update_list(id, name).await?;

    Ok(RenderHtml(
        "lists/_card".to_string(),
        state.template_engine,
        json!({
            "list": list,
            "editing_name": false,
        }),
    )
    .into_response())
}
