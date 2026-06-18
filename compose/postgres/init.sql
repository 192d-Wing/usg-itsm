-- Schema-per-service layout (ADR-0002). Each service owns one schema and
-- connects with a role scoped to it. For local dev all schemas live in one
-- database and the single `itsm` role owns them; production uses distinct
-- least-privilege roles.

CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS ticketing;
CREATE SCHEMA IF NOT EXISTS catalog;
CREATE SCHEMA IF NOT EXISTS workflow;
CREATE SCHEMA IF NOT EXISTS notification;
CREATE SCHEMA IF NOT EXISTS audit;

-- Useful extensions (available in the postgres:16 image).
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "citext";     -- case-insensitive text (emails)
