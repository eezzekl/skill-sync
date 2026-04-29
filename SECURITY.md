# Security Policy

## Supported Versions

Only the latest stable release receives security fixes.

| Version | Supported |
|---------|-----------|
| latest  | ✅        |
| older   | ❌        |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Report security issues privately via [GitHub Security Advisories](https://github.com/ezzek/skill-sync/security/advisories/new).

### What to Include

- A clear description of the vulnerability
- Steps to reproduce
- The potential impact (path traversal, data corruption, unintended file writes, etc.)
- Any suggested mitigations you've identified

## Scope

skill-sync is a local-first CLI tool that reads `SKILL.md` files from disk and writes them to configured target directories. The attack surface is intentionally small.

**In scope:**
- Path traversal via malicious `targets` entries in the config file
- Writes outside the expected target directories
- Malicious content in a `SKILL.md` that propagates silently to all targets
- Hash collision allowing a lower-version skill to overwrite a newer one

**Out of scope:**
- Issues requiring physical access to the machine where skill-sync is installed
- Issues that require the attacker to already have write access to the user's home directory

## Recognition

Responsible disclosures are recognized in the release notes of the version that contains the fix.
