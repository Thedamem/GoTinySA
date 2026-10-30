<p align="center">
  <img src="assets/logo.png" alt="tinysa_go" width="148">
</p>

# tinysa_go

**English** | [Русский](README.ru.md)

tinysa_go is a desktop spectrum-analyzer application for the **tinySA** and **tinySA Ultra**, written in Go with a Wails/Svelte frontend. It is a port of [QtTinySA](https://github.com/g4ixt/QtTinySA) and keeps the original feature set while moving serial I/O, sweep parsing, logging, and file work off the UI thread.

The application ships as a single binary for Linux and Windows. A built-in HTTP server provides a separate phone interface for tuning an antenna from the same local network.

## Features

- Connection to tinySA and tinySA Ultra over USB CDC, with port discovery and diagnostics.
- Live spectrum and waterfall on canvas, with max/avg/min traces, zoom/pan, and a point tooltip.
- Sweep control: start/stop/center/span, points, RBW auto/manual, attenuator auto/manual, LNA, and Spurs.
- Peak frequency, peak level, median noise floor, SNR, and max-hold readout.
- CSV export of the current sweep.
- Raw USB traffic logging (text and binary) for diagnostics and replay.
- `tinysa-cli` for port listing and device commands without the GUI.
- Phone web UI with the live spectrum, waterfall, readouts, band presets, sweep settings, and CSV download.
- English and Russian UI; English is the default.

## Requirements

- Go 1.25 or newer
- Wails CLI v2.17
- Node.js and npm
- Linux GUI dependencies: `libgtk-3-dev libwebkit2gtk-4.1-dev`
- A tinySA or tinySA Ultra connected over USB for measurements

## Build

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0

wails build -tags webkit2_41          # Linux
wails build -platform windows/amd64   # Windows cross-build
```

For development:

```sh
wails dev -tags webkit2_41
```

`npm run dev` alone starts Vite without the Wails runtime; Go bindings will not work in that mode.

The committed `frontend/dist` directory is embedded into the Go binary. Run `npm run build` in `frontend/` after frontend changes, or let `wails build` rebuild it.

## Phone web UI

1. Connect the analyzer and start the app.
2. Open **Web** and press **Enable**. The dialog shows the current LAN address and a QR code.
3. Open the address on a phone in the same Wi-Fi network.
4. The phone UI exposes the spectrum, waterfall, peak/noise/SNR, band presets, start/stop/span, points, RBW, attenuator, LNA, Spurs, traces, and CSV download.
5. **Peak to center** and **Zoom to peak** retune around the strongest signal. **Apply** restarts the sweep with edited settings.

The server listens on `0.0.0.0:8420` or a free port if 8420 is busy. It has **no authentication**: use it only on a trusted local network and never expose the port to the internet. The server runs only while the application is open.

If Wi-Fi does not reach the antenna, enable a hotspot on the phone and connect the laptop to it. The web address stays the same; only the IP changes, and the **Web** dialog shows the current address.

## Language

The desktop and phone interfaces are available in English and Russian.

- Desktop: **Settings → Language → Русский**.
- Phone: **EN / RU** selector in the header.

The choice is stored locally in the webview or browser and is not sent to the device.

## CLI

`tinysa-cli` uses the same serial layer as the GUI and is useful for diagnostics:

```sh
go run ./cmd/tinysa-cli -list
go run ./cmd/tinysa-cli -port /dev/ttyACM0 version
```

On Windows, use a COM port such as `-port COM5`.

## Tests

```sh
cd frontend && npm ci && npm run build
go vet -tags webkit2_41 ./...
go build -tags webkit2_41 ./...
go test -tags webkit2_41 -race ./...
```

Tests use fake serial ports and backends; no hardware is required.

## CI and releases

GitHub Actions runs the checks in `.github/workflows/ci.yml` and builds Linux and Windows artifacts in `.github/workflows/build.yml`. Tags matching `v*` publish a release with the application and `tinysa-cli` binaries.

## Windows: USB device not found

1. Use a USB cable with data lines, not a charge-only cable.
2. In **Device Manager → Ports (COM & LPT)** the device should appear as a COM port (`tinySA` / `USB Serial Device`). Windows 10/11 installs the usbser driver automatically.
3. Make sure no other program holds the port: QtTinySA, a terminal, or a second copy of this application.
4. Diagnose without the GUI: `tinysa-cli.exe -list` lists ports; `tinysa-cli.exe -port COM5 version` checks the link. In the app, use **Settings → Debug… → Show ports**.

## Project layout

- `app.go`, `web.go`, `main.go` — Wails application, desktop API, and embedded web server.
- `internal/serial` — serial transport, port ownership, framing, and raw logging.
- `internal/tinysa` — protocol, sweep parser, traces, and CSV rendering.
- `internal/webui` — phone HTTP/SSE server and assets.
- `frontend/` — Svelte desktop UI and Wails bindings.
- `cmd/tinysa-cli` — command-line diagnostics.
- `docs/` — protocol, reference, feature, and UI notes.

## License

GPL-3.0-or-later. This is a derivative work of QtTinySA (GPL-3.0-or-later). The tinySA name and logo are not used as trademarks.
