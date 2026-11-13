package posterrepository

import (
	"context"
	"errors"
	"fmt"
	"looqa/model"
	"time"

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

func (r *PostgresRepository) GetPosterAll() ([]model.Poster, error) {
	rows, err := r.pool.Query(context.Background(), `
		SELECT id,
		       COALESCE(label, ''), COALESCE(text, ''), author_id,
		       COALESCE(comments_count, 0), COALESCE(likes_count, 0),
		       publication_at
		FROM "Poster"
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("%w: query posters: %v", model.ErrInternal, err)
	}
	defer rows.Close()

	posters := make([]model.Poster, 0)
	for rows.Next() {
		var p model.Poster
		if err := rows.Scan(&p.Id, &p.Label, &p.Text, &p.AuthorID, &p.CommentsCount, &p.LikesCount, &p.PublicationAt); err != nil {
			return nil, fmt.Errorf("%w: scan poster: %v", model.ErrInternal, err)
		}
		posters = append(posters, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate posters: %v", model.ErrInternal, err)
	}
	return posters, nil
}

func (r *PostgresRepository) GetPosterByID(id uuid.UUID) (model.Poster, error) {
	var p model.Poster
	err := r.pool.QueryRow(context.Background(), `
		SELECT id,
		       COALESCE(label, ''), COALESCE(text, ''), author_id,
		       COALESCE(comments_count, 0), COALESCE(likes_count, 0),
		       publication_at
		FROM "Poster"
		WHERE id = $1`, id).
		Scan(&p.Id, &p.Label, &p.Text, &p.AuthorID, &p.CommentsCount, &p.LikesCount, &p.PublicationAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Poster{}, model.ErrNotFound
	}
	if err != nil {
		return model.Poster{}, fmt.Errorf("%w: query poster by id: %v", model.ErrInternal, err)
	}
	return p, nil
}

func (r *PostgresRepository) CreatePoster(poster model.Poster) error {
	publicationAt := poster.PublicationAt
	if publicationAt.IsZero() {
		publicationAt = time.Now()
	}

	tag, err := r.pool.Exec(context.Background(), `
		INSERT INTO "Poster" (id, label, text, author_id, comments_count, likes_count, publication_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING`,
		poster.Id, poster.Label, poster.Text, poster.AuthorID,
		poster.CommentsCount, poster.LikesCount, publicationAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == fkViolationCode {
			return fmt.Errorf("create poster: author %s not found: %v", poster.AuthorID, err)
		}
		return fmt.Errorf("%w: insert poster: %v", model.ErrInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrConflict
	}
	return nil
}

func (r *PostgresRepository) DeletePoster(id uuid.UUID) error {
	tag, err := r.pool.Exec(context.Background(), `
		DELETE FROM "Poster"
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("%w: delete poster: %v", model.ErrInternal, err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
