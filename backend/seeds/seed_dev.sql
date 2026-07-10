-- Deeb development seed data
-- ---------------------------------------------------------------------------
-- Populates users, policies, environments, database instances, projects,
-- databases and project members with realistic sample data for local dev and
-- for exercising search / filtering across every resource.
--
-- Idempotent: every row uses a fixed UUID and ON CONFLICT (id) DO NOTHING, so
-- re-running is safe. Apply with:  make -C backend seed   (see Makefile target)
-- or:  psql "$DATABASE_URL" -f backend/seeds/seed_dev.sql
--
-- All seeded users share the password:  Password123!
-- (bcrypt hash below). Log in as admin@deeb.dev to explore everything.
-- ---------------------------------------------------------------------------

BEGIN;

-- ============================================================
-- USERS  (password for all = "Password123!")
-- ============================================================
INSERT INTO users (id, email, password_hash, display_name, organization_role, is_active, created_at, updated_at) VALUES
  ('10000000-0000-0000-0000-000000000001', 'admin@deeb.dev',      '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG', 'Ada Admin',      'OrganizationAdmin',     true,  NOW() - INTERVAL '120 days', NOW() - INTERVAL '120 days'),
  ('10000000-0000-0000-0000-000000000002', 'alice.dba@deeb.dev',  '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG', 'Alice Nguyen',   'OrganizationDBA',       true,  NOW() - INTERVAL '90 days',  NOW() - INTERVAL '90 days'),
  ('10000000-0000-0000-0000-000000000003', 'bob.dba@deeb.dev',    '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG', 'Bob Martins',    'OrganizationDBA',       true,  NOW() - INTERVAL '75 days',  NOW() - INTERVAL '75 days'),
  ('10000000-0000-0000-0000-000000000004', 'carol.dev@deeb.dev',  '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG', 'Carol Silva',    'OrganizationDeveloper', true,  NOW() - INTERVAL '60 days',  NOW() - INTERVAL '60 days'),
  ('10000000-0000-0000-0000-000000000005', 'dave.dev@deeb.dev',   '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG', 'Dave Oliveira',  'OrganizationDeveloper', true,  NOW() - INTERVAL '45 days',  NOW() - INTERVAL '45 days'),
  ('10000000-0000-0000-0000-000000000006', 'erin.dev@deeb.dev',   '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG', 'Erin Costa',     'OrganizationDeveloper', true,  NOW() - INTERVAL '30 days',  NOW() - INTERVAL '30 days'),
  ('10000000-0000-0000-0000-000000000007', 'frank.dev@deeb.dev',  '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG', 'Frank Rocha',    'OrganizationDeveloper', false, NOW() - INTERVAL '20 days',  NOW() - INTERVAL '20 days'),
  ('10000000-0000-0000-0000-000000000008', 'grace.dev@deeb.dev',  '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG', 'Grace Lima',     'OrganizationDeveloper', true,  NOW() - INTERVAL '14 days',  NOW() - INTERVAL '14 days'),
  ('10000000-0000-0000-0000-000000000009', 'heidi.dba@deeb.dev',  '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG', 'Heidi Souza',    'OrganizationDBA',       true,  NOW() - INTERVAL '7 days',   NOW() - INTERVAL '7 days'),
  ('10000000-0000-0000-0000-00000000000a', 'ivan.dev@deeb.dev',   '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG', 'Ivan Pereira',   'OrganizationDeveloper', true,  NOW() - INTERVAL '2 days',   NOW() - INTERVAL '2 days')
ON CONFLICT (id) DO NOTHING;

-- 100 additional bulk users (userN@deeb.dev, same password) for pagination/search load.
-- Roles cycle across the three org roles; every 10th user is inactive.
INSERT INTO users (id, email, password_hash, display_name, organization_role, is_active, created_at, updated_at)
SELECT
  ('11000000-0000-0000-0000-' || lpad(g::text, 12, '0'))::uuid,
  'user' || g || '@deeb.dev',
  '$2a$10$.3OW1jjmqK1M/ydOp0Gvf.oT3pc3AAdOPp9.33yz2ej3OCNCT6RTG',
  'Seed User ' || lpad(g::text, 3, '0'),
  (ARRAY['OrganizationDeveloper', 'OrganizationDeveloper', 'OrganizationDBA', 'OrganizationDeveloper'])[1 + (g % 4)],
  (g % 10 <> 0),
  NOW() - (g || ' days')::interval,
  NOW() - (g || ' days')::interval
FROM generate_series(1, 100) AS g
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- POLICIES  (SQL_REVIEW)
-- ============================================================
INSERT INTO policies (id, name, description, type, policy_config, created_by_id, created_at, updated_at) VALUES
  ('30000000-0000-0000-0000-000000000001', 'Standard SQL Review',      'Baseline SQL review rules applied to non-production environments.',
     'SQL_REVIEW',
     '{"rules": {"statement.disallow-drop-table": {"type": "statement.disallow-drop-table", "level": "warning", "engine": "mysql", "comment": "Discourage DROP TABLE"}}}'::jsonb,
     '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '100 days', NOW() - INTERVAL '100 days'),
  ('30000000-0000-0000-0000-000000000002', 'Strict Production Review', 'Strict, error-level SQL review rules enforced for production databases.',
     'SQL_REVIEW',
     '{"rules": {"statement.disallow-drop-table": {"type": "statement.disallow-drop-table", "level": "error", "engine": "postgres", "comment": "Block DROP TABLE in production"}, "naming.table": {"type": "naming.table", "level": "error", "engine": "postgres", "comment": "Tables must be snake_case"}}}'::jsonb,
     '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '95 days', NOW() - INTERVAL '95 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- ENVIRONMENTS
-- ============================================================
INSERT INTO environments (id, name, description, policy_id, allow_self_approval, created_by_id, created_at, updated_at) VALUES
  ('40000000-0000-0000-0000-000000000001', 'Development', 'Local and shared development environment.', '30000000-0000-0000-0000-000000000001', true,  '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '98 days', NOW() - INTERVAL '98 days'),
  ('40000000-0000-0000-0000-000000000002', 'Staging',     'Pre-production staging environment.',        '30000000-0000-0000-0000-000000000001', false, '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '97 days', NOW() - INTERVAL '97 days'),
  ('40000000-0000-0000-0000-000000000003', 'Production',  'Live production environment.',               '30000000-0000-0000-0000-000000000002', false, '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '96 days', NOW() - INTERVAL '96 days'),
  ('40000000-0000-0000-0000-000000000004', 'QA',          'Quality assurance / test environment.',      NULL,                                   true,  '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '50 days', NOW() - INTERVAL '50 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DATABASE INSTANCES  (connection_details_encrypted left NULL for seed data)
-- ============================================================
INSERT INTO database_instances (id, name, description, database_engine, host, port, username, use_rds_iam_auth, require_ssl, status, environment_id, created_by_id, created_at, updated_at) VALUES
  ('50000000-0000-0000-0000-000000000001', 'mysql-dev-01',     'MySQL development instance.',    'mysql',    'mysql-dev-01.internal',     3306, 'deeb_app', false, false, 'Active',      '40000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '90 days', NOW() - INTERVAL '90 days'),
  ('50000000-0000-0000-0000-000000000002', 'pg-dev-01',        'Postgres development instance.', 'postgres', 'pg-dev-01.internal',        5432, 'deeb_app', false, false, 'Active',      '40000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '89 days', NOW() - INTERVAL '89 days'),
  ('50000000-0000-0000-0000-000000000003', 'mysql-staging-01', 'MySQL staging instance.',        'mysql',    'mysql-staging-01.internal', 3306, 'deeb_app', false, true,  'Active',      '40000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '80 days', NOW() - INTERVAL '80 days'),
  ('50000000-0000-0000-0000-000000000004', 'pg-prod-01',       'Primary production Postgres.',   'postgres', 'pg-prod-01.internal',       5432, 'deeb_app', true,  true,  'Active',      '40000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '70 days', NOW() - INTERVAL '70 days'),
  ('50000000-0000-0000-0000-000000000005', 'pg-prod-02',       'Secondary production Postgres.', 'postgres', 'pg-prod-02.internal',       5432, 'deeb_app', true,  true,  'Active',      '40000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '65 days', NOW() - INTERVAL '65 days'),
  ('50000000-0000-0000-0000-000000000006', 'mysql-qa-01',      'MySQL QA instance.',             'mysql',    'mysql-qa-01.internal',      3306, 'deeb_app', false, false, 'Maintenance', '40000000-0000-0000-0000-000000000004', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '40 days', NOW() - INTERVAL '40 days')
ON CONFLICT (id) DO NOTHING;

-- 200 additional bulk database instances for pagination/search/filter load.
-- Engine alternates mysql/postgres (with matching port), status and environment cycle.
INSERT INTO database_instances (id, name, description, database_engine, host, port, username, use_rds_iam_auth, require_ssl, status, environment_id, created_by_id, created_at, updated_at)
SELECT
  ('51000000-0000-0000-0000-' || lpad(g::text, 12, '0'))::uuid,
  'instance-' || lpad(g::text, 3, '0'),
  'Seeded database instance ' || lpad(g::text, 3, '0') || '.',
  (ARRAY['mysql', 'postgres'])[1 + (g % 2)],
  'db-' || lpad(g::text, 3, '0') || '.internal',
  (ARRAY[3306, 5432])[1 + (g % 2)],
  'deeb_app',
  false,
  (g % 3 <> 0),
  (ARRAY['Active', 'Active', 'Active', 'Inactive', 'Maintenance'])[1 + (g % 5)],
  (ARRAY[
     '40000000-0000-0000-0000-000000000001',
     '40000000-0000-0000-0000-000000000002',
     '40000000-0000-0000-0000-000000000003',
     '40000000-0000-0000-0000-000000000004'
   ])[1 + (g % 4)]::uuid,
  '10000000-0000-0000-0000-000000000001',
  NOW() - (g || ' hours')::interval,
  NOW() - (g || ' hours')::interval
FROM generate_series(1, 200) AS g
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- PROJECTS
-- ============================================================
INSERT INTO projects (id, name, description, created_by_id, created_at, updated_at) VALUES
  ('20000000-0000-0000-0000-000000000001', 'payments',       'Payment processing and wallet services.', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '88 days', NOW() - INTERVAL '88 days'),
  ('20000000-0000-0000-0000-000000000002', 'checkout',       'Checkout and cart services.',             '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '85 days', NOW() - INTERVAL '85 days'),
  ('20000000-0000-0000-0000-000000000003', 'identity',       'Authentication and user identity.',       '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '82 days', NOW() - INTERVAL '82 days'),
  ('20000000-0000-0000-0000-000000000004', 'catalog',        'Product catalog and listings.',           '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '78 days', NOW() - INTERVAL '78 days'),
  ('20000000-0000-0000-0000-000000000005', 'notifications',  'Email, SMS and push notifications.',      '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '70 days', NOW() - INTERVAL '70 days'),
  ('20000000-0000-0000-0000-000000000006', 'analytics',      'Event analytics and reporting.',          '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '60 days', NOW() - INTERVAL '60 days'),
  ('20000000-0000-0000-0000-000000000007', 'billing',        'Invoicing and subscription billing.',     '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '50 days', NOW() - INTERVAL '50 days'),
  ('20000000-0000-0000-0000-000000000008', 'search-service', 'Search indexing and query APIs.',         '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '40 days', NOW() - INTERVAL '40 days'),
  ('20000000-0000-0000-0000-000000000009', 'inventory',      'Stock and inventory management.',          '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '25 days', NOW() - INTERVAL '25 days'),
  ('20000000-0000-0000-0000-00000000000a', 'shipping',       'Fulfilment and shipping logistics.',      '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days'),
  ('20000000-0000-0000-0000-00000000000b', 'fraud-detection','Fraud detection and risk scoring.',        '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '86 days', NOW() - INTERVAL '86 days'),
  ('20000000-0000-0000-0000-00000000000c', 'loyalty',        'Loyalty and rewards program.',             '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '72 days', NOW() - INTERVAL '72 days'),
  ('20000000-0000-0000-0000-00000000000d', 'recommendations','Product recommendation engine.',           '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '64 days', NOW() - INTERVAL '64 days'),
  ('20000000-0000-0000-0000-00000000000e', 'pricing',        'Dynamic pricing and promotions.',          '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '56 days', NOW() - INTERVAL '56 days'),
  ('20000000-0000-0000-0000-00000000000f', 'tax',            'Tax calculation and compliance.',          '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '44 days', NOW() - INTERVAL '44 days'),
  ('20000000-0000-0000-0000-000000000010', 'warehouse',      'Warehouse management and picking.',        '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '33 days', NOW() - INTERVAL '33 days'),
  ('20000000-0000-0000-0000-000000000011', 'returns',        'Returns and refunds processing.',          '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '28 days', NOW() - INTERVAL '28 days'),
  ('20000000-0000-0000-0000-000000000012', 'gift-cards',     'Gift cards and store credit.',             '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '18 days', NOW() - INTERVAL '18 days'),
  ('20000000-0000-0000-0000-000000000013', 'marketing',      'Campaigns and marketing automation.',      '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '12 days', NOW() - INTERVAL '12 days'),
  ('20000000-0000-0000-0000-000000000014', 'support-desk',   'Customer support and ticketing.',          '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '4 days',  NOW() - INTERVAL '4 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- DATABASES  (unique per (instance, name); mixed statuses)
-- ============================================================
INSERT INTO databases (id, project_id, database_instance_id, environment_id, database_name, description, status, created_by_id, created_at, updated_at) VALUES
  ('60000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000003', 'payments_prod',        'Payments production database.',        'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '80 days', NOW() - INTERVAL '80 days'),
  ('60000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000002', 'payments_staging',     'Payments staging database.',           'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '79 days', NOW() - INTERVAL '79 days'),
  ('60000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000001', '50000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', 'payments_dev',         'Payments development database.',       'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '78 days', NOW() - INTERVAL '78 days'),
  ('60000000-0000-0000-0000-000000000004', '20000000-0000-0000-0000-000000000002', '50000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000003', 'checkout_prod',        'Checkout production database.',        'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '75 days', NOW() - INTERVAL '75 days'),
  ('60000000-0000-0000-0000-000000000005', '20000000-0000-0000-0000-000000000002', '50000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000001', 'checkout_dev',         'Checkout development database.',       'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '74 days', NOW() - INTERVAL '74 days'),
  ('60000000-0000-0000-0000-000000000006', '20000000-0000-0000-0000-000000000003', '50000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000003', 'identity_prod',        'Identity production database.',        'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '70 days', NOW() - INTERVAL '70 days'),
  ('60000000-0000-0000-0000-000000000007', '20000000-0000-0000-0000-000000000003', '50000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000001', 'identity_dev',         'Identity development database.',       'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '69 days', NOW() - INTERVAL '69 days'),
  ('60000000-0000-0000-0000-000000000008', '20000000-0000-0000-0000-000000000004', '50000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000003', 'catalog_prod',         'Catalog production database.',         'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '65 days', NOW() - INTERVAL '65 days'),
  ('60000000-0000-0000-0000-000000000009', '20000000-0000-0000-0000-000000000004', '50000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000004', 'catalog_qa',           'Catalog QA database.',                 'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '38 days', NOW() - INTERVAL '38 days'),
  ('60000000-0000-0000-0000-00000000000a', '20000000-0000-0000-0000-000000000005', '50000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', 'notifications_dev',    'Notifications development database.',  'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '60 days', NOW() - INTERVAL '60 days'),
  ('60000000-0000-0000-0000-00000000000b', '20000000-0000-0000-0000-000000000005', '50000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000002', 'notifications_staging','Notifications staging database.',      'Inactive', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '58 days', NOW() - INTERVAL '58 days'),
  ('60000000-0000-0000-0000-00000000000c', '20000000-0000-0000-0000-000000000006', '50000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000003', 'analytics_prod',       'Analytics production database.',       'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '55 days', NOW() - INTERVAL '55 days'),
  ('60000000-0000-0000-0000-00000000000d', '20000000-0000-0000-0000-000000000006', '50000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000001', 'analytics_dev',        'Analytics development database.',      'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '54 days', NOW() - INTERVAL '54 days'),
  ('60000000-0000-0000-0000-00000000000e', '20000000-0000-0000-0000-000000000007', '50000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000003', 'billing_prod',         'Billing production database.',         'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '48 days', NOW() - INTERVAL '48 days'),
  ('60000000-0000-0000-0000-00000000000f', '20000000-0000-0000-0000-000000000007', '50000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000003', 'billing_next',         'Billing v2 database (awaiting creation).', 'Pending', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '5 days',  NOW() - INTERVAL '5 days'),
  ('60000000-0000-0000-0000-000000000010', '20000000-0000-0000-0000-000000000008', '50000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000003', 'search_prod',          'Search-service production database.',  'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '40 days', NOW() - INTERVAL '40 days'),
  ('60000000-0000-0000-0000-000000000011', '20000000-0000-0000-0000-000000000008', '50000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000004', 'search_qa',            'Search-service QA database.',          'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '36 days', NOW() - INTERVAL '36 days'),
  ('60000000-0000-0000-0000-000000000012', '20000000-0000-0000-0000-000000000009', '50000000-0000-0000-0000-000000000004', '40000000-0000-0000-0000-000000000003', 'inventory_prod',       'Inventory production database.',       'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '24 days', NOW() - INTERVAL '24 days'),
  ('60000000-0000-0000-0000-000000000013', '20000000-0000-0000-0000-000000000009', '50000000-0000-0000-0000-000000000003', '40000000-0000-0000-0000-000000000002', 'inventory_staging',    'Inventory staging database.',          'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '23 days', NOW() - INTERVAL '23 days'),
  ('60000000-0000-0000-0000-000000000014', '20000000-0000-0000-0000-00000000000a', '50000000-0000-0000-0000-000000000005', '40000000-0000-0000-0000-000000000003', 'shipping_prod',        'Shipping production database.',        'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '9 days',  NOW() - INTERVAL '9 days'),
  ('60000000-0000-0000-0000-000000000015', '20000000-0000-0000-0000-00000000000a', '50000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', 'shipping_dev',         'Shipping development database.',       'Active',   '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '8 days',  NOW() - INTERVAL '8 days'),
  ('60000000-0000-0000-0000-000000000016', '20000000-0000-0000-0000-000000000004', '50000000-0000-0000-0000-000000000006', '40000000-0000-0000-0000-000000000004', 'legacy_orders',        'Legacy orders database (decommissioning).', 'Inactive', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '35 days', NOW() - INTERVAL '35 days')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- PROJECT MEMBERS
-- ============================================================
INSERT INTO project_members (id, project_id, user_id, project_role, added_by_id, added_at) VALUES
  ('70000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002', 'ProjectOwner',     '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '88 days'),
  ('70000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000004', 'ProjectDeveloper', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '87 days'),
  ('70000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000005', 'ProjectDeveloper', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '87 days'),
  ('70000000-0000-0000-0000-000000000004', '20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000003', 'ProjectOwner',     '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '85 days'),
  ('70000000-0000-0000-0000-000000000005', '20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000006', 'ProjectDeveloper', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '84 days'),
  ('70000000-0000-0000-0000-000000000006', '20000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000002', 'ProjectApprover',  '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '82 days'),
  ('70000000-0000-0000-0000-000000000007', '20000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000008', 'ProjectDeveloper', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '81 days'),
  ('70000000-0000-0000-0000-000000000008', '20000000-0000-0000-0000-000000000004', '10000000-0000-0000-0000-000000000009', 'ProjectOwner',     '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '78 days'),
  ('70000000-0000-0000-0000-000000000009', '20000000-0000-0000-0000-000000000004', '10000000-0000-0000-0000-00000000000a', 'ProjectViewer',    '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '77 days'),
  ('70000000-0000-0000-0000-00000000000a', '20000000-0000-0000-0000-000000000007', '10000000-0000-0000-0000-000000000002', 'ProjectOwner',     '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '50 days'),
  ('70000000-0000-0000-0000-00000000000b', '20000000-0000-0000-0000-00000000000b', '10000000-0000-0000-0000-000000000003', 'ProjectOwner',     '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '85 days'),
  ('70000000-0000-0000-0000-00000000000c', '20000000-0000-0000-0000-00000000000c', '10000000-0000-0000-0000-000000000004', 'ProjectDeveloper', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '70 days'),
  ('70000000-0000-0000-0000-00000000000d', '20000000-0000-0000-0000-00000000000d', '10000000-0000-0000-0000-000000000008', 'ProjectDeveloper', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '62 days'),
  ('70000000-0000-0000-0000-00000000000e', '20000000-0000-0000-0000-000000000013', '10000000-0000-0000-0000-000000000006', 'ProjectDeveloper', '10000000-0000-0000-0000-000000000001', NOW() - INTERVAL '11 days')
ON CONFLICT (id) DO NOTHING;

COMMIT;

-- Summary of seeded rows (informational)
SELECT
  (SELECT count(*) FROM users WHERE email LIKE '%@deeb.dev')            AS users,
  (SELECT count(*) FROM policies)                                       AS policies,
  (SELECT count(*) FROM environments)                                   AS environments,
  (SELECT count(*) FROM database_instances)                             AS instances,
  (SELECT count(*) FROM projects)                                       AS projects,
  (SELECT count(*) FROM databases)                                      AS databases,
  (SELECT count(*) FROM project_members)                                AS project_members;
