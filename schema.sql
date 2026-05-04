-- schema.sql
-- ExamDash Learner Check-in API v4
-- Run this first to set up your database tables

-- Table 1: tracks (lookup/reference table)
CREATE TABLE IF NOT EXISTS tracks (
    id   SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

-- Table 2: users (new in Week 4)
CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    email      TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'learner' CHECK (role IN ('learner', 'reviewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Table 3: checkins (now linked to users via user_id)
CREATE TABLE IF NOT EXISTS checkins (
    id           SERIAL      PRIMARY KEY,
    user_id      INTEGER     NOT NULL REFERENCES users(id),
    learner_name TEXT        NOT NULL,
    track_id     INTEGER     NOT NULL REFERENCES tracks(id),
    status       TEXT        NOT NULL CHECK (status IN ('pending', 'submitted', 'reviewed')),
    submitted_at TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_checkins_status  ON checkins(status);
CREATE INDEX IF NOT EXISTS idx_checkins_track   ON checkins(track_id);
CREATE INDEX IF NOT EXISTS idx_checkins_user    ON checkins(user_id);