# AGENTS.md

TinySA spectrum-analyzer GUI: Go 1.25 + Wails v2.17 desktop app (Svelte 5), an
embedded phone web UI, and `tinysa-cli`. Port of QtTinySA; the pinned reference
checkout is gitignored at `reference/QtTinySA` (commit `1cc609d`, see
`docs/REFERENCE.md`). GPL-3.0-or-later.

## Build and run

- Install the matching Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0`.
- Linux GUI deps: `libgtk-3-dev libwebkit2gtk-4.1-dev`.
- Dev: `wails dev -tags webkit2_41` starts Vite and regenerates bindings.
  `npm run dev` alone has no Wails runtime, so binding calls fail.
- Build: `wails build -tags webkit2_41` (Linux) or
  `wails build -platform windows/amd64`. A plain `go build` does not rebuild
  the frontend and embeds whatever is already in `frontend/dist`.
- CLI: `go run ./cmd/tinysa-cli -list`;
  `go run ./cmd/tinysa-cli -port /dev/ttyACM0 version`.
- Runtime log: `~/.config/tinysa_go/tinysa_go.log`
  (`%AppData%\tinysa_go\...` on Windows).

## Checks

CI is GitHub Actions (`.github/workflows/ci.yml`) and runs, in order:

1. `cd frontend && npm ci && npm run build`
2. `go vet -tags webkit2_41 ./...`
3. `go build -tags webkit2_41 ./...`
4. `go test -tags webkit2_41 -race ./...`

- Single test: `go test -tags webkit2_41 ./internal/serial -run TestQueryVersion -v`.
- Tests use fake serial ports/backends; no hardware is required. Do not add
  device-dependent tests.
- `npm run check` is not in CI and currently reports 2 pre-existing errors in
  `frontend/src/Spectrum.svelte` plus warnings. Use it for new diagnostics, not
  as a clean gate; don't mistake those for your regression.
- `frontend/dist` is committed and embedded via `go:embed all:frontend/dist`.
  Run `npm run build` there after UI changes, or plain Go builds/tests embed
  stale assets. `wails build` rebuilds it automatically.
- `frontend/wailsjs` is generated and committed. After changing exported `App`
  methods or bound types, run `wails generate module` (or `wails build`); do
  not hand-edit it. Keep `package-lock.json` authoritative and use `npm ci`.

## Architecture

- `app.go` is the Wails-bound `App`: exported methods are the desktop API, and
  updates are pushed with `runtime.EventsEmit` (`sweep`, `sweep:error`,
  `connect:step`).
- `internal/serial` has one pump goroutine per port; queries, writes, and
  streams serialize. A running `scanraw` stream owns the pump, so do not issue
  commands during a sweep: stop it, or use `App.ApplySweep`, which stops and
  awaits the old stream before starting a new one.
- `internal/tinysa` is the protocol/sweep engine. Units are easy to mix up:
  Start/Stop in Hz, RBW in kHz (`0` = auto), Atten in dB (`-1` = auto), Points
  2..2000. dBm = uint16/32 − scale (`128` basic, `174` Ultra/Ultra+). See
  `docs/PROTOCOL.md`.
- Desktop UI is `frontend/src/App.svelte` plus `Spectrum.svelte` and
  `Waterfall.svelte`. `docs/UI.md` is a target mockup, not a file map; some
  components it lists do not exist yet.
- `internal/webui` is the separate phone UI: plain HTML/CSS/JS in
  `internal/webui/assets`, no build step, unauthenticated on `0.0.0.0:8420`
  (falls back to a free port). Change phone UI there, desktop UI in
  `frontend/src`. New endpoints go through `webui.Backend`; update the fake in
  `internal/webui/server_test.go`. Its CSP forbids inline script/style.
- `cmd/tinysa-cli` shares the same serial/usblog code; keep diagnostics and GUI
  behavior consistent.

## Conventions and traps

- Desktop and phone UI are bilingual (English default, Russian available) via
  `frontend/src/i18n.ts` and the dictionary in `internal/webui/assets/app.js`.
  Keep both dictionaries in sync when changing UI text. CLI output, Go
  identifiers, comments, and diagnostic errors are English.
- New Go source files carry `// SPDX-License-Identifier: GPL-3.0-or-later`,
  matching most of the tree.
- Windows serial support is deliberately defensive: 12 s connect budget, baud
  fallbacks, DTR/RTS assertion, close-port-before-wait in `Client.Close`, and
  bounded cleanup. Preserve the timeouts and ordering; do not simplify them.
- `App` uses separate mutexes (`mu`, `frameMu`, `sweepMu`, `webMu`) and must not
  hold `mu` across serial I/O. Follow the existing lock discipline.
- The web server has no auth and is LAN-only by design; do not expose it to the
  internet or assume remote-access support.
- `reference/QtTinySA` is a gitignored, pinned checkout: read it for parity, do
  not commit it or bump the pin casually.
