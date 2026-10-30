<script lang="ts">
  import { onMount } from 'svelte'

  export let minDBM = -120
  export let maxDBM = -20
  export let start = 88e6
  export let stop = 108e6
  export let viewStart: number | null = null
  export let viewStop: number | null = null

  let wrap: HTMLDivElement
  let canvas: HTMLCanvasElement
  let width = 0
  let height = 0
  let lastSeq = -1
  let ready = false
  let lo = minDBM
  let hi = maxDBM
  let dataStart = 0
  let dataStop = 0

  function color(v: number): [number, number, number] {
    const t = Math.max(0, Math.min(1, (v - lo) / (hi - lo)))
    const stops: Array<[number, number, number, number]> = [
      [0.0, 8, 8, 24],
      [0.25, 40, 60, 120],
      [0.5, 30, 130, 130],
      [0.75, 120, 190, 90],
      [1.0, 250, 220, 60],
    ]
    for (let i = 1; i < stops.length; i++) {
      if (t <= stops[i][0]) {
        const [t0, r0, g0, b0] = stops[i - 1]
        const [t1, r1, g1, b1] = stops[i]
        const k = (t - t0) / (t1 - t0)
        return [
          Math.round(r0 + (r1 - r0) * k),
          Math.round(g0 + (g1 - g0) * k),
          Math.round(b0 + (b1 - b0) * k),
        ]
      }
    }
    return [250, 220, 60]
  }

  function clear(ctx: CanvasRenderingContext2D) {
    ctx.fillStyle = '#0b0d12'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
  }

  export function push(f: { seq: number; start?: number; stop?: number; dbm: number[] }) {
    if (!canvas || width <= 0 || height <= 0) return
    if (f.seq === lastSeq) return
    lastSeq = f.seq
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // Rows scanned at different ranges cannot share a waterfall: clear it so
    // the image never mixes two frequency scales.
    if (f.start !== undefined && f.stop !== undefined && (f.start !== dataStart || f.stop !== dataStop)) {
      dataStart = f.start
      dataStop = f.stop
      ready = false
    }

    // Auto colour range follows the signal with a slow EMA, so weak and strong
    // bands both stay readable instead of turning uniformly green.
    let mn = Infinity
    let mx = -Infinity
    for (const v of f.dbm) {
      if (v < mn) mn = v
      if (v > mx) mx = v
    }
    if (isFinite(mn) && isFinite(mx)) {
      const tlo = Math.max(-190, mn - 4)
      const thi = Math.min(30, mx + 6)
      lo = lo * 0.9 + tlo * 0.1
      hi = hi * 0.9 + thi * 0.1
      if (hi < lo + 10) hi = lo + 10
    }

    if (!ready) {
      canvas.width = Math.max(1, width)
      canvas.height = Math.max(1, height)
      clear(ctx)
      ready = true
    }

    // scroll down by one row, newest on top
    ctx.drawImage(canvas, 0, 1)

    const img = ctx.createImageData(canvas.width, 1)
    const n = f.dbm.length
    const rowStart = f.start ?? (dataStart || start)
    const rowStop = f.stop ?? (dataStop || stop)
    const vs = viewStart ?? rowStart
    const ve = viewStop ?? rowStop
    const span = rowStop - rowStart
    for (let x = 0; x < canvas.width; x++) {
      const freq = vs + ((ve - vs) * x) / Math.max(1, canvas.width - 1)
      let idx = 0
      if (span > 0) idx = Math.round(((freq - rowStart) / span) * (n - 1))
      idx = Math.max(0, Math.min(n - 1, idx))
      const [r, g, b] = color(f.dbm[idx])
      const o = x * 4
      img.data[o] = r
      img.data[o + 1] = g
      img.data[o + 2] = b
      img.data[o + 3] = 255
    }
    ctx.putImageData(img, 0, 0)
  }

  function resize() {
    if (!wrap) return
    width = wrap.clientWidth
    height = wrap.clientHeight
    ready = false
    if (canvas) {
      canvas.width = Math.max(1, width)
      canvas.height = Math.max(1, height)
      canvas.style.width = width + 'px'
      canvas.style.height = height + 'px'
      const ctx = canvas.getContext('2d')
      if (ctx) {
        ctx.fillStyle = '#0b0d12'
        ctx.fillRect(0, 0, canvas.width, canvas.height)
      }
    }
  }

  onMount(() => {
    resize()
    const ro = new ResizeObserver(resize)
    ro.observe(wrap)
    return () => ro.disconnect()
  })
</script>

<div class="wrap" bind:this={wrap}>
  <canvas bind:this={canvas}></canvas>
</div>

<style>
  .wrap {
    width: 100%;
    height: 100%;
    overflow: hidden;
  }
</style>
