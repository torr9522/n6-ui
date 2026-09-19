# N6 Fresh Install Acceptance

Date: 2026-09-19

The available Debian 12 host already had an active N5 `x-ui` service and an
existing `/etc/x-ui/x-ui.db`, so it was not safe or accurate to overwrite it.
Instead, the official N6 installer was executed inside a root `unshare -m`
mount namespace with isolated `/etc/x-ui`, `/usr/local/x-ui`, `/usr/bin`,
systemd, and logrotate paths. The host application and database remained
untouched.

The test used only N6 sources:

- installer: `raw.githubusercontent.com/torr9522/n6-ui/main/install.sh`
- application package: N6 `v0.1.0-rc.1`
- runtime package: N6 `n6-runtime-26.5.3-amd64`

Results:

- binary package download and extraction: PASS
- runtime extraction: PASS
- Xray version: 26.5.3
- Xray binary SHA256: `128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e`
- generated isolated database and random admin settings: PASS
- n6-ui login HTML and browser endpoint: PASS
- installer fallback to N5 or another panel repository: none

This is an isolated clean-install acceptance. A separate physical clean Debian
machine is still required for the final production-install gate.
