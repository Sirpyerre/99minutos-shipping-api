import { post } from './http.js';
import { BASE_URL } from '../config.js';

export function sendEvent(token, event) {
  return post(`${BASE_URL}/v1/events`, event, token);
}

export function sendBatch(token, events) {
  return post(`${BASE_URL}/v1/events/batch`, events, token);
}

/**
 * Builds a tracking event payload.
 * @param {string} trackingNumber
 * @param {string} status
 * @param {Object} [overrides]
 */
export function buildEvent(trackingNumber, status, overrides = {}) {
  return {
    tracking_number: trackingNumber,
    status,
    timestamp: new Date().toISOString(),
    source: 'k6_test',
    location: { lat: 19.4326, lng: -99.1332 },
    ...overrides,
  };
}
