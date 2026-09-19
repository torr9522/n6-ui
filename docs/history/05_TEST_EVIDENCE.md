# Test Evidence Summary

This file is a redacted public summary. It intentionally does not include raw
logs, DB dumps, browser traces, tokens, cookies, server addresses, domains,
source IPs, or credentials.

## Automated Checks

The verified candidate and release commit passed:

- `go test ./...`
- `go build ./...`
- `go vet ./...`
- `go test -race ./web/service/n5/... ./web/controller/n5/...`
- `git diff --check`

## Clean Install Gate

- OS family: Debian
- Validated version: Debian 11 Bullseye
- Architecture: amd64/x86_64
- Pre-install runtime state: no previous `/etc/x-ui`, `/usr/local/x-ui`, old DB,
  or old service state was used
- DB: created fresh by the application
- Service: `x-ui` active and enabled
- Timer: `xui-portlimit-sync.timer` active and enabled
- Runtime: Xray 26.5.3 amd64

## Browser Gates

Desktop viewport:

- 1366x768
- real login page used
- core pages checked: system status, inbound list, N5 egress, egress rules,
  traffic rules, rule group detail, N5 settings, subscription, Access IP
- no critical console errors, page errors, failed requests, or HTTP 500

Mobile viewports:

- 320x720
- 390x844
- additional audit viewports: 360x800 and 412x915
- N5 egress, egress rules, traffic rules, manage rules, and N5 settings were
  usable
- edit/delete/test/manage/save/cancel controls were reachable

Known non-blocker:

- `/favicon.ico` may return 404

## Runtime Feature Matrix

Inbound:

- VMess TCP: create/edit/enable/disable PASS
- VLESS TCP: create/edit/enable/disable PASS
- VLESS Reality TCP Vision: creation/configuration PASS

Egress:

- SOCKS egress: create/edit/test PASS
- additional egress smoke used distinguishable fixtures where needed

Routing:

- ALL -> egress target PASS
- AI -> egress target PASS
- custom group -> egress target PASS
- direct Custom -> egress target PASS

Priority:

- direct custom exact/suffix/keyword/regexp outranks custom group and builtins
- custom group outranks AI/Game/Streaming/ALL
- creation order does not affect Simple routing result
- conflict warning appears and explains which rule wins

Subscription:

- default/base64 output PASS
- clash output PASS
- mihomo output PASS
- QR generation PASS

Access IP:

- verified with real proxy traffic
- source IP and inbound port were observed by the feature
- actual source IP is not published

## P1/P2 Regression Gates

- P1-01 egress reference protection PASS
- P1-02 inbound delete cleanup PASS
- P1-03 Advanced/Simple ownership protection PASS
- P2-01 Simple transaction rollback PASS
- P2-02 disabled builtin backend protection PASS

## SQLite Busy Gate

- handled concurrency returned controlled responses
- no HTTP 500 for handled busy/locked cases
- no raw SQL leaked
- no raw SQLite lock text leaked

## Network E2E

- real TCP E2E PASS
- real UDP E2E PASS
- outbound routing tag evidence was checked locally and summarized here

## Reboot Gate

- real OS reboot PASS
- service came back active/enabled
- DB persisted
- created inbound, egress, routing, rule group, and subscription data persisted
- post-reboot TCP PASS
- post-reboot UDP PASS

## DB Consistency

All checked counters were zero:

- orphan policy
- orphan rule
- orphan binding
- missing egress target
- missing inbound binding
- metadata missing rule IDs
- duplicate binding
- pool member missing egress
