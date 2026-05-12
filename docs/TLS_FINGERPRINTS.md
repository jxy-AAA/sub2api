# TLS Fingerprints

## Purpose

Sub2API can simulate account-specific TLS client fingerprints when talking to
upstream providers.

## Relevant config

- Global switch: `gateway.tls_fingerprint.enabled`
- Data model: `TLSFingerprintProfile`
- Runtime conversion: `TLSFingerprintProfile.ToTLSProfile()`

## How it works

- The global config keeps the feature available.
- Individual accounts or account-linked profiles decide whether a custom
  fingerprint is actually used.
- Empty profile slices fall back to built-in defaults in the TLS fingerprint package.

## Operational notes

- Combine TLS fingerprints with dedicated proxies when you want stronger
  per-account isolation.
- Keep the feature disabled for accounts that do not need it; start with the
  default profile before adding custom ones.
