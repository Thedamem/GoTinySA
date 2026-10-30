<script lang="ts">
  import { onMount } from 'svelte'
  import { translator, type Lang } from './i18n'

  interface Frame {
    seq: number
    start?: number
    stop?: number
    dbm: number[]
    max?: number[]
    avg?: number[]
    min?: number[]
    noise?: number
  }

  export let start = 88e6
  export let stop = 108e6
  export let viewStart: number | null = null
  export let viewStop: number | null = null
  export let frame: Frame | null = null
  export let showMax = true
  export let showAvg = true
  export let showMin = true
  export let lang: Lang = 'en'
  $: tr = translator(lang)
  export let onViewChange: (s: number, e: number) => void = () => {}
  export let onViewReset: () => void = () => {}

  const L = 56
  const R = 14
  const T = 10
  const B = 24

  const COLORS = {
    live: '#ffd700',
    max: '#87ceeb',
    avg: '#dda0dd',
    min: '#98fb98',
  }

  let wrap: HTMLDivElement
  let canvas: HTMLCanvasElement
  let width = 0
  let height = 0
  let yMin = -120
  let yMax = -20
  let mouse: { x: number; y: number } | null = null
  let dragging = false
  let dragX = 0
  let dragY = 0
  let dragViewStart = 0
  let dragViewSpan = 0
  let dragYMin = 0
  let dragYSpan = 0

  // The frame carries the range the device actually scanned. Prefer it over
  // the form values so labels stay correct even if a config event is delayed.
  $: dataStart = frame && frame.start !== undefined && frame.stop !== undefined && frame.stop > frame.start ? frame.start : start
  $: dataStop = frame && frame.start !== undefined && frame.stop !== undefined && frame.stop > frame.start ? frame.stop : stop
  $: vs = Math.min(viewStart ?? dataStart, viewStop ?? dataStop)
  $: ve = Math.max(viewStart ?? dataStart, viewStop ?? dataStop)

  const clamp = (v: number, a: number, b: number) => Math.max(a, Math.min(b, v))

  function fmtFreq(hz: number, span: number): string {
    if (hz >= 1e9) return (hz / 1e9).toFixed(span >= 1e9 ? 0 : 3) + tr('unit.ghzShort')
    if (hz >= 1e6) return (hz / 1e6).toFixed(span >= 100e6 ? 0 : span >= 10e6 ? 1 : 3) + tr('unit.mhzShort')
    if (hz >= 1e3) return (hz / 1e3).toFixed(0) + tr('unit.khzAxis')
    return hz.toFixed(0)
  }

  function fmtExact(hz: number): string {
    if (hz >= 1e9) return (hz / 1e9).toFixed(6) + ' ' + tr('unit.ghz')
    if (hz >= 1e6) return (hz / 1e6).toFixed(6) + ' ' + tr('unit.mhz')
    return (hz / 1e3).toFixed(3) + ' ' + tr('unit.khz')
  }

  function niceStep(raw: number): number {
    const p = Math.pow(10, Math.floor(Math.log10(raw)))
    const n = raw / p
    return (n <= 1 ? 1 : n <= 2 ? 2 : n <= 5 ? 5 : 10) * p
  }

  // tracePath maps every point through the *visible* window; the caller clips
  // to the plot rect so segments crossing the edge stay smooth.
  function tracePath(ctx: CanvasRenderingContext2D, data: number[], pw: number, ph: number) {
    const n = data.length
    if (n < 2) return
    ctx.beginPath()
    for (let i = 0; i < n; i++) {
      const freq = dataStart + ((dataStop - dataStart) * i) / (n - 1)
      const x = L + ((freq - vs) / (ve - vs)) * pw
      const v = clamp(data[i], yMin, yMax)
      const y = T + (1 - (v - yMin) / (yMax - yMin)) * ph
      if (i === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    }
  }

  function draw() {
    if (!canvas || width <= 0 || height <= 0) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const dpr = window.devicePixelRatio || 1
    canvas.width = Math.round(width * dpr)
    canvas.height = Math.round(height * dpr)
    canvas.style.width = width + 'px'
    canvas.style.height = height + 'px'
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)

    const pw = Math.max(10, width - L - R)
    const ph = Math.max(10, height - T - B)

    ctx.fillStyle = '#000'
    ctx.fillRect(0, 0, width, height)

    ctx.font = '11px ui-monospace, monospace'
    ctx.textBaseline = 'middle'

    const yStep = niceStep((yMax - yMin) / 6)
    const yFirst = Math.ceil(yMin / yStep) * yStep
    for (let dbm = yFirst; dbm <= yMax; dbm += yStep) {
      const y = T + (1 - (dbm - yMin) / (yMax - yMin)) * ph
      ctx.strokeStyle = '#2a2a2a'
      ctx.beginPath()
      ctx.moveTo(L, y)
      ctx.lineTo(L + pw, y)
      ctx.stroke()
      ctx.fillStyle = '#b8b8b8'
      ctx.textAlign = 'right'
      ctx.fillText(yStep >= 1 ? dbm.toFixed(0) : dbm.toFixed(1), L - 8, y)
    }

    const ticks = 8
    for (let i = 0; i <= ticks; i++) {
      const x = L + (i / ticks) * pw
      ctx.strokeStyle = '#2a2a2a'
      ctx.beginPath()
      ctx.moveTo(x, T)
      ctx.lineTo(x, T + ph)
      ctx.stroke()
      const f = vs + ((ve - vs) * i) / ticks
      ctx.fillStyle = '#b8b8b8'
      ctx.textAlign = i === 0 ? 'left' : i === ticks ? 'right' : 'center'
      ctx.fillText(fmtFreq(f, ve - vs), x, T + ph + 12)
    }

    const n = frame?.dbm?.length ?? 0
    if (n > 1) {
      ctx.save()
      ctx.beginPath()
      ctx.rect(L, T, pw, ph)
      ctx.clip()

      if (showMin && frame?.min) {
        tracePath(ctx, frame.min, pw, ph)
        ctx.strokeStyle = COLORS.min
        ctx.lineWidth = 1
        ctx.stroke()
      }
      if (showAvg && frame?.avg) {
        tracePath(ctx, frame.avg, pw, ph)
        ctx.strokeStyle = COLORS.avg
        ctx.lineWidth = 1
        ctx.stroke()
      }
      if (showMax && frame?.max) {
        tracePath(ctx, frame.max, pw, ph)
        ctx.strokeStyle = COLORS.max
        ctx.lineWidth = 1
        ctx.stroke()
      }
      tracePath(ctx, frame.dbm, pw, ph)
      ctx.strokeStyle = COLORS.live
      ctx.lineWidth = 1.4
      ctx.stroke()

      ctx.restore()
    }

    // Crosshair and tooltip
    if (mouse && n > 1 && mouse.x >= L && mouse.x <= L + pw && mouse.y >= T && mouse.y <= T + ph) {
      const freqAt = vs + ((mouse.x - L) / pw) * (ve - vs)
      const i = Math.round(((freqAt - dataStart) / (dataStop - dataStart)) * (n - 1))
      if (i >= 0 && i < n) {
        const freq = dataStart + ((dataStop - dataStart) * i) / (n - 1)
        const x = L + ((freq - vs) / (ve - vs)) * pw
        const v = clamp(frame!.dbm[i], yMin, yMax)
        const y = T + (1 - (v - yMin) / (yMax - yMin)) * ph

        ctx.strokeStyle = '#707070'
        ctx.setLineDash([3, 3])
        ctx.beginPath()
        ctx.moveTo(x, T)
        ctx.lineTo(x, T + ph)
        ctx.stroke()
        ctx.setLineDash([])

        ctx.fillStyle = COLORS.live
        ctx.beginPath()
        ctx.arc(x, y, 3, 0, Math.PI * 2)
        ctx.fill()

        const rows: Array<{ label: string; value: string; color: string }> = [
          { label: tr('trace.live'), value: frame!.dbm[i].toFixed(1) + ' dBm', color: COLORS.live },
        ]
        if (showMax && frame?.max) {
          rows.push({ label: tr('trace.max'), value: frame.max[i].toFixed(1) + ' dBm', color: COLORS.max })
        }
        if (showAvg && frame?.avg) {
          rows.push({ label: tr('trace.average'), value: frame.avg[i].toFixed(1) + ' dBm', color: COLORS.avg })
        }
        if (showMin && frame?.min) {
          rows.push({ label: tr('trace.min'), value: frame.min[i].toFixed(1) + ' dBm', color: COLORS.min })
        }
        if (frame?.noise !== undefined) {
          rows.push({ label: 'SNR', value: (frame.dbm[i] - frame.noise).toFixed(1) + ' dB', color: '#d0d0d0' })
        }

        const bw = 178
        const bh = 20 + rows.length * 15
        let bx = mouse.x + 14
        let by = mouse.y + 14
        if (bx + bw > width - 4) bx = mouse.x - 14 - bw
        if (by + bh > height - 4) by = mouse.y - 14 - bh

        ctx.fillStyle = 'rgba(24, 24, 24, 0.94)'
        ctx.fillRect(bx, by, bw, bh)
        ctx.strokeStyle = '#666'
        ctx.strokeRect(bx + 0.5, by + 0.5, bw - 1, bh - 1)

        ctx.textAlign = 'left'
        ctx.fillStyle = '#e8e8e8'
        ctx.fillText(fmtExact(freq), bx + 8, by + 11)
        rows.forEach((row, idx) => {
          const ry = by + 26 + idx * 15
          ctx.fillStyle = row.color
          ctx.fillRect(bx + 8, ry - 4, 8, 8)
          ctx.fillStyle = '#d0d0d0'
          ctx.fillText(`${row.label}  ${row.value}`, bx + 22, ry)
        })
      }
    }
  }

  function resize() {
    if (!wrap) return
    width = wrap.clientWidth
    height = wrap.clientHeight
  }

  function onWheel(e: WheelEvent) {
    e.preventDefault()
    const pw = Math.max(10, width - L - R)
    const ph = Math.max(10, height - T - B)
    const factor = e.deltaY > 0 ? 1.3 : 1 / 1.3

    const spanX = ve - vs
    const spanY = yMax - yMin
    const minSpanX = Math.max(1e3, (dataStop - dataStart) / 20000)
    const newSpanX = clamp(spanX * factor, minSpanX, dataStop - dataStart)
    const newSpanY = clamp(spanY * factor, 5, 260)

    const fx = clamp((e.offsetX - L) / pw, 0, 1)
    const fy = clamp((e.offsetY - T) / ph, 0, 1)
    const anchorFreq = vs + fx * spanX
    const anchorVal = yMax - fy * spanY

    let a = anchorFreq - fx * newSpanX
    let b = a + newSpanX
    if (a < dataStart) {
      a = dataStart
      b = dataStart + newSpanX
    }
    if (b > dataStop) {
      b = dataStop
      a = dataStop - newSpanX
    }

    let top = anchorVal + fy * newSpanY
    let bottom = top - newSpanY
    if (bottom < -220) {
      bottom = -220
      top = bottom + newSpanY
    }
    if (top > 80) {
      top = 80
      bottom = top - newSpanY
    }

    onViewChange(a, b)
    yMin = bottom
    yMax = top
  }

  function onDown(e: MouseEvent) {
    if (e.button !== 0) return
    dragging = true
    dragX = e.offsetX
    dragY = e.offsetY
    dragViewStart = vs
    dragViewSpan = ve - vs
    dragYMin = yMin
    dragYSpan = yMax - yMin
  }

  function onMove(e: MouseEvent) {
    mouse = { x: e.offsetX, y: e.offsetY }
    if (!dragging) return

    const pw = Math.max(10, width - L - R)
    const ph = Math.max(10, height - T - B)
    const dx = e.offsetX - dragX
    const dy = e.offsetY - dragY

    let a = dragViewStart - (dx / pw) * dragViewSpan
    let b = a + dragViewSpan
    if (a < dataStart) {
      a = dataStart
      b = dataStart + dragViewSpan
    }
    if (b > dataStop) {
      b = dataStop
      a = dataStop - dragViewSpan
    }

    let y0 = dragYMin + (dy / ph) * dragYSpan
    if (y0 < -220) y0 = -220
    if (y0 + dragYSpan > 80) y0 = 80 - dragYSpan

    onViewChange(a, b)
    yMin = y0
    yMax = y0 + dragYSpan
  }

  function onUp() {
    dragging = false
  }

  function onLeave() {
    mouse = null
    dragging = false
  }

  function onDblClick() {
    yMin = -120
    yMax = -20
    onViewReset()
  }

  onMount(() => {
    resize()
    const ro = new ResizeObserver(resize)
    ro.observe(wrap)
    return () => ro.disconnect()
  })

  $: draw(width, height, vs, ve, frame, mouse, showMax, showAvg, showMin, yMin, yMax, lang)
</script>

<div class="wrap" bind:this={wrap}>
  <canvas
    bind:this={canvas}
    on:wheel={onWheel}
    on:mousedown={onDown}
    on:mousemove={onMove}
    on:mouseup={onUp}
    on:mouseleave={onLeave}
    on:dblclick={onDblClick}
  ></canvas>
</div>

<style>
  .wrap {
    width: 100%;
    height: 100%;
    overflow: hidden;
    cursor: crosshair;
  }
</style>
