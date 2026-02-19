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
db.auth_users.createIndex({ email: 1 }, { unique: true });

// ── Seed users ────────────────────────────────────────────────────────────────
// bcrypt hash of "password123" (cost 12)
const PASSWORD_HASH = "$2a$12$bBXOztiJVYqEE7E6Dm/ag.pE607fDxB9QOR9WWHo1WeV8ihtedG2y";

const NOW = Math.floor(new Date().getTime() / 1000);

db.auth_users.insertMany([
  {
    username:      "admin_user",
    email:         "admin@99minutos.com",
    password_hash: PASSWORD_HASH,
    role:          "admin",
    client_id:     null,
    created_at:    NOW,
    updated_at:    NOW,
  },
  {
    username:      "client_user_001",
    email:         "client001@99minutos.com",
    password_hash: PASSWORD_HASH,
    role:          "client",
    client_id:     "client_001",
    created_at:    NOW,
    updated_at:    NOW,
  },
]);

print("✅  mongo-init: indexes and seed users created");
