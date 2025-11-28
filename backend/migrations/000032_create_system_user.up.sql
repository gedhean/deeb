-- Create system user for automated operations
-- This user is used for operations like automatic project assignments
-- Note: Using 00000001-... instead of all zeros to avoid uuid.Nil validation issues
INSERT INTO users (
    id,
    email,
    password_hash,
    organization_role,
    is_active,
    created_at,
    updated_at
) VALUES (
    '00000001-0000-0000-0000-000000000000',
    'system@deeb1.dev',
    '',
    'OrganizationAdmin',
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (email) DO NOTHING;
