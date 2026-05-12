# Debug Environment Variables

Sub2API currently supports three documented `SUB2API_DEBUG_*` switches in the
gateway service.

## Accepted boolean values

For boolean-style switches, the parser treats the following as enabled:

- `1`
- `true`
- `yes`
- `on`

## Variables

### `SUB2API_DEBUG_GATEWAY_BODY`

Purpose: write original/forwarded gateway bodies to a file for deep request-path debugging.

Accepted forms:

- `SUB2API_DEBUG_GATEWAY_BODY=1` — write to `gateway_debug.log`
- `SUB2API_DEBUG_GATEWAY_BODY=/path/to/file.log` — write to that exact file
- `SUB2API_DEBUG_GATEWAY_BODY=/path/to/dir` — write `gateway_debug.log` in that directory

Notes:

- The file can contain sensitive request and response content.
- Enable only for short-lived incident debugging.

### `SUB2API_DEBUG_MODEL_ROUTING`

Purpose: emit extra routing diagnostics around requested-model normalization,
account selection, fallback, and Anthropic routing decisions.

Use this when a request lands on an unexpected upstream model or account.

### `SUB2API_DEBUG_CLAUDE_MIMIC`

Purpose: emit extra diagnostics around Claude-mimic request shaping and
compatibility behavior.

Use this when debugging request transformation between client behavior and
Claude-compatible gateway expectations.
