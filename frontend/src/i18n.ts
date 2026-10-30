export type Lang = 'en' | 'ru'

const STORAGE_KEY = 'tinysa_lang'

const en: Record<string, string> = {
  'lang.label': 'Language',
  'menu.file': 'File',
  'menu.settings': 'Settings',
  'menu.help': 'Help',
  'menu.saveCsv': 'Save CSV…',
  'menu.quit': 'Quit',
  'menu.webDialog': 'Web interface…',
  'menu.debug': 'Debug…',
  'menu.guide': 'Guide…',
  'menu.about': 'About…',
  'status.notConnected': 'not connected',
  'common.battery': 'Battery',
  'common.auto': 'auto',
  'common.connect': 'Connect',
  'common.disconnect': 'Disconnect',
  'common.cancel': 'Cancel',
  'common.refresh': 'Refresh',
  'common.run': 'Run',
  'common.stop': 'Stop',
  'common.send': 'Send',
  'side.device': 'Device',
  'side.options': 'Options',
  'side.attenuatorDb': 'Attenuator, dB',
  'side.traces': 'Traces',
  'trace.live': 'Live',
  'trace.normal': 'Normal',
  'trace.max': 'Max',
  'trace.average': 'Average',
  'trace.min': 'Min',
  'hint.battery': 'Battery: {value}',
  'hint.noise': 'Noise',
  'hint.peak': 'peak',
  'hint.plotControls': 'Wheel zooms, drag pans, double-click resets.',
  'hint.noPorts': 'No ports found: check the USB cable (data lines) and the COM-port driver.',
  'notice.harmonic': 'Above 7.3 GHz is harmonic mode; level is not calibrated',
  'notice.csvSaved': 'CSV saved: {path}',
  'notice.portNotSelected': 'port not selected',
  'notice.webStarted': 'Web interface: {url}',
  'notice.webStopped': 'Web interface stopped',
  'notice.copied': 'Address copied: {url}',
  'notice.copyManual': 'Copy manually: {url}',
  'debug.title': 'Debug',
  'debug.rawUsbLog': 'Raw USB log',
  'debug.enable': 'Enable',
  'debug.disable': 'Disable',
  'debug.logHint': 'Writes text and binary traffic to the directory. Enable before connecting.',
  'debug.logPath': 'Log: {path}',
  'debug.showPorts': 'Show ports',
  'debug.usbDiagnostics': 'USB diagnostics',
  'debug.commandConsole': 'Command console',
  'debug.commandPlaceholder': 'version, info, vbat…',
  'debug.send': 'Send',
  'debug.emptyResponse': '(empty)',
  'debug.usbLogPrefix': 'USB log:\n',
  'debug.noPorts':
    'No ports found.\nCheck the USB cable (it must carry data, not only power) and the driver:\n' +
    'Device Manager → Ports (COM & LPT). The device should appear as a COM port.',
  'about.title': 'About',
  'about.description': 'Go + Wails port of QtTinySA. Licensed under GPL-3.0-or-later.',
  'about.license': 'GPL-3.0-or-later',
  'web.title': 'Web interface for phone',
  'web.intro':
    'Spectrum and waterfall in a phone browser. The phone and computer must be on the same Wi-Fi network. ' +
    'The interface has no password — do not expose it to the internet.',
  'web.enable': 'Enable',
  'web.disable': 'Disable',
  'web.refresh': 'Refresh',
  'web.qr': 'QR',
  'web.qrTitle': 'QR code for phone access',
  'web.qrAlt': 'QR code for phone access',
  'web.qrHint': 'Point the phone camera at the code',
  'web.noLan':
    'No LAN address: connect the laptop to Wi-Fi or the phone hotspot and press QR again',
  'web.openOnPhone': 'Open on your phone:',
  'web.copy': 'Copy',
  'web.portHint':
    'Port {port}. If the page does not open, check that the phone is on the same network and allow incoming connections in the firewall.',
  'web.notRunning': 'The server runs while the app is open. Start it with the button.',
  'web.computerOffline':
    'The computer is not connected to Wi-Fi/LAN — the phone cannot open the page. ' +
    'Connect to a network or the phone hotspot and press Refresh.',
  'web.toggleTitle': 'Local web UI for phone',
  'guide.title': 'Guide',
  'guide.lna.title': 'LNA',
  'guide.lna.body':
    'Internal low-noise amplifier (+20 dB, Ultra/Ultra+ only). It improves sensitivity to weak signals but reduces the maximum input level and dynamic range. Use it for weak signals; turn it off for strong ones to avoid overload and false components. Enabling LNA sets the attenuator to 0 dB.',
  'guide.spurs.title': 'Spurs (spurious suppression)',
  'guide.spurs.body':
    'Suppresses internal spurious components caused by local-oscillator harmonics. It roughly doubles sweep time. Enable it above 800 MHz; Auto does this automatically.',
  'guide.attenuator.title': 'Attenuator',
  'guide.attenuator.body':
    'Input attenuator, 0–31 dB. Protects the input and reduces distortion on strong signals. Auto selects attenuation automatically. Maximum input without attenuation: +6 dBm (Ultra).',
  'guide.rbw.title': 'RBW (resolution bandwidth)',
  'guide.rbw.body':
    'Smaller RBW improves frequency resolution and lowers the noise floor but slows the sweep. Auto is about 7 kHz per 1 MHz of span. Use 100–600 kHz for fast overview sweeps and 1–10 kHz to resolve close signals.',
  'guide.points.title': 'Points',
  'guide.points.body':
    'Number of scan points (2–2000; the Ultra displays up to 450 points on its own screen). More points give a more detailed trace but take longer and transfer more data over USB.',
  'guide.traces.title': 'Traces',
  'guide.traces.body':
    'Normal is the current sweep. Max and Min are the extremes since Run. Average is the running average since Run. Traces reset on a new Run and are included in CSV export.',
  'guide.plot.title': 'Plot',
  'guide.plot.body':
    'Mouse wheel zooms frequency and level around the cursor; drag pans; double-click resets. The <> button applies the visible range to the sweep (the device rescans that range). Hover shows the exact frequency and trace levels.',
  'guide.battery.title': 'Battery',
  'guide.battery.body':
    'Supply voltage read with the vbat command when connecting (for example, 4.1 V).',
  'guide.bands.title': 'Frequency ranges',
  'guide.bands.body':
    'tinySA basic: 100 kHz–960 MHz. Ultra/Ultra+: 100 kHz–800/900 MHz direct, above that harmonic mode. On Ultra+ ZS407 the level is calibrated to 7.3 GHz; above that (up to about 11–12 GHz) it works as an uncalibrated overview receiver.',
  'guide.usb.title': 'Firmware USB commands',
  'guide.usb.body':
    'Full command list: run help on the device. Common commands: version, info, deviceid, vbat, help; ' +
    'scanraw start stop points 3 (auto-repeating stream) and abort; pause, resume, restart, wait, repeat N; ' +
    'attenuate auto|0..31, lna on|off, lna2, spur off|on|auto, rbw auto|kHz; sweep start|stop Hz, sweeptime, zero; ' +
    'mode low|high|genlow|genhigh, ultra on|off, direct, avoid, if; marker, trace, trigger, level, leveloffset, levelchange; ' +
    'save/recall, saveconfig/clearconfig; sd_list, sd_read, sd_delete; output, modulation; capture, touch, refresh, release. ' +
    'Transport: USB CDC (VID:PID 0483:5740), CR-terminated commands, responses end with the ch> prompt.',
  'bottom.points': 'Points',
  'bottom.rbwKhz': 'RBW, kHz',
  'bottom.band': 'Band',
  'bottom.startMhz': 'Start, MHz',
  'bottom.center': 'Center',
  'bottom.stop': 'Stop',
  'bottom.span': 'Span',
  'bottom.zoomApply': 'Sweep the selected range',
  'unit.ghz': 'GHz',
  'unit.mhz': 'MHz',
  'unit.khz': 'kHz',
  'unit.hz': 'Hz',
  'unit.khzShort': 'kHz',
  'unit.ghzShort': 'G',
  'unit.mhzShort': 'M',
  'unit.khzAxis': 'k',
  'unit.dbShort': 'dB',
  'band.none': '—',
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
  'band.wifi6': 'Wi-Fi 6',
  'band.full': 'Full range',
}

const ru: Record<string, string> = {
  'lang.label': 'Язык',
  'menu.file': 'Файл',
  'menu.settings': 'Настройки',
  'menu.help': 'Справка',
  'menu.saveCsv': 'Сохранить CSV…',
  'menu.quit': 'Выход',
  'menu.webDialog': 'Веб-интерфейс…',
  'menu.debug': 'Отладка…',
  'menu.guide': 'Руководство…',
  'menu.about': 'О программе…',
  'status.notConnected': 'не подключено',
  'common.battery': 'Батарея',
  'common.auto': 'авто',
  'common.connect': 'Подключить',
  'common.disconnect': 'Отключить',
  'common.cancel': 'Отмена',
  'common.refresh': 'Обновить',
  'common.run': 'Запуск',
  'common.stop': 'Стоп',
  'common.send': 'Отправить',
  'side.device': 'Устройство',
  'side.options': 'Параметры',
  'side.attenuatorDb': 'Аттенюатор, дБ',
  'side.traces': 'Трассы',
  'trace.live': 'Текущий',
  'trace.normal': 'Текущий',
  'trace.max': 'Максимум',
  'trace.average': 'Среднее',
  'trace.min': 'Минимум',
  'hint.battery': 'Батарея: {value}',
  'hint.noise': 'Шум',
  'hint.peak': 'пик',
  'hint.plotControls': 'Колесо — зум, перетаскивание — сдвиг, двойной клик — сброс.',
  'hint.noPorts': 'Портов не найдено: проверьте USB-кабель (линии данных) и драйвер COM-порта.',
  'notice.harmonic': 'Выше 7.3 ГГц — гармонический режим, уровень не калиброван',
  'notice.csvSaved': 'CSV сохранён: {path}',
  'notice.portNotSelected': 'порт не выбран',
  'notice.webStarted': 'Веб-интерфейс: {url}',
  'notice.webStopped': 'Веб-интерфейс выключен',
  'notice.copied': 'Адрес скопирован: {url}',
  'notice.copyManual': 'Скопируйте вручную: {url}',
  'debug.title': 'Отладка',
  'debug.rawUsbLog': 'Сырой USB-лог',
  'debug.enable': 'Включить',
  'debug.disable': 'Выключить',
  'debug.logHint': 'Пишет текст и бинарный трафик в каталог. Включается до подключения.',
  'debug.logPath': 'Лог: {path}',
  'debug.showPorts': 'Показать порты',
  'debug.usbDiagnostics': 'диагностика USB',
  'debug.commandConsole': 'Консоль команд',
  'debug.commandPlaceholder': 'version, info, vbat…',
  'debug.send': 'Отправить',
  'debug.emptyResponse': '(пусто)',
  'debug.usbLogPrefix': 'USB-лог:\n',
  'debug.noPorts':
    'Портов не найдено.\nПроверьте USB-кабель (нужен кабель с линиями данных, не только заряд) и драйвер:\n' +
    'Диспетчер устройств → Порты (COM и LPT). Прибор должен быть виден как COM-порт.',
  'about.title': 'О программе',
  'about.description': 'Порт QtTinySA на Go + Wails. Лицензия GPL-3.0-or-later.',
  'about.license': 'GPL-3.0-or-later',
  'web.title': 'Веб-интерфейс для телефона',
  'web.intro':
    'Спектр и водопад в браузере телефона. Телефон и компьютер должны быть в одной Wi-Fi сети. ' +
    'Интерфейс без пароля — не открывайте его из интернета.',
  'web.enable': 'Включить',
  'web.disable': 'Выключить',
  'web.refresh': 'Обновить',
  'web.qr': 'QR',
  'web.qrTitle': 'QR-код для подключения с телефона',
  'web.qrAlt': 'QR-код для подключения с телефона',
  'web.qrHint': 'Наведите камеру телефона на код',
  'web.noLan':
    'Нет LAN-адреса: подключите ноутбук к Wi-Fi или точке доступа телефона и нажмите «QR» ещё раз',
  'web.openOnPhone': 'Откройте на телефоне:',
  'web.copy': 'Копировать',
  'web.portHint':
    'Порт {port}. Если страница не открывается — проверьте, что телефон в той же сети, и разрешите входящие соединения в файрволе.',
  'web.notRunning': 'Сервер работает, пока открыто приложение. Запускается по кнопке.',
  'web.computerOffline':
    'Компьютер не подключён к Wi-Fi/LAN — телефон не сможет открыть страницу. ' +
    'Подключитесь к сети или точке доступа телефона и нажмите «Обновить».',
  'web.toggleTitle': 'Локальная веб-морда для телефона',
  'guide.title': 'Руководство',
  'guide.lna.title': 'LNA',
  'guide.lna.body':
    'Внутренний малошумящий усилитель +20 дБ (только Ultra/Ultra+). Повышает чувствительность к слабым сигналам, но снижает максимальный входной уровень и динамический диапазон. Включайте для слабых сигналов; при сильных — выключайте, иначе перегрузка и ложные составляющие. При включении аттенюатор ставится в 0 дБ.',
  'guide.spurs.title': 'Spurs (подавление спуров)',
  'guide.spurs.body':
    'Убирает внутренние ложные составляющие от гармоник гетеродина. Включение примерно вдвое замедляет свип. Выше 800 МГц включайте обязательно (режим Auto делает это сам).',
  'guide.attenuator.title': 'Аттенюатор',
  'guide.attenuator.body':
    'Входной аттенюатор 0–31 дБ. Защищает вход и уменьшает искажения при сильных сигналах; Auto подбирает затухание автоматически. Максимальный вход без аттенюатора: +6 дБм (Ultra).',
  'guide.rbw.title': 'RBW (полоса разрешения)',
  'guide.rbw.body':
    'Чем меньше RBW, тем лучше разрешение по частоте и ниже шумовая дорожка, но тем медленнее свип. Auto в tinySA ≈ 7 кГц на 1 МГц полосы обзора. Для быстрых обзорных свипов берите 100–600 кГц, для разбора близких сигналов — 1–10 кГц.',
  'guide.points.title': 'Точки',
  'guide.points.body':
    'Число точек сканирования (2–2000; у Ultra до 450 точек на экране прибора). Больше точек — детальнее картинка, но дольше свип и больше данных по USB.',
  'guide.traces.title': 'Трассы',
  'guide.traces.body':
    'Normal — текущий свип. Max и Min — максимумы и минимумы с момента Run. Average — среднее с момента Run. Все трассы сбрасываются при новом Run и сохраняются в CSV.',
  'guide.plot.title': 'График',
  'guide.plot.body':
    'Колесо мыши — зум по частоте и уровню вокруг курсора; перетаскивание — сдвиг; двойной клик — сброс. Кнопка «<>» применяет видимый участок к свипу (прибор переснимает этот диапазон). Наведение показывает точную частоту и уровни трасс.',
  'guide.battery.title': 'Батарея',
  'guide.battery.body':
    'Напряжение питания прибора, считывается командой vbat при подключении (например, 4.1 В).',
  'guide.bands.title': 'Диапазоны',
  'guide.bands.body':
    'tinySA basic: 100 кГц–960 МГц. Ultra/Ultra+: 100 кГц–800/900 МГц напрямую, выше — гармонический режим. У Ultra+ ZS407 уровень калиброван до 7.3 ГГц; выше (до ~11–12 ГГц) прибор работает как обзорный приёмник без калибровки уровня.',
  'guide.usb.title': 'USB-команды прошивки',
  'guide.usb.body':
    'Полный список: команда help на приборе. Основные команды: version, info, deviceid, vbat, help; ' +
    'scanraw start stop points 3 (поток с авто-повтором) и abort; pause, resume, restart, wait, repeat N; ' +
    'attenuate auto|0..31, lna on|off, lna2, spur off|on|auto, rbw auto|кГц; sweep start|stop Гц, sweeptime, zero; ' +
    'mode low|high|genlow|genhigh, ultra on|off, direct, avoid, if; marker, trace, trigger, level, leveloffset, levelchange; ' +
    'save/recall, saveconfig/clearconfig; sd_list, sd_read, sd_delete; output, modulation; capture, touch, refresh, release. ' +
    'Транспорт: USB CDC (VID:PID 0483:5740), команды завершаются CR, ответ заканчивается приглашением ch>.',
  'bottom.points': 'Точек',
  'bottom.rbwKhz': 'RBW, кГц',
  'bottom.band': 'Диапазон',
  'bottom.startMhz': 'Старт, МГц',
  'bottom.center': 'Центр',
  'bottom.stop': 'Конец',
  'bottom.span': 'Полоса',
  'bottom.zoomApply': 'Сканировать выделенный участок',
  'unit.ghz': 'ГГц',
  'unit.mhz': 'МГц',
  'unit.khz': 'кГц',
  'unit.hz': 'Гц',
  'unit.khzShort': 'кГц',
  'unit.ghzShort': 'Г',
  'unit.mhzShort': 'М',
  'unit.khzAxis': 'к',
  'unit.dbShort': 'дБ',
  'band.none': '—',
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
  'band.wifi6': 'Wi-Fi 6',
  'band.full': 'Весь диапазон',
}

const messages: Record<Lang, Record<string, string>> = { en, ru }

export const defaultLang: Lang = 'en'

export function loadLang(): Lang {
  try {
    const value = localStorage.getItem(STORAGE_KEY)
    if (value === 'ru' || value === 'en') return value
  } catch {
    // localStorage can be unavailable in a sandboxed webview.
  }
  return defaultLang
}

export function saveLang(lang: Lang) {
  try {
    localStorage.setItem(STORAGE_KEY, lang)
  } catch {
    // Ignore storage failures; the in-memory choice still works.
  }
}

export function translator(lang: Lang) {
  return (key: string, vars?: Record<string, string | number>): string => {
    let text = messages[lang][key] ?? messages.en[key] ?? key
    if (vars) {
      for (const [name, value] of Object.entries(vars)) {
        text = text.split(`{${name}}`).join(String(value))
      }
    }
    return text
  }
}
