/**
 * K6 end-to-end test — Full shipment lifecycle
 *
 * Scenario:
 *   1. Register client + admin users
 *   2. Client creates a shipment
 *   3. Driver app sends events: created → picked_up → in_warehouse → in_transit → delivered
 *   4. After each transition, verify status + history length
 *   5. Admin can query the shipment at any point
 *   6. Attempt a post-delivery transition (should be rejected / no-op)
 *
 * Run: k6 run test/k6/e2e.test.js
 */
import { check, group, sleep } from 'k6';
import { options as baseOptions, EVENT_SETTLE_MS } from './config.js';
import { setupUser, setupAdmin } from './helpers/auth.js';
import { createShipment, getShipment, listShipments } from './helpers/shipment.js';
import { sendEvent, buildEvent } from './helpers/events.js';
import { waitForStatus } from './helpers/poll.js';
import { parse } from './helpers/http.js';
import { fail } from 'k6';

export const options = {
  ...baseOptions,
  thresholds: {
    ...baseOptions.thresholds,
    'checks{scenario:e2e}': ['rate==1.0'],
  },
};

export function setup() {
  const ts = Date.now();
  const client = setupUser(`e2e_c_${ts}`);
  const admin  = setupAdmin(`e2e_adm_${ts}`);
  return { client, admin };
}

export default function (data) {
  const { client, admin } = data;

  // ── Step 1: Create shipment ──────────────────────────────────────────────

  let trackingNumber;

  group('Step 1: Create shipment', () => {
    const res = createShipment(client.token, {
      sender: { name: 'Ana Logística', email: 'ana@99minutos.com', phone: '+525555555555' },
      service_type: 'next_day',
    });
    const body = parse(res);

    check(res, {
      'shipment created (201)':      r => r.status === 201,
      'tracking_number assigned':    r => !!parse(r)?.tracking_number,
      'initial status is created':   r => parse(r)?.status === 'created',
      'estimated_delivery present':  r => !!parse(r)?.estimated_delivery,
    });

    trackingNumber = body?.tracking_number;
    if (!trackingNumber) fail('e2e: no tracking_number in create response');
  });

  // ── Step 2: Verify initial GET ───────────────────────────────────────────

  group('Step 2: Verify initial shipment state', () => {
    const res = getShipment(client.token, trackingNumber);
    const body = parse(res);
    check(res, {
      'GET returns 200':          r => r.status === 200,
      'status is created':        r => parse(r)?.status === 'created',
      'history has 1 entry':      r => parse(r)?.status_history?.length === 1,
      'sender name matches':      r => parse(r)?.sender?.name === 'Ana Logística',
    });

    // Admin can also see it
    const adminRes = getShipment(admin.token, trackingNumber);
    check(adminRes, { 'admin can view shipment': r => r.status === 200 });
  });

  // ── Step 3: Transitions through the full state machine ───────────────────

  const transitions = [
    { status: 'picked_up',    expectedHistory: 2 },
    { status: 'in_warehouse', expectedHistory: 3 },
    { status: 'in_transit',   expectedHistory: 4 },
    { status: 'delivered',    expectedHistory: 5 },
  ];

  for (const { status, expectedHistory } of transitions) {
    group(`Step 3: Transition → ${status}`, () => {
      const res = sendEvent(client.token, buildEvent(trackingNumber, status, {
        source: 'driver_app',
        location: { lat: 19.4326, lng: -99.1332 },
      }));

      check(res, { [`event ${status} accepted (202)`]: r => r.status === 202 });

      const updated = waitForStatus(client.token, trackingNumber, status, 14, 0.5);
      check(updated, {
        [`status is now ${status}`]:             s => s?.status === status,
        [`history has ${expectedHistory} entries`]: s => s?.status_history?.length === expectedHistory,
        [`last history entry is ${status}`]:     s => s?.status_history?.slice(-1)[0]?.status === status,
        'location stored in history notes':       s => !!s?.status_history?.slice(-1)[0]?.notes,
      });
    });
  }

  // ── Step 4: Post-delivery — no further transitions allowed ───────────────

  group('Step 4: Post-delivery transition is a no-op (invalid transition)', () => {
    const res = sendEvent(client.token, buildEvent(trackingNumber, 'cancelled'));
    check(res, {
      '202 accepted (async rejection)': r => r.status === 202,
    });

    sleep(EVENT_SETTLE_MS);

    const after = parse(getShipment(client.token, trackingNumber));
    check(after, {
      'status remains delivered': s => s?.status === 'delivered',
      'history length unchanged (5)': s => s?.status_history?.length === 5,
    });
  });

  // ── Step 5: List endpoint reflects final state ───────────────────────────

  group('Step 5: List shows shipment with status delivered', () => {
    const res = listShipments(client.token, { status: 'delivered' });
    const body = parse(res);
    check(res, {
      'status 200':                   r => r.status === 200,
      'delivered shipment in list':   () =>
        (body?.data ?? []).some(s => s.tracking_number === trackingNumber),
      'list item has no status_history (lightweight)': () =>
        body?.data?.[0]?.status_history === undefined,
    });
  });

  // ── Step 6: RBAC — other client cannot access this shipment ──────────────

  group('Step 6: RBAC — other user cannot see this shipment', () => {
    const ts = Date.now();
    const other = setupUser(`e2e_other_${ts}`);
    const res = getShipment(other.token, trackingNumber);
    check(res, { 'status 404 for other client': r => r.status === 404 });
  });
}
