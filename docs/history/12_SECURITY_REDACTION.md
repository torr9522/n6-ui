# Security Redaction Policy

This history archive is public-safe by design.

## Never Commit

- real test server IP or SSH host
- SSH username or password
- panel administrator username or password
- API tokens
- GitHub tokens
- `Authorization` or `Bearer` headers
- cookies or sessions
- SSH private or public keys
- TLS private keys
- database backups
- production `config.json`
- production `x-ui.db`
- subscription tokens
- real UUIDs or proxy credentials from runtime UAT
- Reality private keys or real Short IDs
- SOCKS or Shadowsocks passwords
- private test domains
- raw access logs
- browser profiles
- packet captures
- `.env`
- shell history
- signed or temporary download URLs

## Redaction Tokens

Use these placeholders in public docs:

- `<TEST_SERVER>`
- `<TEST_DOMAIN>`
- `<TEST_USER>`
- `<REDACTED>`
- `<REDACTED_TOKEN>`
- `<REDACTED_SOURCE_IP>`

## Allowed Examples

- loopback examples such as `127.0.0.1`
- wildcard bind addresses such as `0.0.0.0`
- RFC 5737 documentation networks such as `192.0.2.0/24`,
  `198.51.100.0/24`, and `203.0.113.0/24`
- public project URLs such as `https://github.com/torr9522/n5-ui`
- generic routing examples such as `openai.com`

## Evidence Policy

Raw evidence stays in private local archives. Public history stores only
redacted structured summaries.
