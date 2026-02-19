import http from 'k6/http';

/**
 * Build a standard JSON params object, optionally with extra headers.
 * @param {string} [token]
 * @param {Record<string,string>} [extra]
 */
export function jsonParams(token, extra = {}) {
  const headers = { 'Content-Type': 'application/json', ...extra };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  return { headers };
}

export function post(url, body, token, extra = {}) {
  return http.post(url, JSON.stringify(body), jsonParams(token, extra));
}

export function get(url, token) {
  return http.get(url, jsonParams(token));
}

/** Parse response body safely; returns null on error. */
export function parse(res) {
  try { return JSON.parse(res.body); } catch { return null; }
}

/** Build a URL with query params, skipping undefined/null/empty values. */
export function withQuery(base, params = {}) {
  const parts = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== null && v !== '')
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`);
  return parts.length ? `${base}?${parts.join('&')}` : base;
}
