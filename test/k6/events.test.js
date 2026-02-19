/**
 * K6 integration tests — Event endpoints
 *
 * Covers:
 *   POST /v1/events         — happy path, invalid status, missing fields, auth
 *   POST /v1/events/batch   — happy path, empty batch, partial validation
 *   Idempotency             — duplicate event is silently ignored
 *   State machine           — invalid transition is accepted (202) but not applied
 *
 * Run: k6 run test/k6/events.test.js
 */
import { check, group, sleep } from 'k6';
import { options as baseOptions, EVENT_SETTLE_MS } from './config.js';
import { setupUser } from './helpers/auth.js';
import { setupShipment, getShipment } from './helpers/shipment.js';
import { sendEvent, sendBatch, buildEvent } from './helpers/events.js';
import { waitForStatus } from './helpers/poll.js';
import { parse } from './helpers/http.js';

export const options = baseOptions;

export function setup() {
  const ts = Date.now();
  const client = setupUser(`evt_c_${ts}`);
  // Each test group that transitions status needs its own fresh shipment
  const trackingHappy    = setupShipment(client.token);
  const trackingDedup    = setupShipment(client.token);
  const trackingBatch    = setupShipment(client.token);
  const trackingInvalid  = setupShipment(client.token); // for invalid-transition test
  return { client, trackingHappy, trackingDedup, trackingBatch, trackingInvalid };
}

export default function (data) {
  const { client, trackingHappy, trackingDedup, trackingBatch, trackingInvalid } = data;
  const token = client.token;

  // ───────────────────────────── POST /v1/events — single event ────────────

  group('POST /v1/events — happy path (created → picked_up)', () => {
    const res = sendEvent(token, buildEvent(trackingHappy, 'picked_up'));
    check(res, {
      'status 202': r => r.status === 202,
      'message accepted': r => !!parse(r)?.message,
    });

    // Async: poll until shipment reflects new status
    const updated = waitForStatus(token, trackingHappy, 'picked_up');
    check(updated, {
      'status updated to picked_up': s => s?.status === 'picked_up',
      'status_history has 2 entries': s => s?.status_history?.length === 2,
      'latest history entry is picked_up': s =>
        s?.status_history?.slice(-1)[0]?.status === 'picked_up',
    });
  });

  group('POST /v1/events — second transition (picked_up → in_warehouse)', () => {
    // Depends on previous group having advanced trackingHappy to picked_up
    const res = sendEvent(token, buildEvent(trackingHappy, 'in_warehouse'));
    check(res, { 'status 202': r => r.status === 202 });

    const updated = waitForStatus(token, trackingHappy, 'in_warehouse');
    check(updated, {
      'status updated to in_warehouse': s => s?.status === 'in_warehouse',
    });
  });

  group('POST /v1/events — invalid status value → 422', () => {
    const res = sendEvent(token, buildEvent(trackingHappy, 'flying'));
    check(res, {
      'status 422': r => r.status === 422,
      'error present': r => !!parse(r)?.error,
    });
  });

  group('POST /v1/events — missing tracking_number → 422', () => {
    const res = sendEvent(token, {
      status: 'picked_up',
      timestamp: new Date().toISOString(),
      source: 'test',
    });
    check(res, { 'status 422': r => r.status === 422 });
  });

  group('POST /v1/events — missing source → 422', () => {
    const res = sendEvent(token, {
      tracking_number: trackingHappy,
      status: 'picked_up',
      timestamp: new Date().toISOString(),
    });
    check(res, { 'status 422': r => r.status === 422 });
  });

  group('POST /v1/events — missing timestamp → 422', () => {
    const res = sendEvent(token, {
      tracking_number: trackingHappy,
      status: 'picked_up',
      source: 'test',
    });
    check(res, { 'status 422': r => r.status === 422 });
  });

  group('POST /v1/events — invalid transition accepted (202) but status unchanged', () => {
    // trackingInvalid is still in "created" state; "delivered" is not a valid transition
    const before = getShipment(token, trackingInvalid);
    const statusBefore = parse(before)?.status;

    const res = sendEvent(token, buildEvent(trackingInvalid, 'delivered'));
    check(res, {
      'status 202 (async — rejection happens in worker)': r => r.status === 202,
    });

    sleep(EVENT_SETTLE_MS);

    const after = getShipment(token, trackingInvalid);
    check(after, {
      'status unchanged after invalid transition': r => parse(r)?.status === statusBefore,
    });
  });

  group('POST /v1/events — no auth → 401', () => {
    const res = sendEvent('', buildEvent(trackingHappy, 'picked_up'));
    check(res, { 'status 401': r => r.status === 401 });
  });

  // ──────────────────────────────────────── Idempotency ────────────────────

  group('POST /v1/events — duplicate event is silently ignored', () => {
    const timestamp = new Date().toISOString();
    const event = buildEvent(trackingDedup, 'picked_up', { timestamp });

    // First event transitions created → picked_up
    sendEvent(token, event);
    const first = waitForStatus(token, trackingDedup, 'picked_up');
    check(first, { 'first event applied': s => s?.status === 'picked_up' });

    const historyLenAfterFirst = first?.status_history?.length ?? 0;

    // Second identical event must be deduplicated — no extra history entry
    const res2 = sendEvent(token, event);
    check(res2, { 'duplicate returns 202': r => r.status === 202 });
    sleep(EVENT_SETTLE_MS);

    const after = parse(getShipment(token, trackingDedup));
    check(after, {
      'history length unchanged after duplicate': s =>
        s?.status_history?.length === historyLenAfterFirst,
    });
  });

  // ────────────────────────────────── POST /v1/events/batch ────────────────

  group('POST /v1/events/batch — happy path (2 events, sequential transitions)', () => {
    // trackingBatch is in "created"; advance it through picked_up → in_warehouse in one batch
    const ts = new Date();
    const t1 = new Date(ts.getTime()).toISOString();
    const t2 = new Date(ts.getTime() + 1000).toISOString(); // 1 second later

    const events = [
      buildEvent(trackingBatch, 'picked_up',    { timestamp: t1 }),
      buildEvent(trackingBatch, 'in_warehouse', { timestamp: t2 }),
    ];

    const res = sendBatch(token, events);
    check(res, {
      'status 202':   r => r.status === 202,
      'count is 2':   r => parse(r)?.count === 2,
      'message present': r => !!parse(r)?.message,
    });

    const updated = waitForStatus(token, trackingBatch, 'in_warehouse', 16, 0.5);
    check(updated, {
      'status advanced to in_warehouse': s => s?.status === 'in_warehouse',
      'history has 3 entries':           s => s?.status_history?.length === 3,
    });
  });

  group('POST /v1/events/batch — empty array → 400', () => {
    const res = sendBatch(token, []);
    check(res, {
      'status 400': r => r.status === 400,
      'error present': r => !!parse(r)?.error,
    });
  });

  group('POST /v1/events/batch — one invalid event in batch → 422', () => {
    const events = [
      buildEvent(trackingBatch, 'picked_up'),  // would be invalid transition here, but validation is structural
      { tracking_number: '', status: 'picked_up', timestamp: new Date().toISOString(), source: 'test' },
    ];
    const res = sendBatch(token, events);
    // Empty tracking_number fails the `required` validator → 422
    check(res, { 'status 422': r => r.status === 422 });
  });

  group('POST /v1/events/batch — no auth → 401', () => {
    const res = sendBatch('', [buildEvent(trackingBatch, 'picked_up')]);
    check(res, { 'status 401': r => r.status === 401 });
  });

  group('POST /v1/events/batch — non-array body → 400', () => {
    const res = sendBatch(token, { not: 'an array' });
    check(res, { 'status 4xx': r => r.status >= 400 });
  });
}
