package repository

import (
	"context"
	"database/sql"
	"errors"

	pb "ecommerce-microservices/proto"
)

var (
	ErrNotFound      = errors.New("product not found")
	ErrAlreadyExists = errors.New("product already exists")
)

type ProductRepository interface {
	Create(context.Context, *pb.Product) error
	Get(context.Context, string) (*pb.Product, error)
	Update(context.Context, *pb.Product) error
	Delete(context.Context, string) error
}

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(ctx context.Context, db *sql.DB) (*SQLiteRepository, error) {
	repo := &SQLiteRepository{db: db}
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS products (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			price REAL NOT NULL,
			stock INTEGER NOT NULL
		)`)
	return repo, err
}

func (r *SQLiteRepository) Create(ctx context.Context, product *pb.Product) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO products (id, name, description, price, stock) VALUES (?, ?, ?, ?, ?)`,
		product.GetId(), product.GetName(), product.GetDescription(), product.GetPrice(), product.GetStock())
	if err != nil {
		if isConstraintError(err) {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (*pb.Product, error) {
	product := &pb.Product{Id: id}
	err := r.db.QueryRowContext(ctx,
		`SELECT name, description, price, stock FROM products WHERE id = ?`, id).
		Scan(&product.Name, &product.Description, &product.Price, &product.Stock)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, product *pb.Product) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE products SET name = ?, description = ?, price = ?, stock = ? WHERE id = ?`,
		product.GetName(), product.GetDescription(), product.GetPrice(), product.GetStock(), product.GetId())
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
	result, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE id = ?`, id)
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

func isConstraintError(err error) bool {
	return err != nil && (errors.Is(err, sql.ErrNoRows) == false &&
		(len(err.Error()) >= 10 && err.Error()[:10] == "constraint"))
}
