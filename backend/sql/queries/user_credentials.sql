-- name: UpsertCredential :one
INSERT INTO user_credentials (
    user_id,
    provider,
    issuer,
    subject,
    password_hash,
    metadata
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (provider, issuer, subject)
DO UPDATE SET
    user_id = EXCLUDED.user_id,
    password_hash = COALESCE(EXCLUDED.password_hash, user_credentials.password_hash),
    metadata = EXCLUDED.metadata,
    updated_at = NOW()
RETURNING *;

-- name: GetCredentialByUserAndProvider :one
SELECT *
FROM user_credentials
WHERE user_id = $1 AND provider = $2 AND issuer = $3;

-- name: GetCredentialBySubject :one
SELECT *
FROM user_credentials
WHERE provider = $1 AND issuer = $2 AND subject = $3;

-- name: ListCredentialsForUser :many
SELECT *
FROM user_credentials
WHERE user_id = $1
ORDER BY created_at DESC;
