package userrepository

import (
	"context"
	"errors"
	"fmt"
	"looqa/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetUserAll() ([]model.User, error) {
	rows, err := r.pool.Query(context.Background(), `
		SELECT id,
		       COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(avatar_url, '')
		FROM "User"
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("%w: query users: %v", model.ErrInternal, err)
	}
	defer rows.Close()

	users := make([]model.User, 0)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.Id, &u.FirstName, &u.LastName, &u.AvatarURL); err != nil {
			return nil, fmt.Errorf("%w: scan user: %v", model.ErrInternal, err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate users: %v", model.ErrInternal, err)
	}
	return users, nil
}

func (r *PostgresRepository) GetUserByID(id uuid.UUID) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(context.Background(), `
		SELECT id,
		       COALESCE(first_name, ''), COALESCE(last_name, ''), COALESCE(avatar_url, '')
		FROM "User"
		WHERE id = $1`, id).
		Scan(&u.Id, &u.FirstName, &u.LastName, &u.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, model.ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("%w: query user by id: %v", model.ErrInternal, err)
	}
	return u, nil
}

func (r *PostgresRepository) CreateUser(user model.User) error {
	tag, err := r.pool.Exec(context.Background(), `
		INSERT INTO "User" (id, first_name, last_name, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING`,
		user.Id, user.FirstName, user.LastName, user.AvatarURL)
	if err != nil {
		return fmt.Errorf("%w: insert user: %v", model.ErrInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrConflict
	}
	return nil
}

func (r *PostgresRepository) DeleteUser(id uuid.UUID) error {
	tag, err := r.pool.Exec(context.Background(), `
		DELETE FROM "User"
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("%w: delete user: %v", model.ErrInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
