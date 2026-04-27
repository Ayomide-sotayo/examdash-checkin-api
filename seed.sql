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

-- Seed checkins (track_id matches the order inserted above)
INSERT INTO checkins (id, learner_name, track_id, status, submitted_at) VALUES
    ('1', 'Ada Okafor',    1, 'submitted', '2026-04-14T09:00:00Z'),
    ('2', 'Emeka Nwosu',   2, 'pending',   '2026-04-15T10:30:00Z'),
    ('3', 'Zara Ahmed',    3, 'reviewed',  '2026-04-15T11:00:00Z'),
    ('4', 'Tolu Balogun',  1, 'pending',   '2026-04-16T08:00:00Z'),
    ('5', 'Chisom Eze',    4, 'submitted', '2026-04-16T09:15:00Z'),
    ('6', 'Fatima Musa',   5, 'reviewed',  '2026-04-16T10:00:00Z'),
    ('7', 'David Osei',    2, 'pending',   '2026-04-17T07:45:00Z'),
    ('8', 'Ngozi Adeyemi', 1, 'reviewed',  '2026-04-17T09:30:00Z')
ON CONFLICT (id) DO NOTHING;