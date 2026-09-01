# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

Please report security vulnerabilities privately via GitHub Security Advisories:

1. Go to https://github.com/sergiorandria/hsm-go-client/security/advisories/new
2. Describe the vulnerability, impact, and reproduction steps.

We aim to acknowledge reports within 48 hours and provide a fix or mitigation plan within 7 days.

Do not report vulnerabilities via public issues.

## Handling Secrets

- Never commit `BearerToken` values or HSM credentials.
- Use environment variables (`HSM_BEARER_TOKEN`, `HSM_BASE_URL`) in CI/examples.
