use crate::error::AppError;
use serde::{Deserialize, Serialize};
use sqlx::pool::Pool;
use sqlx::postgres::Postgres;

#[derive(sqlx::FromRow, Deserialize, Serialize)]
pub struct List {
    pub id: i64,
    pub name: String,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

#[derive(Clone)]
pub struct Repo {
    pool: Pool<Postgres>,
}

impl Repo {
    pub fn new(pool: Pool<Postgres>) -> Repo {
        Repo { pool }
    }

    pub async fn get_list(&self, id: i64) -> Result<List, AppError> {
        let sql = "SELECT * FROM list WHERE id = $1".to_string();

        let list = sqlx::query_as::<_, List>(&sql)
            .bind(id)
            .fetch_one(&self.pool)
            .await?;

        Ok(list)
    }

    pub async fn filter_lists(&self) -> Result<Vec<List>, AppError> {
        let sql = "SELECT * FROM list ORDER BY id".to_string();

        let results = sqlx::query_as::<_, List>(&sql)
            .fetch_all(&self.pool)
            .await?;

        Ok(results)
    }

    pub async fn create_list(&self, name: &str) -> Result<List, AppError> {
        let sql = "INSERT INTO list (name) VALUES ($1) RETURNING *".to_string();

        let list = sqlx::query_as::<_, List>(&sql)
            .bind(name)
            .fetch_one(&self.pool)
            .await?;

        Ok(list)
    }

    pub async fn delete_list(&self, id: i64) -> Result<(), AppError> {
        let sql = "DELETE FROM list WHERE id = $1".to_string();

        sqlx::query(&sql).bind(id).execute(&self.pool).await?;

        Ok(())
    }

    pub async fn update_list(&self, id: i64, name: &str) -> Result<List, AppError> {
        let mut transaction = self.pool.begin().await?;

        let sql = "SELECT * FROM list WHERE id = $1".to_string();

        let list = sqlx::query_as::<_, List>(&sql)
            .bind(id)
            .fetch_one(&mut *transaction)
            .await?;

        if list.name == name {
            return Ok(list);
        }

        let sql =
            "UPDATE list SET name = $1, updated_at = NOW() WHERE id = $2 RETURNING *".to_string();

        let list = sqlx::query_as::<_, List>(&sql)
            .bind(name)
            .bind(id)
            .fetch_one(&mut *transaction)
            .await?;

        transaction.commit().await?;

        Ok(list)
    }
}
