<script lang="ts">
  import type { ChartPoint } from './types';

  // A dependency-free SVG line/area chart. Self-contained so it works under the
  // pool's strict Content-Security-Policy (no external chart library).
  let {
    points = [],
    value,
    format,
    label,
  }: {
    points: ChartPoint[];
    value: (p: ChartPoint) => number;
    format: (v: number) => string;
    label: string;
  } = $props();

  const W = 600;
  const H = 160;
  const pad = { l: 4, r: 4, t: 10, b: 4 };

  const series = $derived(
    points.map((p) => ({ x: p.x, y: value(p) })).filter((p) => Number.isFinite(p.y)),
  );

  const bounds = $derived.by(() => {
    if (series.length < 2) return null;
    const xs = series.map((p) => p.x);
    const ys = series.map((p) => p.y);
    const xMin = Math.min(...xs);
    const xMax = Math.max(...xs);
    const yMax = Math.max(...ys, 1);
    // A little floor headroom so a steady series isn't a flat line at the top.
    const yMin = Math.min(Math.min(...ys), yMax * 0.75);
    return { xMin, xMax, yMin, yMax };
  });

  function px(x: number, b: { xMin: number; xMax: number }): number {
    const span = b.xMax - b.xMin || 1;
    return pad.l + ((x - b.xMin) / span) * (W - pad.l - pad.r);
  }
  function py(y: number, b: { yMin: number; yMax: number }): number {
    const span = b.yMax - b.yMin || 1;
    return pad.t + (1 - (y - b.yMin) / span) * (H - pad.t - pad.b);
  }

  const line = $derived.by(() => {
    const b = bounds;
    if (!b) return '';
    return series.map((p, i) => `${i ? 'L' : 'M'}${px(p.x, b).toFixed(1)},${py(p.y, b).toFixed(1)}`).join(' ');
  });
  const area = $derived.by(() => {
    const b = bounds;
    if (!b) return '';
    const x0 = px(series[0].x, b).toFixed(1);
    const x1 = px(series[series.length - 1].x, b).toFixed(1);
    const base = (H - pad.b).toFixed(1);
    return `${line} L${x1},${base} L${x0},${base} Z`;
  });

  const peak = $derived(series.length ? Math.max(...series.map((p) => p.y)) : 0);
</script>

<div class="chart">
  <div class="chart-head">
    <span class="chart-label">{label}</span>
    {#if series.length >= 2}<span class="chart-peak">peak {format(peak)}</span>{/if}
  </div>
  {#if series.length >= 2}
    <svg class="chart-svg" viewBox="0 0 {W} {H}" preserveAspectRatio="none" role="img" aria-label={label}>
      <path class="chart-fill" d={area} />
      <path class="chart-line" d={line} />
    </svg>
  {:else}
    <p class="chart-empty">Not enough data yet — check back soon.</p>
  {/if}
</div>

<style>
  .chart {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 14px 16px;
    margin: 18px 0;
  }
  .chart-head {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 8px;
  }
  .chart-label {
    font-weight: 600;
    color: var(--charcoal);
  }
  .chart-peak {
    font-size: 12px;
    color: var(--muted);
  }
  .chart-svg {
    display: block;
    width: 100%;
    height: 150px;
  }
  .chart-line {
    fill: none;
    stroke: var(--etc-green);
    stroke-width: 2;
    stroke-linejoin: round;
    vector-effect: non-scaling-stroke;
  }
  .chart-fill {
    fill: var(--etc-green);
    opacity: 0.12;
    stroke: none;
  }
  .chart-empty {
    color: var(--muted);
    font-size: 13px;
    margin: 8px 0 0;
  }
</style>
