package repo

import (
	"context"
	"database/sql"
	"htmxtodo/gen/htmxtodo_dev/public/model"

	. "github.com/go-jet/jet/v2/postgres"
	. "htmxtodo/gen/htmxtodo_dev/public/table"
)

type Repository interface {
	FilterLists(ctx context.Context) ([]model.Lists, error)
	GetListById(ctx context.Context, id int64) (model.Lists, error)
	CreateList(ctx context.Context, name string) (model.Lists, error)
	UpdateListById(ctx context.Context, id int64, list model.Lists) (model.Lists, error)
	DeleteListById(ctx context.Context, id int64) error
}

func NewRepository(db *sql.DB) Repository {
	return &repository{
		db: db,
	}
}

type DBTX interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type repository struct {
	db *sql.DB
}

func (r *repository) FilterLists(ctx context.Context) ([]model.Lists, error) {
	stmt := Lists.SELECT(Lists.AllColumns).ORDER_BY(Lists.Name.ASC())

	var results []model.Lists
	if err := stmt.QueryContext(ctx, r.db, &results); err != nil {
		return nil, err
	}

	if results == nil {
		results = make([]model.Lists, 0)
	}

	return results, nil
}

func (r *repository) GetListById(ctx context.Context, id int64) (model.Lists, error) {
	stmt := Lists.SELECT(Lists.AllColumns).WHERE(Lists.ID.EQ(Int(id))).LIMIT(1)

	var result model.Lists
	if err := stmt.QueryContext(ctx, r.db, &result); err != nil {
		return result, err
	}

	if result.ID == 0 {
		return result, sql.ErrNoRows
	}

	return result, nil
}

func (r *repository) CreateList(ctx context.Context, name string) (model.Lists, error) {
	var result model.Lists

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	stmt := Lists.INSERT(Lists.Name).
		VALUES(name).
		RETURNING(Lists.AllColumns)

	if err := stmt.QueryContext(ctx, r.db, &result); err != nil {
		return result, err
	}

	if err = tx.Commit(); err != nil {
		return result, err
	}

	return result, nil
}

func (r *repository) UpdateListById(ctx context.Context, id int64, list model.Lists) (model.Lists, error) {
	var result model.Lists

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	// TODO: get the list first?

	updateStmt := Lists.UPDATE(Lists.Name).
		SET(list.Name).
		WHERE(Lists.ID.EQ(Int(id))).
		RETURNING(Lists.AllColumns)

	if err = updateStmt.QueryContext(ctx, tx, &result); err != nil {
		return result, err
	}

	if err = tx.Commit(); err != nil {
		return result, err
	}

	return result, nil
}
