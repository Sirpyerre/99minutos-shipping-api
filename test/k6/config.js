/** @type {import('k6/options').Options} */
export const options = {
  vus: 1,
  iterations: 1,
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<3000'],
    checks: ['rate==1.0'], // all checks must pass
  },
};

export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

/** Seconds to wait after sending an event before polling for the new status. */
export const EVENT_SETTLE_MS = 1.5;
