package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/99minutos/shipping-system/internal/core/domain"
	"github.com/99minutos/shipping-system/internal/core/ports"
)

// EventRepository implements ports.EventRepository using MongoDB.
type EventRepository struct {
	db *mongo.Database
}

// NewEventRepository creates a new EventRepository.
func NewEventRepository(db *mongo.Database) ports.EventRepository {
	return &EventRepository{db: db}
}

// UpdateShipmentStatus atomically sets the shipment status and appends a history entry.
func (r *EventRepository) UpdateShipmentStatus(
	ctx context.Context,
	trackingNumber string,
	status domain.ShipmentStatus,
	ts time.Time,
	source string,
	location *domain.Coordinates,
) error {
	historyEntry := bson.M{
		"status":    string(status),
		"timestamp": ts.UTC(),
		"notes":     source,
	}

	filter := bson.M{"tracking_number": trackingNumber}
	update := bson.M{
		"$set":  bson.M{"status": string(status)},
		"$push": bson.M{"status_history": historyEntry},
	}

	_, err := r.db.Collection("shipments").UpdateOne(ctx, filter, update)
	return err
}

// InsertEvent persists a tracking event to the status_events audit collection.
func (r *EventRepository) InsertEvent(ctx context.Context, event *domain.TrackingEvent) error {
	doc := bson.M{
		"tracking_number": event.TrackingNumber,
		"status":          string(event.Status),
		"timestamp":       event.Timestamp.UTC(),
		"source":          event.Source,
		"processed_at":    time.Now().UTC(),
	}
	if event.Location != nil {
		doc["location"] = bson.M{
			"lat": event.Location.Lat,
			"lng": event.Location.Lng,
		}
	}

	_, err := r.db.Collection("status_events").InsertOne(ctx, doc)
	return err
}
