# Channel Monitoring

## What it covers

Sub2API ships a built-in channel monitor for upstream health checks, availability
tracking, and historical rollups.

## Main pieces

- Feature switch: `channel_monitor_enabled`
- Default interval: `channel_monitor_default_interval_seconds`
- Entities: `ChannelMonitor`, `ChannelMonitorHistory`, `ChannelMonitorDailyRollup`
- Reusable request definitions: `ChannelMonitorRequestTemplate`

## Admin workflow

1. Enable the feature in system settings.
2. Create or reuse a request template for shared headers/body overrides.
3. Create one or more monitors per upstream/channel/model combination.
4. Trigger a manual run or wait for the periodic runner.
5. Inspect latest status, history, and daily rollups in the admin UI.

## Operational notes

- The runner uses distributed locking so only one instance executes a given
  scheduled check at a time.
- Validation is SSRF-aware and avoids exposing internal topology details in
  error messages.
- Cleanup and daily maintenance run through the ops cleanup path.
