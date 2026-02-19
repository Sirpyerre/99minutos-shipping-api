package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/99minutos/shipping-system/internal/core/domain"
	"github.com/99minutos/shipping-system/internal/core/ports"
)

// DedupChecker abstracts the idempotency store (Redis).
type DedupChecker interface {
	IsDuplicate(ctx context.Context, trackingNumber, status string, ts time.Time) (bool, error)
	Mark(ctx context.Context, trackingNumber, status string, ts time.Time) error
}

type eventService struct {
	shipmentRepo ports.ShipmentRepository
	eventRepo    ports.EventRepository
	dedup        DedupChecker
	log          zerolog.Logger
}

// NewEventService returns an EventService implementation.
func NewEventService(
	shipmentRepo ports.ShipmentRepository,
	eventRepo ports.EventRepository,
	dedup DedupChecker,
	log zerolog.Logger,
) ports.EventService {
	return &eventService{
		shipmentRepo: shipmentRepo,
		eventRepo:    eventRepo,
		dedup:        dedup,
		log:          log,
	}
}

// Process validates, deduplicates, and persists a single tracking event.
func (s *eventService) Process(ctx context.Context, in ports.TrackingEventInput) error {
	newStatus := domain.ShipmentStatus(in.Status)

	// 1. Idempotency check — silently skip duplicates.
	isDup, err := s.dedup.IsDuplicate(ctx, in.TrackingNumber, in.Status, in.Timestamp)
	if err != nil {
		s.log.Warn().Err(err).Str("tracking", in.TrackingNumber).Msg("dedup check failed, processing anyway")
	} else if isDup {
		s.log.Debug().Str("tracking", in.TrackingNumber).Str("status", in.Status).Msg("duplicate event skipped")
		return nil
	}

	// 2. Find shipment (no client filter — events come from external sources).
	shipment, err := s.shipmentRepo.FindByTrackingNumber(ctx, in.TrackingNumber, "")
	if err != nil {
		return fmt.Errorf("process event: %w", err)
	}

	// 3. Validate state machine transition.
	if !shipment.Status.CanTransitionTo(newStatus) {
		return fmt.Errorf("process event: %w (from %s to %s)", domain.ErrInvalidTransition, shipment.Status, newStatus)
	}

	// 4. Mark as processed before writing (prevents duplicate processing on retry).
	if markErr := s.dedup.Mark(ctx, in.TrackingNumber, in.Status, in.Timestamp); markErr != nil {
		s.log.Warn().Err(markErr).Str("tracking", in.TrackingNumber).Msg("failed to set dedup key")
	}

	// 5. Build optional location.
	var loc *domain.Coordinates
	if in.Location != nil {
		loc = &domain.Coordinates{Lat: in.Location.Lat, Lng: in.Location.Lng}
	}

	// 6. Atomically update shipment status + history.
	if err := s.eventRepo.UpdateShipmentStatus(ctx, in.TrackingNumber, newStatus, in.Timestamp, in.Source, loc); err != nil {
		return fmt.Errorf("process event: update status: %w", err)
	}

	// 7. Insert into audit trail (non-fatal on failure).
	auditEvent := &domain.TrackingEvent{
		TrackingNumber: in.TrackingNumber,
		Status:         newStatus,
		Timestamp:      in.Timestamp,
		Source:         in.Source,
		Location:       loc,
	}
	if err := s.eventRepo.InsertEvent(ctx, auditEvent); err != nil {
		s.log.Warn().Err(err).Str("tracking", in.TrackingNumber).Msg("failed to insert audit event")
	}

	s.log.Info().
		Str("tracking", in.TrackingNumber).
		Str("status", in.Status).
		Str("source", in.Source).
		Msg("event processed")

	return nil
}