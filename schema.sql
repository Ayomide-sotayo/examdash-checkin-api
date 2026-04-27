-- schema.sql
-- ExamDash Learner Check-in API v3
-- Run this first to set up your database tables

-- Table 1: tracks (lookup/reference table)
-- Stores the valid tracks learners can belong to
CREATE TABLE IF NOT EXISTS tracks (
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE  -- e.g. "Backend", "Frontend"
);

-- Table 2: checkins (main table — references tracks)
-- Each checkin belongs to one track via foreign key
CREATE TABLE IF NOT EXISTS checkins (
    id           TEXT PRIMARY KEY,
    learner_name TEXT        NOT NULL,
    track_id     INTEGER     NOT NULL REFERENCES tracks(id),
    status       TEXT        NOT NULL CHECK (status IN ('pending', 'submitted', 'reviewed')),
    submitted_at TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index to speed up filtering by status
CREATE INDEX IF NOT EXISTS idx_checkins_status ON checkins(status);

-- Index to speed up filtering by track
CREATE INDEX IF NOT EXISTS idx_checkins_track ON checkins(track_id);