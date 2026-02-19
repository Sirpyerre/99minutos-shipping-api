package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/99minutos/shipping-system/internal/core/domain"
	"github.com/99minutos/shipping-system/internal/core/ports"
)

// ---------------------------------------------------------------------------
// In-memory stub repository
// ---------------------------------------------------------------------------

type stubShipmentRepo struct {
	byTracking     map[string]*domain.Shipment
	byIdempotency  map[string]*domain.Shipment
	lastFindFilter string // clientID passed to the last FindByTrackingNumber call
	createErr      error  // if set, Create returns this error
}

func newStubShipmentRepo() *stubShipmentRepo {
	return &stubShipmentRepo{
		byTracking:    make(map[string]*domain.Shipment),
		byIdempotency: make(map[string]*domain.Shipment),
	}
}

func (r *stubShipmentRepo) Create(_ context.Context, s *domain.Shipment) error {
	if r.createErr != nil {
		return r.createErr
	}
	clone := *s
	r.byTracking[s.TrackingNumber] = &clone
	if s.IdempotencyKey != "" {
		r.byIdempotency[s.IdempotencyKey] = &clone
	}
	return nil
}

func (r *stubShipmentRepo) FindByTrackingNumber(_ context.Context, trackingNumber, clientID string) (*domain.Shipment, error) {
	r.lastFindFilter = clientID
	s, ok := r.byTracking[trackingNumber]
	if !ok {
		return nil, domain.ErrShipmentNotFound
	}
	// Enforce client filter (mirrors the real Mongo query)
	if clientID != "" && s.ClientID != clientID {
		return nil, domain.ErrShipmentNotFound
	}
	clone := *s
	return &clone, nil
}

func (r *stubShipmentRepo) FindByIdempotencyKey(_ context.Context, key string) (*domain.Shipment, error) {
	s, ok := r.byIdempotency[key]
	if !ok {
		return nil, domain.ErrShipmentNotFound
	}
	clone := *s
	return &clone, nil
}

// List applies the same filters the real Mongo repo would use.
func (r *stubShipmentRepo) List(_ context.Context, f ports.ListShipmentsFilter) ([]*domain.Shipment, int64, error) {
	if r.createErr != nil {
		return nil, 0, r.createErr
	}

	var matched []*domain.Shipment
	for _, s := range r.byTracking {
		if f.ClientID != "" && s.ClientID != f.ClientID {
			continue
		}
		if f.Status != "" && string(s.Status) != f.Status {
			continue
		}
		if f.ServiceType != "" && s.ServiceType != f.ServiceType {
			continue
		}
		if !f.DateFrom.IsZero() && s.CreatedAt.Before(f.DateFrom) {
			continue
		}
		if !f.DateTo.IsZero() && s.CreatedAt.After(f.DateTo) {
			continue
		}
		if f.Search != "" {
			trackingMatch := strings.Contains(strings.ToLower(s.TrackingNumber), strings.ToLower(f.Search))
			nameMatch := strings.Contains(strings.ToLower(s.Sender.Name), strings.ToLower(f.Search))
			if !trackingMatch && !nameMatch {
				continue
			}
		}
		clone := *s
		matched = append(matched, &clone)
	}

	total := int64(len(matched))

	// Apply pagination
	limit := f.Limit
	if limit <= 0 {
		limit = len(matched)
	}
	skip := (f.Page - 1) * limit
	if skip < 0 {
		skip = 0
	}
	if skip > len(matched) {
		return []*domain.Shipment{}, total, nil
	}
	end := skip + limit
	if end > len(matched) {
		end = len(matched)
	}
	return matched[skip:end], total, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var discardLogger = zerolog.Nop()

func minimalInput(clientID, serviceType string) ports.CreateShipmentInput {
	return ports.CreateShipmentInput{
		Sender:      ports.SenderInput{Name: "Pedro", Email: "pedro@example.com", Phone: "+52"},
		Origin:      ports.AddressInput{Address: "Av 1", City: "CDMX", ZipCode: "06600"},
		Destination: ports.AddressInput{Address: "Calle 2", City: "Puebla", ZipCode: "72000"},
		Package: ports.PackageInput{
			WeightKg:      2.5,
			DeclaredValue: 100,
			Currency:      "MXN",
			Description:   "test",
		},
		ServiceType: serviceType,
		ClientID:    clientID,
	}
}

// ---------------------------------------------------------------------------
// CreateShipment tests
// ---------------------------------------------------------------------------

func TestShipmentService_Create_Success(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)

	result, err := svc.CreateShipment(context.Background(), minimalInput("client_1", "next_day"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(result.TrackingNumber, "99M-") {
		t.Errorf("tracking number format wrong: %s", result.TrackingNumber)
	}
	if result.Status != string(domain.StatusCreated) {
		t.Errorf("expected status %q, got %q", domain.StatusCreated, result.Status)
	}
	if result.AlreadyExisted {
		t.Error("expected AlreadyExisted=false for new shipment")
	}
	if result.CreatedAt.IsZero() {
		t.Error("CreatedAt must not be zero")
	}
	if result.EstimatedDelivery.IsZero() {
		t.Error("EstimatedDelivery must not be zero")
	}
}

func TestShipmentService_Create_SetsInitialStatusHistory(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)

	result, _ := svc.CreateShipment(context.Background(), minimalInput("client_1", "standard"))

	stored := repo.byTracking[result.TrackingNumber]
	if len(stored.StatusHistory) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(stored.StatusHistory))
	}
	if stored.StatusHistory[0].Status != domain.StatusCreated {
		t.Errorf("expected initial status %q, got %q", domain.StatusCreated, stored.StatusHistory[0].Status)
	}
	if stored.StatusHistory[0].Timestamp.IsZero() {
		t.Error("history timestamp must not be zero")
	}
}

func TestShipmentService_Create_StoresClientID(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)

	result, _ := svc.CreateShipment(context.Background(), minimalInput("client_42", "standard"))

	stored := repo.byTracking[result.TrackingNumber]
	if stored.ClientID != "client_42" {
		t.Errorf("expected client_id %q, got %q", "client_42", stored.ClientID)
	}
}

func TestShipmentService_Create_RepoError(t *testing.T) {
	repo := newStubShipmentRepo()
	repo.createErr = errors.New("db unavailable")
	svc := NewShipmentService(repo, discardLogger)

	_, err := svc.CreateShipment(context.Background(), minimalInput("client_1", "standard"))
	if err == nil {
		t.Fatal("expected error when repo fails, got nil")
	}
}

// ---------------------------------------------------------------------------
// Idempotency tests
// ---------------------------------------------------------------------------

func TestShipmentService_Create_IdempotencyReplay(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)

	input := minimalInput("client_1", "next_day")
	input.IdempotencyKey = "key-abc-123"

	first, err := svc.CreateShipment(context.Background(), input)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	second, err := svc.CreateShipment(context.Background(), input)
	if err != nil {
		t.Fatalf("second create (replay) failed: %v", err)
	}

	if second.TrackingNumber != first.TrackingNumber {
		t.Errorf("replay must return same tracking number: got %q, want %q", second.TrackingNumber, first.TrackingNumber)
	}
	if !second.AlreadyExisted {
		t.Error("replay must set AlreadyExisted=true")
	}
	// Only one shipment should be stored.
	if len(repo.byTracking) != 1 {
		t.Errorf("expected 1 stored shipment, got %d", len(repo.byTracking))
	}
}

func TestShipmentService_Create_NoIdempotencyKey_AlwaysCreates(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)

	_, _ = svc.CreateShipment(context.Background(), minimalInput("client_1", "standard"))
	_, _ = svc.CreateShipment(context.Background(), minimalInput("client_1", "standard"))

	if len(repo.byTracking) != 2 {
		t.Errorf("without idempotency key, each call must create a new shipment; got %d", len(repo.byTracking))
	}
}

// ---------------------------------------------------------------------------
// Estimated delivery tests
// ---------------------------------------------------------------------------

func TestShipmentService_Create_EstimatedDelivery(t *testing.T) {
	// Fix a reference time: 2026-02-19 10:00:00 UTC
	ref := time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		serviceType string
		wantDate    time.Time
	}{
		{"same_day", time.Date(2026, 2, 19, 18, 0, 0, 0, time.UTC)},
		{"next_day", time.Date(2026, 2, 20, 18, 0, 0, 0, time.UTC)},
		{"standard", time.Date(2026, 2, 22, 18, 0, 0, 0, time.UTC)},
		{"unknown", time.Date(2026, 2, 22, 18, 0, 0, 0, time.UTC)}, // defaults to +3 days
	}

	for _, tc := range cases {
		got := estimatedDelivery(tc.serviceType, ref)
		if !got.Equal(tc.wantDate) {
			t.Errorf("serviceType=%q: expected %v, got %v", tc.serviceType, tc.wantDate, got)
		}
	}
}

// ---------------------------------------------------------------------------
// GetShipment tests
// ---------------------------------------------------------------------------

func seedShipment(repo *stubShipmentRepo, trackingNumber, clientID string) *domain.Shipment {
	now := time.Now().UTC()
	s := &domain.Shipment{
		TrackingNumber:    trackingNumber,
		ClientID:          clientID,
		Status:            domain.StatusCreated,
		ServiceType:       "next_day",
		CreatedAt:         now,
		EstimatedDelivery: now.AddDate(0, 0, 1),
		Sender:            domain.Person{Name: "Pedro", Email: "pedro@example.com"},
		StatusHistory: []domain.StatusHistoryEntry{
			{Status: domain.StatusCreated, Timestamp: now},
		},
	}
	repo.byTracking[trackingNumber] = s
	return s
}

func TestShipmentService_Get_AdminSeesAll(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)
	seedShipment(repo, "99M-AAAABBBB", "client_1")

	_, err := svc.GetShipment(context.Background(), ports.GetShipmentInput{
		TrackingNumber: "99M-AAAABBBB",
		Role:           domain.RoleAdmin,
		ClientID:       "",
	})
	if err != nil {
		t.Fatalf("admin should see any shipment, got error: %v", err)
	}
	// Admin must not filter by clientID.
	if repo.lastFindFilter != "" {
		t.Errorf("admin query must not pass clientID filter, got %q", repo.lastFindFilter)
	}
}

func TestShipmentService_Get_ClientFiltersById(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)
	seedShipment(repo, "99M-AAAABBBB", "client_1")

	_, err := svc.GetShipment(context.Background(), ports.GetShipmentInput{
		TrackingNumber: "99M-AAAABBBB",
		Role:           domain.RoleClient,
		ClientID:       "client_1",
	})
	if err != nil {
		t.Fatalf("client should see own shipment, got error: %v", err)
	}
	// Client must pass its own clientID so the repo enforces ownership.
	if repo.lastFindFilter != "client_1" {
		t.Errorf("expected clientID filter %q, got %q", "client_1", repo.lastFindFilter)
	}
}

func TestShipmentService_Get_ClientCannotSeeOtherClientShipment(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)
	seedShipment(repo, "99M-AAAABBBB", "client_1")

	_, err := svc.GetShipment(context.Background(), ports.GetShipmentInput{
		TrackingNumber: "99M-AAAABBBB",
		Role:           domain.RoleClient,
		ClientID:       "client_999", // wrong client
	})
	if !errors.Is(err, domain.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound for unauthorized client, got %v", err)
	}
}

func TestShipmentService_Get_NotFound(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)

	_, err := svc.GetShipment(context.Background(), ports.GetShipmentInput{
		TrackingNumber: "99M-NOTEXIST",
		Role:           domain.RoleAdmin,
	})
	if !errors.Is(err, domain.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got %v", err)
	}
}

func TestShipmentService_Get_MapsDetailCorrectly(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)
	seeded := seedShipment(repo, "99M-DETAIL01", "client_1")

	detail, err := svc.GetShipment(context.Background(), ports.GetShipmentInput{
		TrackingNumber: "99M-DETAIL01",
		Role:           domain.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detail.TrackingNumber != seeded.TrackingNumber {
		t.Errorf("TrackingNumber: want %q, got %q", seeded.TrackingNumber, detail.TrackingNumber)
	}
	if detail.Status != string(seeded.Status) {
		t.Errorf("Status: want %q, got %q", seeded.Status, detail.Status)
	}
	if detail.ServiceType != seeded.ServiceType {
		t.Errorf("ServiceType: want %q, got %q", seeded.ServiceType, detail.ServiceType)
	}
	if !detail.CreatedAt.Equal(seeded.CreatedAt) {
		t.Errorf("CreatedAt: want %v, got %v", seeded.CreatedAt, detail.CreatedAt)
	}
	if len(detail.StatusHistory) != len(seeded.StatusHistory) {
		t.Fatalf("StatusHistory len: want %d, got %d", len(seeded.StatusHistory), len(detail.StatusHistory))
	}
	if detail.StatusHistory[0].Status != string(seeded.StatusHistory[0].Status) {
		t.Errorf("history[0].Status: want %q, got %q", seeded.StatusHistory[0].Status, detail.StatusHistory[0].Status)
	}
	if !detail.StatusHistory[0].Timestamp.Equal(seeded.StatusHistory[0].Timestamp) {
		t.Errorf("history[0].Timestamp: want %v, got %v", seeded.StatusHistory[0].Timestamp, detail.StatusHistory[0].Timestamp)
	}
}

func TestShipmentService_Get_MapsFullStatusHistory(t *testing.T) {
	repo := newStubShipmentRepo()
	svc := NewShipmentService(repo, discardLogger)

	now := time.Now().UTC()
	repo.byTracking["99M-HIST0001"] = &domain.Shipment{
		TrackingNumber: "99M-HIST0001",
		ClientID:       "client_1",
		Status:         domain.StatusInTransit,
		StatusHistory: []domain.StatusHistoryEntry{
			{Status: domain.StatusCreated, Timestamp: now.Add(-3 * time.Hour)},
			{Status: domain.StatusPickedUp, Timestamp: now.Add(-2 * time.Hour), Notes: "collected"},
			{Status: domain.StatusInWarehouse, Timestamp: now.Add(-1 * time.Hour)},
			{Status: domain.StatusInTransit, Timestamp: now},
		},
	}

	detail, err := svc.GetShipment(context.Background(), ports.GetShipmentInput{
		TrackingNumber: "99M-HIST0001",
		Role:           domain.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(detail.StatusHistory) != 4 {
		t.Fatalf("expected 4 history entries, got %d", len(detail.StatusHistory))
	}
	if detail.StatusHistory[1].Notes != "collected" {
		t.Errorf("expected notes %q on entry 1, got %q", "collected", detail.StatusHistory[1].Notes)
	}
}

// ---------------------------------------------------------------------------
// ListShipments tests
// ---------------------------------------------------------------------------

func seedViaService(t *testing.T, svc ports.ShipmentService, overrides func(*ports.CreateShipmentInput)) *ports.ShipmentResult {
t.Helper()
in := ports.CreateShipmentInput{
ClientID:    "client_001",
ServiceType: "next_day",
Sender:      ports.SenderInput{Name: "Pedro", Email: "p@e.com", Phone: "+521"},
Origin:      ports.AddressInput{Address: "A", City: "CDMX", ZipCode: "06600"},
Destination: ports.AddressInput{Address: "B", City: "Puebla", ZipCode: "72000"},
Package:     ports.PackageInput{WeightKg: 1},
}
if overrides != nil {
overrides(&in)
}
result, err := svc.CreateShipment(context.Background(), in)
if err != nil {
t.Fatalf("seed: %v", err)
}
return result
}

func TestListShipments_AdminSeesAll(t *testing.T) {
repo := newStubShipmentRepo()
svc := NewShipmentService(repo, zerolog.Nop())

seedViaService(t, svc, func(i *ports.CreateShipmentInput) { i.ClientID = "client_001" })
seedViaService(t, svc, func(i *ports.CreateShipmentInput) { i.ClientID = "client_002" })

res, err := svc.ListShipments(context.Background(), ports.ListShipmentsInput{
Role: "admin", ClientID: "", Page: 1, Limit: 10,
})
if err != nil {
t.Fatal(err)
}
if int(res.Total) != 2 {
t.Errorf("admin: expected 2 total, got %d", res.Total)
}
}

func TestListShipments_ClientSeesOwn(t *testing.T) {
repo := newStubShipmentRepo()
svc := NewShipmentService(repo, zerolog.Nop())

seedViaService(t, svc, func(i *ports.CreateShipmentInput) { i.ClientID = "client_001" })
seedViaService(t, svc, func(i *ports.CreateShipmentInput) { i.ClientID = "client_002" })

res, err := svc.ListShipments(context.Background(), ports.ListShipmentsInput{
Role: "client", ClientID: "client_001", Page: 1, Limit: 10,
})
if err != nil {
t.Fatal(err)
}
if int(res.Total) != 1 {
t.Errorf("client: expected 1, got %d", res.Total)
}
if res.Items[0].TrackingNumber == "" {
t.Error("expected a tracking number in result")
}
}

func TestListShipments_LimitCappedAt100(t *testing.T) {
repo := newStubShipmentRepo()
svc := NewShipmentService(repo, zerolog.Nop())

res, err := svc.ListShipments(context.Background(), ports.ListShipmentsInput{
Role: "admin", Limit: 999, Page: 1,
})
if err != nil {
t.Fatal(err)
}
if res.Limit != 100 {
t.Errorf("expected limit 100, got %d", res.Limit)
}
}

func TestListShipments_DefaultLimit(t *testing.T) {
repo := newStubShipmentRepo()
svc := NewShipmentService(repo, zerolog.Nop())

res, err := svc.ListShipments(context.Background(), ports.ListShipmentsInput{
Role: "admin", Limit: 0, Page: 0,
})
if err != nil {
t.Fatal(err)
}
if res.Limit != 20 {
t.Errorf("expected default limit 20, got %d", res.Limit)
}
}

func TestListShipments_PaginationMath(t *testing.T) {
repo := newStubShipmentRepo()
svc := NewShipmentService(repo, zerolog.Nop())

for i := 0; i < 5; i++ {
seedViaService(t, svc, nil)
}

res, err := svc.ListShipments(context.Background(), ports.ListShipmentsInput{
Role: "admin", Limit: 2, Page: 1,
})
if err != nil {
t.Fatal(err)
}
if res.Total != 5 {
t.Errorf("total: expected 5, got %d", res.Total)
}
if res.TotalPages != 3 {
t.Errorf("total_pages: expected 3, got %d", res.TotalPages)
}
if res.Page != 1 {
t.Errorf("page: expected 1, got %d", res.Page)
}
if len(res.Items) != 2 {
t.Errorf("items: expected 2, got %d", len(res.Items))
}
}

func TestListShipments_FilterByStatus(t *testing.T) {
repo := newStubShipmentRepo()
svc := NewShipmentService(repo, zerolog.Nop())

seedViaService(t, svc, nil) // status=created

res, err := svc.ListShipments(context.Background(), ports.ListShipmentsInput{
Role: "admin", Status: "created", Page: 1, Limit: 10,
})
if err != nil {
t.Fatal(err)
}
if int(res.Total) != 1 {
t.Errorf("filter by created: expected 1, got %d", res.Total)
}

res2, _ := svc.ListShipments(context.Background(), ports.ListShipmentsInput{
Role: "admin", Status: "delivered", Page: 1, Limit: 10,
})
if int(res2.Total) != 0 {
t.Errorf("filter by delivered: expected 0, got %d", res2.Total)
}
}

func TestListShipments_FilterByServiceType(t *testing.T) {
repo := newStubShipmentRepo()
svc := NewShipmentService(repo, zerolog.Nop())

seedViaService(t, svc, func(i *ports.CreateShipmentInput) { i.ServiceType = "next_day" })
seedViaService(t, svc, func(i *ports.CreateShipmentInput) { i.ServiceType = "same_day" })

res, err := svc.ListShipments(context.Background(), ports.ListShipmentsInput{
Role: "admin", ServiceType: "same_day", Page: 1, Limit: 10,
})
if err != nil {
t.Fatal(err)
}
if int(res.Total) != 1 {
t.Errorf("filter by same_day: expected 1, got %d", res.Total)
}
}

func TestListShipments_SearchBySenderName(t *testing.T) {
repo := newStubShipmentRepo()
svc := NewShipmentService(repo, zerolog.Nop())

seedViaService(t, svc, func(i *ports.CreateShipmentInput) { i.Sender.Name = "Pedro García" })
seedViaService(t, svc, func(i *ports.CreateShipmentInput) { i.Sender.Name = "Ana Torres" })

res, err := svc.ListShipments(context.Background(), ports.ListShipmentsInput{
Role: "admin", Search: "pedro", Page: 1, Limit: 10,
})
if err != nil {
t.Fatal(err)
}
if int(res.Total) != 1 {
t.Errorf("search: expected 1, got %d", res.Total)
}
}

func TestListShipments_DateRangeFilter(t *testing.T) {
repo := newStubShipmentRepo()
svc := NewShipmentService(repo, zerolog.Nop())

seedViaService(t, svc, nil)

yesterday := time.Now().UTC().AddDate(0, 0, -1)
tomorrow := time.Now().UTC().AddDate(0, 0, 1)

res, err := svc.ListShipments(context.Background(), ports.ListShipmentsInput{
Role: "admin", DateFrom: yesterday, DateTo: tomorrow, Page: 1, Limit: 10,
})
if err != nil {
t.Fatal(err)
}
if int(res.Total) != 1 {
t.Errorf("date range: expected 1, got %d", res.Total)
}
}
