# n6-ui Runtime Release Pin

The N6 runtime asset is the binary inherited from the validated N5 v0.2.0 release during bootstrap. It is not rebuilt or modified.

- Version: Xray 26.5.3 amd64
- Asset name: `Xray-linux-64.zip`
- Asset SHA256: `98e1cfe7b8a85d833edcd5101530f2d67609505d832eac15ac2a236f3374bbbe`
- Extracted binary SHA256: `128f9c34811ee74b3770eef7010d011e3946e85dfab28f2ed1804e380461b05e`
- N6 final hosting: `https://github.com/torr9522/n6-ui/releases/download/n6-runtime-26.5.3-amd64/Xray-linux-64.zip`

The installer and in-panel updater verify the extracted binary SHA256 before accepting the runtime. A mismatch is a hard failure; there is no N5 or third-party runtime fallback.
