package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/99minutos/shipping-system/internal/core/domain"
)

const collectionShipments = "shipments"

type ShipmentRepository struct {
	col *mongo.Collection
}

func NewShipmentRepository(db *mongo.Database) *ShipmentRepository {
	return &ShipmentRepository{col: db.Collection(collectionShipments)}
}

// Create inserts a new shipment document.
func (r *ShipmentRepository) Create(ctx context.Context, s *domain.Shipment) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err := r.col.InsertOne(ctx, s)
	if err != nil {
		return err
	}
	return nil
}

// FindByTrackingNumber retrieves a shipment by tracking number.
// When clientID is non-empty, an additional filter by client_id is applied.
func (r *ShipmentRepository) FindByTrackingNumber(ctx context.Context, trackingNumber string, clientID string) (*domain.Shipment, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	filter := bson.M{"tracking_number": trackingNumber}
	if clientID != "" {
		filter["client_id"] = clientID
	}

	var s domain.Shipment
	err := r.col.FindOne(ctx, filter).Decode(&s)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrShipmentNotFound
		}
		return nil, err
	}
	return &s, nil
}

// FindByIdempotencyKey retrieves an existing shipment that was created with the given key.
func (r *ShipmentRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Shipment, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var s domain.Shipment
	err := r.col.FindOne(ctx, bson.M{"idempotency_key": key}).Decode(&s)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrShipmentNotFound
		}
		return nil, err
	}
	return &s, nil
}

// EnsureIndexes creates necessary indexes on the shipments collection.
func (r *ShipmentRepository) EnsureIndexes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tracking_number", Value: 1}}},
		{Keys: bson.D{{Key: "client_id", Value: 1}}},
		{Keys: bson.D{{Key: "idempotency_key", Value: 1}}},
	}

	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	return err
}
