import { sleep } from 'k6';
import { getShipment } from './shipment.js';
import { parse } from './http.js';

/**
 * Polls GET /v1/shipments/:trackingNumber until status equals expectedStatus
 * or retries are exhausted. Returns the shipment body or null on timeout.
 *
 * @param {string} token
 * @param {string} trackingNumber
 * @param {string} expectedStatus
 * @param {number} [maxRetries=12]
 * @param {number} [intervalSec=0.5]
 */
export function waitForStatus(token, trackingNumber, expectedStatus, maxRetries = 12, intervalSec = 0.5) {
  for (let i = 0; i < maxRetries; i++) {
    const res = getShipment(token, trackingNumber);
    if (res.status === 200) {
      const body = parse(res);
      if (body && body.status === expectedStatus) return body;
    }
    sleep(intervalSec);
  }
  return null;
}
