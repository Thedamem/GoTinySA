# Инвентарь функций QtTinySA → чек-лист паритета

Источник: `reference/QtTinySA` @ `1cc609d` (v2.0.2), `changelog.txt`, `.ui`-файлы.
Статусы: `[ ]` не начато · `[~]` в работе · `[x]` принято.

## Главное окно (`spectrum.ui`: 80 виджетов, 4 меню)

Меню:
- [ ] File: Browse TinySA (SD-файлы), Recordings (загрузка `.npy`), Quit
- [ ] Settings: Settings, Presets (bands/markers), Correction
- [ ] Measurements: Phase Noise, Fading, Antenna Pattern
- [ ] Help: About

Панели/контролы (имена виджетов из `.ui`):
- [ ] Devices: `dev0..dev3` (4 прибора), `split_scan`
- [ ] Input: `atten_box` + `atten_auto`, `lna_box`, `spur_box` (auto/on/off), `rbw_box` + `rbw_auto`
- [ ] Frequency Points: `points_box` + `points_auto`, `sampleRepeat`, `mixerMode` (LNB)
- [ ] Свип: `scan_button` (Start/Stop), `start_freq`, `centre_freq`, `stop_freq`, `span_freq`, zero span, sync time
- [ ] Traces ×4: `trace1..trace4`, типы `t1_type..t4_type` (Live/Min/Max/Avg), `memBox`, `avgBox`
- [ ] Waterfall: show/hide (`waterfall_size`), `waterfall_auto` (auto range), `wf_2D` (2D/3D), `histogram`
- [ ] Record/Play: `record`, `play`, `stop`, `eject`, `speed`, `vortex` (слайдер позиции)
- [ ] Маркеры: `m1_type..m4_type`, `m1track..m4track`, `mkr_start`, `mkr_centre`, `setToMkr`, `presetMarker` (+`presetLabel`), `addFix`; перетаскивание; клик — центрирование
- [ ] Limit lines: best (−25), maximum (0), damage (+6), threshold (movable), lowF/highF (границы min/max-детекции), reference (movable)
- [ ] `band_box` + `filterBox` (пресеты bands/типы), `setRange` (`<>`)
- [ ] Статусбар: батарея (vbat), индикатор скорости обновления

## Диалоги

- [ ] `bands.ui` — Presets: таблица частот (bands/markers) + таблица типов пресетов, add/delete/filter, import/export
- [ ] `settings.ui` — предпочтения: темы, цвета трасс, интервал обновления GUI, `saveFolder`, порог детекции, max scan freq (>6 ГГц), syncTime, RBW списки и пр.
- [ ] `offset.ui` — Correction: таблица поправок (`mode`, `entry`, `frequency`, `dB`), import/export, применение к отображению
- [ ] `filebrowse.ui` — SD tinySA: список файлов (`sd_list`), превью bmp, `sd_read` выбранного, save
- [ ] `phasenoise.ui` — Phase Noise: dBc/Hz от offset (log X), кнопка centre tone, baseline-поправка
- [ ] `fading.ui` — Fading: уровень выбранных маркеров во времени (мультиграфик), `timePoints`
- [ ] `pattern.ui` — Antenna Pattern: полярный график (4 канала × 40 точек), measure, старт/стоп

## Движок/устройства

- [ ] tinySA (basic) и tinySA4 (Ultra/Ultra+); детект `0483:5740` по product string; до 4 приборов
- [ ] `scanraw` auto-repeat (FW ≥ 1.4.177) и legacy-режим; корректная синхронизация (`}{`), `abort`
- [ ] Лимиты/шкалы: scale 128/174; лимиты частот; RBW clip; `spur` (basic не имеет auto)
- [ ] `sampleRepeat` (`repeat`), sync time (`time b ...`), `pause/resume/reset`
- [ ] Zero span, split scan
- [ ] NanoVNA — экспериментально (низкий приоритет, после основного)

## Запись и файлы

- [ ] Запись свипов в `.npy` (совместимо с оригиналом): row0 = [serial, freqs…], col0 = timestamps; лимит ~10e6 полей; до 4 записей
- [ ] Плейбек: play/pause, слайдер позиции, speed, progress
- [ ] SD-браузер tinySA с превью bmp и сохранением
- [ ] Экспорт CSV: обычный, Shure WWB, Sennheiser WSM; экспорт изображений графика
- [ ] Настройки SQLite `QtTSAprefs.db` + импорт/экспорт preset/band/correction CSV

Таблицы БД настроек (шаблон v2.0.2): `boolean`, `combo`, `rbw`, `mapping`, `trace`, `freqtype`,
`SVGColour`, `settings`, `correction`, `recording`, `checkboxes`, `numbers`, `frequencies` (196 строк).

## Ассеты для переноса

- [ ] `correction/`, `frequency_lists/`, `Sample_preset_markers/` (CSV), примеры `sweep_files/`
- [ ] `modules/*_baseline.txt` (для Phase Noise)

## Прогресс

- **M1 (готово):** serial-слой (владение портом, фрейминг), сырой USB-лог (текст+bin), CLI, консоль в UI, сборки Linux/Windows.
- **M2 (готово, ждёт проверки на железе):** свип-движок `scanraw` (stream, парсер кадров, `abort`),
  спектр и водопад на canvas, Run/Stop, start/center/stop/span, points, RBW auto/manual,
  atten auto/manual, LNA, Spurs, счётчик fps.

## Отличия нашего порта (осознанные)

- UI: Svelte/TS (Wails) вместо Qt-виджетов — функциональная эквивалентность, не пиксель-в-пиксель.
- Многопоточность: UI-поток не делает I/O; воркеры свипа/записи/файлов на горутинах (это и есть цель проекта).
- `+` сырой USB-лог в debug-режиме (текст + бинарник для replay/фикстур).
- `+` локальный веб-интерфейс для телефона: спектр, водопад, пик/SNR, управление свипом и CSV.
- `+` прошивка tinySA из приложения (DFU, с дамп-бэкапом) — этап M8.
- `+` CI/CD: сборки Linux/Windows и релизы по тегам `v*`.
