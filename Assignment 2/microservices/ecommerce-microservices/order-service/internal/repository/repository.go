package repository

import (
	"context"
	"database/sql"
	"errors"

	pb "ecommerce-microservices/proto"
)

var ErrNotFound = errors.New("order not found")

type OrderRepository interface {
	Create(context.Context, *pb.Order) error
	Get(context.Context, string) (*pb.Order, error)
	Update(context.Context, *pb.Order) error
	Delete(context.Context, string) error
}

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(ctx context.Context, db *sql.DB) (*SQLiteRepository, error) {
	repo := &SQLiteRepository{db: db}
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS orders (
			id TEXT PRIMARY KEY,
			product_id TEXT NOT NULL,
			quantity INTEGER NOT NULL,
			unit_price REAL NOT NULL,
			total_price REAL NOT NULL
		)`)
	return repo, err
}

func (r *SQLiteRepository) Create(ctx context.Context, order *pb.Order) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO orders (id, product_id, quantity, unit_price, total_price) VALUES (?, ?, ?, ?, ?)`,
		order.GetId(), order.GetProductId(), order.GetQuantity(), order.GetUnitPrice(), order.GetTotalPrice())
	return err
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (*pb.Order, error) {
	order := &pb.Order{Id: id}
	err := r.db.QueryRowContext(ctx,
		`SELECT product_id, quantity, unit_price, total_price FROM orders WHERE id = ?`, id).
		Scan(&order.ProductId, &order.Quantity, &order.UnitPrice, &order.TotalPrice)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, order *pb.Order) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE orders SET product_id = ?, quantity = ?, unit_price = ?, total_price = ? WHERE id = ?`,
		order.GetProductId(), order.GetQuantity(), order.GetUnitPrice(), order.GetTotalPrice(), order.GetId())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM orders WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
