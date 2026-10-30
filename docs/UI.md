# UI reference (target mockup)

Based on the QtTinySA interface (v1.0.2) and `spectrum.ui` from reference v2.0.2. The port is
functionally equivalent, not pixel-for-pixel, but follows the same structure.

## Overall layout

```
┌──────────────────────────────────────────────────────────────────────┐
│ File  Settings  Help                                          (title) │
├───────────────┬──────────────────────────────────────────────────────┤
│ left panel    │  spectrum (dBm × MHz, grid, limits, markers 1–4)     │
│ ~300 px       │                                                      │
│               ├──────────────────────────────────────────────────────┤
│               │  waterfall                                           │
│               ├──────────────────────────────────────────────────────┤
│               │  bottom control panel (2 rows)                       │
└───────────────┴──────────────────────────────────────────────────────┘
```

## Left panel (top to bottom)

1. **Options**: `LNA`, `Spurs` — checkboxes.
2. **Attenuator**: auto checkbox + numeric dB field.
3. **RBW**: auto checkbox + drop-down list in kHz.
4. **Points**: auto checkbox + point count + sample repeat count.
5. **Traces** ×4: checkbox + type (`Normal`, `Max`, `Average`, `Min`).
6. Waterfall height slider (25…), battery/voltage indicator ("No Device" without a device).
7. **LNB Mode** button.

## Bottom panel (2 rows)

Row 1: points, RBW/div, band preset selector (`2G WiFi`), span/center, `<>` button,
arrows, `Max`, `Off` ×3, checkbox, **Run**.

Row 2: attenuator, start freq, band preset (`amateur`), span, `+`, `<>`,
colored marker buttons 1–4, checkbox, **3D**.

## Markers and limits

- Markers M1–M4: colored flags on the plot (yellow/cyan/orange/…), draggable;
  clicking the plot centers on the marker.
- Horizontal lines: best (−25 dB), maximum (0), damage (+6), threshold (movable),
  reference (movable); lowF/highF are the min/max detection bounds.

## What we implement in M2/M3

- M2: spectrum + waterfall + Run/Stop + points/RBW/atten/LNA/spur + frequencies.
- M3: markers, limits, band presets, LNB, traces (max/avg/min), bottom panel.
- Svelte components: `Spectrum.svelte`, `Waterfall.svelte`, `Sidebar.svelte`,
  `BottomBar.svelte`, `Markers.svelte`, `StatusBar.svelte`.

Reference screenshots are not committed to the repository; if needed, add an anonymized mockup to `docs/`.
