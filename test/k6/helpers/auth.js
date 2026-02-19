import { post, parse } from './http.js';
import { BASE_URL } from '../config.js';
import { fail } from 'k6';

export function register(username, password, email, role = 'client', clientId = '') {
  return post(`${BASE_URL}/auth/register`, {
    username,
    password,
    email,
    role,
    client_id: clientId,
  });
}

export function login(email, password) {
  return post(`${BASE_URL}/auth/login`, { email, password });
}

/**
 * Register a new user and login; returns { token, user }.
 * Calls fail() if either step fails — stops the test immediately.
 */
export function setupUser(suffix) {
  const username = `u_${suffix}`;
  const email = `u_${suffix}@test.com`;
  const password = 'Password123!';
  const role = 'client';
  const clientId = `client_${suffix}`;

  const regRes = register(username, password, email, role, clientId);
  if (regRes.status !== 201) {
    fail(`setup: register failed (${regRes.status}): ${regRes.body}`);
  }

  const loginRes = login(email, password);
  if (loginRes.status !== 200) {
    fail(`setup: login failed (${loginRes.status}): ${loginRes.body}`);
  }

  const body = parse(loginRes);
  return { token: body.token, user: body.user, email, password, username, clientId };
}

/**
 * Register an admin user and login; returns { token, user }.
 */
export function setupAdmin(suffix) {
  const username = `admin_${suffix}`;
  const email = `admin_${suffix}@test.com`;
  const password = 'Password123!';

  const regRes = register(username, password, email, 'admin', '');
  if (regRes.status !== 201) {
    fail(`setup: admin register failed (${regRes.status}): ${regRes.body}`);
  }

  const loginRes = login(email, password);
  if (loginRes.status !== 200) {
    fail(`setup: admin login failed (${loginRes.status}): ${loginRes.body}`);
  }

  const body = parse(loginRes);
  return { token: body.token, user: body.user, email, password, username };
}
