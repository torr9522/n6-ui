# N6 Browser UAT

Date: 2026-09-19

The N6 application was exercised with Chromium using an isolated temporary
`/etc/x-ui` mount and port `18080`. The verified Xray 26.5.3 binary was used
for the local fixture; no production database was modified.

## Results

- Desktop 1440x1000: login, n6-ui branding, N5 settings, SS2022 inbound/egress
  flows, Simple and Advanced routing, subscription Base64/Mihomo responses,
  Classic Clash guidance, QR, and egress edit/delete: PASS.
- Mobile 320x720: SS2022 inbound/key generation, share-link import, routing,
  subscription, cleanup: PASS.
- Mobile 390x844: SS2022 inbound/key generation, share-link import, routing,
  subscription, cleanup: PASS.
- Browser page errors: 0.
- Browser console errors: 0.
- Failed requests and HTTP 5xx responses: 0.

The temporary fixture records were removed after the mobile runs.
