# Risk Control and Content Audit

## Scope

The current in-repo "risk control" functionality is centered on content
moderation and auditability for gateway traffic.

## Content moderation

The moderation service supports three modes:

- `off` — disable checks
- `observe` — score and record hits without blocking traffic
- `pre_block` — block flagged requests before they reach the upstream

## Key settings

The moderation config is stored in system settings rather than `config.yaml`.
Important fields include:

- `enabled`
- `mode`
- `base_url`
- `model`
- `api_key` / `api_keys`
- `sample_rate`
- `all_groups` / `group_ids`
- `record_non_hits`
- `thresholds`
- `worker_count`
- `queue_size`
- `block_status`
- `block_message`
- `email_on_hit`
- `auto_ban_enabled`
- `ban_threshold`
- `violation_window_hours`
- `hit_retention_days`
- `non_hit_retention_days`
- `pre_hash_check_enabled`

## Gateway coverage

Moderation checks are wired into:

- Anthropic-compatible message routes
- OpenAI chat/completions routes
- OpenAI responses routes, including WebSocket ingress
- Gemini routes
- OpenAI image routes

## Audit workflow

- Every moderation decision can be logged for later review.
- Admin APIs support status inspection, log listing, unbanning, and flagged-hash cleanup.
- Use `observe` before `pre_block` when introducing new thresholds.
