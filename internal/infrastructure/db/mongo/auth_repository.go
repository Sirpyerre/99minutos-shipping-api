package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/99minutos/shipping-system/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const authCollection = "auth_users"

type MongoAuthRepository struct {
	coll *mongo.Collection
}

func NewAuthRepository(db *mongo.Database) *MongoAuthRepository {
	return &MongoAuthRepository{coll: db.Collection(authCollection)}
}

type mongoUser struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Username     string             `bson:"username"`
	PasswordHash string             `bson:"password_hash"`
	Role         string             `bson:"role"`
	ClientID     string             `bson:"client_id,omitempty"`
	CreatedAt    int64              `bson:"created_at"`
	UpdatedAt    int64              `bson:"updated_at"`
}

func (r *MongoAuthRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	doc := mongoUser{
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
		ClientID:     user.ClientID,
		CreatedAt:    user.CreatedAt.Unix(),
		UpdatedAt:    user.UpdatedAt.Unix(),
	}

	_, err := r.coll.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, domain.ErrUserExists
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	// fetch back to get ID
	created, err := r.FindByUsername(ctx, user.Username)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *MongoAuthRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var mu mongoUser
	if err := r.coll.FindOne(ctx, bson.M{"username": username}).Decode(&mu); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	return &domain.User{
		ID:           mu.ID.Hex(),
		Username:     mu.Username,
		PasswordHash: mu.PasswordHash,
		Role:         mu.Role,
		ClientID:     mu.ClientID,
		CreatedAt:    unixToTime(mu.CreatedAt),
		UpdatedAt:    unixToTime(mu.UpdatedAt),
	}, nil
}

func unixToTime(ts int64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0).UTC()
}
