package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/rs/zerolog"

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

	return &ports.ShipmentResult{
		TrackingNumber:    shipment.TrackingNumber,
		Status:            string(shipment.Status),
		CreatedAt:         shipment.CreatedAt,
		EstimatedDelivery: shipment.EstimatedDelivery,
	}, nil
}

// generateTrackingNumber returns a unique tracking number in the format 99M-XXXXXXXX.
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
