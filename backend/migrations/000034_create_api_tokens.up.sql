-- API tokens (service-account tokens) for unattended automation.
-- A token binds to a user and inherits that user's role through the existing RBAC.
-- Only the SHA-256 hash of the token is stored; the plaintext is shown once at issue time.
CREATE TABLE api_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    token_prefix VARCHAR(32)  NOT NULL,          -- e.g. "deeb_pat_ab12cd34" for display/identification
    token_hash   VARCHAR(64)  NOT NULL UNIQUE,   -- sha256 hex of the full plaintext (UNIQUE provides the lookup index)
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at   TIMESTAMP WITH TIME ZONE,       -- NULL = long-lived
    revoked_at   TIMESTAMP WITH TIME ZONE,       -- NULL = active
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_tokens_user_id ON api_tokens(user_id);
