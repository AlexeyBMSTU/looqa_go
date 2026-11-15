package authrepository

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
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// refreshDoc — документ refresh-токена в MongoDB. В _id лежит
// SHA-256-хеш токена (сам токен не хранится), family_id связывает цепочку
// ротированных токенов одной сессии.
type refreshDoc struct {
	ID        string    `bson:"_id"`
	FamilyID  string    `bson:"family_id"`
	UserID    string    `bson:"user_id"`
	ExpiresAt time.Time `bson:"expires_at"`
	Rotated   bool      `bson:"rotated"`
}

func newRefreshDoc(token model.RefreshToken) refreshDoc {
	return refreshDoc{
		ID:        token.Hash,
		FamilyID:  token.FamilyID.String(),
		UserID:    token.UserID.String(),
		ExpiresAt: token.ExpiresAt,
	}
}

func (d refreshDoc) toModel() (model.RefreshToken, error) {
	familyID, err := uuid.Parse(d.FamilyID)
	if err != nil {
		return model.RefreshToken{}, fmt.Errorf("%w: parse family_id %q: %v", model.ErrInternal, d.FamilyID, err)
	}
	userID, err := uuid.Parse(d.UserID)
	if err != nil {
		return model.RefreshToken{}, fmt.Errorf("%w: parse user_id %q: %v", model.ErrInternal, d.UserID, err)
	}
	return model.RefreshToken{Hash: d.ID, FamilyID: familyID, UserID: userID, ExpiresAt: d.ExpiresAt}, nil
}

// MongoRepository — хранилище refresh-токенов в MongoDB.
type MongoRepository struct {
	coll *mongo.Collection
}

// NewMongoRepository создаёт репозиторий и TTL-индекс по expires_at:
// MongoDB сама удаляет истёкшие токены, отдельная чистка не нужна.
func NewMongoRepository(ctx context.Context, coll *mongo.Collection) (*MongoRepository, error) {
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{bson.E{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	if err != nil {
		return nil, fmt.Errorf("create ttl index: %w", err)
	}
	return &MongoRepository{coll: coll}, nil
}

// AddRefresh сохраняет новый refresh-токен (точнее, его хеш).
func (r *MongoRepository) AddRefresh(ctx context.Context, token model.RefreshToken) error {
	_, err := r.coll.InsertOne(ctx, newRefreshDoc(token))
	if mongo.IsDuplicateKeyError(err) {
		return model.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("%w: insert refresh token: %v", model.ErrInternal, err)
	}
	return nil
}

// RotateRefresh атомарно помечает токен ротированным и возвращает его.
// Ошибки:
//   - ErrNotFound — токен неизвестен или истёк;
//   - ErrConflict — токен уже ротирован: его предъявили повторно (возможная
//     кража), вызывающий должен отозвать всё семейство (RevokeFamily);
//   - ErrInternal — прочие ошибки хранилища.
func (r *MongoRepository) RotateRefresh(ctx context.Context, hash string) (model.RefreshToken, error) {
	var doc refreshDoc
	err := r.coll.FindOneAndUpdate(ctx,
		bson.M{"_id": hash, "rotated": false, "expires_at": bson.M{"$gt": time.Now()}},
		bson.M{"$set": bson.M{"rotated": true}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		// Не совпало с «ещё не ротирован и жив»: либо токена нет/истёк,
		// либо его уже ротировали — различаем повторным поиском.
		var d refreshDoc
		err := r.coll.FindOne(ctx, bson.M{"_id": hash, "expires_at": bson.M{"$gt": time.Now()}}).Decode(&d)
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			return model.RefreshToken{}, model.ErrNotFound
		case err != nil:
			return model.RefreshToken{}, fmt.Errorf("%w: query refresh token: %v", model.ErrInternal, err)
		}
		token, err := d.toModel()
		if err != nil {
			return model.RefreshToken{}, err
		}
		return token, model.ErrConflict // живой, но уже ротированный: переиспользование
	}
	if err != nil {
		return model.RefreshToken{}, fmt.Errorf("%w: rotate refresh token: %v", model.ErrInternal, err)
	}
	return doc.toModel()
}

// RevokeFamily удаляет все refresh-токены сессии familyID (logout
// или детект переиспользования). Идемпотентна: отсутствие токенов — не ошибка.
func (r *MongoRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	_, err := r.coll.DeleteMany(ctx, bson.M{"family_id": familyID.String()})
	if err != nil {
		return fmt.Errorf("%w: revoke family: %v", model.ErrInternal, err)
	}
	return nil
}

// PostgresUserRepository — пользователи и их учётные данные
// (таблицы "User" и "Credential" в postgres).
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository возвращает репозиторий пользователей
// поверх пула подключений к postgres.
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

const uniqueViolationCode = "23505" // unique_violation

// CreateUser регистрирует пользователя: в одной транзакции создаёт
// профиль в "User" и учётные данные в "Credential".
func (r *PostgresUserRepository) CreateUser(ctx context.Context, user model.User, cred model.Credential) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: begin register: %v", model.ErrInternal, err)
	}
	defer func() {
		// no-op после успешного Commit
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `
		INSERT INTO "User" (id, first_name, last_name, avatar_url)
		VALUES ($1, $2, $3, $4)`,
		user.Id, user.FirstName, user.LastName, user.AvatarURL)
	if err != nil {
		return fmt.Errorf("%w: insert user: %v", model.ErrInternal, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO "Credential" (user_id, email, password_hash)
		VALUES ($1, $2, $3)`,
		cred.UserID, cred.Email, cred.PasswordHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return fmt.Errorf("email %s already registered: %w", cred.Email, model.ErrConflict)
		}
		return fmt.Errorf("%w: insert credential: %v", model.ErrInternal, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: commit register: %v", model.ErrInternal, err)
	}
	return nil
}

// GetCredentialByEmail возвращает учётные данные по email (для входа).
func (r *PostgresUserRepository) GetCredentialByEmail(ctx context.Context, email string) (model.Credential, error) {
	var cred model.Credential
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, email, password_hash
		FROM "Credential"
		WHERE email = $1`, email).
		Scan(&cred.UserID, &cred.Email, &cred.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Credential{}, model.ErrNotFound
	}
	if err != nil {
		return model.Credential{}, fmt.Errorf("%w: query credential: %v", model.ErrInternal, err)
	}
	return cred, nil
}
