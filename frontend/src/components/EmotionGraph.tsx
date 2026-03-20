/**
 * EmotionGraph — full-screen interactive force-directed graph.
 *
 * Each node = 1 analyzed song.
 * 8 emotion poles (euphoria, tension, …) act as gravity wells arranged
 * in a circle. A song is attracted to every pole proportional to its emotion
 * value, so songs cluster in regions that reflect their emotional "fingerprint".
 * Songs between two emotions float at their weighted centroid.
 *
 * Visual encoding (audio features → appearance):
 *   Position  → emotion gravity (primary)
 *   Size      → energy
 *   Color     → valence heat-map (cold-blue=sad ↔ warm-gold=happy)
 *   Glow      → energy intensity
 *   Ring      → BPM speed (faster = brighter outer ring)
 *   Dashes    → cosine-similar songs (shown on hover)
 *
 * Future-proof by design:
 *   - Emotion poles are generated from the API response keys automatically.
 *     Add a new emotion dimension → it gets a pole.
 *   - Audio feature display in the details panel iterates all response keys.
 *   - Node visual encoding is data-driven and easy to extend.
 */

import { useEffect, useRef, useState, useCallback } from 'react';
import axios from 'axios';

const ANALYTICS = import.meta.env.VITE_ANALYTICS_API_URL || '/analytics-api';

// ─── Types ────────────────────────────────────────────────────────────────────

interface RawFeatures {
  bpm: number | null;
  key: string | null;
  mode: string | null;
  duration_seconds: number | null;
  energy: number | null;
  danceability: number | null;
  valence: number | null;
  acousticness: number | null;
  instrumentalness: number | null;
  speechiness: number | null;
  liveness: number | null;
  melodicness: number | null;
  loudness_db: number | null;
  [key: string]: number | string | null;
}

interface RawTrack {
  id: number;
  file_name: string;
  analysis_status: string;
  features: RawFeatures;
}

interface RawEmotions {
  track_id: number;
  computed_at: string;
  [key: string]: number | string;
}

interface Pole {
  name: string;
  x: number;
  y: number;
  color: string;
  rgb: [number, number, number];
}

interface GraphNode {
  id: number;
  name: string;
  shortName: string;
  features: RawFeatures;
  emotions: Record<string, number>;
  x: number; vx: number;
  y: number; vy: number;
  radius: number;
  colorRgb: [number, number, number];
  bpmNorm: number;
}

// ─── Palette (unknown future dimensions get auto-generated colours) ────────────

const KNOWN_PALETTE: Record<string, [string, [number, number, number]]> = {
  euphoria:   ['#fbbf24', [251, 191,  36]],
  tension:    ['#ef4444', [239,  68,  68]],
  melancholy: ['#3b82f6', [ 59, 130, 246]],
  aggression: ['#f97316', [249, 115,  22]],
  serenity:   ['#10b981', [ 16, 185, 129]],
  nostalgia:  ['#d97706', [217, 119,   6]],
  sensuality: ['#ec4899', [236,  72, 153]],
  power:      ['#8b5cf6', [139,  92, 246]],
};

function getPoleColor(name: string, index: number): [string, [number, number, number]] {
  if (KNOWN_PALETTE[name]) return KNOWN_PALETTE[name];
  const hue = (index * 137.5) % 360;
  const h = hue / 360, s = 0.7, l = 0.6;
  const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
  const p = 2 * l - q;
  const hue2rgb = (t: number) => {
    t = ((t % 1) + 1) % 1;
    if (t < 1/6) return p + (q - p) * 6 * t;
    if (t < 1/2) return q;
    if (t < 2/3) return p + (q - p) * (2/3 - t) * 6;
    return p;
  };
  const r = Math.round(hue2rgb(h + 1/3) * 255);
  const g = Math.round(hue2rgb(h) * 255);
  const b = Math.round(hue2rgb(h - 1/3) * 255);
  return [`rgb(${r},${g},${b})`, [r, g, b]];
}

// ─── Physics constants ────────────────────────────────────────────────────────

const POLE_ORBIT  = 310;   // world-unit radius for pole circle
const NEBULA_R    = 215;   // soft glow radius per pole
const ATTRACT     = 0.016; // spring strength toward emotion poles
const REPEL       = 2400;  // node-node repulsion coefficient
const CENTER_PULL = 0.0006; // keeps graph from drifting
const DAMPING     = 0.87;
const NODE_MIN_R  = 4;
const NODE_MAX_R  = 14;

// ─── Helpers ──────────────────────────────────────────────────────────────────

const clamp01 = (v: number) => Math.max(0, Math.min(1, v));
const lerp    = (a: number, b: number, t: number) => a + (b - a) * clamp01(t);

function valenceRgb(v: number): [number, number, number] {
  return [
    Math.round(lerp(59,  251, v)),
    Math.round(lerp(130, 191, v)),
    Math.round(lerp(246,  36, v)),
  ];
}

function drawRoundRect(
  ctx: CanvasRenderingContext2D,
  x: number, y: number, w: number, h: number, r: number,
) {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.lineTo(x + w - r, y); ctx.arcTo(x+w, y,   x+w, y+r,   r);
  ctx.lineTo(x + w, y + h - r); ctx.arcTo(x+w, y+h, x+w-r, y+h, r);
  ctx.lineTo(x + r, y + h); ctx.arcTo(x, y+h, x, y+h-r, r);
  ctx.lineTo(x, y + r); ctx.arcTo(x, y, x+r, y, r);
  ctx.closePath();
}

function cosineSim(a: Record<string, number>, b: Record<string, number>, dims: string[]): number {
  let dot = 0, magA = 0, magB = 0;
  for (const d of dims) {
    const av = a[d] ?? 0, bv = b[d] ?? 0;
    dot += av * bv; magA += av * av; magB += bv * bv;
  }
  const denom = Math.sqrt(magA * magB);
  return denom > 0 ? dot / denom : 0;
}

// ─── Build poles from emotion key list (future-proof) ─────────────────────────

function buildPoles(emotionKeys: string[]): Pole[] {
  return emotionKeys.map((name, i) => {
    const angle = (i / emotionKeys.length) * 2 * Math.PI - Math.PI / 2;
    const [color, rgb] = getPoleColor(name, i);
    return {
      name,
      x: POLE_ORBIT * Math.cos(angle),
      y: POLE_ORBIT * Math.sin(angle),
      color,
      rgb,
    };
  });
}

// ─── Build graph nodes from tracks + emotions ──────────────────────────────────

function buildNodes(
  tracks: RawTrack[],
  emotionsMap: Map<number, Record<string, number>>,
  poles: Pole[],
): GraphNode[] {
  return tracks.filter(t => emotionsMap.has(t.id)).map(t => {
    const emotions = emotionsMap.get(t.id)!;
    const f = t.features;

    // Initial position: weighted centroid of poles
    let sx = 0, sy = 0, tw = 0;
    poles.forEach(p => { const w = emotions[p.name] ?? 0; sx += p.x * w; sy += p.y * w; tw += w; });
    const ix = tw > 0 ? sx / tw : 0;
    const iy = tw > 0 ? sy / tw : 0;

    // Clean display name (strip YouTube ID + extension)
    const raw = t.file_name;
    const clean = raw.replace(/\s*\[[\w-]+\](\.[^.]+)?$/, '').replace(/\.[^.]+$/, '').trim() || raw;
    const shortName = clean.length > 36 ? clean.slice(0, 35) + '\u2026' : clean;

    const energy  = f.energy  ?? 0.5;
    const valence = f.valence ?? 0.5;
    const bpmNorm = f.bpm != null ? clamp01(f.bpm / 200) : 0.5;

    return {
      id: t.id,
      name: raw,
      shortName,
      features: f,
      emotions,
      x:  ix + (Math.random() - 0.5) * 60,
      y:  iy + (Math.random() - 0.5) * 60,
      vx: 0, vy: 0,
      radius: NODE_MIN_R + energy * (NODE_MAX_R - NODE_MIN_R),
      colorRgb: valenceRgb(valence),
      bpmNorm,
    };
  });
}

// ─── One physics step (O(n²) repulsion, fine up to ~800 nodes) ────────────────

function stepSimulation(nodes: GraphNode[], poles: Pole[]): boolean {
  let totalKE = 0;
  for (let i = 0; i < nodes.length; i++) {
    const n = nodes[i];
    let fx = 0, fy = 0;

    for (const p of poles) {
      const w = n.emotions[p.name] ?? 0;
      if (w < 0.01) continue;
      fx += (p.x - n.x) * w * ATTRACT;
      fy += (p.y - n.y) * w * ATTRACT;
    }

    for (let j = i + 1; j < nodes.length; j++) {
      const m = nodes[j];
      const dx = n.x - m.x, dy = n.y - m.y;
      const d2 = dx * dx + dy * dy + 1;
      const d  = Math.sqrt(d2);
      const minD = n.radius + m.radius + 10;
      if (d < minD * 3) {
        const f = REPEL / d2;
        const nx = dx / d, ny = dy / d;
        fx += nx * f;   fy += ny * f;
        m.vx -= nx * f; m.vy -= ny * f;
      }
    }

    fx -= n.x * CENTER_PULL;
    fy -= n.y * CENTER_PULL;

    n.vx = (n.vx + fx) * DAMPING;
    n.vy = (n.vy + fy) * DAMPING;
    n.x += n.vx;
    n.y += n.vy;
    totalKE += Math.abs(n.vx) + Math.abs(n.vy);
  }
  return totalKE / Math.max(1, nodes.length) > 0.08;
}

// ─── Component ────────────────────────────────────────────────────────────────

export default function EmotionGraph() {
  const canvasRef    = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const [loading, setLoading]   = useState(true);
  const [error, setError]       = useState<string | null>(null);
  const [nodeCount, setCount]   = useState(0);
  const [selected, setSelected] = useState<GraphNode | null>(null);
  const [filterEmotion, setFE]  = useState<string | null>(null);
  const [threshold, setThresh]  = useState(0.5);
  const [settled, setSettled]   = useState(false);

  // Mutable simulation state (updated every frame — not React state)
  const nodesRef    = useRef<GraphNode[]>([]);
  const polesRef    = useRef<Pole[]>([]);
  const tf          = useRef({ x: 0, y: 0, scale: 1 });
  const hoveredRef  = useRef<GraphNode | null>(null);
  const selectedRef = useRef<GraphNode | null>(null);
  const filterRef   = useRef<{ emotion: string | null; threshold: number }>({ emotion: null, threshold: 0.5 });
  const dimsRef     = useRef<string[]>([]);
  const simActive   = useRef(true);
  const animRef     = useRef(0);
  const drag        = useRef({ on: false, sx: 0, sy: 0, ox: 0, oy: 0 });

  // Keep refs in sync with React state
  useEffect(() => { filterRef.current = { emotion: filterEmotion, threshold }; }, [filterEmotion, threshold]);
  useEffect(() => { selectedRef.current = selected; }, [selected]);

  // ── Fetch data ─────────────────────────────────────────────────────────────
  useEffect(() => {
    let dead = false;
    (async () => {
      try {
        const [trRes, emRes] = await Promise.all([
          axios.get(`${ANALYTICS}/api/v1/tracks`, { params: { status: 'completed', limit: 2000 }, timeout: 15000 }),
          axios.get(`${ANALYTICS}/api/v1/emotions`, { params: { limit: 2000 }, timeout: 15000 }),
        ]);
        if (dead) return;

        const tracks: RawTrack[]      = trRes.data?.tracks   ?? [];
        const rawEm:  RawEmotions[]   = emRes.data?.emotions ?? [];

        const emotionsMap = new Map<number, Record<string, number>>();
        const dims = new Set<string>();
        rawEm.forEach(e => {
          const m: Record<string, number> = {};
          Object.entries(e).forEach(([k, v]) => {
            if (k !== 'track_id' && k !== 'computed_at' && typeof v === 'number') {
              m[k] = v; dims.add(k);
            }
          });
          emotionsMap.set(e.track_id as number, m);
        });

        const keys = dims.size > 0 ? [...dims] : Object.keys(KNOWN_PALETTE);
        dimsRef.current = keys;

        const poles = buildPoles(keys);
        const nodes = buildNodes(tracks, emotionsMap, poles);
        polesRef.current = poles;
        nodesRef.current = nodes;
        simActive.current = true;

        const cont = containerRef.current;
        if (cont) tf.current = { x: cont.clientWidth / 2, y: cont.clientHeight / 2, scale: 1 };

        setCount(nodes.length);
        setError(null);
      } catch (e: any) {
        if (!dead) setError(e?.message ?? 'Failed to load graph data');
      } finally {
        if (!dead) setLoading(false);
      }
    })();
    return () => { dead = true; };
  }, []);

  // ── Size canvas to container ───────────────────────────────────────────────
  useEffect(() => {
    const canvas = canvasRef.current, cont = containerRef.current;
    if (!canvas || !cont) return;
    const ro = new ResizeObserver(() => { canvas.width = cont.clientWidth; canvas.height = cont.clientHeight; });
    ro.observe(cont);
    canvas.width = cont.clientWidth; canvas.height = cont.clientHeight;
    return () => ro.disconnect();
  }, []);

  // ── Render loop ────────────────────────────────────────────────────────────
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const frame = () => {
      const ctx = canvas.getContext('2d');
      if (!ctx) { animRef.current = requestAnimationFrame(frame); return; }

      const W = canvas.width, H = canvas.height;
      const t = tf.current;
      const nodes = nodesRef.current;
      const poles = polesRef.current;

      if (simActive.current && nodes.length > 0) {
        if (!stepSimulation(nodes, poles)) {
          simActive.current = false;
          setSettled(true);
        }
      }

      // Background
      ctx.fillStyle = '#080e1a';
      ctx.fillRect(0, 0, W, H);
      const bg = ctx.createRadialGradient(W/2, H/2, 0, W/2, H/2, Math.min(W, H) * 0.75);
      bg.addColorStop(0, 'rgba(25,35,65,0.6)');
      bg.addColorStop(1, 'rgba(8,14,26,0)');
      ctx.fillStyle = bg;
      ctx.fillRect(0, 0, W, H);

      ctx.save();
      ctx.translate(t.x, t.y);
      ctx.scale(t.scale, t.scale);

      // Emotion nebulae (soft zone glow)
      for (const p of poles) {
        const [r, g, b] = p.rgb;
        const grd = ctx.createRadialGradient(p.x, p.y, 0, p.x, p.y, NEBULA_R);
        grd.addColorStop(0,   `rgba(${r},${g},${b},0.15)`);
        grd.addColorStop(0.45,`rgba(${r},${g},${b},0.06)`);
        grd.addColorStop(1,   `rgba(${r},${g},${b},0)`);
        ctx.fillStyle = grd;
        ctx.beginPath();
        ctx.arc(p.x, p.y, NEBULA_R, 0, Math.PI * 2);
        ctx.fill();
      }

      const hov = hoveredRef.current;
      const sel = selectedRef.current;
      const flt = filterRef.current;
      const dims = dimsRef.current;

      // Similarity threads on hover (cosine similarity dashed lines)
      if (hov && nodes.length > 1) {
        const sims = nodes
          .filter(n => n !== hov)
          .map(n => ({ n, s: cosineSim(hov.emotions, n.emotions, dims) }))
          .sort((a, b) => b.s - a.s)
          .slice(0, 6);
        ctx.setLineDash([4, 6]);
        ctx.lineWidth = 1;
        for (const { n, s } of sims) {
          ctx.strokeStyle = `rgba(148,163,184,${(s * 0.65).toFixed(2)})`;
          ctx.beginPath();
          ctx.moveTo(hov.x, hov.y);
          ctx.lineTo(n.x, n.y);
          ctx.stroke();
        }
        ctx.setLineDash([]);
      }

      // Song nodes
      for (const node of nodes) {
        const isHov = node === hov;
        const isSel = node === sel;
        const emVal  = flt.emotion ? (node.emotions[flt.emotion] ?? 0) : 1;
        const dimmed = flt.emotion !== null && emVal < flt.threshold;

        const [r, g, b] = node.colorRgb;
        const radius = node.radius + (isHov || isSel ? 3 : 0);
        const glow   = (node.features.energy ?? 0.5) * 20;

        ctx.save();
        ctx.globalAlpha = dimmed ? 0.1 : 1;

        // BPM speed ring
        if (!dimmed && node.bpmNorm > 0.38) {
          ctx.shadowColor  = `rgb(${r},${g},${b})`;
          ctx.shadowBlur   = 3;
          ctx.strokeStyle  = `rgba(${r},${g},${b},${(node.bpmNorm * 0.7).toFixed(2)})`;
          ctx.lineWidth    = 1.5;
          ctx.beginPath();
          ctx.arc(node.x, node.y, radius + 5, 0, Math.PI * 2);
          ctx.stroke();
        }

        // Glow + fill
        ctx.shadowColor   = `rgb(${r},${g},${b})`;
        ctx.shadowBlur    = dimmed ? 0 : (isHov || isSel ? glow * 2.2 : glow);
        ctx.fillStyle     = `rgb(${r},${g},${b})`;
        ctx.beginPath();
        ctx.arc(node.x, node.y, radius, 0, Math.PI * 2);
        ctx.fill();

        // Selected border
        if (isSel) {
          ctx.shadowBlur  = 0;
          ctx.strokeStyle = '#ffffff';
          ctx.lineWidth   = 2;
          ctx.stroke();
        }

        // Inner specular highlight
        if (!dimmed) {
          ctx.shadowBlur  = 0;
          ctx.fillStyle   = 'rgba(255,255,255,0.22)';
          ctx.beginPath();
          ctx.arc(node.x - radius * 0.28, node.y - radius * 0.3, radius * 0.4, 0, Math.PI * 2);
          ctx.fill();
        }
        ctx.restore();
      }

      // Emotion poles (always on top)
      for (const p of poles) {
        const [r, g, b] = p.rgb;
        ctx.save();
        // Outer pulse ring
        ctx.strokeStyle = `rgba(${r},${g},${b},0.3)`;
        ctx.lineWidth   = 1;
        ctx.beginPath();
        ctx.arc(p.x, p.y, 30, 0, Math.PI * 2);
        ctx.stroke();
        // Main disc
        ctx.shadowColor = p.color;
        ctx.shadowBlur  = 35;
        ctx.fillStyle   = p.color;
        ctx.beginPath();
        ctx.arc(p.x, p.y, 20, 0, Math.PI * 2);
        ctx.fill();
        // White inner dot
        ctx.shadowBlur  = 0;
        ctx.fillStyle   = 'rgba(255,255,255,0.8)';
        ctx.beginPath();
        ctx.arc(p.x, p.y, 5, 0, Math.PI * 2);
        ctx.fill();
        ctx.restore();
        // Label (not affected by transform alpha)
        ctx.font        = 'bold 11px system-ui, sans-serif';
        ctx.textAlign   = 'center';
        ctx.fillStyle   = '#e2e8f0';
        ctx.shadowBlur  = 0;
        ctx.fillText(p.name.charAt(0).toUpperCase() + p.name.slice(1), p.x, p.y + 40);
      }

      // Hover tooltip
      if (hov) {
        ctx.font = '12px system-ui, sans-serif';
        const tw = ctx.measureText(hov.shortName).width;
        const pad = 8, bh = 22;
        const bw = tw + pad * 2;
        const bx = hov.x - bw / 2;
        const by = hov.y - hov.radius - bh - 8;
        ctx.fillStyle = 'rgba(8,14,26,0.9)';
        drawRoundRect(ctx, bx, by, bw, bh, 5);
        ctx.fill();
        ctx.fillStyle = '#f1f5f9';
        ctx.textAlign = 'center';
        ctx.fillText(hov.shortName, hov.x, by + 15);
      }

      ctx.restore();
      animRef.current = requestAnimationFrame(frame);
    };

    animRef.current = requestAnimationFrame(frame);
    return () => cancelAnimationFrame(animRef.current);
  }, []);

  // ── Non-passive wheel listener ─────────────────────────────────────────────
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const rect = canvas.getBoundingClientRect();
      const mx = e.clientX - rect.left, my = e.clientY - rect.top;
      const factor = e.deltaY < 0 ? 1.12 : 0.9;
      const t = tf.current;
      const ns = Math.max(0.06, Math.min(14, t.scale * factor));
      tf.current = { scale: ns, x: mx - (mx - t.x) * (ns / t.scale), y: my - (my - t.y) * (ns / t.scale) };
    };
    canvas.addEventListener('wheel', onWheel, { passive: false });
    return () => canvas.removeEventListener('wheel', onWheel);
  }, []);

  // ── Interaction helpers ────────────────────────────────────────────────────
  const toWorld = useCallback((cx: number, cy: number) => {
    const r = canvasRef.current!.getBoundingClientRect(), t = tf.current;
    return { x: (cx - r.left - t.x) / t.scale, y: (cy - r.top - t.y) / t.scale };
  }, []);

  const hitTest = useCallback((wx: number, wy: number): GraphNode | null => {
    for (const n of nodesRef.current) {
      const dx = n.x - wx, dy = n.y - wy;
      if (dx*dx + dy*dy <= (n.radius + 6) ** 2) return n;
    }
    return null;
  }, []);

  const onMouseMove = useCallback((e: React.MouseEvent<HTMLCanvasElement>) => {
    if (drag.current.on) {
      const d = drag.current;
      tf.current = { ...tf.current, x: d.ox + e.clientX - d.sx, y: d.oy + e.clientY - d.sy };
      return;
    }
    const w = toWorld(e.clientX, e.clientY);
    hoveredRef.current = hitTest(w.x, w.y);
    canvasRef.current!.style.cursor = hoveredRef.current ? 'pointer' : 'grab';
  }, [toWorld, hitTest]);

  const onMouseDown = useCallback((e: React.MouseEvent<HTMLCanvasElement>) => {
    drag.current = { on: true, sx: e.clientX, sy: e.clientY, ox: tf.current.x, oy: tf.current.y };
    canvasRef.current!.style.cursor = 'grabbing';
  }, []);

  const onMouseUp = useCallback((e: React.MouseEvent<HTMLCanvasElement>) => {
    const d = drag.current;
    const moved = Math.abs(e.clientX - d.sx) > 4 || Math.abs(e.clientY - d.sy) > 4;
    drag.current.on = false;
    canvasRef.current!.style.cursor = 'grab';
    if (!moved) {
      const w = toWorld(e.clientX, e.clientY);
      const node = hitTest(w.x, w.y);
      setSelected(prev => prev?.id === node?.id ? null : node ?? null);
    }
  }, [toWorld, hitTest]);

  const onMouseLeave = useCallback(() => { drag.current.on = false; hoveredRef.current = null; }, []);

  const resetView = useCallback(() => {
    const c = containerRef.current;
    if (c) tf.current = { x: c.clientWidth / 2, y: c.clientHeight / 2, scale: 1 };
  }, []);

  const rerunSim = useCallback(() => {
    simActive.current = true;
    setSettled(false);
    nodesRef.current.forEach(n => { n.vx = (Math.random() - 0.5) * 5; n.vy = (Math.random() - 0.5) * 5; });
  }, []);

  const poles = polesRef.current;

  // ── UI ─────────────────────────────────────────────────────────────────────

  if (loading) return (
    <div className="flex items-center justify-center h-full bg-[#080e1a]">
      <div className="text-center space-y-3">
        <div className="w-10 h-10 border-2 border-violet-500 border-t-transparent rounded-full animate-spin mx-auto" />
        <p className="text-slate-400">Mapping emotion space…</p>
        <p className="text-slate-600 text-sm">Fetching tracks &amp; computing positions</p>
      </div>
    </div>
  );

  if (error) return (
    <div className="flex items-center justify-center h-full bg-[#080e1a]">
      <div className="text-center">
        <p className="text-red-400 text-lg mb-2">Graph unavailable</p>
        <p className="text-slate-500 text-sm">{error}</p>
        <p className="text-slate-600 text-xs mt-2">Is the analytics API running on port 8080?</p>
      </div>
    </div>
  );

  return (
    <div className="flex h-full bg-[#080e1a] overflow-hidden">

      {/* ── Left sidebar: emotion filter ───────────────────────────────── */}
      <div className="w-52 bg-[#0d1424] border-r border-slate-800 flex flex-col flex-shrink-0 overflow-y-auto">
        <div className="px-4 py-3 border-b border-slate-800">
          <h3 className="text-slate-400 font-bold text-xs uppercase tracking-widest">Emotion Filter</h3>
        </div>

        <div className="p-2 space-y-0.5 flex-1">
          <button
            onClick={() => setFE(null)}
            className={`w-full text-left px-3 py-1.5 rounded text-xs transition ${
              filterEmotion === null
                ? 'bg-slate-700 text-white'
                : 'text-slate-500 hover:text-white hover:bg-slate-800'
            }`}
          >
            All ({nodeCount})
          </button>
          {poles.map(p => (
            <button
              key={p.name}
              onClick={() => setFE(filterEmotion === p.name ? null : p.name)}
              className={`w-full text-left px-3 py-1.5 rounded text-xs transition flex items-center gap-2 ${
                filterEmotion === p.name
                  ? 'bg-slate-700 text-white'
                  : 'text-slate-500 hover:text-white hover:bg-slate-800'
              }`}
            >
              <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ backgroundColor: p.color }} />
              <span className="capitalize">{p.name}</span>
            </button>
          ))}
        </div>

        {filterEmotion && (
          <div className="p-3 border-t border-slate-800 space-y-2">
            <div className="flex justify-between text-xs">
              <span className="text-slate-500 capitalize">{filterEmotion}</span>
              <span className="text-white font-mono">{Math.round(threshold * 100)}%</span>
            </div>
            <input type="range" min={0} max={100} value={Math.round(threshold * 100)}
              onChange={e => setThresh(Number(e.target.value) / 100)}
              className="w-full accent-violet-500"
            />
            <p className="text-slate-700 text-xs">Dim below threshold</p>
          </div>
        )}

        <div className="p-2 border-t border-slate-800 space-y-0.5">
          <button onClick={resetView}
            className="w-full text-left px-3 py-1.5 rounded text-xs text-slate-500 hover:text-white hover:bg-slate-800 transition">
            ⊙&nbsp; Reset view
          </button>
          <button onClick={rerunSim}
            className="w-full text-left px-3 py-1.5 rounded text-xs text-slate-500 hover:text-white hover:bg-slate-800 transition">
            ↺&nbsp; Re-run physics
          </button>
        </div>

        <div className="p-3 border-t border-slate-800">
          <p className="text-slate-700 text-xs font-semibold mb-1 uppercase tracking-wider">Legend</p>
          <div className="text-slate-700 text-xs space-y-0.5">
            <div>Position → emotion mix</div>
            <div>Size → energy</div>
            <div>Colour → valence</div>
            <div>Ring → BPM speed</div>
            <div>Glow → intensity</div>
            <div className="text-slate-800">─── hover ───</div>
            <div>Dashes → similar songs</div>
          </div>
        </div>
      </div>

      {/* ── Canvas area ────────────────────────────────────────────────── */}
      <div ref={containerRef} className="flex-1 relative overflow-hidden">
        <canvas
          ref={canvasRef}
          className="absolute inset-0"
          style={{ cursor: 'grab' }}
          onMouseMove={onMouseMove}
          onMouseDown={onMouseDown}
          onMouseUp={onMouseUp}
          onMouseLeave={onMouseLeave}
        />
        {!settled && nodeCount > 0 && (
          <div className="absolute top-3 right-4 bg-slate-900/70 text-slate-500 text-xs px-2 py-1 rounded-full backdrop-blur">
            Settling…
          </div>
        )}
        <div className="absolute bottom-3 right-4 text-slate-700 text-xs pointer-events-none select-none">
          Scroll = zoom&nbsp;·&nbsp;Drag = pan&nbsp;·&nbsp;Click = details
        </div>
      </div>

      {/* ── Right sidebar: selected node details ───────────────────────── */}
      {selected && (
        <div className="w-68 min-w-64 bg-[#0d1424] border-l border-slate-800 flex flex-col flex-shrink-0 overflow-y-auto">
          <div className="px-4 py-3 border-b border-slate-800 flex justify-between items-start gap-2">
            <p className="text-white text-sm font-semibold leading-snug">{selected.shortName}</p>
            <button onClick={() => setSelected(null)}
              className="text-slate-600 hover:text-white text-xl leading-none flex-shrink-0">×</button>
          </div>

          {/* Emotion breakdown */}
          <div className="px-4 py-3 border-b border-slate-800">
            <p className="text-slate-600 text-xs uppercase tracking-widest mb-3">Emotions</p>
            <div className="space-y-2">
              {poles.map(p => {
                const v = selected.emotions[p.name] ?? 0;
                return (
                  <div key={p.name}>
                    <div className="flex justify-between text-xs mb-0.5">
                      <span className="text-slate-400 capitalize">{p.name}</span>
                      <span className="text-slate-300 font-mono">{(v * 100).toFixed(0)}%</span>
                    </div>
                    <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
                      <div className="h-full rounded-full transition-all"
                        style={{ width: `${v * 100}%`, backgroundColor: p.color }} />
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Audio features — all keys from API response (auto-adapts to new features) */}
          <div className="px-4 py-3">
            <p className="text-slate-600 text-xs uppercase tracking-widest mb-3">Audio Features</p>
            <div className="space-y-1.5">
              {Object.entries(selected.features)
                .filter(([, v]) => v !== null && v !== undefined)
                .map(([k, v]) => {
                  const label = k.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
                  let display: string;
                  if (k === 'bpm') display = `${Number(v).toFixed(1)} BPM`;
                  else if (k === 'duration_seconds') {
                    const s = Number(v);
                    display = `${Math.floor(s/60)}:${String(Math.floor(s) % 60).padStart(2, '0')}`;
                  } else if (k === 'loudness_db') display = `${Number(v).toFixed(1)} dB`;
                  else if (typeof v === 'number' && v >= 0 && v <= 1)
                    display = `${(v * 100).toFixed(0)}%`;
                  else display = String(v);
                  return (
                    <div key={k} className="flex justify-between text-xs">
                      <span className="text-slate-600">{label}</span>
                      <span className="text-slate-300 font-mono">{display}</span>
                    </div>
                  );
                })}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
