-- seed.sql
-- ExamDash Learner Check-in API v5
-- Run AFTER schema.sql to populate sample data for frontend testing

-- Seed the tracks lookup table
INSERT INTO tracks (name) VALUES
    ('Backend'),
    ('Frontend'),
    ('Product Design'),
    ('Product Management'),
    ('Growth')
ON CONFLICT (name) DO NOTHING;

-- Seed users (passwords are bcrypt hashes of 'password123')
INSERT INTO users (email, password, role) VALUES
    ('ada@examdash.com',    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'learner'),
    ('emeka@examdash.com',  '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'learner'),
    ('reviewer@examdash.com','$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'reviewer')
ON CONFLICT (email) DO NOTHING;

-- Seed check-ins linked to seeded users
-- Ada (learner 1) check-ins
INSERT INTO checkins (user_id, learner_name, track_id, status, submitted_at, created_at, updated_at)
SELECT u.id, 'Ada Okafor', t.id, 'submitted', '2026-04-14T09:00:00Z', NOW(), NOW()
FROM users u, tracks t WHERE u.email = 'ada@examdash.com' AND t.name = 'Backend';

INSERT INTO checkins (user_id, learner_name, track_id, status, submitted_at, created_at, updated_at)
SELECT u.id, 'Ada Okafor', t.id, 'reviewed', '2026-04-21T09:00:00Z', NOW(), NOW()
FROM users u, tracks t WHERE u.email = 'ada@examdash.com' AND t.name = 'Backend';

INSERT INTO checkins (user_id, learner_name, track_id, status, submitted_at, created_at, updated_at)
SELECT u.id, 'Ada Okafor', t.id, 'pending', '2026-04-28T09:00:00Z', NOW(), NOW()
FROM users u, tracks t WHERE u.email = 'ada@examdash.com' AND t.name = 'Backend';

-- Emeka (learner 2) check-ins
INSERT INTO checkins (user_id, learner_name, track_id, status, submitted_at, created_at, updated_at)
SELECT u.id, 'Emeka Nwosu', t.id, 'pending', '2026-04-15T10:30:00Z', NOW(), NOW()
FROM users u, tracks t WHERE u.email = 'emeka@examdash.com' AND t.name = 'Frontend';

INSERT INTO checkins (user_id, learner_name, track_id, status, submitted_at, created_at, updated_at)
SELECT u.id, 'Emeka Nwosu', t.id, 'submitted', '2026-04-22T10:30:00Z', NOW(), NOW()
FROM users u, tracks t WHERE u.email = 'emeka@examdash.com' AND t.name = 'Frontend';

-- Login credentials for frontend testing:
-- ada@examdash.com      / password123  (learner)
-- emeka@examdash.com    / password123  (learner)
-- reviewer@examdash.com / password123  (reviewer)