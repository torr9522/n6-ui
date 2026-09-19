# N6 Security Scan

Scan scope: tracked source, templates, scripts, release metadata and documentation required for N6 operation.

- GitHub tokens, passwords, private keys and cookies: none found.
- Real test server IP/domain and UAT subscription credentials: removed from the N6 tree.
- Loopback addresses and RFC 5737 documentation networks in tests: allowed fixtures.
- Xray runtime key: only the public SHA256 pin is recorded.
- N5 URLs: retained only in historical/provenance documents; executable N6 fetch paths contain no N5 fallback.

Status: PASS for the current local tree. Re-run before every public release and after adding UAT artifacts.
