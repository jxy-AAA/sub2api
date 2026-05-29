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

### OpenAI-compatible clients

- Use this mode for GPT-style tools such as Cursor and any OpenAI SDK-compatible client.
- Point the client Base URL at your Sub2API deployment and keep using the usual routes:
  - `/v1/chat/completions`
  - `/v1/responses`
  - `/v1/models`
- The user-facing `/model-market` page is the fastest way to confirm which models are exposed through OpenAI-compatible upstreams and what their pricing looks like.

## Anthropic-compatible gateway

Sub2API also supports Anthropic-compatible routing for Claude-style clients.

### Anthropic-compatible clients

- Use this mode for Claude Code and Anthropic SDK-compatible clients.
- Point the client Base URL at your Sub2API deployment and use:
  - `/v1/messages`
  - `/v1/models` when that model listing route is exposed by the upstream integration
- Check `/model-market` to see which models are marked as Anthropic-compatible before wiring Claude Code to a specific deployment.

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
