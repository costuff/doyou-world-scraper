/**
 * EmotionGraph — Hierarchical Spatial Emotion Map.
 *
 * Layer 0 (Overview): 8 emotion zones shown as coloured territories.
 *   Each zone is a circle sized by track count. Click to drill in.
 *
 * Layer 1 (Zone): All tracks in the selected zone rendered as dots,
 *   positioned by their 2-D projection coordinates (proj_x, proj_y).
 *   Click a dot to see track details in the right panel.
 *
 * Navigation: breadcrumb trail at the top + Back button.
 *
 * Data sources:
 *   GET /analytics-api/api/v1/projections?limit=10000  — proj_x, proj_y, zone per track
 *   GET /analytics-api/api/v1/tracks/:id               — full track details (on click)
 */

import { useEffect, useRef, useState, useCallback } from 'react';
import axios from 'axios';

const ANALYTICS = import.meta.env.VITE_ANALYTICS_API_URL || '/analytics-api';

// ─── Types ────────────────────────────────────────────────────────────────────

interface ProjectionPoint {
  track_id: number;
  proj_x: number;   // [-1, 1] arousal axis
  proj_y: number;   // [-1, 1] valence axis
  zone: string;
}

interface TrackDetail {
  id: number;
  file_name: string;
  analysis_status: string;
  audio_features?: {
    bpm?: number;
    key?: string;
    mode?: string;
    duration_seconds?: number;
    energy?: number;
    danceability?: number;
    valence?: number;
    acousticness?: number;
    instrumentalness?: number;
    speechiness?: number;
    liveness?: number;
    melodicness?: number;
    loudness_db?: number;
    [key: string]: number | string | undefined;
  };
  emotions?: {
    euphoria?: number;
    tension?: number;
    melancholy?: number;
    aggression?: number;
    serenity?: number;
    nostalgia?: number;
    sensuality?: number;
    power?: number;
  };
}

// ─── Zone layout ─────────────────────────────────────────────────────────────

interface ZoneMeta {
  color: string;
  hex: string;
  label: string;
  nx: number;   // normalised x in [-1, 1] on the overview canvas (arousal)
  ny: number;   // normalised y in [-1, 1] on the overview canvas (valence, y-up)
}

// Fixed positions create a semantic 2D map:
//   X-axis = arousal (left=calm, right=energetic)
//   Y-axis = valence (bottom=negative, top=positive) — canvas y is flipped below
const ZONES: Record<string, ZoneMeta> = {
  euphoria:   { color: 'rgba(251,191, 36,0.85)', hex: '#fbbf24', label: 'Euphoria',   nx:  0.68, ny: -0.05 },
  power:      { color: 'rgba(249,115, 22,0.85)', hex: '#f97316', label: 'Power',      nx:  0.05, ny: -0.72 },
  aggression: { color: 'rgba(220, 38, 38,0.85)', hex: '#dc2626', label: 'Aggression', nx: -0.62, ny: -0.55 },
  tension:    { color: 'rgba(239, 68, 68,0.85)', hex: '#ef4444', label: 'Tension',    nx: -0.75, ny:  0.05 },
  melancholy: { color: 'rgba( 99,102,241,0.85)', hex: '#6366f1', label: 'Melancholy', nx: -0.60, ny:  0.58 },
  nostalgia:  { color: 'rgba(167,139,250,0.85)', hex: '#a78bfa', label: 'Nostalgia',  nx:  0.05, ny:  0.72 },
  serenity:   { color: 'rgba( 52,211,153,0.85)', hex: '#34d399', label: 'Serenity',   nx:  0.65, ny:  0.55 },
  sensuality: { color: 'rgba(244,114,182,0.85)', hex: '#f472b6', label: 'Sensuality', nx:  0.72, ny: -0.40 },
};

const ZONE_NAMES = Object.keys(ZONES);

// ─── Canvas helpers ───────────────────────────────────────────────────────────

function toCanvas(nx: number, ny: number, cx: number, cy: number, radius: number) {
  return { x: cx + nx * radius, y: cy + ny * radius };
}

function drawZoneCircle(
  ctx: CanvasRenderingContext2D,
  x: number, y: number, r: number,
  meta: ZoneMeta, label: string, count: number,
  hovered: boolean,
) {
  ctx.beginPath();
  ctx.arc(x, y, r, 0, Math.PI * 2);

  const grd = ctx.createRadialGradient(x - r * 0.2, y - r * 0.2, r * 0.1, x, y, r);
  grd.addColorStop(0, meta.color.replace('0.85', '0.95'));
  grd.addColorStop(1, meta.color.replace('0.85', '0.4'));
  ctx.fillStyle = grd;
  ctx.fill();

  ctx.strokeStyle = hovered ? '#fff' : 'rgba(255,255,255,0.25)';
  ctx.lineWidth = hovered ? 2.5 : 1;
  ctx.stroke();

  ctx.fillStyle = '#fff';
  ctx.font = `bold ${Math.max(10, Math.min(16, r * 0.28))}px Inter,sans-serif`;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillText(label, x, y - 6);

  ctx.font = `${Math.max(9, Math.min(13, r * 0.22))}px Inter,sans-serif`;
  ctx.fillStyle = 'rgba(255,255,255,0.7)';
  ctx.fillText(`${count.toLocaleString()} tracks`, x, y + 12);
}

// ─── Component ────────────────────────────────────────────────────────────────

type Layer = 'overview' | 'zone';

export default function EmotionGraph() {
  const canvasRef    = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const animRef      = useRef<number>(0);

  const [loading, setLoading]          = useState(true);
  const [error, setError]              = useState<string | null>(null);
  const [layer, setLayer]              = useState<Layer>('overview');
  const [activeZone, setActiveZone]    = useState<string | null>(null);
  const [selectedTrack, setSelected]   = useState<TrackDetail | null>(null);
  const [loadingDetail, setLoadingDet] = useState(false);

  const projectionsRef  = useRef<ProjectionPoint[]>([]);
  const hoveredZoneRef  = useRef<string | null>(null);
  const hoveredDotRef   = useRef<number | null>(null);

  // ── Fetch projections ───────────────────────────────────────────────────────
  useEffect(() => {
    let cancelled = false;
    axios.get(`${ANALYTICS}/api/v1/projections?limit=10000`).then(({ data }) => {
      if (!cancelled) { projectionsRef.current = data.projections ?? []; setLoading(false); }
    }).catch(err => {
      if (!cancelled) { setError(err instanceof Error ? err.message : 'Failed to load'); setLoading(false); }
    });
    return () => { cancelled = true; };
  }, []);

  // ── Fetch track detail on click ─────────────────────────────────────────────
  const fetchTrack = useCallback(async (id: number) => {
    setLoadingDet(true);
    try {
      const { data } = await axios.get(`${ANALYTICS}/api/v1/tracks/${id}`);
      setSelected(data);
    } catch { setSelected(null); }
    finally { setLoadingDet(false); }
  }, []);

  // ── Canvas rendering + mouse events ─────────────────────────────────────────
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || loading) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const dpr = window.devicePixelRatio || 1;

    function resize() {
      if (!canvas || !containerRef.current) return;
      const { width, height } = containerRef.current.getBoundingClientRect();
      canvas.width  = width  * dpr;
      canvas.height = height * dpr;
      canvas.style.width  = `${width}px`;
      canvas.style.height = `${height}px`;
      ctx!.scale(dpr, dpr);
    }
    resize();
    const ro = new ResizeObserver(resize);
    if (containerRef.current) ro.observe(containerRef.current);

    const projections = projectionsRef.current;
    const zoneCounts: Record<string, number> = {};
    ZONE_NAMES.forEach(z => { zoneCounts[z] = 0; });
    projections.forEach(p => { if (p.zone in zoneCounts) zoneCounts[p.zone]++; });
    const maxCount = Math.max(1, ...Object.values(zoneCounts));

    // ── Draw overview ───────────────────────────────────────────────────────────
    function drawOverview() {
      if (!ctx || !canvas) return;
      const W = canvas.width / dpr, H = canvas.height / dpr;
      const cx = W / 2, cy = H / 2;
      const mapRadius = Math.min(W, H) * 0.42;

      ctx.clearRect(0, 0, W, H);
      ctx.fillStyle = '#0f1729';
      ctx.fillRect(0, 0, W, H);

      // Axis guides
      ctx.strokeStyle = 'rgba(255,255,255,0.05)';
      ctx.lineWidth = 1;
      ctx.beginPath(); ctx.moveTo(cx - mapRadius, cy); ctx.lineTo(cx + mapRadius, cy); ctx.stroke();
      ctx.beginPath(); ctx.moveTo(cx, cy - mapRadius); ctx.lineTo(cx, cy + mapRadius); ctx.stroke();

      ctx.font = '11px Inter,sans-serif';
      ctx.fillStyle = 'rgba(255,255,255,0.18)';
      ctx.textAlign = 'left';  ctx.fillText('← Calm',       cx - mapRadius + 4,  cy - 7);
      ctx.textAlign = 'right'; ctx.fillText('Energetic →',  cx + mapRadius - 4,  cy - 7);
      ctx.textAlign = 'center';
      ctx.fillText('Positive ↑', cx, cy - mapRadius + 14);
      ctx.fillText('↓ Negative', cx, cy + mapRadius - 4);

      const zoneOrder = [...ZONE_NAMES].sort((a, b) => zoneCounts[b] - zoneCounts[a]);
      for (const zone of zoneOrder.reverse()) {
        const meta = ZONES[zone];
        const count = zoneCounts[zone];
        const minR = 46, maxR = Math.min(mapRadius * 0.38, 108);
        const r = minR + (count / maxCount) * (maxR - minR);
        const { x, y } = toCanvas(meta.nx, meta.ny, cx, cy, mapRadius);
        drawZoneCircle(ctx, x, y, r, meta, meta.label, count, hoveredZoneRef.current === zone);
      }
    }

    // ── Draw zone layer ─────────────────────────────────────────────────────────
    function drawZoneView(zone: string) {
      if (!ctx || !canvas) return;
      const W = canvas.width / dpr, H = canvas.height / dpr;
      const meta = ZONES[zone];
      if (!meta) return;

      ctx.clearRect(0, 0, W, H);
      ctx.fillStyle = '#0f1729';
      ctx.fillRect(0, 0, W, H);

      const zoneTracks = projections.filter(p => p.zone === zone);
      if (zoneTracks.length === 0) {
        ctx.fillStyle = 'rgba(255,255,255,0.35)';
        ctx.font = '15px Inter,sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText('No tracks in this zone yet', W / 2, H / 2);
        return;
      }

      let xMin = Infinity, xMax = -Infinity, yMin = Infinity, yMax = -Infinity;
      for (const p of zoneTracks) {
        if (p.proj_x < xMin) xMin = p.proj_x; if (p.proj_x > xMax) xMax = p.proj_x;
        if (p.proj_y < yMin) yMin = p.proj_y; if (p.proj_y > yMax) yMax = p.proj_y;
      }
      const padX = Math.max(0.05, (xMax - xMin) * 0.12);
      const padY = Math.max(0.05, (yMax - yMin) * 0.12);
      xMin -= padX; xMax += padX; yMin -= padY; yMax += padY;
      const xRange = xMax - xMin || 1, yRange = yMax - yMin || 1;
      const margin = 52;
      const mapW = W - margin * 2, mapH = H - margin * 2;

      const toCV = (p: ProjectionPoint) => ({
        cx: margin + ((p.proj_x - xMin) / xRange) * mapW,
        cy: margin + ((p.proj_y - yMin) / yRange) * mapH,
      });

      // Zone tone background
      ctx.fillStyle = meta.color.replace('0.85', '0.06');
      ctx.fillRect(margin, margin, mapW, mapH);

      for (const p of zoneTracks) {
        const { cx: px, cy: py } = toCV(p);
        const isHov  = hoveredDotRef.current  === p.track_id;
        const isSel  = selectedTrack?.id      === p.track_id;
        const r = isHov || isSel ? 7 : 4;

        if (isSel) {
          ctx.beginPath(); ctx.arc(px, py, r + 5, 0, Math.PI * 2);
          ctx.fillStyle = 'rgba(255,255,255,0.22)'; ctx.fill();
        }
        ctx.beginPath(); ctx.arc(px, py, r, 0, Math.PI * 2);
        ctx.fillStyle = isHov || isSel ? '#fff' : meta.hex;
        ctx.globalAlpha = isHov || isSel ? 1 : 0.65;
        ctx.fill();
        ctx.globalAlpha = 1;
      }

      // Hovered dot tooltip
      if (hoveredDotRef.current !== null) {
        const hp = zoneTracks.find(p => p.track_id === hoveredDotRef.current);
        if (hp) {
          const { cx: px, cy: py } = toCV(hp);
          const tip = `Track #${hp.track_id}`;
          ctx.font = '11px Inter,sans-serif';
          const tw = ctx.measureText(tip).width + 12;
          const tx = Math.min(W - tw - 4, Math.max(4, px - tw / 2));
          const ty = py - 28;
          ctx.fillStyle = 'rgba(0,0,0,0.8)';
          ctx.beginPath();
          ctx.roundRect?.(tx, ty, tw, 20, 4);
          ctx.fill();
          ctx.fillStyle = '#e2e8f0'; ctx.textAlign = 'left';
          ctx.fillText(tip, tx + 6, ty + 14);
        }
      }
    }

    function draw() {
      if (layer === 'overview') drawOverview();
      else if (activeZone) drawZoneView(activeZone);
      animRef.current = requestAnimationFrame(draw);
    }
    animRef.current = requestAnimationFrame(draw);

    // ── Mouse events ────────────────────────────────────────────────────────────
    function getXY(e: MouseEvent) {
      const rect = canvas!.getBoundingClientRect();
      return { mx: e.clientX - rect.left, my: e.clientY - rect.top };
    }

    function handleMouseMove(e: MouseEvent) {
      const { mx, my } = getXY(e);
      const W = canvas!.width / dpr, H = canvas!.height / dpr;

      if (layer === 'overview') {
        const cx = W / 2, cy = H / 2;
        const mapRadius = Math.min(W, H) * 0.42;
        let hit: string | null = null;
        for (const zone of ZONE_NAMES) {
          const meta = ZONES[zone];
          const { x, y } = toCanvas(meta.nx, meta.ny, cx, cy, mapRadius);
          const count = zoneCounts[zone];
          const minR = 46, maxR = Math.min(mapRadius * 0.38, 108);
          const r = minR + (count / maxCount) * (maxR - minR);
          if ((mx - x) ** 2 + (my - y) ** 2 < r * r) { hit = zone; break; }
        }
        if (hoveredZoneRef.current !== hit) {
          hoveredZoneRef.current = hit;
          canvas!.style.cursor = hit ? 'pointer' : 'default';
        }
      } else if (activeZone) {
        const zoneTracks = projections.filter(p => p.zone === activeZone);
        let xMin = Infinity, xMax = -Infinity, yMin = Infinity, yMax = -Infinity;
        zoneTracks.forEach(p => {
          if (p.proj_x < xMin) xMin = p.proj_x; if (p.proj_x > xMax) xMax = p.proj_x;
          if (p.proj_y < yMin) yMin = p.proj_y; if (p.proj_y > yMax) yMax = p.proj_y;
        });
        const padX = Math.max(0.05, (xMax - xMin) * 0.12);
        const padY = Math.max(0.05, (yMax - yMin) * 0.12);
        xMin -= padX; xMax += padX; yMin -= padY; yMax += padY;
        const xRange = xMax - xMin || 1, yRange = yMax - yMin || 1;
        const margin = 52, mapW = W - margin * 2, mapH = H - margin * 2;
        let hit: number | null = null;
        const PICK = 10;
        for (const p of zoneTracks) {
          const px = margin + ((p.proj_x - xMin) / xRange) * mapW;
          const py = margin + ((p.proj_y - yMin) / yRange) * mapH;
          if ((mx - px) ** 2 + (my - py) ** 2 < PICK * PICK) { hit = p.track_id; break; }
        }
        if (hoveredDotRef.current !== hit) {
          hoveredDotRef.current = hit;
          canvas!.style.cursor = hit ? 'pointer' : 'default';
        }
      }
    }

    function handleClick(e: MouseEvent) {
      const { mx, my } = getXY(e);
      const W = canvas!.width / dpr, H = canvas!.height / dpr;

      if (layer === 'overview') {
        const cx = W / 2, cy = H / 2;
        const mapRadius = Math.min(W, H) * 0.42;
        for (const zone of ZONE_NAMES) {
          const meta = ZONES[zone];
          const { x, y } = toCanvas(meta.nx, meta.ny, cx, cy, mapRadius);
          const count = zoneCounts[zone];
          const minR = 46, maxR = Math.min(mapRadius * 0.38, 108);
          const r = minR + (count / maxCount) * (maxR - minR);
          if ((mx - x) ** 2 + (my - y) ** 2 < r * r) {
            setActiveZone(zone); setLayer('zone'); setSelected(null); hoveredDotRef.current = null;
            return;
          }
        }
      } else if (activeZone) {
        const zoneTracks = projections.filter(p => p.zone === activeZone);
        let xMin = Infinity, xMax = -Infinity, yMin = Infinity, yMax = -Infinity;
        zoneTracks.forEach(p => {
          if (p.proj_x < xMin) xMin = p.proj_x; if (p.proj_x > xMax) xMax = p.proj_x;
          if (p.proj_y < yMin) yMin = p.proj_y; if (p.proj_y > yMax) yMax = p.proj_y;
        });
        const padX = Math.max(0.05, (xMax - xMin) * 0.12);
        const padY = Math.max(0.05, (yMax - yMin) * 0.12);
        xMin -= padX; xMax += padX; yMin -= padY; yMax += padY;
        const xRange = xMax - xMin || 1, yRange = yMax - yMin || 1;
        const margin = 52, mapW = W - margin * 2, mapH = H - margin * 2;
        const PICK = 10;
        for (const p of zoneTracks) {
          const px = margin + ((p.proj_x - xMin) / xRange) * mapW;
          const py = margin + ((p.proj_y - yMin) / yRange) * mapH;
          if ((mx - px) ** 2 + (my - py) ** 2 < PICK * PICK) { fetchTrack(p.track_id); return; }
        }
        setSelected(null);
      }
    }

    canvas.addEventListener('mousemove', handleMouseMove);
    canvas.addEventListener('click', handleClick);
    return () => {
      cancelAnimationFrame(animRef.current);
      canvas.removeEventListener('mousemove', handleMouseMove);
      canvas.removeEventListener('click', handleClick);
      ro.disconnect();
    };
  }, [loading, layer, activeZone, selectedTrack, fetchTrack]);

  const zoneMeta = activeZone ? ZONES[activeZone] : null;
  const zoneTrackCount = activeZone
    ? projectionsRef.current.filter(p => p.zone === activeZone).length
    : 0;

  return (
    <div className="flex h-full bg-[#0f1729] text-white">
      {/* Left sidebar */}
      <div className="w-52 flex-shrink-0 border-r border-slate-700 p-4 flex flex-col gap-3 overflow-y-auto">
        <p className="text-xs font-bold text-slate-400 uppercase tracking-widest">Emotion Zones</p>
        {ZONE_NAMES.map(zone => {
          const meta = ZONES[zone];
          const count = projectionsRef.current.filter(p => p.zone === zone).length;
          return (
            <button key={zone}
              onClick={() => { setActiveZone(zone); setLayer('zone'); setSelected(null); hoveredDotRef.current = null; }}
              className={`w-full text-left flex items-center gap-2 px-2 py-1.5 rounded text-sm transition ${
                activeZone === zone && layer === 'zone' ? 'bg-slate-600' : 'hover:bg-slate-700'
              }`}>
              <span className="w-3 h-3 rounded-full flex-shrink-0" style={{ background: meta.hex }} />
              <span className="flex-1">{meta.label}</span>
              <span className="text-slate-400 text-xs tabular-nums">{count}</span>
            </button>
          );
        })}
        <div className="text-xs text-slate-500 mt-auto pt-4 border-t border-slate-700">
          <p className="font-semibold text-slate-400 mb-1">How to use</p>
          <p>Click a zone to explore its tracks. Click a dot to see details.</p>
        </div>
      </div>

      {/* Canvas */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Breadcrumb bar */}
        <div className="flex items-center gap-2 px-4 py-2 border-b border-slate-700 text-sm">
          <button
            onClick={() => { setLayer('overview'); setActiveZone(null); setSelected(null); }}
            className={`transition ${layer === 'overview' ? 'text-white font-semibold' : 'text-slate-400 hover:text-white'}`}>
            All Zones
          </button>
          {activeZone && zoneMeta && (<>
            <span className="text-slate-600">/</span>
            <span className="font-semibold" style={{ color: zoneMeta.hex }}>{zoneMeta.label}</span>
            <span className="text-slate-500 text-xs ml-1">({zoneTrackCount} tracks)</span>
          </>)}
          {layer === 'zone' && (
            <button
              onClick={() => { setLayer('overview'); setActiveZone(null); setSelected(null); }}
              className="ml-auto text-xs text-slate-400 hover:text-white transition px-2 py-1 rounded bg-slate-700 hover:bg-slate-600">
              ← Back
            </button>
          )}
          {layer === 'overview' && !loading && (
            <span className="ml-auto text-xs text-slate-500">
              {projectionsRef.current.length.toLocaleString()} tracks mapped
            </span>
          )}
        </div>

        {/* Canvas container */}
        <div ref={containerRef} className="flex-1 relative">
          {loading && (
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-slate-400 text-sm animate-pulse">Loading emotion map…</span>
            </div>
          )}
          {error && (
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-red-400 text-sm">Error: {error}</span>
            </div>
          )}
          {!loading && !error && projectionsRef.current.length === 0 && (
            <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 text-center">
              <span className="text-slate-400 text-sm">No emotion projections yet.</span>
              <span className="text-slate-600 text-xs max-w-xs">
                Projections are computed automatically as tracks finish analysis. Run the worker to process pending tracks.
              </span>
            </div>
          )}
          <canvas ref={canvasRef} className="absolute inset-0" />
        </div>
      </div>

      {/* Right panel — track detail */}
      {(selectedTrack || loadingDetail) && (
        <div className="w-72 flex-shrink-0 border-l border-slate-700 p-4 overflow-y-auto">
          <div className="flex items-center justify-between mb-3">
            <p className="text-sm font-bold text-slate-300">Track Detail</p>
            <button onClick={() => setSelected(null)}
              className="text-slate-500 hover:text-white text-xl leading-none transition">×</button>
          </div>
          {loadingDetail ? (
            <p className="text-slate-400 text-sm animate-pulse">Loading…</p>
          ) : selectedTrack ? (
            <TrackDetailPanel track={selectedTrack} zoneMeta={zoneMeta} />
          ) : null}
        </div>
      )}
    </div>
  );
}

// ─── Track Detail Panel ───────────────────────────────────────────────────────

function TrackDetailPanel({ track, zoneMeta }: { track: TrackDetail; zoneMeta: ZoneMeta | null }) {
  const name = track.file_name
    .replace(/\s*\[[\w-]+\](\.[^.]+)?$/, '')
    .replace(/\.[^.]+$/, '')
    .trim() || track.file_name;

  const emotions = track.emotions ?? {};
  const features = track.audio_features ?? {};

  const emotionEntries = (Object.entries(emotions).filter(([, v]) => v != null) as [string, number][])
    .sort((a, b) => b[1] - a[1]);

  return (
    <div className="space-y-4 text-sm">
      <div>
        <p className="font-semibold text-white leading-tight break-words">{name}</p>
        {zoneMeta && (
          <span className="mt-1.5 inline-block text-xs px-2 py-0.5 rounded-full"
            style={{ background: zoneMeta.color, color: '#fff' }}>
            {zoneMeta.label}
          </span>
        )}
      </div>

      {emotionEntries.length > 0 && (
        <div>
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-widest mb-2">Emotions</p>
          {emotionEntries.map(([eName, val]) => (
            <div key={eName} className="mb-1.5">
              <div className="flex justify-between text-xs text-slate-300 mb-0.5">
                <span className="capitalize">{eName}</span>
                <span>{(val * 100).toFixed(0)}%</span>
              </div>
              <div className="h-1.5 rounded bg-slate-700 overflow-hidden">
                <div className="h-full rounded"
                  style={{ width: `${val * 100}%`, background: ZONES[eName]?.hex ?? '#60a5fa' }} />
              </div>
            </div>
          ))}
        </div>
      )}

      {Object.keys(features).length > 0 && (
        <div>
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-widest mb-2">Audio Features</p>
          <div className="space-y-1 text-xs">
            {features.bpm != null && <Row label="BPM" value={features.bpm.toFixed(1)} />}
            {features.key != null && <Row label="Key" value={`${features.key} ${features.mode ?? ''}`.trim()} />}
            {features.duration_seconds != null && (
              <Row label="Duration"
                value={`${Math.floor(features.duration_seconds / 60)}:${String(Math.round(features.duration_seconds % 60)).padStart(2, '0')}`} />
            )}
            {features.energy      != null && <Row label="Energy"      value={`${(features.energy * 100).toFixed(0)}%`} />}
            {features.danceability!= null && <Row label="Danceability" value={`${(features.danceability * 100).toFixed(0)}%`} />}
            {features.valence     != null && <Row label="Valence"     value={`${(features.valence * 100).toFixed(0)}%`} />}
            {features.loudness_db != null && <Row label="Loudness"    value={`${features.loudness_db.toFixed(1)} dB`} />}
          </div>
        </div>
      )}
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <span className="text-slate-400">{label}</span>
      <span className="text-slate-200 font-mono">{value}</span>
    </div>
  );
}

