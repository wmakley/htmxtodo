package repo

import (
	"context"
	"database/sql"
	. "github.com/go-jet/jet/v2/postgres"
	"htmxtodo/gen/htmxtodo_dev/public/model"
	. "htmxtodo/gen/htmxtodo_dev/public/table"
)

type Repository interface {
	FilterLists(ctx context.Context) ([]model.List, error)
	GetListById(ctx context.Context, id int64) (model.List, error)
	CreateList(ctx context.Context, name string) (model.List, error)
	UpdateListById(ctx context.Context, id int64, name string) (model.List, error)
	DeleteListById(ctx context.Context, id int64) error
}

// DBTX is an interface that matches the standard library sql.DB and sql.Tx interfaces.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db *sql.DB) Repository {
	return &repository{
		// db is the database connection
		db: db,
		// dbtx is either the database connection or a transaction, allows nested repository calls
		// to use the same transaction
		dbtx: db,
	}
}

type repository struct {
	db   *sql.DB
	dbtx DBTX
}

// withTransaction returns a new repository that uses the provided transaction as dbtx
func (r *repository) withTransaction(tx *sql.Tx) *repository {
	return &repository{
		db:   r.db,
		dbtx: tx,
	}
}

func (r *repository) FilterLists(ctx context.Context) ([]model.List, error) {
	stmt := List.SELECT(List.AllColumns).ORDER_BY(List.Name.ASC())

	var results []model.List
	if err := stmt.QueryContext(ctx, r.dbtx, &results); err != nil {
		return nil, err
	}

	if results == nil {
		results = make([]model.List, 0)
	}

	return results, nil
}

func (r *repository) GetListById(ctx context.Context, id int64) (model.List, error) {
	stmt := List.SELECT(List.AllColumns).WHERE(List.ID.EQ(Int(id))).LIMIT(1)

	var result model.List
	if err := stmt.QueryContext(ctx, r.dbtx, &result); err != nil {
		return result, err
	}

	if result.ID == 0 {
		return result, sql.ErrNoRows
	}

	return result, nil
}

func (r *repository) CreateList(ctx context.Context, name string) (model.List, error) {
	var result model.List

	stmt := List.INSERT(List.Name).
		VALUES(name).
		RETURNING(List.AllColumns)

	if err := stmt.QueryContext(ctx, r.dbtx, &result); err != nil {
		return result, err
	}

	return result, nil
}

func (r *repository) UpdateListById(ctx context.Context, id int64, name string) (model.List, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		panic(err) // unrecoverable
	}
	defer tx.Rollback()
	rtx := r.withTransaction(tx)

	existing, err := rtx.GetListById(ctx, id)

	if existing.Name == name {
		// No update needed
		return existing, nil
	}

	updateStmt := List.UPDATE(List.Name, List.UpdatedAt).
		SET(name, NOW()).
		WHERE(List.ID.EQ(Int(id))).
		RETURNING(List.AllColumns)

	if err = updateStmt.QueryContext(ctx, tx, &existing); err != nil {
		return existing, err
	}

	if err = tx.Commit(); err != nil {
		return existing, err
	}

	return existing, nil
}

func (r *repository) DeleteListById(ctx context.Context, id int64) error {
	deleteStmt := List.DELETE().
		WHERE(List.ID.EQ(Int(id)))

	if _, err := deleteStmt.ExecContext(ctx, r.dbtx); err != nil {
		return err
	}

	return nil
}
