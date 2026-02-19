package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	apimetrics "github.com/99minutos/shipping-system/internal/api/metrics"
	"github.com/99minutos/shipping-system/internal/core/domain"
	"github.com/99minutos/shipping-system/internal/core/ports"
)

type ShipmentService struct {
	repo   ports.ShipmentRepository
	logger zerolog.Logger
}

func NewShipmentService(repo ports.ShipmentRepository, logger zerolog.Logger) *ShipmentService {
	return &ShipmentService{repo: repo, logger: logger}
}

// CreateShipment creates a new shipment. If an idempotency key is provided and
// already seen, the previously created shipment is returned without side effects.
func (s *ShipmentService) CreateShipment(ctx context.Context, input ports.CreateShipmentInput) (*ports.ShipmentResult, error) {
	if input.IdempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, input.IdempotencyKey)
		if err == nil && existing != nil {
			s.logger.Info().Str("idempotency_key", input.IdempotencyKey).Str("tracking_number", existing.TrackingNumber).Msg("idempotent replay")
			return &ports.ShipmentResult{
				TrackingNumber:    existing.TrackingNumber,
				Status:            string(existing.Status),
				CreatedAt:         existing.CreatedAt,
				EstimatedDelivery: existing.EstimatedDelivery,
				AlreadyExisted:    true,
			}, nil
		}
	}

	now := time.Now().UTC()
	shipment := &domain.Shipment{
		TrackingNumber:    generateTrackingNumber(),
		ClientID:          input.ClientID,
		Status:            domain.StatusCreated,
		ServiceType:       input.ServiceType,
		CreatedAt:         now,
		EstimatedDelivery: estimatedDelivery(input.ServiceType, now),
		IdempotencyKey:    input.IdempotencyKey,
		StatusHistory: []domain.StatusHistoryEntry{
			{Status: domain.StatusCreated, Timestamp: now},
		},
		Sender: domain.Person{
			Name:  input.Sender.Name,
			Email: input.Sender.Email,
			Phone: input.Sender.Phone,
		},
		Origin: domain.Address{
			Address: input.Origin.Address,
			City:    input.Origin.City,
			ZipCode: input.Origin.ZipCode,
			Coordinates: domain.Coordinates{
				Lat: input.Origin.Coordinates.Lat,
				Lng: input.Origin.Coordinates.Lng,
			},
		},
		Destination: domain.Address{
			Address: input.Destination.Address,
			City:    input.Destination.City,
			ZipCode: input.Destination.ZipCode,
			Coordinates: domain.Coordinates{
				Lat: input.Destination.Coordinates.Lat,
				Lng: input.Destination.Coordinates.Lng,
			},
		},
		Package: domain.Package{
			WeightKg: input.Package.WeightKg,
			Dimensions: domain.Dimensions{
				LengthCm: input.Package.Dimensions.LengthCm,
				WidthCm:  input.Package.Dimensions.WidthCm,
				HeightCm: input.Package.Dimensions.HeightCm,
			},
			Description:   input.Package.Description,
			DeclaredValue: input.Package.DeclaredValue,
			Currency:      input.Package.Currency,
		},
	}

	if err := s.repo.Create(ctx, shipment); err != nil {
		s.logger.Error().Err(err).Msg("failed to create shipment")
		return nil, err
	}

	s.logger.Info().Str("tracking_number", shipment.TrackingNumber).Str("client_id", input.ClientID).Msg("shipment created")
	apimetrics.ShipmentsCreatedTotal.WithLabelValues(input.ServiceType).Inc()

	return &ports.ShipmentResult{
		TrackingNumber:    shipment.TrackingNumber,
		Status:            string(shipment.Status),
		CreatedAt:         shipment.CreatedAt,
		EstimatedDelivery: shipment.EstimatedDelivery,
	}, nil
}

// GetShipment retrieves a shipment with its full status history.
// Clients can only see their own shipments; admins see all.
func (s *ShipmentService) GetShipment(ctx context.Context, input ports.GetShipmentInput) (*ports.ShipmentDetail, error) {
	// For "client" role, pass clientID so the repo enforces ownership at query level.
	filterClientID := ""
	if input.Role == domain.RoleClient {
		filterClientID = input.ClientID
	}

	shipment, err := s.repo.FindByTrackingNumber(ctx, input.TrackingNumber, filterClientID)
	if err != nil {
		return nil, err // ErrShipmentNotFound is returned as-is
	}

	history := make([]ports.StatusHistoryItem, len(shipment.StatusHistory))
	for i, h := range shipment.StatusHistory {
		history[i] = ports.StatusHistoryItem{
			Status:    string(h.Status),
			Timestamp: h.Timestamp,
			Notes:     h.Notes,
		}
	}

	return &ports.ShipmentDetail{
		TrackingNumber:    shipment.TrackingNumber,
		Status:            string(shipment.Status),
		ServiceType:       shipment.ServiceType,
		CreatedAt:         shipment.CreatedAt,
		EstimatedDelivery: shipment.EstimatedDelivery,
		Sender: ports.SenderInput{
			Name:  shipment.Sender.Name,
			Email: shipment.Sender.Email,
			Phone: shipment.Sender.Phone,
		},
		Origin: ports.AddressInput{
			Address: shipment.Origin.Address,
			City:    shipment.Origin.City,
			ZipCode: shipment.Origin.ZipCode,
			Coordinates: ports.CoordinatesInput{
				Lat: shipment.Origin.Coordinates.Lat,
				Lng: shipment.Origin.Coordinates.Lng,
			},
		},
		Destination: ports.AddressInput{
			Address: shipment.Destination.Address,
			City:    shipment.Destination.City,
			ZipCode: shipment.Destination.ZipCode,
			Coordinates: ports.CoordinatesInput{
				Lat: shipment.Destination.Coordinates.Lat,
				Lng: shipment.Destination.Coordinates.Lng,
			},
		},
		Package: ports.PackageInput{
			WeightKg: shipment.Package.WeightKg,
			Dimensions: ports.DimensionsInput{
				LengthCm: shipment.Package.Dimensions.LengthCm,
				WidthCm:  shipment.Package.Dimensions.WidthCm,
				HeightCm: shipment.Package.Dimensions.HeightCm,
			},
			Description:   shipment.Package.Description,
			DeclaredValue: shipment.Package.DeclaredValue,
			Currency:      shipment.Package.Currency,
		},
		StatusHistory: history,
	}, nil
}

const (
	defaultLimit = 20
	maxLimit     = 100
)

// ListShipments returns a paginated, filtered list of shipments.
// Clients are always scoped to their own shipments; admins see all.
func (s *ShipmentService) ListShipments(ctx context.Context, input ports.ListShipmentsInput) (*ports.ListShipmentsResult, error) {
	// Normalise pagination.
	limit := input.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}

	// RBAC: clients can only see their own shipments.
	clientIDFilter := ""
	if input.Role == domain.RoleClient {
		clientIDFilter = input.ClientID
	}

	filter := ports.ListShipmentsFilter{
		ClientID:    clientIDFilter,
		Status:      input.Status,
		ServiceType: input.ServiceType,
		Search:      input.Search,
		DateFrom:    input.DateFrom,
		DateTo:      input.DateTo,
		Page:        page,
		Limit:       limit,
	}

	shipments, total, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list shipments")
		return nil, err
	}

	items := make([]ports.ShipmentSummary, len(shipments))
	for i, sh := range shipments {
		items[i] = ports.ShipmentSummary{
			TrackingNumber:    sh.TrackingNumber,
			Status:            string(sh.Status),
			ServiceType:       sh.ServiceType,
			ClientID:          sh.ClientID,
			CreatedAt:         sh.CreatedAt,
			EstimatedDelivery: sh.EstimatedDelivery,
			Sender: ports.SenderInput{
				Name:  sh.Sender.Name,
				Email: sh.Sender.Email,
				Phone: sh.Sender.Phone,
			},
			Origin: ports.AddressInput{
				Address: sh.Origin.Address,
				City:    sh.Origin.City,
				ZipCode: sh.Origin.ZipCode,
				Coordinates: ports.CoordinatesInput{
					Lat: sh.Origin.Coordinates.Lat,
					Lng: sh.Origin.Coordinates.Lng,
				},
			},
			Destination: ports.AddressInput{
				Address: sh.Destination.Address,
				City:    sh.Destination.City,
				ZipCode: sh.Destination.ZipCode,
				Coordinates: ports.CoordinatesInput{
					Lat: sh.Destination.Coordinates.Lat,
					Lng: sh.Destination.Coordinates.Lng,
				},
			},
		}
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &ports.ListShipmentsResult{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}
func generateTrackingNumber() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// fallback: use current nanoseconds
		return fmt.Sprintf("99M-%08X", time.Now().UnixNano()&0xFFFFFFFF)
	}
	return fmt.Sprintf("99M-%08X", b)
}

// estimatedDelivery calculates the estimated delivery time based on service type.
func estimatedDelivery(serviceType string, from time.Time) time.Time {
	base := time.Date(from.Year(), from.Month(), from.Day(), 18, 0, 0, 0, time.UTC)
	switch serviceType {
	case "same_day":
		return base
	case "next_day":
		return base.AddDate(0, 0, 1)
	default: // "standard" or unknown → 3 days
		return base.AddDate(0, 0, 3)
	}
}
