# QtTinySA feature inventory → parity checklist

Source: `reference/QtTinySA` @ `1cc609d` (v2.0.2), `changelog.txt`, `.ui` files.
Statuses: `[ ]` not started · `[~]` in progress · `[x]` done.

## Main window (`spectrum.ui`: 80 widgets, 4 menus)

Menus:
- [ ] File: Browse TinySA (SD files), Recordings (load `.npy`), Quit
- [ ] Settings: Settings, Presets (bands/markers), Correction
- [ ] Measurements: Phase Noise, Fading, Antenna Pattern
- [ ] Help: About

Panels/controls (widget names from `.ui`):
- [ ] Devices: `dev0..dev3` (4 devices), `split_scan`
- [ ] Input: `atten_box` + `atten_auto`, `lna_box`, `spur_box` (auto/on/off), `rbw_box` + `rbw_auto`
- [ ] Frequency Points: `points_box` + `points_auto`, `sampleRepeat`, `mixerMode` (LNB)
- [ ] Sweep: `scan_button` (Start/Stop), `start_freq`, `centre_freq`, `stop_freq`, `span_freq`, zero span, sync time
- [ ] Traces ×4: `trace1..trace4`, types `t1_type..t4_type` (Live/Min/Max/Avg), `memBox`, `avgBox`
- [ ] Waterfall: show/hide (`waterfall_size`), `waterfall_auto` (auto range), `wf_2D` (2D/3D), `histogram`
- [ ] Record/Play: `record`, `play`, `stop`, `eject`, `speed`, `vortex` (position slider)
- [ ] Markers: `m1_type..m4_type`, `m1track..m4track`, `mkr_start`, `mkr_centre`, `setToMkr`, `presetMarker` (+`presetLabel`), `addFix`; draggable; click to center
- [ ] Limit lines: best (−25), maximum (0), damage (+6), threshold (movable), lowF/highF (min/max detection bounds), reference (movable)
- [ ] `band_box` + `filterBox` (band presets/types), `setRange` (`<>`)
- [ ] Status bar: battery (vbat), refresh-rate indicator

## Dialogs

- [ ] `bands.ui` — Presets: frequency table (bands/markers) + preset type table, add/delete/filter, import/export
- [ ] `settings.ui` — preferences: themes, trace colors, GUI refresh interval, `saveFolder`, detection threshold, max scan freq (>6 GHz), syncTime, RBW lists, etc.
- [ ] `offset.ui` — Correction: correction table (`mode`, `entry`, `frequency`, `dB`), import/export, apply to display
- [ ] `filebrowse.ui` — tinySA SD: file list (`sd_list`), bmp preview, `sd_read` selected file, save
- [ ] `phasenoise.ui` — Phase Noise: dBc/Hz vs offset (log X), center tone button, baseline correction
- [ ] `fading.ui` — Fading: selected marker levels over time (multi-plot), `timePoints`
- [ ] `pattern.ui` — Antenna Pattern: polar plot (4 channels × 40 points), measure, start/stop

## Engine/devices

- [ ] tinySA (basic) and tinySA4 (Ultra/Ultra+); detect `0483:5740` by product string; up to 4 devices
- [ ] `scanraw` auto-repeat (FW ≥ 1.4.177) and legacy mode; correct synchronization (`}{`), `abort`
- [ ] Limits/scales: scale 128/174; frequency limits; RBW clip; `spur` (basic has no auto)
- [ ] `sampleRepeat` (`repeat`), sync time (`time b ...`), `pause/resume/reset`
- [ ] Zero span, split scan
- [ ] NanoVNA — experimental (low priority, after the core)

## Recording and files

- [ ] Record sweeps to `.npy` (compatible with the original): row0 = [serial, freqs…], col0 = timestamps; limit ~10e6 fields; up to 4 recordings
- [ ] Playback: play/pause, position slider, speed, progress
- [ ] tinySA SD browser with bmp preview and save
- [ ] CSV export: generic, Shure WWB, Sennheiser WSM; plot image export
- [ ] SQLite settings `QtTSAprefs.db` + import/export of preset/band/correction CSV

Settings database tables (v2.0.2 template): `boolean`, `combo`, `rbw`, `mapping`, `trace`, `freqtype`,
`SVGColour`, `settings`, `correction`, `recording`, `checkboxes`, `numbers`, `frequencies` (196 rows).

## Assets to port

- [ ] `correction/`, `frequency_lists/`, `Sample_preset_markers/` (CSV), `sweep_files/` examples
- [ ] `modules/*_baseline.txt` (for Phase Noise)

## Progress

- **M1 (done):** serial layer (port ownership, framing), raw USB log (text + binary), CLI, in-app console, Linux/Windows builds.
- **M2 (done, pending hardware verification):** `scanraw` sweep engine (stream, frame parser, `abort`),
  canvas spectrum and waterfall, Run/Stop, start/center/stop/span, points, RBW auto/manual,
  atten auto/manual, LNA, Spurs, fps counter.

## Deliberate differences in our port

- UI: Svelte/TS (Wails) instead of Qt widgets — functional equivalence, not pixel-for-pixel.
- Concurrency: the UI thread does not perform I/O; sweep/recording/file workers run on goroutines (this is the point of the project).
- `+` raw USB log in debug mode (text + binary for replay/fixtures).
- `+` local web UI for phones: spectrum, waterfall, peak/SNR, sweep controls, and CSV.
- `+` tinySA firmware update from the app (DFU, with dump backup) — milestone M8.
- `+` CI/CD: Linux/Windows/AppImage builds and releases on `v*` tags.
