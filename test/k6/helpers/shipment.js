import { post, get, parse, withQuery } from './http.js';
import { jsonParams } from './http.js';
import http from 'k6/http';
import { BASE_URL } from '../config.js';
import { fail } from 'k6';

export const defaultShipmentPayload = {
  sender: {
    name: 'Pedro Test',
    email: 'pedro@test.com',
    phone: '+525512345678',
  },
  origin: {
    address: 'Av. Reforma 222, Juárez',
    city: 'CDMX',
    zip_code: '06600',
    coordinates: { lat: 19.427, lng: -99.167 },
  },
  destination: {
    address: 'Calle Falsa 123',
    city: 'Puebla',
    zip_code: '72000',
    coordinates: { lat: 19.041, lng: -98.206 },
  },
  package: {
    weight_kg: 2.5,
    dimensions: { length_cm: 30, width_cm: 20, height_cm: 15 },
    description: 'Documentos de prueba',
    declared_value: 2500.50,
    currency: 'MXN',
  },
  service_type: 'next_day',
};

export function createShipment(token, overrides = {}, idempotencyKey = '') {
  const payload = Object.assign({}, defaultShipmentPayload, overrides);
  const extra = idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : {};
  return post(`${BASE_URL}/v1/shipments`, payload, token, extra);
}

export function getShipment(token, trackingNumber) {
  return get(`${BASE_URL}/v1/shipments/${trackingNumber}`, token);
}

export function listShipments(token, params = {}) {
  return get(withQuery(`${BASE_URL}/v1/shipments`, params), token);
}

/**
 * Creates a shipment and returns its tracking_number; calls fail() on error.
 */
export function setupShipment(token, overrides = {}) {
  const res = createShipment(token, overrides);
  if (res.status !== 201) {
    fail(`setup: create shipment failed (${res.status}): ${res.body}`);
  }
  return parse(res).tracking_number;
}
