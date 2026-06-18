-- Ticketing core schema (ADR-0008: incidents and service requests are one
-- work-item type). Runs with search_path pinned to the ticketing schema.

-- Human-facing number sequences, one per work-item type.
CREATE SEQUENCE IF NOT EXISTS incident_seq START 1001;
CREATE SEQUENCE IF NOT EXISTS request_seq START 1001;

CREATE TABLE IF NOT EXISTS work_items (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    number           text        NOT NULL UNIQUE,
    type             text        NOT NULL CHECK (type IN ('incident', 'service_request')),
    title            text        NOT NULL,
    description      text        NOT NULL DEFAULT '',
    status           text        NOT NULL DEFAULT 'new',
    priority         text        NOT NULL CHECK (priority IN ('critical', 'high', 'moderate', 'low')),
    requester_id     text        NOT NULL,
    assignee_id      text,
    assignment_group text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    closed_at        timestamptz
);

CREATE INDEX IF NOT EXISTS work_items_status_idx    ON work_items (status);
CREATE INDEX IF NOT EXISTS work_items_assignee_idx  ON work_items (assignee_id);
CREATE INDEX IF NOT EXISTS work_items_requester_idx ON work_items (requester_id);
CREATE INDEX IF NOT EXISTS work_items_type_idx      ON work_items (type);

CREATE TABLE IF NOT EXISTS comments (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    work_item_id uuid        NOT NULL REFERENCES work_items (id) ON DELETE CASCADE,
    author_id    text        NOT NULL,
    body         text        NOT NULL,
    internal     boolean     NOT NULL DEFAULT false,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS comments_work_item_idx ON comments (work_item_id, created_at);

-- Append-only history / audit trail for each work item.
CREATE TABLE IF NOT EXISTS events (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    work_item_id uuid        NOT NULL REFERENCES work_items (id) ON DELETE CASCADE,
    actor_id     text        NOT NULL,
    kind         text        NOT NULL,
    data         jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS events_work_item_idx ON events (work_item_id, created_at);
