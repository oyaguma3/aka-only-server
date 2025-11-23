package db

import (
	"context"
	"fmt"

	"aka-server/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Pool *pgxpool.Pool
}

func NewRepository(dbURL string) (*Repository, error) {
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	return &Repository{Pool: pool}, nil
}

func (r *Repository) Close() {
	r.Pool.Close()
}

func (r *Repository) CreateSubscriber(ctx context.Context, sub *model.Subscriber) error {
	query := `
		INSERT INTO public.subscribers (imsi, ki, opc, sqn, amf)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.Pool.Exec(ctx, query, sub.IMSI, sub.Ki, sub.Opc, sub.SQN, sub.AMF)
	return err
}

func (r *Repository) GetSubscriber(ctx context.Context, imsi string) (*model.Subscriber, error) {
	query := `
		SELECT imsi, ki, opc, sqn, amf, created_at
		FROM public.subscribers
		WHERE imsi = $1
	`
	row := r.Pool.QueryRow(ctx, query, imsi)

	var sub model.Subscriber
	err := row.Scan(&sub.IMSI, &sub.Ki, &sub.Opc, &sub.SQN, &sub.AMF, &sub.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *Repository) UpdateSubscriber(ctx context.Context, sub *model.Subscriber) error {
	query := `
		UPDATE public.subscribers
		SET ki = $2, opc = $3, sqn = $4, amf = $5
		WHERE imsi = $1
	`
	_, err := r.Pool.Exec(ctx, query, sub.IMSI, sub.Ki, sub.Opc, sub.SQN, sub.AMF)
	return err
}

func (r *Repository) DeleteSubscriber(ctx context.Context, imsi string) error {
	query := `DELETE FROM public.subscribers WHERE imsi = $1`
	_, err := r.Pool.Exec(ctx, query, imsi)
	return err
}

func (r *Repository) UpdateSQN(ctx context.Context, imsi, newSQN string) error {
	query := `UPDATE public.subscribers SET sqn = $2 WHERE imsi = $1`
	_, err := r.Pool.Exec(ctx, query, imsi, newSQN)
	return err
}

func (r *Repository) GetSubscriberCount(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM public.subscribers`
	var count int64
	err := r.Pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}

func (r *Repository) ListSubscribers(ctx context.Context) ([]*model.Subscriber, error) {
	query := `
		SELECT imsi, ki, opc, sqn, amf, created_at
		FROM public.subscribers
		ORDER BY imsi ASC
	`
	rows, err := r.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscribers []*model.Subscriber
	for rows.Next() {
		var sub model.Subscriber
		if err := rows.Scan(&sub.IMSI, &sub.Ki, &sub.Opc, &sub.SQN, &sub.AMF, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subscribers = append(subscribers, &sub)
	}
	return subscribers, rows.Err()
}
