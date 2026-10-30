# Референс QtTinySA

- **Репозиторий:** https://github.com/g4ixt/QtTinySA
- **Зафиксированный коммит:** `1cc609d6876184fe308de72f78846decf4752246` (2026-10-11, «v2.0.2»)
- **Локально:** `reference/QtTinySA` (в `.gitignore`, в репозиторий не коммитится)
- **Лицензия референса:** GPL-3.0-or-later → `tinysa_go` — производная работа, тоже GPL-3.0-or-later.

## Как получить референс заново

```sh
git clone --depth 1 https://github.com/g4ixt/QtTinySA reference/QtTinySA
git -C reference/QtTinySA log -1 --format='%H %ci %s'   # должен совпасть с пином выше
```

Обновлять пин — только осознанно; фиксируйте причину и новый коммит в этом файле.

## Карта референса (что где лежит)

| Путь | Что это |
|---|---|
| `src/QtTinySA.py` | Главный файл (~81 КБ): класс `Analyser` (роутер данных), `ModelView`, вся логика UI и меню |
| `src/modules/devices.py` | Serial/устройства: классы `USBdevice`, `Tiny` (basic/Ultra), `Nano`, `Recorder`; `scanraw`-цикл; эвристика таймаутов |
| `src/modules/graphs.py` | Графики: спектр, водопад, гистограмма, PhaseNoiseGraph, PolarGraph |
| `src/modules/exporters.py` | Экспорт CSV: обычный + Shure WWB + Sennheiser WSM |
| `src/modules/*.ui` | Qt Designer: `spectrum.ui` (главное окно), `settings`, `bands`, `offset`, `filebrowse`, `phasenoise`, `fading`, `pattern` |
| `src/modules/*_baseline.txt` | Базовые уровни (10/1152 точек) — используются, в частности, для Phase Noise |
| `src/QtTSAprefs.db` | SQLite настроек (шаблон; при первом запуске копируется в конфиг-каталог) |
| `correction/` | Таблицы поправок (пример `23041023/table.csv`) |
| `frequency_lists/` | Списки частот (маяки, WiFi, PMR) в CSV |
| `Sample_preset_markers/` | Примеры пресетов (broadcast, ТВ-каналы, event-маркеры) |
| `sweep_files/` | Примеры записей `.npy` (формат совместимости!) |
| `flow_diagram/`, `Screenshots/` | Диаграмма потока данных, скриншоты |
