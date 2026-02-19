/**
 * K6 integration tests — Shipment endpoints
 *
 * Covers:
 *   POST /v1/shipments   — happy path, idempotency, validation, auth
 *   GET  /v1/shipments/:tracking_number — happy path, not found, RBAC
 *   GET  /v1/shipments   — pagination, filters, RBAC isolation
 *
 * Run: k6 run test/k6/shipments.test.js
 */
import { check, group, sleep } from 'k6';
import { BASE_URL, options as baseOptions } from './config.js';
import { setupUser, setupAdmin } from './helpers/auth.js';
import {
  createShipment,
  getShipment,
  listShipments,
  setupShipment,
  defaultShipmentPayload,
} from './helpers/shipment.js';
import { get, parse } from './helpers/http.js';

export const options = baseOptions;

export function setup() {
  const ts = Date.now();
  const client1 = setupUser(`ship_c1_${ts}`);
  const client2 = setupUser(`ship_c2_${ts}`);
  const admin   = setupAdmin(`ship_adm_${ts}`);

  // Pre-create a shipment for GET / List tests
  const tracking = setupShipment(client1.token);

  return { client1, client2, admin, tracking };
}

export default function (data) {
  const { client1, client2, admin, tracking } = data;

  // ──────────────────────────────────────── POST /v1/shipments ─────────────

  group('POST /v1/shipments — happy path', () => {
    const res = createShipment(client1.token);
    const body = parse(res);
    check(res, {
      'status 201':               r => r.status === 201,
      'tracking_number present':  r => !!parse(r)?.tracking_number,
      'tracking_number format':   r => /^99M-[A-F0-9]{8}$/.test(parse(r)?.tracking_number ?? ''),
      'status is created':        r => parse(r)?.status === 'created',
      'created_at present':       r => !!parse(r)?.created_at,
      'estimated_delivery present': r => !!parse(r)?.estimated_delivery,
      '_links.self present':      r => !!parse(r)?._links?.self,
      '_links.events present':    r => !!parse(r)?._links?.events,
    });
  });

  group('POST /v1/shipments — Idempotency-Key deduplication', () => {
    const ikey = `idem-${Date.now()}`;
    const res1 = createShipment(client1.token, {}, ikey);
    const res2 = createShipment(client1.token, {}, ikey);
    check(res1, { 'first call 201': r => r.status === 201 });
    check(res2, {
      'second call 200 or 201':   r => r.status === 200 || r.status === 201,
      'same tracking_number':     () =>
        parse(res1)?.tracking_number === parse(res2)?.tracking_number,
    });
  });

  group('POST /v1/shipments — service_type same_day', () => {
    const res = createShipment(client1.token, { service_type: 'same_day' });
    check(res, { 'status 201': r => r.status === 201 });
  });

  group('POST /v1/shipments — service_type standard', () => {
    const res = createShipment(client1.token, { service_type: 'standard' });
    check(res, { 'status 201': r => r.status === 201 });
  });

  group('POST /v1/shipments — missing required field (sender) → 422', () => {
    const payload = Object.assign({}, defaultShipmentPayload);
    delete payload.sender;
    const res = createShipment(client1.token, payload);
    // Merge leaves sender undefined; use explicit payload
    const res2 = createShipment(client1.token, { sender: undefined });
    // Either 400 or 422 is acceptable for validation failure
    check(res2, { 'status 4xx': r => r.status === 400 || r.status === 422 });
  });

  group('POST /v1/shipments — invalid service_type → 422', () => {
    const res = createShipment(client1.token, { service_type: 'teleport' });
    check(res, {
      'status 422': r => r.status === 422,
      'error present': r => !!parse(r)?.error,
    });
  });

  group('POST /v1/shipments — invalid sender email → 422', () => {
    const res = createShipment(client1.token, {
      sender: { name: 'X', email: 'not-an-email', phone: '+521' },
    });
    check(res, { 'status 422': r => r.status === 422 });
  });

  group('POST /v1/shipments — negative package weight → 422', () => {
    const overrides = {
      package: Object.assign({}, defaultShipmentPayload.package, { weight_kg: -1 }),
    };
    const res = createShipment(client1.token, overrides);
    check(res, { 'status 422': r => r.status === 422 });
  });

  group('POST /v1/shipments — no auth → 401', () => {
    const res = createShipment('');
    check(res, { 'status 401': r => r.status === 401 });
  });

  // ─────────────────────────── GET /v1/shipments/:tracking_number ──────────

  group('GET /v1/shipments/:id — happy path', () => {
    const res = getShipment(client1.token, tracking);
    const body = parse(res);
    check(res, {
      'status 200':                r => r.status === 200,
      'tracking_number matches':   r => parse(r)?.tracking_number === tracking,
      'status_history present':    r => Array.isArray(parse(r)?.status_history),
      'initial history entry':     r => parse(r)?.status_history?.length >= 1,
      'first status is created':   r => parse(r)?.status_history?.[0]?.status === 'created',
      'sender present':            r => !!parse(r)?.sender,
      'origin present':            r => !!parse(r)?.origin,
      'destination present':       r => !!parse(r)?.destination,
      'package present':           r => !!parse(r)?.package,
      '_links present':            r => !!parse(r)?._links,
    });
  });

  group('GET /v1/shipments/:id — admin can see any shipment', () => {
    // Shipment created by client1, admin should be able to fetch it
    const res = getShipment(admin.token, tracking);
    check(res, { 'status 200': r => r.status === 200 });
  });

  group('GET /v1/shipments/:id — client cannot see another client shipment → 404', () => {
    // tracking belongs to client1; client2 should not see it
    const res = getShipment(client2.token, tracking);
    check(res, { 'status 404': r => r.status === 404 });
  });

  group('GET /v1/shipments/:id — not found → 404', () => {
    const res = getShipment(client1.token, '99M-NOTEXIST');
    check(res, {
      'status 404': r => r.status === 404,
      'error present': r => !!parse(r)?.error,
    });
  });

  group('GET /v1/shipments/:id — no auth → 401', () => {
    const res = getShipment('', tracking);
    check(res, { 'status 401': r => r.status === 401 });
  });

  // ────────────────────────────────────── GET /v1/shipments (list) ─────────

  group('GET /v1/shipments — happy path (client sees own)', () => {
    const res = listShipments(client1.token);
    const body = parse(res);
    check(res, {
      'status 200':           r => r.status === 200,
      'data is array':        r => Array.isArray(parse(r)?.data),
      'pagination present':   r => !!parse(r)?.pagination,
      'total >= 1':           r => (parse(r)?.pagination?.total ?? 0) >= 1,
      'no status_history':    r => parse(r)?.data?.[0]?.status_history === undefined,
      '_links in each item':  r => !!parse(r)?.data?.[0]?._links,
    });
  });

  group('GET /v1/shipments — admin sees all', () => {
    // Admin can see shipments from any client
    const adminList = listShipments(admin.token);
    const client1List = listShipments(client1.token);
    check(adminList, {
      'status 200': r => r.status === 200,
      'admin total >= client total': () =>
        (parse(adminList)?.pagination?.total ?? 0) >=
        (parse(client1List)?.pagination?.total ?? 0),
    });
  });

  group('GET /v1/shipments — client isolation (no cross-client data)', () => {
    // client2 shipments should not appear in client1 list
    const res = listShipments(client2.token);
    const body = parse(res);
    check(res, {
      'status 200': r => r.status === 200,
      // client2 has no shipments yet, so tracking from client1 must not appear
      'client1 tracking not in client2 list': () =>
        !(body?.data ?? []).some(s => s.tracking_number === tracking),
    });
  });

  group('GET /v1/shipments — filter by status=created', () => {
    const res = listShipments(client1.token, { status: 'created' });
    const body = parse(res);
    check(res, {
      'status 200': r => r.status === 200,
      'all items have status=created': () =>
        (body?.data ?? []).every(s => s.status === 'created'),
    });
  });

  group('GET /v1/shipments — filter by service_type=next_day', () => {
    const res = listShipments(client1.token, { service_type: 'next_day' });
    const body = parse(res);
    check(res, {
      'status 200': r => r.status === 200,
      'all items match service_type': () =>
        (body?.data ?? []).every(s => s.service_type === 'next_day'),
    });
  });

  group('GET /v1/shipments — search by tracking_number', () => {
    // Use the first few chars of the known tracking number as search term
    const searchTerm = tracking.slice(0, 6); // e.g. "99M-7A"
    const res = listShipments(client1.token, { search: searchTerm });
    check(res, {
      'status 200': r => r.status === 200,
    });
  });

  group('GET /v1/shipments — pagination: page=1 limit=2', () => {
    const res = listShipments(client1.token, { page: 1, limit: 2 });
    const body = parse(res);
    check(res, {
      'status 200':             r => r.status === 200,
      'limit honored (≤2)':     () => (body?.data?.length ?? 0) <= 2,
      'pagination.limit is 2':  () => body?.pagination?.limit === 2,
      'total_pages present':    () => typeof body?.pagination?.total_pages === 'number',
    });
  });

  group('GET /v1/shipments — limit capped at 100', () => {
    const res = listShipments(client1.token, { limit: 9999 });
    const body = parse(res);
    check(res, {
      'status 200':             r => r.status === 200,
      'limit capped at 100':    () => body?.pagination?.limit === 100,
    });
  });

  group('GET /v1/shipments — date_from filter (today)', () => {
    const today = new Date().toISOString().split('T')[0];
    const res = listShipments(client1.token, { date_from: today });
    check(res, { 'status 200': r => r.status === 200 });
  });

  group('GET /v1/shipments — invalid date_from → 400', () => {
    const res = listShipments(client1.token, { date_from: 'not-a-date' });
    check(res, { 'status 400': r => r.status === 400 });
  });

  group('GET /v1/shipments — no auth → 401', () => {
    const res = get(`${BASE_URL}/v1/shipments`, '');
    check(res, { 'status 401': r => r.status === 401 });
  });
}
