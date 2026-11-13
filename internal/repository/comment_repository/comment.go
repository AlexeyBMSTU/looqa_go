package commentrepository

import (
	"context"
	"errors"
	"fmt"
	"looqa/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const fkViolationCode = "23503" // foreign_key_violation

func (r *PostgresRepository) GetCommentAll() ([]model.Comment, error) {
	rows, err := r.pool.Query(context.Background(), `
		SELECT id, author_id, COALESCE(likes_count, 0)
		FROM "Comment"
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("%w: query comments: %v", model.ErrInternal, err)
	}
	defer rows.Close()

	comments := make([]model.Comment, 0)
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.Id, &c.AuthorID, &c.LikesCount); err != nil {
			return nil, fmt.Errorf("%w: scan comment: %v", model.ErrInternal, err)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate comments: %v", model.ErrInternal, err)
	}
	return comments, nil
}

func (r *PostgresRepository) GetCommentByID(id uuid.UUID) (model.Comment, error) {
	var c model.Comment
	err := r.pool.QueryRow(context.Background(), `
		SELECT id, author_id, COALESCE(likes_count, 0)
		FROM "Comment"
		WHERE id = $1`, id).
		Scan(&c.Id, &c.AuthorID, &c.LikesCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Comment{}, model.ErrNotFound
	}
	if err != nil {
		return model.Comment{}, fmt.Errorf("%w: query comment by id: %v", model.ErrInternal, err)
	}
	return c, nil
}

func (r *PostgresRepository) CreateComment(comment model.Comment) error {
	tag, err := r.pool.Exec(context.Background(), `
		INSERT INTO "Comment" (id, author_id, likes_count)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING`,
		comment.Id, comment.AuthorID, comment.LikesCount)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == fkViolationCode {
			return fmt.Errorf("create comment: author %s not found: %v", comment.AuthorID, err)
		}
		return fmt.Errorf("%w: insert comment: %v", model.ErrInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrConflict
	}
	return nil
}

func (r *PostgresRepository) DeleteComment(id uuid.UUID) error {
	tag, err := r.pool.Exec(context.Background(), `
		DELETE FROM "Comment"
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("%w: delete comment: %v", model.ErrInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
