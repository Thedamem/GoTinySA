<script lang="ts">
  import { onMount } from 'svelte'
  import Spectrum from './Spectrum.svelte'
  import Waterfall from './Waterfall.svelte'
  import {
    ApplySweep,
    CancelConnect,
    Connect,
    Disconnect,
    DisableUSBLog,
    EnableUSBLog,
    ListPorts,
    LogPath,
    Query,
    SaveCSV,
    StartWebServer,
    Status,
    StopSweep,
    StopWebServer,
    Version,
    WebServerQRCode,
    WebServerStatus,
  } from '../wailsjs/go/main/App.js'
  import { EventsOn, EventsOff, Quit } from '../wailsjs/runtime/runtime.js'
  import { loadLang, saveLang, translator, type Lang } from './i18n'

  interface PortInfo {
    name: string
    vid: string
    pid: string
    serial: string
    product: string
    isTinySA: boolean
  }

  interface Model {
    ultra: boolean
    scale: number
    maxFreq: number
    firmware: string
  }

  interface SweepConfig {
    start: number
    stop: number
    points: number
    rbw: number
    atten: number
    lna: boolean
    spur: boolean
  }

  interface StatusInfo {
    connected: boolean
    connecting: boolean
    port: string
    usbDir: string
    sweeping: boolean
    model: Model
    battery: string
    config?: SweepConfig | null
  }

  interface WebInfo {
    running: boolean
    port: number
    urls: string[]
    started: string
  }

  interface Frame {
    seq: number
    start?: number
    stop?: number
    dbm: number[]
    max?: number[]
    avg?: number[]
    min?: number[]
    noise?: number
    peak?: number
    peakFreq?: number
    snr?: number
  }

  let lang: Lang = loadLang()
  let t = translator(lang)
  $: t = translator(lang)
  $: document.documentElement.lang = lang

  let version = ''
  let ports: PortInfo[] = []
  let selectedPort = ''
  let connected = false
  let connectedPort = ''
  let connecting = false
  let usbDir = ''
  let logDir = 'logs'
  let model: Model = { ultra: false, scale: 128, maxFreq: 960e6, firmware: '' }
  let battery = ''

  let running = false
  let frame: Frame | null = null
  let waterfall: any
  let fps = 0
  let lastFrameAt = 0
  let errorText = ''
  let notice = ''

  const BANDS = [
    { id: 'none', start: 0, stop: 0 },
    { id: 'fm', start: 88, stop: 108 },
    { id: 'aviation', start: 108, stop: 137 },
    { id: 'twoMeter', start: 144, stop: 146 },
    { id: 'dab', start: 174, stop: 240 },
    { id: 'seventyCm', start: 430, stop: 440 },
    { id: 'ism433', start: 433.05, stop: 434.79 },
    { id: 'pmr446', start: 446.0, stop: 446.2 },
    { id: 'wifi24', start: 2400, stop: 2483.5 },
    { id: 'wifi5', start: 5150, stop: 5850 },
    { id: 'wifi6', start: 5925, stop: 7125 },
  ]
  let band = 'fm'
  $: bandOptions = BANDS.map(b => ({ ...b, label: t(`band.${b.id}`) }))

  function buildGuide(tr: (key: string) => string) {
    return [
      { title: tr('guide.lna.title'), body: tr('guide.lna.body') },
      { title: tr('guide.spurs.title'), body: tr('guide.spurs.body') },
      { title: tr('guide.attenuator.title'), body: tr('guide.attenuator.body') },
      { title: tr('guide.rbw.title'), body: tr('guide.rbw.body') },
      { title: tr('guide.points.title'), body: tr('guide.points.body') },
      { title: tr('guide.traces.title'), body: tr('guide.traces.body') },
      { title: tr('guide.plot.title'), body: tr('guide.plot.body') },
      { title: tr('guide.battery.title'), body: tr('guide.battery.body') },
      { title: tr('guide.bands.title'), body: tr('guide.bands.body') },
      { title: tr('guide.usb.title'), body: tr('guide.usb.body') },
    ]
  }
  $: guideSections = buildGuide(t)

  function setLang(next: Lang) {
    lang = next
    saveLang(next)
  }

  let startMHz = 88
  let stopMHz = 108
  let points = 400
  let rbwAuto = true
  let rbw = 100
  let attenAuto = true
  let atten = 0
  let lna = false
  let spur = false
  let lastConfigKey = ''

  let viewStart: number | null = null
  let viewStop: number | null = null
  let showMax = true
  let showAvg = true
  let showMin = true

  let openMenu: string | null = null
  let debugOpen = false
  let aboutOpen = false
  let guideOpen = false
  let cmd = ''
  let consoleOut = ''
  let logPath = ''

  let webOpen = false
  let webBusy = false
  let webError = ''
  let webInfo: WebInfo = { running: false, port: 0, urls: [], started: '' }
  let qrData = ''
  let qrBusy = false

  $: centerMHz = (startMHz + stopMHz) / 2
  $: spanMHz = stopMHz - startMHz

  function fmtPeak(hz: number): string {
    if (hz >= 1e9) return (hz / 1e9).toFixed(4) + ' ' + t('unit.ghz')
    if (hz >= 1e6) return (hz / 1e6).toFixed(3) + ' ' + t('unit.mhz')
    return (hz / 1e3).toFixed(1) + ' ' + t('unit.khz')
  }

  function configKey(c: SweepConfig): string {
    return [c.start, c.stop, c.points, c.rbw, c.atten, c.lna ? 1 : 0, c.spur ? 1 : 0].join('|')
  }

  // The sweep config is shared by the desktop and the phone UI: when either
  // side applies a sweep, both update their controls and plot scale. A status
  // poll only applies a config that differs from the last one; a sweep:started
  // event is applied unconditionally because it describes a new measurement.
  function syncSweepConfig(cfg: SweepConfig | null | undefined, force = false) {
    if (!cfg) return
    const key = configKey(cfg)
    if (!force && key === lastConfigKey) return
    lastConfigKey = key
    startMHz = +(cfg.start / 1e6).toFixed(6)
    stopMHz = +(cfg.stop / 1e6).toFixed(6)
    points = cfg.points
    rbwAuto = cfg.rbw <= 0
    if (!rbwAuto) rbw = cfg.rbw
    attenAuto = cfg.atten < 0
    if (!attenAuto) atten = cfg.atten
    lna = cfg.lna
    spur = cfg.spur
    const match = BANDS.find(b => b.start > 0 && Math.abs(b.start - startMHz) < 0.0005 && Math.abs(b.stop - stopMHz) < 0.0005)
    band = match ? match.id : 'none'
    resetView()
  }

  const toggleMenu = (m: string) => (openMenu = openMenu === m ? null : m)
  const closeMenus = () => (openMenu = null)

  function resetView() {
    viewStart = null
    viewStop = null
  }

  function clampRange() {
    if (!isFinite(startMHz) || startMHz < 0.1) startMHz = 0.1
    if (!isFinite(stopMHz) || stopMHz <= startMHz) stopMHz = startMHz + 1
    resetView()
  }

  function onCenter(e: Event) {
    const v = parseFloat((e.currentTarget as HTMLInputElement).value)
    if (!isFinite(v)) return
    const span = Math.max(0.001, spanMHz)
    startMHz = Math.max(0.1, v - span / 2)
    stopMHz = startMHz + span
    resetView()
  }

  function onSpan(e: Event) {
    const v = parseFloat((e.currentTarget as HTMLInputElement).value)
    if (!isFinite(v) || v <= 0) return
    const c = centerMHz
    startMHz = Math.max(0.1, c - v / 2)
    stopMHz = startMHz + v
    resetView()
  }

  async function refreshPorts() {
    try {
      ports = (await ListPorts()) as PortInfo[]
      if (!selectedPort) {
        const tiny = ports.find(p => p.isTinySA)
        selectedPort = (tiny ?? ports[0])?.name ?? ''
      }
    } catch (e) {
      errorText = String(e)
    }
  }

  async function refreshStatus() {
    const st = (await Status()) as StatusInfo
    connected = st.connected
    connecting = st.connecting
    connectedPort = st.port
    usbDir = st.usbDir
    model = st.model
    battery = st.battery
    running = st.sweeping
    syncSweepConfig(st.config)
  }

  onMount(() => {
    Version().then(v => (version = v))
    LogPath().then(p => (logPath = p))
    WebServerStatus()
      .then(info => (webInfo = info as WebInfo))
      .catch(() => {})
    refreshPorts()
    refreshStatus()

    EventsOn('sweep', (f: any) => {
      const now = performance.now()
      if (lastFrameAt > 0) {
        const dt = (now - lastFrameAt) / 1000
        if (dt > 0) fps = fps ? fps * 0.8 + (1 / dt) * 0.2 : 1 / dt
      }
      lastFrameAt = now
      frame = f
      waterfall?.push(f)
    })
    EventsOn('sweep:error', (msg: string) => {
      errorText = msg
      running = false
    })
    EventsOn('sweep:started', (cfg: SweepConfig) => {
      running = true
      syncSweepConfig(cfg, true)
    })
    EventsOn('sweep:stopped', () => {
      running = false
    })
    EventsOn('connect:step', (msg: string) => {
      notice = msg
      if (msg.startsWith('connected') || msg === 'cancelled' || msg === 'disconnected' || msg.startsWith('timeout')) {
        refreshStatus()
        setTimeout(() => (notice = ''), 4000)
      }
    })

    const timer = setInterval(() => {
      if (lastFrameAt > 0 && performance.now() - lastFrameAt > 3000) fps = 0
    }, 1000)

    return () => {
      clearInterval(timer)
      EventsOff('sweep', 'sweep:error', 'sweep:started', 'sweep:stopped', 'connect:step')
    }
  })

  async function toggleConnect() {
    errorText = ''
    if (connecting) {
      try {
        await CancelConnect()
      } catch (e) {
        errorText = String(e)
      }
      connecting = false
      await refreshStatus()
      return
    }
    notice = ''
    try {
      if (connected) {
        await StopSweep()
        await Disconnect()
        frame = null
        running = false
        resetView()
      } else {
        if (!selectedPort) {
          errorText = t('notice.portNotSelected')
          return
        }
        connecting = true
        await Connect(selectedPort)
      }
      await refreshStatus()
    } catch (e) {
      errorText = String(e)
    } finally {
      connecting = false
      await refreshStatus()
    }
  }

  async function toggleRun() {
    errorText = ''
    if (!connected) return
    if (running) {
      await StopSweep()
      return
    }
    clampRange()
    resetView()
    if (model.ultra && stopMHz > 7300) {
      notice = t('notice.harmonic')
      setTimeout(() => (notice = ''), 6000)
    }
    running = true
    try {
      await ApplySweep({
        start: startMHz * 1e6,
        stop: stopMHz * 1e6,
        points,
        rbw: rbwAuto ? 0 : rbw,
        atten: attenAuto ? -1 : atten,
        lna,
        spur,
      } as any)
    } catch (e) {
      running = false
      errorText = String(e)
    }
  }

  async function toggleLog() {
    errorText = ''
    try {
      if (usbDir) {
        await DisableUSBLog()
        consoleOut = ''
      } else {
        const paths = await EnableUSBLog(logDir)
        consoleOut = t('debug.usbLogPrefix') + paths
      }
      await refreshStatus()
    } catch (e) {
      errorText = String(e)
    }
  }

  async function sendCmd() {
    const c = cmd.trim()
    if (!c) return
    cmd = ''
    try {
      consoleOut = (await Query(c)) || t('debug.emptyResponse')
    } catch (e) {
      consoleOut = String(e)
    }
  }

  async function showPorts() {
    try {
      const list = (await ListPorts()) as PortInfo[]
      if (!list.length) {
        consoleOut = t('debug.noPorts')
        return
      }
      consoleOut = list
        .map(
          p =>
            `${p.name}  ${p.vid}:${p.pid}  ${p.serial}  ${p.product}${p.isTinySA ? '  <- tinySA' : ''}`,
        )
        .join('\n')
    } catch (e) {
      consoleOut = String(e)
    }
  }

  function onCmdKey(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      sendCmd()
    }
  }

  async function saveCSV() {
    openMenu = null
    try {
      const path = await SaveCSV()
      if (path) {
        notice = t('notice.csvSaved', { path })
        setTimeout(() => (notice = ''), 5000)
      }
    } catch (e) {
      errorText = String(e)
    }
  }

  async function refreshWeb() {
    try {
      webInfo = (await WebServerStatus()) as WebInfo
      qrData = ''
    } catch (e) {
      webError = String(e)
    }
  }

  async function toggleWeb() {
    webBusy = true
    webError = ''
    qrData = ''
    try {
      if (webInfo.running) {
        await StopWebServer()
        notice = t('notice.webStopped')
      } else {
        webInfo = (await StartWebServer()) as WebInfo
        notice = t('notice.webStarted', { url: webInfo.urls[0] || t('web.title') })
      }
      setTimeout(() => (notice = ''), 5000)
      await refreshWeb()
    } catch (e) {
      webError = String(e)
    } finally {
      webBusy = false
    }
  }

  async function showQR() {
    qrBusy = true
    webError = ''
    try {
      webInfo = (await WebServerStatus()) as WebInfo
      const url = webInfo.urls[0] || ''
      if (!url || url.includes('127.0.0.1')) {
        throw new Error(t('web.noLan'))
      }
      qrData = await WebServerQRCode(url)
    } catch (e) {
      qrData = ''
      webError = String(e)
    } finally {
      qrBusy = false
    }
  }

  async function copyWebURL(url: string) {
    let copied = false
    try {
      await navigator.clipboard.writeText(url)
      copied = true
    } catch {
      const ta = document.createElement('textarea')
      ta.value = url
      ta.style.position = 'fixed'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.select()
      try {
        copied = document.execCommand('copy')
      } catch {
        copied = false
      }
      ta.remove()
    }
    notice = copied ? t('notice.copied', { url }) : t('notice.copyManual', { url })
    setTimeout(() => (notice = ''), 5000)
  }

  function onViewChange(s: number, e: number) {
    viewStart = s
    viewStop = e
  }

  function onViewReset() {
    resetView()
  }

  function applyBand() {
    const b = BANDS.find(x => x.id === band)
    if (!b || b.start <= 0) return
    startMHz = b.start
    stopMHz = b.stop
    resetView()
  }

  function applyViewToSweep() {
    if (viewStart === null || viewStop === null) return
    startMHz = +(viewStart / 1e6).toFixed(6)
    stopMHz = +(viewStop / 1e6).toFixed(6)
    resetView()
  }
</script>

<svelte:window on:click={closeMenus} />

<div class="app">
  <div class="menubar">
    <div class="menuwrap">
      <button class="menu" class:active={openMenu === 'file'} on:click|stopPropagation={() => toggleMenu('file')}>{t('menu.file')}</button>
      {#if openMenu === 'file'}
        <div class="dropdown" on:click|stopPropagation>
          <button on:click={saveCSV} disabled={!frame}>{t('menu.saveCsv')}</button>
          <div class="sep"></div>
          <button on:click={() => Quit()}>{t('menu.quit')}</button>
        </div>
      {/if}
    </div>

    <div class="menuwrap">
      <button class="menu" class:active={openMenu === 'settings'} on:click|stopPropagation={() => toggleMenu('settings')}>{t('menu.settings')}</button>
      {#if openMenu === 'settings'}
        <div class="dropdown" on:click|stopPropagation>
          <label class="langsel">{t('lang.label')}
            <select bind:value={lang} on:change={() => saveLang(lang)}>
              <option value="en">English</option>
              <option value="ru">Русский</option>
            </select>
          </label>
          <div class="sep"></div>
          <button on:click={() => { openMenu = null; webOpen = true }}>{t('menu.webDialog')}</button>
          <button on:click={() => { openMenu = null; debugOpen = true }}>{t('menu.debug')}</button>
        </div>
      {/if}
    </div>

    <div class="menuwrap">
      <button class="menu" class:active={openMenu === 'help'} on:click|stopPropagation={() => toggleMenu('help')}>{t('menu.help')}</button>
      {#if openMenu === 'help'}
        <div class="dropdown" on:click|stopPropagation>
          <button on:click={() => { openMenu = null; guideOpen = true }}>{t('menu.guide')}</button>
          <button on:click={() => { openMenu = null; aboutOpen = true }}>{t('menu.about')}</button>
        </div>
      {/if}
    </div>

    <button class="menu webtoggle" class:on={webInfo.running} on:click|stopPropagation={() => (webOpen = true)} title={t('web.toggleTitle')}>
      Web<span class="webdot" class:on={webInfo.running}></span>
    </button>

    <div class="spacer"></div>

    <div class="statusline">
      {#if connected}
        <span>{model.ultra ? 'Ultra' : 'tinySA'}</span>
        <span class="dim">{model.firmware.length > 48 ? model.firmware.slice(0, 48) + '…' : model.firmware}</span>
        <span class="dim">{connectedPort}</span>
      {:else}
        <span class="dim">{t('status.notConnected')}</span>
      {/if}
      {#if battery}<span class="dim">{battery}</span>{/if}
      <span class="dim">{fps.toFixed(1)} fps</span>
    </div>
  </div>

  {#if errorText}<div class="errorbar">{errorText}</div>{/if}
  {#if notice}<div class="noticebar">{notice}</div>{/if}

  <div class="main">
    <aside>
      <section>
        <h2>{t('side.device')}</h2>
        <select bind:value={selectedPort} disabled={connected || running || connecting}>
          {#each ports as p}
            <option value={p.name}>{p.name}{p.isTinySA ? ' — tinySA' : ''}</option>
          {/each}
        </select>
        <div class="row">
          <button on:click={refreshPorts} disabled={connected || running || connecting}>{t('common.refresh')}</button>
          <button class="grow" on:click={toggleConnect} disabled={running || (!connected && !connecting && !selectedPort)}>
            {connecting ? t('common.cancel') : connected ? t('common.disconnect') : t('common.connect')}
          </button>
        </div>
        {#if ports.length === 0}
          <div class="hint">{t('hint.noPorts')}</div>
        {/if}
      </section>

      <section>
        <h2>{t('side.options')}</h2>
        <label class="check"><input type="checkbox" bind:checked={lna} disabled={!model.ultra || running} /> LNA</label>
        <label class="check"><input type="checkbox" bind:checked={spur} disabled={running} /> Spurs</label>
      </section>

      <section>
        <h2>{t('side.attenuatorDb')}</h2>
        <label class="check"><input type="checkbox" bind:checked={attenAuto} disabled={running} /> {t('common.auto')}</label>
        <input type="number" min="0" max="31" bind:value={atten} disabled={attenAuto || running} />
      </section>

      <section>
        <h2>{t('side.traces')}</h2>
        <label class="check"><input type="checkbox" checked disabled /> <span class="dot" style="background:#ffd700"></span> {t('trace.normal')}</label>
        <label class="check"><input type="checkbox" bind:checked={showMax} /> <span class="dot" style="background:#87ceeb"></span> {t('trace.max')}</label>
        <label class="check"><input type="checkbox" bind:checked={showAvg} /> <span class="dot" style="background:#dda0dd"></span> {t('trace.average')}</label>
        <label class="check"><input type="checkbox" bind:checked={showMin} /> <span class="dot" style="background:#98fb98"></span> {t('trace.min')}</label>
      </section>

      <div class="sidefoot">
        <div class="hint">{t('hint.battery', { value: battery || '—' })}</div>
        {#if frame?.noise !== undefined}
          <div class="hint">
            {t('hint.noise')}: {frame.noise.toFixed(1)} dBm · {t('hint.peak')}: {frame.peak?.toFixed(1)} dBm
            {#if frame.peakFreq !== undefined}@ {fmtPeak(frame.peakFreq)}{/if}
            · SNR: {frame.snr?.toFixed(1)} dB
          </div>
        {/if}
        <div class="hint">{t('hint.plotControls')}</div>
      </div>
    </aside>

    <div class="plots">
      <div class="spectrum">
        <Spectrum
          start={startMHz * 1e6}
          stop={stopMHz * 1e6}
          {viewStart}
          {viewStop}
          {frame}
          {showMax}
          {showAvg}
          {showMin}
          {lang}
          onViewChange={onViewChange}
          onViewReset={onViewReset}
        />
      </div>
      <div class="waterfall">
        <Waterfall bind:this={waterfall} start={startMHz * 1e6} stop={stopMHz * 1e6} {viewStart} {viewStop} />
      </div>
    </div>
  </div>

  <div class="bottom">
    <label>{t('bottom.points')}
      <input type="number" min="2" max="2000" bind:value={points} disabled={running} />
    </label>
    <label>{t('bottom.rbwKhz')}
      <div class="inline">
        <label class="check mini"><input type="checkbox" bind:checked={rbwAuto} disabled={running} /> {t('common.auto')}</label>
        <input type="number" min="0.1" step="1" bind:value={rbw} disabled={rbwAuto || running} />
      </div>
    </label>
    <label>{t('bottom.band')}
      <select bind:value={band} on:change={applyBand} disabled={running}>
        {#each bandOptions as b}
          <option value={b.id}>{b.label}</option>
        {/each}
      </select>
    </label>
    <label>{t('bottom.startMhz')}
      <input type="number" step="0.001" bind:value={startMHz} on:change={clampRange} disabled={running} />
    </label>
    <label>{t('bottom.center')}
      <input type="number" step="0.001" value={centerMHz.toFixed(4)} on:change={onCenter} disabled={running} />
    </label>
    <label>{t('bottom.stop')}
      <input type="number" step="0.001" bind:value={stopMHz} on:change={clampRange} disabled={running} />
    </label>
    <label>{t('bottom.span')}
      <input type="number" step="0.001" value={spanMHz.toFixed(4)} on:change={onSpan} disabled={running} />
    </label>
    <button class="zoom-apply" on:click={applyViewToSweep} disabled={running || viewStart === null} title={t('bottom.zoomApply')}>
      &lt;&gt;
    </button>
    <button class="run" class:stop={running} on:click={toggleRun} disabled={!connected}>
      {running ? t('common.stop') : t('common.run')}
    </button>
  </div>
</div>

{#if debugOpen}
  <div class="overlay" on:click={() => (debugOpen = false)}>
    <div class="dialog" on:click|stopPropagation>
      <div class="dialog-title">
        <span>{t('debug.title')}</span>
        <button class="x" on:click={() => (debugOpen = false)}>×</button>
      </div>
      <div class="dialog-body">
        <h3>{t('debug.rawUsbLog')}</h3>
        <div class="row">
          <input bind:value={logDir} disabled={!!usbDir || running} />
          <button on:click={toggleLog} disabled={running || connected || connecting}>{usbDir ? t('debug.disable') : t('debug.enable')}</button>
        </div>
        <div class="hint">{t('debug.logHint')}</div>
        {#if usbDir}<div class="hint mono">{usbDir}</div>{/if}
        {#if logPath}<div class="hint mono">{t('debug.logPath', { path: logPath })}</div>{/if}

        <div class="row">
          <button on:click={showPorts}>{t('debug.showPorts')}</button>
          <span class="hint">{t('debug.usbDiagnostics')}</span>
        </div>

        <h3>{t('debug.commandConsole')}</h3>
        <div class="row">
          <input bind:value={cmd} placeholder={t('debug.commandPlaceholder')} on:keydown={onCmdKey} disabled={running} />
          <button on:click={sendCmd} disabled={running || !connected}>{t('debug.send')}</button>
        </div>
        {#if consoleOut}<pre class="out mono">{consoleOut}</pre>{/if}
      </div>
    </div>
  </div>
{/if}

{#if aboutOpen}
  <div class="overlay" on:click={() => (aboutOpen = false)}>
    <div class="dialog" on:click|stopPropagation>
      <div class="dialog-title">
        <span>{t('about.title')}</span>
        <button class="x" on:click={() => (aboutOpen = false)}>×</button>
      </div>
      <div class="dialog-body">
        <div><b>tinysa_go</b> <span class="mono">{version}</span></div>
        <div class="hint">{t('about.description')}</div>
      </div>
    </div>
  </div>
{/if}

{#if guideOpen}
  <div class="overlay" on:click={() => (guideOpen = false)}>
    <div class="dialog guide-dialog" on:click|stopPropagation>
      <div class="dialog-title">
        <span>{t('guide.title')}</span>
        <button class="x" on:click={() => (guideOpen = false)}>×</button>
      </div>
      <div class="dialog-body guide">
        {#each guideSections as section}
          <h3>{section.title}</h3>
          <p>{section.body}</p>
        {/each}
      </div>
    </div>
  </div>
{/if}

{#if webOpen}
  <div class="overlay" role="presentation" on:click={() => (webOpen = false)}>
    <div class="dialog" role="dialog" aria-modal="true" aria-label={t('web.title')} on:click|stopPropagation>
      <div class="dialog-title">
        <span>{t('web.title')}</span>
        <button class="x" on:click={() => (webOpen = false)}>×</button>
      </div>
      <div class="dialog-body">
        <div class="hint">{t('web.intro')}</div>

        <div class="row">
          <button class="grow" on:click={toggleWeb} disabled={webBusy}>
            {webBusy ? '…' : webInfo.running ? t('web.disable') : t('web.enable')}
          </button>
          {#if webInfo.running}
            <button on:click={refreshWeb} disabled={webBusy}>{t('web.refresh')}</button>
            <button on:click={showQR} disabled={qrBusy} title={t('web.qrTitle')}>
              {qrBusy ? '…' : t('web.qr')}
            </button>
          {/if}
        </div>

        {#if qrData}
          <div class="qrbox">
            <img src={qrData} alt={t('web.qrAlt')} />
            <div class="hint">{t('web.qrHint')}</div>
          </div>
        {/if}

        {#if webError}<div class="weberror">{webError}</div>{/if}

        {#if webInfo.running}
          <div class="hint">{t('web.openOnPhone')}</div>
          {#each webInfo.urls as u}
            <div class="weburl">
              <span class="mono">{u}</span>
              <button on:click={() => copyWebURL(u)}>{t('web.copy')}</button>
            </div>
          {/each}
          {#if webInfo.urls.length <= 1}
            <div class="weberror">{t('web.computerOffline')}</div>
          {/if}
          <div class="hint">{t('web.portHint', { port: webInfo.port })}</div>
        {:else}
          <div class="hint">{t('web.notRunning')}</div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .app {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .menubar {
    display: flex;
    align-items: center;
    height: 28px;
    background: #f0f0f0;
    border-bottom: 1px solid #c8c8c8;
    padding: 0 2px;
    position: relative;
    z-index: 10;
  }

  .menuwrap {
    position: relative;
  }

  .menu {
    background: transparent;
    border: none;
    border-radius: 0;
    padding: 4px 10px;
    font-size: 12px;
    color: #1e1e1e;
  }

  .menu:hover,
  .menu.active {
    background: #dcdcdc;
  }

  .webtoggle {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .webtoggle.on {
    color: #1d5c2d;
    font-weight: 600;
  }

  .webdot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: #b0b0b0;
  }

  .webdot.on {
    background: #3fa45b;
    box-shadow: 0 0 6px rgba(63, 164, 91, 0.65);
  }

  .dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    min-width: 170px;
    background: #fafafa;
    border: 1px solid #a8a8a8;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.18);
    padding: 2px 0;
    z-index: 30;
  }

  .dropdown button {
    display: block;
    width: 100%;
    text-align: left;
    background: none;
    border: none;
    border-radius: 0;
    padding: 5px 14px;
    font-size: 12px;
  }

  .dropdown button:hover:not(:disabled) {
    background: #dbe9f6;
  }

  .langsel {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 5px 14px;
    font-size: 12px;
  }

  .langsel select {
    width: auto;
    min-width: 104px;
  }

  .sep {
    height: 1px;
    background: #d0d0d0;
    margin: 3px 0;
  }

  .spacer {
    flex: 1;
  }

  .statusline {
    display: flex;
    gap: 10px;
    font-size: 11px;
    padding-right: 8px;
    white-space: nowrap;
    overflow: hidden;
    min-width: 0;
  }

  .statusline span {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .statusline .dim {
    color: #666;
  }

  .errorbar {
    background: #f8d7da;
    color: #7a1f26;
    border-bottom: 1px solid #e0aeb2;
    padding: 4px 10px;
    font-size: 12px;
  }

  .noticebar {
    background: #e2f0d9;
    color: #2f5c2f;
    border-bottom: 1px solid #b8d4b0;
    padding: 4px 10px;
    font-size: 12px;
  }

  .main {
    flex: 1;
    display: flex;
    min-height: 0;
  }

  aside {
    width: 225px;
    background: #f0f0f0;
    border-right: 1px solid #c8c8c8;
    padding: 8px;
    overflow: auto;
    display: flex;
    flex-direction: column;
  }

  aside section {
    margin-bottom: 10px;
  }

  aside h2 {
    font-size: 11px;
    font-weight: 600;
    color: #333;
    margin: 0 0 4px;
  }

  .check {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 12px;
    margin: 3px 0;
  }

  .check input[type='checkbox'] {
    width: 13px;
    height: 13px;
    margin: 0;
    padding: 0;
  }

  .dot {
    display: inline-block;
    width: 9px;
    height: 9px;
    border: 1px solid #666;
  }

  .row {
    display: flex;
    gap: 4px;
    margin-top: 4px;
  }

  .row input {
    flex: 1;
    min-width: 0;
  }

  .grow {
    flex: 1;
  }

  input[type='number'] {
    width: 100%;
  }

  .hint {
    color: #666;
    font-size: 11px;
    line-height: 1.4;
  }

  .sidefoot {
    margin-top: auto;
    padding-top: 8px;
    border-top: 1px solid #d5d5d5;
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .plots {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    background: #000;
  }

  .spectrum {
    flex: 1;
    min-height: 0;
  }

  .waterfall {
    height: 150px;
    border-top: 1px solid #555;
  }

  .bottom {
    display: flex;
    align-items: flex-end;
    gap: 8px;
    padding: 6px 8px;
    background: #f0f0f0;
    border-top: 1px solid #c8c8c8;
  }

  .bottom label {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 10px;
    color: #444;
  }

  .bottom input {
    width: 88px;
  }

  .bottom select {
    width: 130px;
  }

  .inline {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .check.mini {
    margin: 0;
    font-size: 11px;
    white-space: nowrap;
  }

  .zoom-apply {
    align-self: flex-end;
    width: 34px;
  }

  .bottom .hint {
    align-self: center;
  }

  .run {
    margin-left: auto;
    min-width: 78px;
    font-weight: 600;
  }

  .run.stop {
    background: #f2dede;
    border-color: #c88;
  }

  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.25);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
  }

  .dialog {
    width: 470px;
    max-width: 92vw;
    background: #f5f5f5;
    border: 1px solid #999;
    box-shadow: 0 4px 18px rgba(0, 0, 0, 0.3);
  }

  .dialog-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: #e4e4e4;
    border-bottom: 1px solid #bbb;
    padding: 5px 10px;
    font-size: 12px;
    font-weight: 600;
  }

  .dialog-body {
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .dialog-body h3 {
    font-size: 11px;
    font-weight: 600;
    color: #333;
    margin: 6px 0 0;
  }

  .x {
    border: none;
    background: none;
    font-size: 16px;
    line-height: 1;
    padding: 0 4px;
  }

  .out {
    background: #fff;
    border: 1px solid #c0c0c0;
    padding: 6px;
    margin: 0;
    font-size: 11px;
    color: #222;
    max-height: 140px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .weburl {
    display: flex;
    align-items: center;
    gap: 8px;
    border: 1px solid #c8c8c8;
    background: #fff;
    padding: 3px 4px 3px 7px;
  }

  .weburl .mono {
    flex: 1;
    min-width: 0;
    overflow: auto;
    white-space: nowrap;
    font-size: 11.5px;
  }

  .weberror {
    background: #f8d7da;
    color: #7a1f26;
    border: 1px solid #e0aeb2;
    padding: 5px 8px;
    font-size: 12px;
  }

  .qrbox {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 5px;
    padding: 8px;
    background: #fff;
    border: 1px solid #c8c8c8;
  }

  .qrbox img {
    width: 328px;
    max-width: 100%;
    height: auto;
    image-rendering: pixelated;
  }

  .guide-dialog {
    width: 720px;
    max-height: 82vh;
    display: flex;
    flex-direction: column;
  }

  .guide {
    overflow: auto;
    max-height: calc(82vh - 36px);
  }

  .guide h3 {
    margin: 10px 0 2px;
    font-size: 12px;
  }

  .guide p {
    margin: 0 0 4px;
    font-size: 12px;
    line-height: 1.5;
    color: #222;
  }
</style>
