# Gateway Modes

## OpenAI gateway

Sub2API exposes OpenAI-compatible routes and also supports OpenAI Responses over
WebSocket when enabled.

Important config:

- `gateway.openai_ws.enabled`
- `gateway.openai_ws.force_http`
- `gateway.openai_ws.responses_websockets`
- `gateway.openai_ws.responses_websockets_v2`
- `gateway.openai_ws.ingress_mode_default`
- `gateway.openai_ws.max_conns_per_account`
- `gateway.openai_ws.queue_limit_per_conn`

Use `force_http` as the fast rollback switch if WebSocket behavior needs to be
disabled without removing the surrounding config.

## Gemini gateway

Gemini traffic can be backed by OAuth accounts and optional quota policy config:

- `gemini.oauth.*`
- `gemini.quota.policy`

## Antigravity gateway

Antigravity accounts expose dedicated Claude and Gemini entrypoints and can also
participate in hybrid scheduling.

Important config:

- `gateway.antigravity_fallback_cooldown_minutes`
- `gateway.antigravity_extra_retries`

## Cross-cutting notes

- Keep `gateway.connection_pool_isolation` at `account_proxy` unless you have a
  measured reason to relax isolation.
- `gateway.force_codex_cli` and
  `gateway.codex_image_generation_bridge_enabled` are compatibility switches for
  Codex-specific request handling.
