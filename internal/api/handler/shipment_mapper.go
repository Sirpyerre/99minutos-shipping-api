package handler

import (
	"github.com/99minutos/shipping-system/internal/core/ports"
)

// --- Request → Service input ---

func toCreateInput(req createShipmentRequest, clientID, idempotencyKey string) ports.CreateShipmentInput {
	return ports.CreateShipmentInput{
		Sender: ports.SenderInput{
			Name:  req.Sender.Name,
			Email: req.Sender.Email,
			Phone: req.Sender.Phone,
		},
		Origin:      toAddressInput(req.Origin),
		Destination: toAddressInput(req.Destination),
		Package:     toPackageInput(req.Package),
		ServiceType:    req.ServiceType,
		ClientID:       clientID,
		IdempotencyKey: idempotencyKey,
	}
}

func toAddressInput(a addressRequest) ports.AddressInput {
	return ports.AddressInput{
		Address: a.Address,
		City:    a.City,
		ZipCode: a.ZipCode,
		Coordinates: ports.CoordinatesInput{
			Lat: a.Coordinates.Lat,
			Lng: a.Coordinates.Lng,
		},
	}
}

func toPackageInput(p packageRequest) ports.PackageInput {
	return ports.PackageInput{
		WeightKg: p.WeightKg,
		Dimensions: ports.DimensionsInput{
			LengthCm: p.Dimensions.LengthCm,
			WidthCm:  p.Dimensions.WidthCm,
			HeightCm: p.Dimensions.HeightCm,
		},
		Description:   p.Description,
		DeclaredValue: p.DeclaredValue,
		Currency:      p.Currency,
	}
}

// --- Service result → HTTP response ---

func toCreateResponse(r *ports.ShipmentResult) createShipmentResponse {
	return createShipmentResponse{
		TrackingNumber:    r.TrackingNumber,
		Status:            r.Status,
		CreatedAt:         r.CreatedAt.UTC(),
		EstimatedDelivery: r.EstimatedDelivery.UTC(),
		Links: shipmentLinks{
			Self:   "/shipments/" + r.TrackingNumber,
			Events: "/events/" + r.TrackingNumber,
		},
	}
}

func toGetResponse(d *ports.ShipmentDetail) getShipmentResponse {
	return getShipmentResponse{
		TrackingNumber:    d.TrackingNumber,
		Status:            d.Status,
		ServiceType:       d.ServiceType,
		CreatedAt:         d.CreatedAt.UTC(),
		EstimatedDelivery: d.EstimatedDelivery.UTC(),
		Sender: senderResponse{
			Name:  d.Sender.Name,
			Email: d.Sender.Email,
			Phone: d.Sender.Phone,
		},
		Origin:        toAddressResponse(d.Origin),
		Destination:   toAddressResponse(d.Destination),
		Package:       toPackageResponse(d.Package),
		StatusHistory: toStatusHistoryResponse(d.StatusHistory),
		Links: shipmentLinks{
			Self:   "/shipments/" + d.TrackingNumber,
			Events: "/events/" + d.TrackingNumber,
		},
	}
}

func toAddressResponse(a ports.AddressInput) addressResponse {
	return addressResponse{
		Address: a.Address,
		City:    a.City,
		ZipCode: a.ZipCode,
		Coordinates: coordinatesResponse{
			Lat: a.Coordinates.Lat,
			Lng: a.Coordinates.Lng,
		},
	}
}

func toPackageResponse(p ports.PackageInput) packageResponse {
	return packageResponse{
		WeightKg: p.WeightKg,
		Dimensions: dimensionsResponse{
			LengthCm: p.Dimensions.LengthCm,
			WidthCm:  p.Dimensions.WidthCm,
			HeightCm: p.Dimensions.HeightCm,
		},
		Description:   p.Description,
		DeclaredValue: p.DeclaredValue,
		Currency:      p.Currency,
	}
}

func toStatusHistoryResponse(items []ports.StatusHistoryItem) []statusHistoryItemResponse {
	out := make([]statusHistoryItemResponse, len(items))
	for i, item := range items {
		out[i] = statusHistoryItemResponse{
			Status:    item.Status,
			Timestamp: item.Timestamp.UTC(),
			Notes:     item.Notes,
		}
	}
	return out
}

func toListResponse(r *ports.ListShipmentsResult) listShipmentsResponse {
	items := make([]shipmentSummaryResponse, len(r.Items))
	for i, s := range r.Items {
		items[i] = toSummaryResponse(s)
	}
	return listShipmentsResponse{
		Data: items,
		Pagination: paginationResponse{
			Total:      r.Total,
			Page:       r.Page,
			Limit:      r.Limit,
			TotalPages: r.TotalPages,
		},
	}
}

func toSummaryResponse(s ports.ShipmentSummary) shipmentSummaryResponse {
	return shipmentSummaryResponse{
		TrackingNumber:    s.TrackingNumber,
		Status:            s.Status,
		ServiceType:       s.ServiceType,
		CreatedAt:         s.CreatedAt.UTC(),
		EstimatedDelivery: s.EstimatedDelivery.UTC(),
		Sender: senderResponse{
			Name:  s.Sender.Name,
			Email: s.Sender.Email,
			Phone: s.Sender.Phone,
		},
		Origin:      toAddressResponse(s.Origin),
		Destination: toAddressResponse(s.Destination),
		Links: shipmentLinks{
			Self:   "/shipments/" + s.TrackingNumber,
			Events: "/events/" + s.TrackingNumber,
		},
	}
}
