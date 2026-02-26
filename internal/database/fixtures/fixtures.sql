-- fixtures.sql — test data for local development / seed command.
-- Passwords are bcrypt hashes of "password123".

INSERT INTO users (email, name, password) VALUES
    ('alice@example.com', 'Alice',   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'),
    ('bob@example.com',   'Bob',     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy')
ON CONFLICT (email) DO NOTHING;

INSERT INTO tasks (user_id, title, description, status) VALUES
    ((SELECT id FROM users WHERE email = 'alice@example.com'), 'Buy groceries',      'Milk, eggs, bread',          'todo'),
    ((SELECT id FROM users WHERE email = 'alice@example.com'), 'Write report',        'Q4 financial summary',       'in_progress'),
    ((SELECT id FROM users WHERE email = 'bob@example.com'),   'Fix landing page',    'Update hero section copy',   'todo'),
    ((SELECT id FROM users WHERE email = 'bob@example.com'),   'Deploy v2',           'Tag release and deploy',     'done')
ON CONFLICT DO NOTHING;
