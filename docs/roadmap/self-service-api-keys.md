# Self-Service API Key Rotation Roadmap

## Summary
Let end users manage their own personal API keys from the user portal instead of relying on admins. This includes creating, revoking, rotating, and viewing usage/quota per key.

## Implementation Overview

1. **User API Surface**
   - Add `/user/api-keys` CRUD endpoints backed by existing key service but scoped to the user’s personal tenant.
   - Support metadata (name, purpose), per-key rate limits, and expiration dates.

2. **Portal UX**
   - User portal gains an “API Keys” tab showing key list, last used, rate limits, and a “Rotate” button.
   - One-time secret reveal modal just like the admin portal.

3. **Security & Policies**
   - Enforce tenant-configured max keys per user, min rotation intervals, and alerting on inactivity.

## Usage Examples

### Create a Key
```bash
curl -X POST https://router.example.com/user/api-keys \
  -H "Authorization: Bearer sk-user-session" \
  -H "Content-Type: application/json" \
  -d '{"name": "local-dev", "rate_limit": {"rpm": 120}}'
```

### Rotate a Key
```bash
curl -X POST https://router.example.com/user/api-keys/{key_id}/rotate \
  -H "Authorization: Bearer sk-user-session"
```

### View Usage for a Key
```bash
curl https://router.example.com/user/api-keys/{key_id}/usage \
  -H "Authorization: Bearer sk-user-session"
```

## Implementation Details

| Area | Notes |
|------|-------|
| API | Reuse admin key service with scoped queries; ensure secrets only shown once. |
| Rate limits | Allow per-key overrides with inheritance from tenant defaults. |
| Notifications | Optional email/webhook when keys are rotated or revoked for auditing. |
| Portal | Use existing table components; add inline charts for per-key usage. |

## Next Steps
1. Extend key service with user-scoped methods.
2. Build `/user/api-keys` endpoints + tests.
3. Implement portal UI & UX flows.
4. Document rotation best practices for tenants.
