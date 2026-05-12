# TOTP / 2FA

## Required configuration

Config key: `totp.encryption_key`

Environment override: `TOTP_ENCRYPTION_KEY`

## Why this matters

If the encryption key is left empty, the service can generate a random key on
startup. That is acceptable only for throwaway development environments.

For any persistent environment, set a fixed 32-byte key before enabling user
TOTP. Otherwise a restart can make previously enrolled TOTP secrets unreadable.

## Recommended rollout

1. Generate a secret, for example with `openssl rand -hex 32`.
2. Store it in `.env`, `/etc/sub2api/sub2api.env`, or your secret manager.
3. Restart the service once with the fixed key in place.
4. Only then enable TOTP in admin settings or ask users to enroll.
