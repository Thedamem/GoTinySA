'use strict';

/* tinySA local web UI. One page, two canvases, and a small JSON/SSE API from
   internal/webui. Sweep settings are owned by the desktop app, so every status
   snapshot carries the config that was last applied by either UI. */

const $ = (id) => document.getElementById(id);

const MONO = 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace';

const COLORS = {
  live: '#e8b64c',
  max: '#6aa9e0',
  avg: '#b18ad6',
  min: '#54b98a',
  grid: '#1c232c',
  axis: '#84909e',
  marker: '#e6e9ee',
};

const STORAGE_KEY = 'tinysa-web-config-v2';
const LEGACY_STORAGE_KEY = 'tinysa-web-config-v1';
const LANG_KEY = 'tinysa-web-lang';

const I18N = {
  en: {
    'app.title': 'tinySA — spectrum analyzer',
    'lang.label': 'Language',
    'state.disconnected': 'disconnected',
    'state.connecting': 'connecting…',
    'state.sweeping': 'sweeping',
    'state.connected': 'connected',
    'state.noLink': 'no link',
    'readout.aria': 'Last sweep result',
    'readout.peak': 'Peak',
    'readout.noise': 'Noise',
    'readout.range': 'Range',
    'readout.hold': 'Maximum',
    'plots.aria': 'Plots',
    'plot.spectrum': 'Spectrum',
    'plot.waterfall': 'Waterfall',
    'plot.noise': 'noise',
    'legend.aria': 'Visible traces',
    'legend.live': 'Live',
    'legend.max': 'Max',
    'legend.avg': 'Average',
    'legend.min': 'Min',
    'spectrum.aria': 'Spectrum: drag to place a marker',
    'tap.default': 'Drag on the plot to place a marker. Double-tap resets it.',
    'waterfall.note': 'newest rows on top',
    'waterfall.aria': 'Waterfall: newest rows on top',
    'controls.aria': 'Controls',
    'device.title': 'Device',
    'device.portAria': 'USB port',
    'device.refreshAria': 'Refresh port list',
    'device.connect': 'Connect',
    'device.disconnect': 'Disconnect',
    'device.cancel': 'Cancel',
    'device.hintDisconnected': 'Device not connected',
    'device.hintConnecting': 'Connecting…',
    'device.lnaTitle': 'LNA is available only on Ultra/Ultra+',
    'port.current': '{name} (current)',
    'port.none': 'No ports found',
    'range.title': 'Range',
    'range.preset': 'Preset',
    'range.start': 'Start, MHz',
    'range.stop': 'Stop, MHz',
    'range.peakCenter': 'Peak to center',
    'range.peakZoom': 'Zoom to peak',
    'range.limit': 'up to {freq}',
    'sweep.title': 'Sweep',
    'sweep.points': 'Points',
    'sweep.atten': 'Attenuator',
    'sweep.hint': 'Settings are applied to the device. If a sweep is already running, press Apply.',
    'unit.auto': 'auto',
    'unit.ghz': 'GHz',
    'unit.mhz': 'MHz',
    'unit.khz': 'kHz',
    'unit.hz': 'Hz',
    'unit.db': 'dB',
    'unit.khzShort': 'kHz',
    'unit.dbShort': 'dB',
    'data.title': 'Data',
    'data.csv': 'Download CSV',
    'data.hint': 'Frequency, live/max/avg/min levels and SNR of the last sweep.',
    'run.aria': 'Sweep control',
    'run.apply': 'Apply',
    'run.start': 'Start',
    'run.stop': 'Stop',
    'run.applyTitleNoConnection': 'No connection',
    'run.applyTitleStopped': 'Sweep is not running',
    'run.applyTitleClean': 'No changed settings',
    'run.applyTitleDirty': 'Restart the sweep with new settings',
    'footnote': 'Local network, no password. Do not expose this address to the internet. The server runs while the tinySA app is open.',
    'noscript': 'This web interface requires JavaScript.',
    'toast.portsError': 'Could not get ports: {error}',
    'toast.connectionCancelled': 'Connection cancelled',
    'toast.disconnected': 'Device disconnected',
    'toast.selectPort': 'Select a USB port first',
    'toast.connected': 'Connected: {port}',
    'toast.configRemote': 'Sweep settings updated from the desktop',
    'toast.noSweepData': 'No sweep data yet',
    'toast.noEventSource': 'This browser does not support EventSource',
    'toast.serverError': 'Server: {error}',
    'error.startStopRequired': 'Enter the start and stop frequencies',
    'error.startMin': 'Start must be at least 0.1 MHz',
    'error.stopGreater': 'Stop must be greater than start',
    'error.modelMax': 'This model supports up to {freq}',
    'error.noSweepData': 'No sweep data',
    'error.connectFirst': 'Connect the device first',
    'empty.noData': 'No data.',
    'empty.connectStart': 'Connect the device and press Start.',
    'band.none': 'Range preset…',
    'band.fm': 'FM 88–108',
    'band.aviation': 'Aviation 108–137',
    'band.twoMeter': '2 m 144–146',
    'band.dab': 'DAB 174–240',
    'band.seventyCm': '70 cm 430–440',
    'band.ism433': '433 ISM',
    'band.pmr446': 'PMR 446',
    'band.lora868': 'LoRa 868',
    'band.wifi24': 'Wi-Fi 2.4',
    'band.wifi5': 'Wi-Fi 5',
    'band.full': 'Full range',
  },
  ru: {
    'app.title': 'tinySA — анализатор спектра',
    'lang.label': 'Язык',
    'state.disconnected': 'не подключён',
    'state.connecting': 'подключение…',
    'state.sweeping': 'свип идёт',
    'state.connected': 'подключён',
    'state.noLink': 'нет связи',
    'readout.aria': 'Результат последнего свипа',
    'readout.peak': 'Пик',
    'readout.noise': 'Шум',
    'readout.range': 'Диапазон',
    'readout.hold': 'Максимум',
    'plots.aria': 'Графики',
    'plot.spectrum': 'Спектр',
    'plot.waterfall': 'Водопад',
    'plot.noise': 'шум',
    'legend.aria': 'Показываемые трассы',
    'legend.live': 'Текущий',
    'legend.max': 'Max',
    'legend.avg': 'Среднее',
    'legend.min': 'Минимум',
    'spectrum.aria': 'Спектр: перетащите, чтобы поставить маркер',
    'tap.default': 'Перетащите по графику, чтобы поставить маркер. Двойной тап сбрасывает.',
    'waterfall.note': 'новые строки сверху',
    'waterfall.aria': 'Водопад: новые строки сверху',
    'controls.aria': 'Управление',
    'device.title': 'Прибор',
    'device.portAria': 'USB-порт',
    'device.refreshAria': 'Обновить список портов',
    'device.connect': 'Подключить',
    'device.disconnect': 'Отключить',
    'device.cancel': 'Отмена',
    'device.hintDisconnected': 'Прибор не подключён',
    'device.hintConnecting': 'Идёт подключение…',
    'device.lnaTitle': 'LNA доступен только Ultra/Ultra+',
    'port.current': '{name} (текущий)',
    'port.none': 'Порты не найдены',
    'range.title': 'Диапазон',
    'range.preset': 'Пресет',
    'range.start': 'От, МГц',
    'range.stop': 'До, МГц',
    'range.peakCenter': 'Пик в центр',
    'range.peakZoom': 'Зум на пик',
    'range.limit': 'до {freq}',
    'sweep.title': 'Свип',
    'sweep.points': 'Точек',
    'sweep.atten': 'Аттенюатор',
    'sweep.hint': 'Настройки применяются к прибору. Если свип уже идёт, нажмите «Применить».',
    'unit.auto': 'авто',
    'unit.ghz': 'ГГц',
    'unit.mhz': 'МГц',
    'unit.khz': 'кГц',
    'unit.hz': 'Гц',
    'unit.db': 'дБ',
    'unit.khzShort': 'кГц',
    'unit.dbShort': 'дБ',
    'data.title': 'Данные',
    'data.csv': 'Скачать CSV',
    'data.hint': 'Частота, уровни live/max/avg/min и SNR последнего свипа.',
    'run.aria': 'Запуск свипа',
    'run.apply': 'Применить',
    'run.start': 'Старт',
    'run.stop': 'Стоп',
    'run.applyTitleNoConnection': 'Нет подключения',
    'run.applyTitleStopped': 'Свип не запущен',
    'run.applyTitleClean': 'Нет изменённых настроек',
    'run.applyTitleDirty': 'Перезапустить свип с новыми настройками',
    'footnote': 'Локальная сеть, без пароля. Не открывайте этот адрес из интернета. Сервер работает, пока открыто приложение tinySA.',
    'noscript': 'Для работы веб-интерфейса нужен JavaScript.',
    'toast.portsError': 'Не удалось получить порты: {error}',
    'toast.connectionCancelled': 'Подключение отменено',
    'toast.disconnected': 'Прибор отключён',
    'toast.selectPort': 'Сначала выберите USB-порт',
    'toast.connected': 'Подключено: {port}',
    'toast.configRemote': 'Настройки свипа обновлены с компьютера',
    'toast.noSweepData': 'Пока нет данных свипа',
    'toast.noEventSource': 'Браузер не поддерживает EventSource',
    'toast.serverError': 'Сервер: {error}',
    'error.startStopRequired': 'Введите начало и конец диапазона',
    'error.startMin': 'Начало диапазона должно быть не ниже 0.1 МГц',
    'error.stopGreater': 'Конец должен быть больше начала',
    'error.modelMax': 'Для этого прибора максимум {freq}',
    'error.noSweepData': 'Нет данных свипа',
    'error.connectFirst': 'Сначала подключите прибор',
    'empty.noData': 'Нет данных.',
    'empty.connectStart': 'Подключите прибор и нажмите «Старт».',
    'band.none': 'Пресет диапазона…',
    'band.fm': 'FM 88–108',
    'band.aviation': 'Авиация 108–137',
    'band.twoMeter': '2 м 144–146',
    'band.dab': 'DAB 174–240',
    'band.seventyCm': '70 см 430–440',
    'band.ism433': '433 ISM',
    'band.pmr446': 'PMR 446',
    'band.lora868': 'LoRa 868',
    'band.wifi24': 'Wi-Fi 2.4',
    'band.wifi5': 'Wi-Fi 5',
    'band.full': 'Весь диапазон',
  },
};

let lang = 'en';

function t(key, vars) {
  let text = (I18N[lang] && I18N[lang][key]) || I18N.en[key] || key;
  if (vars) {
    for (const [name, value] of Object.entries(vars)) {
      text = text.split('{' + name + '}').join(String(value));
    }
  }
  return text;
}

function loadLang() {
  try {
    const saved = localStorage.getItem(LANG_KEY);
    if (saved === 'en' || saved === 'ru') return saved;
  } catch (_) {}
  return 'en';
}

function saveLang(next) {
  try {
    localStorage.setItem(LANG_KEY, next);
  } catch (_) {}
}

// [id, start MHz, stop MHz]; stop = 'full' means the model's maximum.
const BANDS = [
  ['none', null, null],
  ['fm', 88, 108],
  ['aviation', 108, 137],
  ['twoMeter', 144, 146],
  ['dab', 174, 240],
  ['seventyCm', 430, 440],
  ['ism433', 433.05, 434.79],
  ['pmr446', 446.0, 446.2],
  ['lora868', 863, 870],
  ['wifi24', 2400, 2483.5],
  ['wifi5', 5150, 5850],
  ['full', 0.1, 'full'],
];

const state = {
  status: null,
  frame: null,
  ports: [],
  lastSeq: 0,
  lastFrameAt: 0,
  fps: 0,
  frameStart: 0,
  frameStop: 0,
  lastReadout: 0,
  marker: null,
  yLo: -120,
  yHi: -20,
  wfLo: -120,
  wfHi: -20,
  wfStart: 0,
  wfStop: 0,
  wfRowPx: 1,
  draggingMarker: false,
  lastTapAt: 0,
  lastTapX: 0,
  lastTapY: 0,
  toastTimer: 0,
  cfgKey: '',
  pendingKey: '',
  dirty: false,
  initialized: false,
};

// ---------------------------------------------------------------- utilities

function num(id) {
  const raw = String($(id).value).replace(',', '.').trim();
  const v = parseFloat(raw);
  return Number.isFinite(v) ? v : NaN;
}

function trimZeros(s) {
  return s.replace(/\.0+$/, '').replace(/(\.\d*?)0+$/, '$1');
}

function fmtFreq(hz) {
  if (!Number.isFinite(hz)) return '—';
  if (hz >= 1e9) return (hz / 1e9).toFixed(4) + ' ' + t('unit.ghz');
  if (hz >= 1e6) return (hz / 1e6).toFixed(3) + ' ' + t('unit.mhz');
  if (hz >= 1e3) return (hz / 1e3).toFixed(1) + ' ' + t('unit.khz');
  return hz.toFixed(0) + ' ' + t('unit.hz');
}

function fmtShort(hz) {
  if (!Number.isFinite(hz)) return '—';
  if (hz >= 1e9) return trimZeros((hz / 1e9).toFixed(3)) + ' ' + t('unit.ghz');
  if (hz >= 1e6) {
    const mhz = hz / 1e6;
    const digits = mhz >= 1000 ? 0 : mhz >= 100 ? 1 : 3;
    return trimZeros(mhz.toFixed(digits)) + ' ' + t('unit.mhz');
  }
  if (hz >= 1e3) return trimZeros((hz / 1e3).toFixed(1)) + ' ' + t('unit.khz');
  return hz.toFixed(0) + ' ' + t('unit.hz');
}

function fmtRange(startHz, stopHz) {
  if (!Number.isFinite(startHz) || !Number.isFinite(stopHz) || stopHz <= startHz) return '—';
  if (startHz >= 1e6 && stopHz < 1e9) {
    return trimZeros((startHz / 1e6).toFixed(3)) + ' – ' + trimZeros((stopHz / 1e6).toFixed(3)) + ' ' + t('unit.mhz');
  }
  if (startHz >= 1e9) {
    return trimZeros((startHz / 1e9).toFixed(3)) + ' – ' + trimZeros((stopHz / 1e9).toFixed(3)) + ' ' + t('unit.ghz');
  }
  return fmtShort(startHz) + ' – ' + fmtShort(stopHz);
}

function fmtLevel(v) {
  return Number.isFinite(v) ? v.toFixed(1) : '—';
}

function setText(id, text) {
  const el = $(id);
  if (el && el.textContent !== text) el.textContent = text;
}

function toast(message, isError) {
  const el = $('toast');
  if (!el) return;
  el.textContent = message;
  el.classList.toggle('error', !!isError);
  el.classList.add('show');
  clearTimeout(state.toastTimer);
  state.toastTimer = setTimeout(() => el.classList.remove('show'), isError ? 7000 : 3500);
}

async function api(path, options) {
  const opts = Object.assign({ cache: 'no-store' }, options || {});
  if (opts.body) {
    opts.headers = Object.assign({ 'Content-Type': 'application/json' }, opts.headers || {});
  }
  const res = await fetch(path, opts);
  const text = await res.text();
  let data = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch (_) {
      data = null;
    }
  }
  if (!res.ok) {
    const message = data && data.error ? data.error : res.status + ' ' + res.statusText;
    throw new Error(message);
  }
  return data;
}

// ---------------------------------------------------------------- status/UI

function setLinkState(text, cls) {
  const el = $('linkState');
  if (!el) return;
  el.className = 'state ' + cls;
  el.textContent = text;
}

function applyStatus(st) {
  if (!st) return;
  const prev = state.status;
  const started = !!st.sweeping && !(prev && prev.sweeping);
  state.status = st;

  if (st.connecting) {
    setLinkState(t('state.connecting'), 'wait');
  } else if (st.connected) {
    setLinkState(st.sweeping ? t('state.sweeping') : t('state.connected'), st.sweeping ? 'live' : 'ok');
  } else {
    setLinkState(t('state.disconnected'), 'idle');
  }

  const ultra = !!(st.model && st.model.ultra);
  setText('modelLabel', st.connected ? (ultra ? 'Ultra' : 'tinySA') : '');
  setText('batteryLabel', st.battery || '');
  setText('limitHint', st.model && st.model.maxFreq ? t('range.limit', { freq: fmtShort(st.model.maxFreq) }) : '');

  const firmware = st.model && st.model.firmware ? st.model.firmware : '';
  setText('deviceState', st.connected ? (firmware.length > 28 ? firmware.slice(0, 28) + '…' : firmware) : '');
  $('deviceState').title = firmware;

  const connect = $('btnConnect');
  connect.textContent = st.connecting ? t('device.cancel') : st.connected ? t('device.disconnect') : t('device.connect');
  connect.classList.toggle('primary', !st.connected && !st.connecting);
  connect.classList.toggle('stop', !!st.connected && !st.connecting);

  $('portSelect').disabled = !!st.connected || !!st.connecting;
  $('btnPorts').disabled = !!st.connected || !!st.connecting;

  if (st.connected) {
    setText('deviceHint', st.port || t('state.connected'));
    $('deviceHint').title = st.port || '';
  } else if (st.connecting) {
    setText('deviceHint', t('device.hintConnecting'));
    $('deviceHint').title = '';
  } else {
    setText('deviceHint', t('device.hintDisconnected'));
    $('deviceHint').title = '';
  }

  const lna = $('lna');
  lna.disabled = !ultra;
  lna.title = ultra ? '' : t('device.lnaTitle');
  if (!ultra) lna.checked = false;

  if (st.connected && st.port) ensurePortOption(st.port);
  applyServerConfig(st.config, started);
  updateBandSelect();
  updateButtons();
}

function ensurePortOption(name) {
  const sel = $('portSelect');
  if (!sel) return;
  for (const option of sel.options) {
    if (option.value === name) {
      sel.value = name;
      return;
    }
  }
  const option = document.createElement('option');
  option.value = name;
  option.textContent = t('port.current', { name });
  sel.appendChild(option);
  sel.value = name;
}

function updateButtons() {
  const st = state.status || {};
  const usable = !!st.connected && !st.connecting;
  const running = !!st.sweeping;

  const run = $('btnRun');
  run.textContent = running ? t('run.stop') : t('run.start');
  run.classList.toggle('stop', running);
  run.disabled = !usable;

  const apply = $('btnApply');
  apply.disabled = !usable || !running || !state.dirty;
  apply.classList.toggle('attention', usable && running && state.dirty);
  apply.title = !usable
    ? t('run.applyTitleNoConnection')
    : !running
      ? t('run.applyTitleStopped')
      : !state.dirty
        ? t('run.applyTitleClean')
        : t('run.applyTitleDirty');

  const hasFrame = !!state.frame;
  $('btnPeakCenter').disabled = !usable || !hasFrame;
  $('btnPeakZoom').disabled = !usable || !hasFrame;

  updateRunRange();
}

function updateRunRange() {
  const startMHz = num('startMHz');
  const stopMHz = num('stopMHz');
  setText('runRange', Number.isFinite(startMHz) && Number.isFinite(stopMHz) && stopMHz > startMHz
    ? fmtRange(startMHz * 1e6, stopMHz * 1e6)
    : '—');
}

async function refreshStatus() {
  try {
    applyStatus(await api('/api/status'));
  } catch (_) {
    setLinkState(t('state.noLink'), 'bad');
  }
}

async function refreshPorts() {
  try {
    const data = await api('/api/ports');
    state.ports = (data && data.ports) || [];
    renderPorts();
  } catch (e) {
    toast(t('toast.portsError', { error: e.message }), true);
  }
}

function renderPorts() {
  const sel = $('portSelect');
  const current = sel.value || (state.status && state.status.port) || '';

  const frag = document.createDocumentFragment();
  if (!state.ports.length) {
    const none = document.createElement('option');
    none.value = '';
    none.textContent = t('port.none');
    none.disabled = true;
    frag.appendChild(none);
  } else {
    for (const port of state.ports) {
      const option = document.createElement('option');
      option.value = port.name;
      option.textContent = port.name + (port.isTinySA ? ' — tinySA' : '');
      frag.appendChild(option);
    }
  }
  sel.replaceChildren(frag);

  let pick = current;
  if (!pick || !state.ports.some((p) => p.name === pick)) {
    const tiny = state.ports.find((p) => p.isTinySA);
    pick = (tiny || state.ports[0] || {}).name || '';
  }
  if (pick) sel.value = pick;
  if (current && !state.ports.some((p) => p.name === current)) ensurePortOption(current);
}

async function toggleConnect() {
  const st = state.status || {};
  try {
    if (st.connecting) {
      await api('/api/connect/cancel', { method: 'POST' });
      toast(t('toast.connectionCancelled'));
      return;
    }
    if (st.connected) {
      await api('/api/disconnect', { method: 'POST' });
      toast(t('toast.disconnected'));
      state.frame = null;
      state.lastSeq = 0;
      state.cfgKey = '';
      state.pendingKey = '';
      state.dirty = false;
      state.marker = null;
      updateCSVLink();
      drawSpectrum();
      await refreshStatus();
      return;
    }

    const port = $('portSelect').value;
    if (!port) {
      toast(t('toast.selectPort'), true);
      return;
    }
    const btn = $('btnConnect');
    btn.textContent = t('state.connecting');
    btn.disabled = true;
    await api('/api/connect', { method: 'POST', body: JSON.stringify({ port }) });
    toast(t('toast.connected', { port }));
  } catch (e) {
    toast(e.message, true);
  } finally {
    $('btnConnect').disabled = false;
    await refreshStatus();
  }
}

// ---------------------------------------------------------------- config

function configKey(cfg) {
  return [cfg.start, cfg.stop, cfg.points, cfg.rbw, cfg.atten, cfg.lna ? 1 : 0, cfg.spur ? 1 : 0].join('|');
}

function configFromForm() {
  const startMHz = num('startMHz');
  const stopMHz = num('stopMHz');
  if (!Number.isFinite(startMHz) || !Number.isFinite(stopMHz)) {
    throw new Error(t('error.startStopRequired'));
  }
  if (startMHz < 0.1) throw new Error(t('error.startMin'));
  if (stopMHz <= startMHz) throw new Error(t('error.stopGreater'));

  const maxHz = (state.status && state.status.model && state.status.model.maxFreq) || 960e6;
  if (stopMHz * 1e6 > maxHz + 1) {
    throw new Error(t('error.modelMax', { freq: fmtShort(maxHz) }));
  }

  const rbw = $('rbw').value;
  const atten = $('atten').value;
  return {
    start: Math.round(startMHz * 1e6),
    stop: Math.round(stopMHz * 1e6),
    points: parseInt($('points').value, 10) || 400,
    rbw: rbw === 'auto' ? 0 : parseFloat(rbw),
    atten: atten === 'auto' ? -1 : parseInt(atten, 10),
    lna: $('lna').checked,
    spur: $('spur').checked,
  };
}

function setSelectValue(id, value, label) {
  if (value == null) return;
  const el = $(id);
  const text = String(value);
  for (const option of el.options) {
    if (option.value === text) {
      el.value = text;
      return;
    }
  }
  const option = document.createElement('option');
  option.value = text;
  option.textContent = label;
  if (text !== 'auto' && id === 'rbw') option.dataset.unit = 'khz';
  if (text !== 'auto' && id === 'atten') option.dataset.unit = 'db';
  el.appendChild(option);
  el.value = text;
}

function setFormConfig(cfg) {
  $('startMHz').value = String(+(cfg.start / 1e6).toFixed(6));
  $('stopMHz').value = String(+(cfg.stop / 1e6).toFixed(6));
  setSelectValue('points', cfg.points, String(cfg.points));
  setSelectValue('rbw', cfg.rbw <= 0 ? 'auto' : cfg.rbw, cfg.rbw <= 0 ? t('unit.auto') : cfg.rbw + ' ' + t('unit.khz'));
  setSelectValue('atten', cfg.atten < 0 ? 'auto' : cfg.atten, cfg.atten < 0 ? t('unit.auto') : cfg.atten + ' ' + t('unit.db'));
  $('lna').checked = !!(cfg.lna && state.status && state.status.model && state.status.model.ultra);
  $('spur').checked = !!cfg.spur;

  state.cfgKey = configKey(cfg);
  state.dirty = false;
  updateBandSelect();
  updateButtons();
  saveConfig();
}

function applyServerConfig(cfg, force) {
  if (!cfg) return;
  const key = configKey(cfg);
  if (!force && key === state.cfgKey) return;

  const remote = state.initialized && state.cfgKey !== '' && key !== state.cfgKey && key !== state.pendingKey;
  const rangeChanged = state.frameStart > 0 && (cfg.start !== state.frameStart || cfg.stop !== state.frameStop);

  setFormConfig(cfg);
  if (key === state.pendingKey) state.pendingKey = '';
  if (rangeChanged) {
    state.marker = null;
    resetTapInfo();
  }
  if (remote && !document.hidden) {
    toast(t('toast.configRemote'));
  }
}

// ---------------------------------------------------------------- sweep

async function applySweep() {
  const cfg = configFromForm();
  const key = configKey(cfg);
  state.pendingKey = key;
  try {
    await api('/api/sweep', { method: 'POST', body: JSON.stringify(cfg) });
  } catch (e) {
    state.pendingKey = '';
    throw e;
  }
  state.cfgKey = key;
  state.pendingKey = '';
  state.dirty = false;
  state.status = Object.assign({}, state.status, { sweeping: true });
  state.frameStart = cfg.start;
  state.frameStop = cfg.stop;
  saveConfig();
  updateButtons();
  return cfg;
}

async function stopSweep() {
  await api('/api/sweep/stop', { method: 'POST' });
  state.status = Object.assign({}, state.status, { sweeping: false });
  updateButtons();
}

async function toggleRun() {
  try {
    if (state.status && state.status.sweeping) await stopSweep();
    else await applySweep();
  } catch (e) {
    toast(e.message, true);
  }
}

async function applyCurrent() {
  try {
    await applySweep();
  } catch (e) {
    toast(e.message, true);
  }
}

async function peakAction(zoom) {
  try {
    const f = state.frame;
    if (!f) throw new Error(t('error.noSweepData'));
    if (!state.status || !state.status.connected) throw new Error(t('error.connectFirst'));

    const maxHz = (state.status.model && state.status.model.maxFreq) || 960e6;
    const oldSpan = Math.max(1e3, f.stop - f.start);
    const span = zoom ? Math.max(50e3, oldSpan / 6) : oldSpan;
    const minHz = 0.1e6;

    let startHz = f.peakFreq - span / 2;
    if (startHz < minHz) startHz = minHz;
    let stopHz = startHz + span;
    if (stopHz > maxHz) {
      stopHz = maxHz;
      startHz = Math.max(minHz, stopHz - span);
    }

    $('startMHz').value = String(+(startHz / 1e6).toFixed(6));
    $('stopMHz').value = String(+(stopHz / 1e6).toFixed(6));
    state.dirty = true;
    state.marker = null;
    resetTapInfo();
    updateBandSelect();
    updateRunRange();

    if (state.status.sweeping) {
      await applySweep();
    } else {
      saveConfig();
      updateButtons();
    }
  } catch (e) {
    toast(e.message, true);
  }
}

// ---------------------------------------------------------------- frames

function onFrame(f) {
  if (!f || !Array.isArray(f.dbm) || f.dbm.length < 2) return;

  const now = performance.now();
  const isNew = f.seq !== state.lastSeq;
  if (isNew) {
    if (state.lastFrameAt > 0) {
      const dt = (now - state.lastFrameAt) / 1000;
      if (dt > 0) state.fps = state.fps ? state.fps * 0.8 + (1 / dt) * 0.2 : 1 / dt;
    }
    state.lastSeq = f.seq;
    state.lastFrameAt = now;
  }

  state.frame = f;
  if (Number.isFinite(f.start) && Number.isFinite(f.stop) && f.stop > f.start) {
    if (state.frameStart !== f.start || state.frameStop !== f.stop) {
      state.frameStart = f.start;
      state.frameStop = f.stop;
      syncRangeFromFrame(f);
      if (state.marker && (state.marker.hz < f.start || state.marker.hz > f.stop)) {
        state.marker = null;
        resetTapInfo();
      }
    }
  }
  updateCSVLink();

  if (state.lastReadout === 0 || now - state.lastReadout > 250) {
    updateReadout();
    state.lastReadout = now;
  }

  drawSpectrum();
  if (isNew) pushWaterfall(f);
}

function syncRangeFromFrame(f) {
  // Once a sweep config is known it is the source of truth. Without it (for
  // example a very old server), follow the first frame so the fields are not
  // stuck on a stale range.
  if (state.cfgKey) return;
  if (!Number.isFinite(f.start) || !Number.isFinite(f.stop)) return;

  if (document.activeElement !== $('startMHz')) {
    $('startMHz').value = String(+(f.start / 1e6).toFixed(6));
  }
  if (document.activeElement !== $('stopMHz')) {
    $('stopMHz').value = String(+(f.stop / 1e6).toFixed(6));
  }
  updateRunRange();
  updateBandSelect();
}

function updateReadout() {
  const f = state.frame;
  if (!f) return;

  setText('peakFreq', fmtFreq(f.peakFreq));
  setText('peakLevel', fmtLevel(f.peak) + ' dBm');
  setText('snr', Number.isFinite(f.snr) ? f.snr.toFixed(1) + ' dB' : '—');
  setText('noise', fmtLevel(f.noise) + ' dBm');
  setText('range', fmtRange(f.start, f.stop));

  let holdText = '—';
  if (Array.isArray(f.max) && f.max.length > 1) {
    let best = 0;
    for (let i = 1; i < f.max.length; i++) {
      if (f.max[i] > f.max[best]) best = i;
    }
    const hz = f.start + ((f.stop - f.start) * best) / (f.max.length - 1);
    if (Number.isFinite(f.max[best]) && f.max[best] > -190) {
      holdText = fmtLevel(f.max[best]) + ' dBm\n' + fmtShort(hz);
    }
  }
  setText('hold', holdText);
}

function updateCSVLink() {
  const link = $('csvLink');
  if (!link) return;
  if (state.frame) {
    link.href = '/api/csv';
    link.classList.remove('disabled');
    link.removeAttribute('aria-disabled');
  } else {
    link.href = '#';
    link.classList.add('disabled');
    link.setAttribute('aria-disabled', 'true');
  }
}

// ---------------------------------------------------------------- spectrum

function canvasMetrics(canvas) {
  const rect = canvas.getBoundingClientRect();
  const w = Math.max(1, Math.round(rect.width));
  const h = Math.max(1, Math.round(rect.height));
  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  return { w, h, dpr };
}

function plotMargins(width) {
  return width < 520
    ? { L: 40, R: 10, T: 10, B: 24 }
    : { L: 50, R: 12, T: 10, B: 24 };
}

function plotRange(f) {
  if (f && Number.isFinite(f.start) && Number.isFinite(f.stop) && f.stop > f.start) {
    return [f.start, f.stop];
  }
  const start = num('startMHz') * 1e6;
  const stop = num('stopMHz') * 1e6;
  const safeStart = Number.isFinite(start) ? start : 88e6;
  const safeStop = Number.isFinite(stop) && stop > safeStart ? stop : safeStart + 20e6;
  return [safeStart, safeStop];
}

function drawTrace(ctx, data, color, width, pw, ph, yFor, L) {
  if (!Array.isArray(data) || data.length < 2) return;
  ctx.beginPath();
  for (let i = 0; i < data.length; i++) {
    const x = L + (pw * i) / (data.length - 1);
    const y = yFor(Math.max(state.yLo, Math.min(state.yHi, data[i])));
    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  }
  ctx.strokeStyle = color;
  ctx.lineWidth = width;
  ctx.stroke();
}

function drawSpectrum() {
  const canvas = $('spectrum');
  if (!canvas) return;
  const { w, h, dpr } = canvasMetrics(canvas);
  const pixelW = Math.round(w * dpr);
  const pixelH = Math.round(h * dpr);
  if (canvas.width !== pixelW || canvas.height !== pixelH) {
    canvas.width = pixelW;
    canvas.height = pixelH;
  }

  const ctx = canvas.getContext('2d');
  if (!ctx) return;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.fillStyle = '#080a0d';
  ctx.fillRect(0, 0, w, h);

  const { L, R, T, B } = plotMargins(w);
  const pw = Math.max(10, w - L - R);
  const ph = Math.max(10, h - T - B);
  const f = state.frame;

  let lo = state.yLo;
  let hi = state.yHi;
  if (f) {
    const noise = Number.isFinite(f.noise) ? f.noise : -100;
    const peak = Number.isFinite(f.peak) ? f.peak : -60;
    const targetLo = Math.min(-95, Math.max(-165, noise - 12));
    let targetHi = Math.max(peak + 8, noise + 35, targetLo + 40);
    targetHi = Math.min(20, targetHi);
    lo = state.yLo + (targetLo - state.yLo) * 0.12;
    hi = state.yHi + (targetHi - state.yHi) * 0.12;
    if (hi - lo < 30) hi = lo + 30;
  } else {
    lo = -120;
    hi = -20;
  }
  state.yLo = lo;
  state.yHi = hi;

  const yFor = (v) => T + (1 - (v - lo) / (hi - lo)) * ph;

  ctx.font = '11px ' + MONO;
  ctx.textBaseline = 'middle';

  const span = hi - lo;
  const step = span <= 40 ? 10 : span <= 90 ? 20 : 30;
  for (let db = Math.ceil(lo / step) * step; db <= hi + 0.001; db += step) {
    const y = yFor(db);
    ctx.strokeStyle = COLORS.grid;
    ctx.beginPath();
    ctx.moveTo(L, y);
    ctx.lineTo(L + pw, y);
    ctx.stroke();
    ctx.fillStyle = COLORS.axis;
    ctx.textAlign = 'right';
    ctx.fillText(db.toFixed(0), L - 6, y);
  }

  const [rangeStart, rangeStop] = plotRange(f);
  const ticks = w < 520 ? 3 : 5;
  for (let i = 0; i <= ticks; i++) {
    const x = L + (pw * i) / ticks;
    ctx.strokeStyle = COLORS.grid;
    ctx.beginPath();
    ctx.moveTo(x, T);
    ctx.lineTo(x, T + ph);
    ctx.stroke();
    const hz = rangeStart + ((rangeStop - rangeStart) * i) / ticks;
    ctx.fillStyle = COLORS.axis;
    ctx.textAlign = i === 0 ? 'left' : i === ticks ? 'right' : 'center';
    ctx.fillText(fmtShort(hz), x, T + ph + 13);
  }

  if (!f || !Array.isArray(f.dbm) || f.dbm.length < 2) {
    ctx.fillStyle = COLORS.axis;
    ctx.font = '13px system-ui, -apple-system, "Segoe UI", Roboto, sans-serif';
    ctx.textAlign = 'center';
    const lines = w < 520
      ? [t('empty.noData'), t('empty.connectStart')]
      : [t('empty.noData') + ' ' + t('empty.connectStart')];
    lines.forEach((line, i) => {
      ctx.fillText(line, w / 2, T + ph / 2 - 8 + i * 20);
    });
    return;
  }

  ctx.save();
  ctx.beginPath();
  ctx.rect(L, T, pw, ph);
  ctx.clip();

  if ($('showMin').checked) drawTrace(ctx, f.min, COLORS.min, 1, pw, ph, yFor, L);
  if ($('showAvg').checked) drawTrace(ctx, f.avg, COLORS.avg, 1, pw, ph, yFor, L);
  if ($('showMax').checked) drawTrace(ctx, f.max, COLORS.max, 1, pw, ph, yFor, L);
  drawTrace(ctx, f.dbm, COLORS.live, 1.5, pw, ph, yFor, L);

  if (Number.isFinite(f.noise) && f.noise >= lo && f.noise <= hi) {
    const y = yFor(f.noise);
    ctx.strokeStyle = 'rgba(132, 144, 158, 0.5)';
    ctx.setLineDash([3, 4]);
    ctx.beginPath();
    ctx.moveTo(L, y);
    ctx.lineTo(L + pw, y);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.fillStyle = 'rgba(180, 190, 203, 0.85)';
    ctx.font = '10px ' + MONO;
    ctx.textAlign = 'right';
    ctx.fillText(t('plot.noise'), L + pw - 4, y - 6);
  }

  if (Number.isFinite(f.peakFreq) && f.peakFreq >= rangeStart && f.peakFreq <= rangeStop) {
    const x = L + ((f.peakFreq - rangeStart) / (rangeStop - rangeStart)) * pw;
    const y = yFor(Math.max(lo, Math.min(hi, f.peak)));
    ctx.strokeStyle = 'rgba(232, 182, 76, 0.5)';
    ctx.setLineDash([4, 4]);
    ctx.beginPath();
    ctx.moveTo(x, T);
    ctx.lineTo(x, T + ph);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.fillStyle = COLORS.live;
    ctx.beginPath();
    ctx.arc(x, y, 3, 0, Math.PI * 2);
    ctx.fill();
  }

  if (state.marker && state.marker.hz >= rangeStart && state.marker.hz <= rangeStop) {
    const x = L + ((state.marker.hz - rangeStart) / (rangeStop - rangeStart)) * pw;
    const y = yFor(Math.max(lo, Math.min(hi, state.marker.level)));
    ctx.strokeStyle = 'rgba(230, 233, 238, 0.55)';
    ctx.setLineDash([2, 3]);
    ctx.beginPath();
    ctx.moveTo(x, T);
    ctx.lineTo(x, T + ph);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.fillStyle = COLORS.marker;
    ctx.beginPath();
    ctx.arc(x, y, 3, 0, Math.PI * 2);
    ctx.fill();

    const label = fmtFreq(state.marker.hz) + '  ' + fmtLevel(state.marker.level) + ' dBm';
    ctx.font = '11px ' + MONO;
    const tw = ctx.measureText(label).width + 12;
    let bx = x + 8;
    if (bx + tw > L + pw) bx = x - tw - 8;
    bx = Math.max(L, bx);
    const by = T + 6;
    ctx.fillStyle = 'rgba(16, 21, 27, 0.95)';
    ctx.fillRect(bx, by, tw, 20);
    ctx.strokeStyle = '#46515f';
    ctx.strokeRect(bx + 0.5, by + 0.5, tw - 1, 19);
    ctx.fillStyle = COLORS.marker;
    ctx.textAlign = 'left';
    ctx.fillText(label, bx + 6, by + 10);
  }

  ctx.restore();
}

// ---------------------------------------------------------------- waterfall

function colorFor(value, lo, hi) {
  const t = Math.max(0, Math.min(1, (value - lo) / (hi - lo)));
  const stops = [
    [0.0, 6, 9, 16],
    [0.22, 24, 42, 82],
    [0.46, 24, 105, 132],
    [0.7, 60, 168, 112],
    [0.88, 206, 188, 70],
    [1.0, 232, 128, 62],
  ];
  for (let i = 1; i < stops.length; i++) {
    if (t <= stops[i][0]) {
      const [t0, r0, g0, b0] = stops[i - 1];
      const [t1, r1, g1, b1] = stops[i];
      const k = (t - t0) / (t1 - t0);
      return [
        Math.round(r0 + (r1 - r0) * k),
        Math.round(g0 + (g1 - g0) * k),
        Math.round(b0 + (b1 - b0) * k),
      ];
    }
  }
  return [232, 128, 62];
}

function resizeWaterfall() {
  const canvas = $('waterfall');
  if (!canvas) return;
  const rect = canvas.getBoundingClientRect();
  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  const w = Math.max(1, Math.round(rect.width * dpr));
  const h = Math.max(1, Math.round(rect.height * dpr));
  if (canvas.width === w && canvas.height === h) return;
  canvas.width = w;
  canvas.height = h;
  state.wfRowPx = Math.max(1, Math.round(dpr));
  const ctx = canvas.getContext('2d');
  if (ctx) {
    ctx.fillStyle = '#080a0d';
    ctx.fillRect(0, 0, w, h);
  }
  state.wfLo = -120;
  state.wfHi = -20;
  state.wfStart = 0;
  state.wfStop = 0;
}

function pushWaterfall(f) {
  const canvas = $('waterfall');
  if (!canvas || !f || !Array.isArray(f.dbm) || f.dbm.length < 2) return;
  resizeWaterfall();

  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  const rangeChanged = f.start !== state.wfStart || f.stop !== state.wfStop;
  if (rangeChanged) {
    state.wfStart = f.start;
    state.wfStop = f.stop;
    ctx.fillStyle = '#080a0d';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
  }

  let mn = Infinity;
  let mx = -Infinity;
  for (const v of f.dbm) {
    if (v < mn) mn = v;
    if (v > mx) mx = v;
  }
  if (Number.isFinite(mn) && Number.isFinite(mx)) {
    const targetLo = Math.max(-190, mn - 3);
    const targetHi = Math.min(20, mx + 6);
    state.wfLo = state.wfLo * 0.88 + targetLo * 0.12;
    state.wfHi = state.wfHi * 0.88 + targetHi * 0.12;
    if (state.wfHi < state.wfLo + 12) state.wfHi = state.wfLo + 12;
  }

  const rowPx = state.wfRowPx;
  ctx.drawImage(canvas, 0, rowPx);

  const w = canvas.width;
  const img = ctx.createImageData(w, rowPx);
  const n = f.dbm.length;
  const spanHz = f.stop - f.start;
  for (let x = 0; x < w; x++) {
    let idx = 0;
    if (spanHz > 0) {
      const hz = f.start + (spanHz * x) / Math.max(1, w - 1);
      idx = Math.round(((hz - f.start) / spanHz) * (n - 1));
    }
    idx = Math.max(0, Math.min(n - 1, idx));
    const [r, g, b] = colorFor(f.dbm[idx], state.wfLo, state.wfHi);
    for (let row = 0; row < rowPx; row++) {
      const o = (row * w + x) * 4;
      img.data[o] = r;
      img.data[o + 1] = g;
      img.data[o + 2] = b;
      img.data[o + 3] = 255;
    }
  }
  ctx.putImageData(img, 0, 0);
}

// ---------------------------------------------------------------- marker

function markerFromPointer(ev) {
  const canvas = $('spectrum');
  const f = state.frame;
  if (!canvas || !f) return;

  const rect = canvas.getBoundingClientRect();
  const { L, R, T, B } = plotMargins(rect.width);
  const pw = Math.max(10, rect.width - L - R);
  const ph = Math.max(10, rect.height - T - B);
  const x = ev.clientX - rect.left;
  const y = ev.clientY - rect.top;
  if (x < L || x > L + pw || y < T || y > T + ph) return;

  const start = f.start;
  const stop = f.stop;
  const hz = start + ((x - L) / pw) * (stop - start);
  const n = f.dbm.length;
  let idx = Math.round(((hz - start) / (stop - start)) * (n - 1));
  idx = Math.max(0, Math.min(n - 1, idx));
  state.marker = { hz, level: f.dbm[idx] };
  setText('tapInfo', fmtFreq(hz) + ' · ' + fmtLevel(f.dbm[idx]) + ' dBm');
  drawSpectrum();
}

function resetTapInfo() {
  setText('tapInfo', t('tap.default'));
}

function bindSpectrum() {
  const canvas = $('spectrum');
  if (!canvas) return;

  canvas.addEventListener('pointerdown', (ev) => {
    state.draggingMarker = true;
    if (canvas.setPointerCapture) {
      try {
        canvas.setPointerCapture(ev.pointerId);
      } catch (_) {}
    }
    markerFromPointer(ev);
  });

  canvas.addEventListener('pointermove', (ev) => {
    if (state.draggingMarker) markerFromPointer(ev);
  });

  const end = (ev) => {
    state.draggingMarker = false;
    if (canvas.releasePointerCapture) {
      try {
        canvas.releasePointerCapture(ev.pointerId);
      } catch (_) {}
    }
    if (ev.pointerType === 'touch') {
      const now = Date.now();
      const dx = ev.clientX - state.lastTapX;
      const dy = ev.clientY - state.lastTapY;
      if (now - state.lastTapAt < 320 && Math.hypot(dx, dy) < 26) {
        state.marker = null;
        resetTapInfo();
        drawSpectrum();
      }
      state.lastTapAt = now;
      state.lastTapX = ev.clientX;
      state.lastTapY = ev.clientY;
    }
  };
  canvas.addEventListener('pointerup', end);
  canvas.addEventListener('pointercancel', () => {
    state.draggingMarker = false;
  });

  canvas.addEventListener('dblclick', () => {
    state.marker = null;
    resetTapInfo();
    drawSpectrum();
  });
}

// ---------------------------------------------------------------- config I/O

function saveConfig() {
  const cfg = {
    start: num('startMHz'),
    stop: num('stopMHz'),
    points: $('points').value,
    rbw: $('rbw').value,
    atten: $('atten').value,
    lna: $('lna').checked,
    spur: $('spur').checked,
    showMax: $('showMax').checked,
    showAvg: $('showAvg').checked,
    showMin: $('showMin').checked,
  };
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg));
  } catch (_) {}
}

function restoreConfig() {
  let cfg = null;
  try {
    cfg = JSON.parse(localStorage.getItem(STORAGE_KEY) || localStorage.getItem(LEGACY_STORAGE_KEY) || 'null');
  } catch (_) {}
  if (!cfg) return;

  if (Number.isFinite(cfg.start)) $('startMHz').value = cfg.start;
  if (Number.isFinite(cfg.stop)) $('stopMHz').value = cfg.stop;

  setSelectValue('points', cfg.points, String(cfg.points));
  setSelectValue('rbw', cfg.rbw, cfg.rbw === 'auto' ? t('unit.auto') : cfg.rbw + ' ' + t('unit.khz'));
  setSelectValue('atten', cfg.atten, cfg.atten === 'auto' ? t('unit.auto') : cfg.atten + ' ' + t('unit.db'));

  if (typeof cfg.lna === 'boolean') $('lna').checked = cfg.lna;
  if (typeof cfg.spur === 'boolean') $('spur').checked = cfg.spur;
  if (typeof cfg.showMax === 'boolean') $('showMax').checked = cfg.showMax;
  if (typeof cfg.showAvg === 'boolean') $('showAvg').checked = cfg.showAvg;
  if (typeof cfg.showMin === 'boolean') $('showMin').checked = cfg.showMin;
}

function initBands() {
  const sel = $('bandSelect');
  const current = sel.value;
  const frag = document.createDocumentFragment();
  BANDS.forEach(([id, start, stop]) => {
    const option = document.createElement('option');
    option.value = id;
    option.textContent = t('band.' + id);
    if (start != null) option.dataset.start = String(start);
    if (stop != null) option.dataset.stop = String(stop);
    frag.appendChild(option);
  });
  sel.replaceChildren(frag);
  if (current && BANDS.some(([id]) => id === current)) sel.value = current;
}

function updateBandSelect() {
  const start = num('startMHz');
  const stop = num('stopMHz');
  const maxMHz = ((state.status && state.status.model && state.status.model.maxFreq) || 960e6) / 1e6;
  let match = 'none';
  if (Number.isFinite(start) && Number.isFinite(stop)) {
    for (const [id, bandStart, bandStop] of BANDS) {
      if (bandStart == null) continue;
      const stopMHz = bandStop === 'full' ? maxMHz : bandStop;
      if (Math.abs(start - bandStart) < 0.0005 && Math.abs(stop - stopMHz) < 0.0005) {
        match = id;
        break;
      }
    }
  }
  $('bandSelect').value = match;
}

async function applyBand() {
  const sel = $('bandSelect');
  const option = sel.selectedOptions[0];
  if (!option || option.dataset.start == null) {
    updateBandSelect();
    return;
  }

  const start = parseFloat(option.dataset.start);
  let stop = parseFloat(option.dataset.stop);
  if (option.dataset.stop === 'full') {
    stop = ((state.status && state.status.model && state.status.model.maxFreq) || 960e6) / 1e6;
  }
  if (!Number.isFinite(start) || !Number.isFinite(stop)) return;

  $('startMHz').value = String(start);
  $('stopMHz').value = String(stop);
  state.dirty = true;
  state.marker = null;
  resetTapInfo();
  updateRunRange();
  saveConfig();

  if (state.status && state.status.sweeping) {
    try {
      await applySweep();
    } catch (e) {
      toast(e.message, true);
    }
  } else {
    updateButtons();
  }
}

// ---------------------------------------------------------------- language

function applyStaticLanguage() {
  document.documentElement.lang = lang;
  document.title = t('app.title');
  for (const el of document.querySelectorAll('[data-i18n]')) {
    el.textContent = t(el.dataset.i18n);
  }
  for (const el of document.querySelectorAll('[data-i18n-aria]')) {
    el.setAttribute('aria-label', t(el.dataset.i18nAria));
  }
  for (const el of document.querySelectorAll('[data-i18n-title]')) {
    el.title = t(el.dataset.i18nTitle);
  }
  for (const el of document.querySelectorAll('[data-unit]')) {
    el.textContent = el.value + ' ' + t('unit.' + el.dataset.unit);
  }
}

function applyLanguage() {
  applyStaticLanguage();
  initBands();
  renderPorts();
  if (state.status) applyStatus(state.status);
  else updateButtons();
  updateBandSelect();
  updateRunRange();
  if (state.marker) {
    setText('tapInfo', fmtFreq(state.marker.hz) + ' · ' + fmtLevel(state.marker.level) + ' dBm');
  } else {
    resetTapInfo();
  }
  if (state.frame) updateReadout();
  drawSpectrum();
}

function setLanguage(next) {
  lang = next === 'ru' ? 'ru' : 'en';
  saveLang(lang);
  applyLanguage();
}

// ---------------------------------------------------------------- events

function onConfigInput() {
  try {
    state.dirty = configKey(configFromForm()) !== state.cfgKey;
  } catch (_) {
    state.dirty = true;
  }
  updateButtons();
  updateRunRange();
  updateBandSelect();
}

function bindControls() {
  $('langSelect').addEventListener('change', () => setLanguage($('langSelect').value));
  $('btnPorts').addEventListener('click', refreshPorts);
  $('btnConnect').addEventListener('click', toggleConnect);
  $('btnRun').addEventListener('click', toggleRun);
  $('btnApply').addEventListener('click', applyCurrent);
  $('btnPeakCenter').addEventListener('click', () => peakAction(false));
  $('btnPeakZoom').addEventListener('click', () => peakAction(true));
  $('bandSelect').addEventListener('change', applyBand);

  for (const id of ['startMHz', 'stopMHz']) {
    $(id).addEventListener('input', onConfigInput);
    $(id).addEventListener('change', saveConfig);
  }

  for (const id of ['points', 'rbw', 'atten', 'lna', 'spur']) {
    $(id).addEventListener('change', () => {
      onConfigInput();
      saveConfig();
    });
  }

  for (const id of ['showMax', 'showAvg', 'showMin']) {
    $(id).addEventListener('change', () => {
      saveConfig();
      drawSpectrum();
    });
  }

  $('csvLink').addEventListener('click', (ev) => {
    if (!state.frame) {
      ev.preventDefault();
      toast(t('toast.noSweepData'), true);
    }
  });

  document.addEventListener('visibilitychange', () => {
    if (!document.hidden) {
      refreshStatus();
      if (!state.status || !state.status.connected) refreshPorts();
    }
  });

  if (window.ResizeObserver) {
    const ro = new ResizeObserver(() => {
      drawSpectrum();
      resizeWaterfall();
    });
    ro.observe($('spectrum'));
    ro.observe($('waterfall'));
  } else {
    window.addEventListener('resize', () => {
      drawSpectrum();
      resizeWaterfall();
    });
  }
}

function connectEvents() {
  if (!window.EventSource) {
    toast(t('toast.noEventSource'), true);
    return;
  }
  const es = new EventSource('/api/events');
  es.addEventListener('status', (ev) => {
    try {
      applyStatus(JSON.parse(ev.data));
    } catch (_) {}
  });
  es.addEventListener('frame', (ev) => {
    try {
      onFrame(JSON.parse(ev.data));
    } catch (_) {}
  });
  es.onopen = () => {
    refreshStatus();
  };
  es.onerror = () => {
    setLinkState(t('state.noLink'), 'bad');
  };
}

function tickFPS() {
  if (state.lastFrameAt > 0 && performance.now() - state.lastFrameAt > 3000) {
    state.fps = 0;
  }
  setText('fpsLabel', state.fps.toFixed(1) + ' fps');
}

async function init() {
  lang = loadLang();
  $('langSelect').value = lang;
  applyStaticLanguage();
  initBands();
  restoreConfig();
  bindControls();
  bindSpectrum();
  resizeWaterfall();
  updateBandSelect();
  updateRunRange();
  resetTapInfo();
  drawSpectrum();

  try {
    applyStatus(await api('/api/status'));
    state.initialized = true;
  } catch (e) {
    setLinkState(t('state.noLink'), 'bad');
    toast(t('toast.serverError', { error: e.message }), true);
  }

  try {
    await refreshPorts();
  } catch (_) {}

  try {
    const f = await api('/api/frame');
    if (f && Array.isArray(f.dbm)) onFrame(f);
  } catch (_) {}

  connectEvents();
  setInterval(tickFPS, 1000);
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}
