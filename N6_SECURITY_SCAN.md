# N6 Security Scan

Scan scope: tracked source, templates, scripts, release metadata and documentation required for N6 operation.

- GitHub tokens, passwords, private keys and cookies: none found.
- Real test server IP/domain and UAT subscription credentials: removed from the N6 tree.
- Loopback addresses and RFC 5737 documentation networks in tests: allowed fixtures.
- Xray runtime key: only the public SHA256 pin is recorded.
- N5 URLs: retained only in historical/provenance documents; executable N6 fetch paths contain no N5 fallback.

Additional checks performed before the local freeze:

- executable N5 repository/runtime URLs: 0
- GitHub token/private-key matches: 0
- supplied test-server address or domain: 0
- non-loopback IP literals are confined to protocol fixtures/documentation
  (for example public DNS fixture addresses in tests), not credentials or
  deployment configuration
- `git diff --check`: PASS

Status: PASS for the current local tree. Formal public release remains gated
on a disposable clean-server and reboot acceptance.
