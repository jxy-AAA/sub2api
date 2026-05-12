# Auth Providers

Sub2API currently documents three built-in external auth providers in config:
LinuxDo Connect, generic OIDC, and WeChat Connect.

## LinuxDo Connect

Config key: `linuxdo_connect`

Important fields:

- `enabled`
- `client_id`
- `client_secret`
- `authorize_url`
- `token_url`
- `userinfo_url`
- `redirect_url`
- `frontend_redirect_url`
- `token_auth_method`
- `use_pkce`

## Generic OIDC

Config key: `oidc_connect`

Important fields:

- `enabled`
- `provider_name`
- `client_id`
- `client_secret`
- `issuer_url` or `discovery_url`
- `authorize_url`
- `token_url`
- `userinfo_url`
- `jwks_url`
- `redirect_url`
- `frontend_redirect_url`
- `use_pkce`
- `validate_id_token`
- `allowed_signing_algs`
- `require_email_verified`

## WeChat Connect

Config key: `wechat_connect`

Important fields:

- `enabled`
- `app_id` / `app_secret`
- `open_app_id` / `open_app_secret`
- `mp_app_id` / `mp_app_secret`
- `mobile_app_id` / `mobile_app_secret`
- `open_enabled`
- `mp_enabled`
- `mobile_enabled`
- `mode`
- `scopes`
- `redirect_url`
- `frontend_redirect_url`

## Rollout notes

- Keep callback URLs aligned with the public reverse-proxy origin.
- Treat provider client secrets exactly like `JWT_SECRET` or database passwords.
- Configure provider login together with `totp.encryption_key` if you require
  account-level 2FA for local credentials as well.
