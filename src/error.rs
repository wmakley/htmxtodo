use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use sqlx;
// use axum_template::{engine::Engine, RenderHtml};
use tracing::error;

// Application error type has specific variants for things we want to handle
// specially, and a catch-all variant for everything else that will become
// "500 Internal Server Error".
//
// Errors are logged when converted to a response (couldn't find another way).
#[derive(Debug)]
pub enum AppError {
    RecordNotFound,
    Other(anyhow::Error),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        // TODO: is this the best way to log every error? can't find another place to put it
        // Axum seems unbelievably eager to just eat errors!
        error!("{:?}", self);

        match self {
            AppError::RecordNotFound => (StatusCode::NOT_FOUND, "404 Not Found\n").into_response(),

            AppError::Other(err) => (
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("500 Internal Server Error: {}\n", err),
            )
                .into_response(),
        }
    }
}

impl From<sqlx::Error> for AppError {
    fn from(err: sqlx::Error) -> Self {
        match err {
            sqlx::Error::RowNotFound => Self::RecordNotFound,
            _ => Self::Other(err.into()),
        }
    }
}

impl From<anyhow::Error> for AppError {
    fn from(err: anyhow::Error) -> Self {
        Self::Other(err)
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
