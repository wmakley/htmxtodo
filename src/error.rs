use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use sqlx;
// use axum_template::{engine::Engine, RenderHtml};
// use tracing::debug;

#[derive(Debug)]
pub enum AppError {
    RecordNotFound(String, i64),
    NotFound,
    Sqlx(sqlx::Error),
    Other(anyhow::Error),
}

// Tell axum how to convert `AppError` into a response.
impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        match self {
            AppError::RecordNotFound(record, id) => (
                StatusCode::NOT_FOUND,
                format!("404 Not Found: {} with id {} not found\n", record, id),
            )
                .into_response(),
            AppError::NotFound => (StatusCode::NOT_FOUND, "404 Not Found\n").into_response(),

            // TODO: details should be logged, but not returned to the client
            AppError::Sqlx(err) => (
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("500 Internal Server Error: {}\n", err),
            )
                .into_response(),
            AppError::Other(err) => (
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("500 Internal Server Error: {}\n", err),
            )
                .into_response(),
        }
    }
}

// This enables using `?` on functions that return `Result<_, anyhow::Error>` to turn them into
// `Result<_, AppError>`. That way you don't need to do that manually.
// impl<E> From<E> for AppError
// where
//     E: Into<anyhow::Error>,
// {
//     fn from(err: E) -> Self {
//         Self::Other(err.into())
//     }
// }

impl From<sqlx::Error> for AppError {
    fn from(err: sqlx::Error) -> Self {
        match err {
            sqlx::Error::RowNotFound => Self::NotFound,
            _ => Self::Sqlx(err),
        }
    }
}
