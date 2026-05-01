-- seed.sql
-- ExamDash Learner Check-in API v3
-- Run AFTER schema.sql to populate sample data

-- Seed the tracks lookup table
INSERT INTO tracks (name) VALUES
    ('Backend'),
    ('Frontend'),
    ('Product Design'),
    ('Product Management'),
    ('Growth')
ON CONFLICT (name) DO NOTHING;

-- Seed checkins — no id column, SERIAL handles it automatically
INSERT INTO checkins (learner_name, track_id, status, submitted_at) VALUES
    ('Ada Okafor',    1, 'submitted', '2026-04-14T09:00:00Z'),
    ('Emeka Nwosu',   2, 'pending',   '2026-04-15T10:30:00Z'),
    ('Zara Ahmed',    3, 'reviewed',  '2026-04-15T11:00:00Z'),
    ('Tolu Balogun',  1, 'pending',   '2026-04-16T08:00:00Z'),
    ('Chisom Eze',    4, 'submitted', '2026-04-16T09:15:00Z'),
    ('Fatima Musa',   5, 'reviewed',  '2026-04-16T10:00:00Z'),
    ('David Osei',    2, 'pending',   '2026-04-17T07:45:00Z'),
    ('Ngozi Adeyemi', 1, 'reviewed',  '2026-04-17T09:30:00Z');