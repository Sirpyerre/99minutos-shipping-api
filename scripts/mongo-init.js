// mongo-init.js — runs once when the container is first created.
// Seeds the auth_users collection with two default users.
// Passwords are bcrypt hashes of "password123" (cost 12).

db = db.getSiblingDB("shipping_system");

// ── Indexes ──────────────────────────────────────────────────────────────────
db.shipments.createIndex({ tracking_number: 1 }, { unique: true });
db.shipments.createIndex({ client_id: 1 });
db.shipments.createIndex({ created_at: -1 });

db.status_events.createIndex({ tracking_number: 1, created_at: -1 });

db.auth_users.createIndex({ username: 1 }, { unique: true });

// ── Seed users ────────────────────────────────────────────────────────────────
// bcrypt hash of "password123" (cost 12)
const PASSWORD_HASH = "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6hsxq/LbMi";

db.auth_users.insertMany([
  {
    username:      "admin_user",
    password_hash: PASSWORD_HASH,
    role:          "admin",
    client_id:     null,
    created_at:    new Date(),
    updated_at:    new Date(),
  },
  {
    username:      "client_user_001",
    password_hash: PASSWORD_HASH,
    role:          "client",
    client_id:     "client_001",
    created_at:    new Date(),
    updated_at:    new Date(),
  },
]);

print("✅  mongo-init: indexes and seed users created");
