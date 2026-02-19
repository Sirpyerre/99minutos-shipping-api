/**
 * K6 integration tests — Auth endpoints
 *
 * Covers:
 *   POST /auth/register  — happy path, duplicate, missing fields, bad content-type
 *   POST /auth/login     — happy path, wrong password, unknown email, empty body
 *   Protected routes     — valid token, no token, invalid token, malformed header
 *
 * Run: k6 run test/k6/auth.test.js
 */
import http from 'k6/http';
import { check, group } from 'k6';
import { BASE_URL, options as baseOptions } from './config.js';
import { register, login } from './helpers/auth.js';
import { get, parse } from './helpers/http.js';

export const options = baseOptions;

export function setup() {
  const ts = Date.now();
  // Pre-create a user that will be used for duplicate/login tests
  const email = `auth_test_${ts}@test.com`;
  const res = register(`auth_user_${ts}`, 'Password123!', email, 'client', `c_${ts}`);
  if (res.status !== 201) {
    console.error('setup: failed to pre-create user', res.body);
  }
  return { email, password: 'Password123!', username: `auth_user_${ts}`, ts };
}

export default function (data) {
  const { email, password, username, ts } = data;

  // ─────────────────────────────────────────────────────────── Register ─────

  group('POST /auth/register — happy path', () => {
    const ts2 = `${ts}_2`;
    const res = register(`new_${ts2}`, 'Password123!', `new_${ts2}@test.com`, 'client', `c_${ts2}`);
    check(res, {
      'status 201':           r => r.status === 201,
      'message present':      r => !!parse(r)?.message,
      'user object present':  r => !!parse(r)?.user,
      'username correct':     r => parse(r)?.user?.username === `new_${ts2}`,
      'role is client':       r => parse(r)?.user?.role === 'client',
      'client_id present':    r => !!parse(r)?.user?.client_id,
    });
  });

  group('POST /auth/register — admin role', () => {
    const ts3 = `${ts}_adm`;
    const res = register(`adm_${ts3}`, 'Password123!', `adm_${ts3}@test.com`, 'admin', '');
    check(res, {
      'status 201':     r => r.status === 201,
      'role is admin':  r => parse(r)?.user?.role === 'admin',
    });
  });

  group('POST /auth/register — duplicate username → 409', () => {
    const res = register(username, password, email, 'client', '');
    check(res, {
      'status 409': r => r.status === 409,
      'error field': r => !!parse(r)?.error,
    });
  });

  group('POST /auth/register — missing required fields → 400', () => {
    const res = http.post(`${BASE_URL}/auth/register`,
      JSON.stringify({ username: 'x' }),  // no password, no email
      { headers: { 'Content-Type': 'application/json' } }
    );
    check(res, { 'status 4xx': r => r.status >= 400 });
  });

  group('POST /auth/register — non-JSON body → 400', () => {
    const res = http.post(`${BASE_URL}/auth/register`, 'not-json', {
      headers: { 'Content-Type': 'text/plain' },
    });
    check(res, { 'status 4xx': r => r.status >= 400 });
  });

  // ──────────────────────────────────────────────────────────── Login ────────

  group('POST /auth/login — happy path', () => {
    const res = login(email, password);
    const body = parse(res);
    check(res, {
      'status 200':            r => r.status === 200,
      'token present':         r => !!parse(r)?.token,
      'token_type is Bearer':  r => parse(r)?.token_type === 'Bearer',
      'expires_in > 0':        r => (parse(r)?.expires_in ?? 0) > 0,
      'user object present':   r => !!parse(r)?.user,
      'username matches':      r => parse(r)?.user?.username === username,
      'role matches':          r => parse(r)?.user?.role === 'client',
    });
  });

  group('POST /auth/login — wrong password → 401', () => {
    const res = login(email, 'wrong_password_xyz');
    check(res, {
      'status 401': r => r.status === 401,
      'error field': r => !!parse(r)?.error,
    });
  });

  group('POST /auth/login — unknown email → 401 or 404', () => {
    const res = login('nobody@nowhere.com', 'any');
    check(res, { 'status 4xx': r => r.status === 401 || r.status === 404 });
  });

  group('POST /auth/login — empty body → 400', () => {
    const res = http.post(`${BASE_URL}/auth/login`, '{}', {
      headers: { 'Content-Type': 'application/json' },
    });
    check(res, { 'status 4xx': r => r.status >= 400 });
  });

  // ─────────────────────────────────────────── Protected route auth checks ──

  const loginRes = login(email, password);
  const token = parse(loginRes)?.token;

  group('GET /v1/shipments — valid token → 200', () => {
    const res = get(`${BASE_URL}/v1/shipments`, token);
    check(res, { 'status 200': r => r.status === 200 });
  });

  group('GET /v1/shipments — no token → 401', () => {
    const res = http.get(`${BASE_URL}/v1/shipments`);
    check(res, { 'status 401': r => r.status === 401 });
  });

  group('GET /v1/shipments — invalid JWT → 401', () => {
    const res = get(`${BASE_URL}/v1/shipments`, 'invalid.jwt.token');
    check(res, { 'status 401': r => r.status === 401 });
  });

  group('GET /v1/shipments — malformed Authorization header → 401', () => {
    const res = http.get(`${BASE_URL}/v1/shipments`, {
      headers: { Authorization: 'Basic dXNlcjpwYXNz' },
    });
    check(res, { 'status 401': r => r.status === 401 });
  });
}
