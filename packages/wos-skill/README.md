# WOS Skills

## Install directly using the WOS Go binary (recommended)

The native server/operator executable contains all five skills and installs them
without Node.js, npm, network access, a running WOS server or an API credential.

```bash
wos skill install --agent codex
wos skill install --agent claude
wos skill install --agent hermes
wos skill install --agent openclaw
wos skill install --agent all --dry-run
wos skill install --agent all
wos skill doctor --agent codex
wos skill update --agent codex
wos skill uninstall --agent codex
```

The native installer reads the canonical skill sources embedded during the Go
build, uses the existing `.wos-skill.json` SHA-256 ownership format, preserves
local modifications and refuses symlink/unmanaged destinations. The optional
Node installer source remains only as a compatibility fixture; Node.js is **not** required
when using `wos skill`.

## Public installation

```sh
npm install -g wosctl@latest
wos skill install --agent codex
```

`wosctl` is the single public npm package. This directory contains the canonical embedded skill source and historical Node installer compatibility fixtures; it is not published as a separate npm package. Native Go owns installation, updates, doctor and removal.
