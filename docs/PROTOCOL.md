# tinySA protocol (reference notes)

Source: `reference/QtTinySA/src/modules/devices.py` @ `1cc609d`.
Goal: port the reference 1:1 instead of reinventing it (and check the tinySA documentation when details are ambiguous).

## Transport

- USB CDC ACM; **VID:PID `0483:5740`**; product: `tinySA` (basic) / `tinySA4` (Ultra, Ultra+); up to 4 devices.
- pyserial opens at baudrate 576000 (not actually critical for CDC).
- Commands are ASCII, terminated with `\r`. The device echoes the command and responds with the `ch> ` prompt.
- `serialWrite`: write → read_until(`ch> `).
- `serialQuery`: write → skip the echo up to `\n` → read up to `ch> ` → the response is returned without the prompt.

## Sweep (scanraw): main loop

- Command: `scanraw <startHz> <stopHz> <points> 3\r` — auto-repeat (FW ≥ 1.4.177);
  `... 1\r` — single sweep (send the repeat yourself; legacy).
- Data: **3 bytes** per point: `'c'` + uint16 little-endian.
  Conversion: **dBm = data/32 − scale**, where `scale` is **174 (Ultra/Ultra+)**, **128 (basic)**.
- End of sweep in auto-repeat: the `}{` pair; when stopping manually, do not read extra data, send `abort\r`.
- If the user touches the screen or joystick, a `}` (or another failure) arrives and the loop stops with an info message.
- On desynchronization (`self.usb.read(2) != b'}{'`), reset the buffer and warn "display out of sync".

Read timeout for a block (the `sweepTimeout` heuristic, port as-is):

```
rbw = "auto" ? (stop-start)*7e-6 : rbw_kHz            # clip 0.2..850 (basic: 3..600)
timeout = ((stop-start)/20e3)/(rbw^2) + points/500
if spur on/auto and stop > 800 MHz: timeout *= 2
timeout = timeout * 20 / points + 1                    # minimum 1 s
```

## Protocol commands used

| Command | Purpose |
|---|---|
| `version` | FW version (string `tinySA4_v1.4-…`) |
| `info` | device description |
| `deviceid` | serial number |
| `vbat` | battery voltage |
| `time b 0xYYMMDD 0xHHMMSS` | clock synchronization |
| `scanraw …` | sweep (see above) |
| `abort` / `abort on` / `abort off` | abort/interrupt mode |
| `pause` / `resume` / `reset` | pause/resume/reset |
| `attenuate auto\|0..n` | attenuator |
| `lna on/off` | LNA (Ultra only) |
| `spur auto\|on\|off` | spur suppression (basic: auto → on) |
| `rbw <kHz\|auto>` | RBW |
| `repeat <n>` | number of measurement repeats per point |
| `sweep start/stop <Hz>` | default device range |
| `sd_list`, `sd_read <file>` | device SD card (listing/reading) |
| `example` | example command (used for link checks) |

## `.npy` recordings (compatibility)

- Original format (`Recorder`): 2D float array; **row 0** = `[serial, freq1…freqN]`,
  subsequent rows = measurements; **col 0** = timestamps (sweep acquisition time), then dBm.
- Limit of ~`10e6` fields per array (about 32 MB per file).
- Our port must read and write the same format (and `.npy` with float32/float64).

## CSV export (from `exporters.py`)

- **Generic CSV**: frequency in MHz (3 decimals) + level (4 decimals).
- **Shure WWB**: one trace per file (select trace 1–4), frequency in MHz.
- **Sennheiser WSM**: `;` delimiter, WSM header, frequency in kHz, level "RF level".
