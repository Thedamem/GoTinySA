# QtTinySA reference

- **Repository:** https://github.com/g4ixt/QtTinySA
- **Pinned commit:** `1cc609d6876184fe308de72f78846decf4752246` (2026-10-11, "v2.0.2")
- **Local checkout:** `reference/QtTinySA` (in `.gitignore`, not committed to the repository)
- **Reference license:** GPL-3.0-or-later → `GoTinySA` is a derivative work, also GPL-3.0-or-later.

## Recreating the reference checkout

```sh
git clone --depth 1 https://github.com/g4ixt/QtTinySA reference/QtTinySA
git -C reference/QtTinySA log -1 --format='%H %ci %s'   # must match the pin above
```

Update the pin only deliberately; record the reason and the new commit in this file.

## Reference layout

| Path | Description |
|---|---|
| `src/QtTinySA.py` | Main file (~81 KB): `Analyser` class (data router), `ModelView`, all UI and menu logic |
| `src/modules/devices.py` | Serial/devices: `USBdevice`, `Tiny` (basic/Ultra), `Nano`, `Recorder` classes; `scanraw` loop; timeout heuristics |
| `src/modules/graphs.py` | Plots: spectrum, waterfall, histogram, PhaseNoiseGraph, PolarGraph |
| `src/modules/exporters.py` | CSV export: generic + Shure WWB + Sennheiser WSM |
| `src/modules/*.ui` | Qt Designer files: `spectrum.ui` (main window), `settings`, `bands`, `offset`, `filebrowse`, `phasenoise`, `fading`, `pattern` |
| `src/modules/*_baseline.txt` | Baseline levels (10/1152 points), used for Phase Noise among other things |
| `src/QtTSAprefs.db` | Settings SQLite database (template; copied to the config directory on first run) |
| `correction/` | Correction tables (example `23041023/table.csv`) |
| `frequency_lists/` | Frequency lists (beacons, Wi-Fi, PMR) in CSV |
| `Sample_preset_markers/` | Sample presets (broadcast, TV channels, event markers) |
| `sweep_files/` | Sample `.npy` recordings (compatibility format!) |
| `flow_diagram/`, `Screenshots/` | Data-flow diagram, screenshots |
