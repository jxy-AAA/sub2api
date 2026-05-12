# Subscriptions

## Overview

Sub2API supports paid subscription plans in addition to balance-based billing.
The implementation spans plan configuration, user subscription assignment, and
payment fulfillment.

## Main concepts

- `SubscriptionPlan` — sellable plan definition
- `UserSubscription` — assigned plan on a concrete user/group
- `purchase_subscription_url` — optional purchase entry shown in the UI

## Admin capabilities

- Create, update, list, and delete subscription plans
- Assign or extend subscriptions for users
- Inspect active and historical user subscriptions

## User-facing flow

1. User opens the purchase entry or plan list.
2. A payment order is created against a subscription plan.
3. Successful fulfillment assigns or extends the user subscription.
4. Gateway billing can reference the active subscription for group access.

## Operational notes

- Plan deletion does not erase historical subscription records.
- Subscription maintenance is controlled by:
  - `subscription_cache.*`
  - `subscription_maintenance.worker_count`
  - `subscription_maintenance.queue_size`
