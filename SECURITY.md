# Security Policy

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Use one of the following private channels:

1. **Preferred:** [Report a vulnerability](https://github.com/getvelora/velora/security/advisories/new) via GitHub's private vulnerability reporting.
2. **Email:** michael@michaeldelle.com

When reporting, include:

- A description of the issue and its impact.
- Steps to reproduce, ideally with a minimal example.
- The affected version, commit SHA, or container tag.
- Any suggested mitigations if you have them.

## What to expect

- Acknowledgement within **3 business days**.
- An initial assessment and severity rating within **7 business days**.
- Coordinated disclosure: we will work with you on a fix and a public advisory. Please give us a reasonable window before publishing details.

## Supported versions

Velora is pre-1.0. Only the latest released container tag and the current `develop` branch receive security fixes. Once a stable release line exists, this section will be updated.

## Scope

In scope:

- The Go server in `apps/server`.
- The web client in `apps/web`.
- The published container image and the `./velora` helper.

Out of scope:

- Vulnerabilities in third-party dependencies that are already publicly tracked (please report those upstream).
- Issues that require physical access to a user's host machine.
- Self-inflicted misconfiguration (e.g., exposing the server to the public internet without a reverse proxy and authentication).
