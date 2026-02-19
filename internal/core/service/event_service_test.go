package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/99minutos/shipping-system/internal/core/domain"
	"github.com/99minutos/shipping-system/internal/core/ports"
)

// ---------------------------------------------------------------------------
// Stubs
// ---------------------------------------------------------------------------

type stubEventRepo struct {
	updateErr error
	insertErr error
	updated   []string // tracking numbers updated
	inserted  []*domain.TrackingEvent
}

func (r *stubEventRepo) UpdateShipmentStatus(_ context.Context, tracking string, _ domain.ShipmentStatus, _ time.Time, _ string, _ *domain.Coordinates) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updated = append(r.updated, tracking)
	return nil
}

func (r *stubEventRepo) InsertEvent(_ context.Context, e *domain.TrackingEvent) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	r.inserted = append(r.inserted, e)
	return nil
}

type stubDedup struct {
	dupResult bool
	dupErr    error
	markErr   error
	marked    []string
}

func (d *stubDedup) IsDuplicate(_ context.Context, tracking, status string, _ time.Time) (bool, error) {
	return d.dupResult, d.dupErr
}

func (d *stubDedup) Mark(_ context.Context, tracking, status string, _ time.Time) error {
	if d.markErr != nil {
		return d.markErr
	}
	d.marked = append(d.marked, tracking+":"+status)
	return nil
}

// ---------------------------------------------------------------------------
// Helper: build a service with a seeded shipment in "created" status.
// ---------------------------------------------------------------------------

func newEventSvc(shipRepo *stubShipmentRepo, eventRepo *stubEventRepo, dedup *stubDedup) ports.EventService {
	return NewEventService(shipRepo, eventRepo, dedup, zerolog.Nop())
}

func seededRepo(tracking, clientID string, status domain.ShipmentStatus) *stubShipmentRepo {
	repo := newStubShipmentRepo()
	now := time.Now().UTC()
	repo.byTracking[tracking] = &domain.Shipment{
		TrackingNumber: tracking,
		ClientID:       clientID,
		Status:         status,
		CreatedAt:      now,
		StatusHistory:  []domain.StatusHistoryEntry{{Status: status, Timestamp: now}},
	}
	return repo
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestEventService_Process_HappyPath(t *testing.T) {
	repo := seededRepo("99M-AABBCCDD", "client_1", domain.StatusCreated)
	evRepo := &stubEventRepo{}
	dedup := &stubDedup{}

	svc := newEventSvc(repo, evRepo, dedup)
	err := svc.Process(context.Background(), ports.TrackingEventInput{
		TrackingNumber: "99M-AABBCCDD",
		Status:         "picked_up",
		Timestamp:      time.Now(),
		Source:         "driver_app",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(evRepo.updated) != 1 || evRepo.updated[0] != "99M-AABBCCDD" {
		t.Errorf("expected shipment status updated, got: %v", evRepo.updated)
	}
	if len(evRepo.inserted) != 1 {
		t.Errorf("expected audit event inserted")
	}
	if len(dedup.marked) != 1 {
		t.Errorf("expected dedup key marked")
	}
}

func TestEventService_Process_DuplicateSkipped(t *testing.T) {
	repo := seededRepo("99M-AABBCCDD", "client_1", domain.StatusCreated)
	evRepo := &stubEventRepo{}
	dedup := &stubDedup{dupResult: true} // simulate already processed

	svc := newEventSvc(repo, evRepo, dedup)
	err := svc.Process(context.Background(), ports.TrackingEventInput{
		TrackingNumber: "99M-AABBCCDD",
		Status:         "picked_up",
		Timestamp:      time.Now(),
		Source:         "driver_app",
	})

	if err != nil {
		t.Fatalf("expected no error for duplicate, got: %v", err)
	}
	if len(evRepo.updated) != 0 {
		t.Errorf("expected no update for duplicate event")
	}
}

func TestEventService_Process_ShipmentNotFound(t *testing.T) {
	repo := newStubShipmentRepo() // empty
	evRepo := &stubEventRepo{}
	dedup := &stubDedup{}

	svc := newEventSvc(repo, evRepo, dedup)
	err := svc.Process(context.Background(), ports.TrackingEventInput{
		TrackingNumber: "99M-NOTFOUND",
		Status:         "picked_up",
		Timestamp:      time.Now(),
		Source:         "driver_app",
	})

	if !errors.Is(err, domain.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got: %v", err)
	}
}

func TestEventService_Process_InvalidTransition(t *testing.T) {
	repo := seededRepo("99M-AABBCCDD", "client_1", domain.StatusCreated)
	evRepo := &stubEventRepo{}
	dedup := &stubDedup{}

	svc := newEventSvc(repo, evRepo, dedup)
	err := svc.Process(context.Background(), ports.TrackingEventInput{
		TrackingNumber: "99M-AABBCCDD",
		Status:         "delivered", // invalid: created → delivered not allowed
		Timestamp:      time.Now(),
		Source:         "driver_app",
	})

	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got: %v", err)
	}
	if len(evRepo.updated) != 0 {
		t.Errorf("expected no update on invalid transition")
	}
}

func TestEventService_Process_WithLocation(t *testing.T) {
	repo := seededRepo("99M-AABBCCDD", "client_1", domain.StatusPickedUp)
	evRepo := &stubEventRepo{}
	dedup := &stubDedup{}

	svc := newEventSvc(repo, evRepo, dedup)
	err := svc.Process(context.Background(), ports.TrackingEventInput{
		TrackingNumber: "99M-AABBCCDD",
		Status:         "in_warehouse",
		Timestamp:      time.Now(),
		Source:         "warehouse_scanner",
		Location:       &ports.LocationInput{Lat: 19.4326, Lng: -99.1332},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evRepo.inserted[0].Location == nil {
		t.Error("expected location to be set in audit event")
	}
	if evRepo.inserted[0].Location.Lat != 19.4326 {
		t.Errorf("unexpected location lat: %v", evRepo.inserted[0].Location.Lat)
	}
}

func TestEventService_Process_DedupCheckError_ProcessesAnyway(t *testing.T) {
	repo := seededRepo("99M-AABBCCDD", "client_1", domain.StatusCreated)
	evRepo := &stubEventRepo{}
	dedup := &stubDedup{dupErr: errors.New("redis timeout")} // dedup check fails

	svc := newEventSvc(repo, evRepo, dedup)
	err := svc.Process(context.Background(), ports.TrackingEventInput{
		TrackingNumber: "99M-AABBCCDD",
		Status:         "picked_up",
		Timestamp:      time.Now(),
		Source:         "driver_app",
	})

	// Should still process despite dedup check failure
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(evRepo.updated) != 1 {
		t.Errorf("expected update to proceed when dedup check errors")
	}
}

func TestEventService_Process_AuditFailureIsNonFatal(t *testing.T) {
	repo := seededRepo("99M-AABBCCDD", "client_1", domain.StatusCreated)
	evRepo := &stubEventRepo{insertErr: errors.New("mongo unavailable")}
	dedup := &stubDedup{}

	svc := newEventSvc(repo, evRepo, dedup)
	err := svc.Process(context.Background(), ports.TrackingEventInput{
		TrackingNumber: "99M-AABBCCDD",
		Status:         "picked_up",
		Timestamp:      time.Now(),
		Source:         "driver_app",
	})

	// InsertEvent failure must NOT fail the overall operation
	if err != nil {
		t.Fatalf("expected audit failure to be non-fatal, got: %v", err)
	}
	if len(evRepo.updated) != 1 {
		t.Error("expected shipment status to be updated")
	}
}
