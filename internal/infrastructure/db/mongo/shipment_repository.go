package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/99minutos/shipping-system/internal/core/domain"
	"github.com/99minutos/shipping-system/internal/core/ports"
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

// List returns a page of shipments matching the filter and the total document count.
func (r *ShipmentRepository) List(ctx context.Context, filter ports.ListShipmentsFilter) ([]*domain.Shipment, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	q := buildListFilter(filter)

	total, err := r.col.CountDocuments(ctx, q)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((filter.Page - 1) * filter.Limit)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(filter.Limit))

	cursor, err := r.col.Find(ctx, q, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var shipments []*domain.Shipment
	if err := cursor.All(ctx, &shipments); err != nil {
		return nil, 0, err
	}
	return shipments, total, nil
}

// buildListFilter constructs a dynamic MongoDB filter from the given parameters.
func buildListFilter(f ports.ListShipmentsFilter) bson.M {
	q := bson.M{}

	if f.ClientID != "" {
		q["client_id"] = f.ClientID
	}
	if f.Status != "" {
		q["status"] = f.Status
	}
	if f.ServiceType != "" {
		q["service_type"] = f.ServiceType
	}
	if !f.DateFrom.IsZero() || !f.DateTo.IsZero() {
		dateRange := bson.M{}
		if !f.DateFrom.IsZero() {
			dateRange["$gte"] = f.DateFrom
		}
		if !f.DateTo.IsZero() {
			dateRange["$lte"] = f.DateTo
		}
		q["created_at"] = dateRange
	}
	if f.Search != "" {
		q["$or"] = bson.A{
			bson.M{"tracking_number": bson.M{"$regex": f.Search, "$options": "i"}},
			bson.M{"sender.name": bson.M{"$regex": f.Search, "$options": "i"}},
		}
	}

	return q
}
func (r *ShipmentRepository) EnsureIndexes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tracking_number", Value: 1}}},
		{Keys: bson.D{{Key: "idempotency_key", Value: 1}}},
		// Compound indexes for list queries: sorted by created_at desc, filtered by client+status.
		{Keys: bson.D{{Key: "client_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "client_id", Value: 1}, {Key: "status", Value: 1}}},
	}

	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	return err
}
