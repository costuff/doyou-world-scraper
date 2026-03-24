/**
 * EmotionGraph — interactive force-directed emotion map.
 *
 * Each node = 1 analysed song. 8 emotion poles act as gravity wells.
 * Songs cluster near the poles they resemble most.
 *
 * Visual layers (back → front):
 *   1. KNN terrain  — smooth inverse-distance-weighted colour field showing
 *      which emotion "owns" each area (Voronoi-style soft boundaries).
 *   2. Nebula glows — radial halos around each pole.
 *   3. Persistent edges — threads to each song's top-3 cosine-similar neighbours.
 *   4. Hover edges  — bright dashed lines to the 6 most similar songs on hover.
 *   5. Song nodes   — coloured by valence, sized by energy, ringed by BPM.
 *   6. Emotion poles — glowing discs at the boundary of the space.
 *
 * Data sources:
 *   GET /analytics-api/api/v1/tracks   — all analysed songs + audio features
 *   GET /analytics-api/api/v1/emotions — per-track emotion scores
 */

import { useEffect, useRef, useState, useCallback } from 'react';
import axios from 'axios';
import {
  NEBULA_R, ATTRACT, REPEL, CENTER_PULL, DAMPING,
  NODE_MIN_R, NODE_MAX_R, KNOWN_PALETTE,
  clamp01, valenceRgb, cosineSim, buildPoles,
  type Pole,
} from './emotionGraphUtils';

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

interface GraphNode {
  id: number;
  name: string;
  shortName: string;
  features: RawFeatures;
  emotions: Record<string, number>;
  x: number; vx: number;
  y: number; vy: number;
  rx: number; ry: number;      // rest position for float mode
  floatPhase: number;          // unique oscillation phase per node
  radius: number;
  colorRgb: [number, number, number];
  bpmNorm: number;
}

interface Edge {
  a: GraphNode;
  b: GraphNode;
  weight: number;
}

// ─── drawRoundRect ────────────────────────────────────────────────────────────

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

// ─── Build graph nodes ────────────────────────────────────────────────────────

function buildNodes(
  tracks: RawTrack[],
  emotionsMap: Map<number, Record<string, number>>,
  poles: Pole[],
): GraphNode[] {
  return tracks.filter(t => emotionsMap.has(t.id) && t.features != null).map(t => {
    const emotions = emotionsMap.get(t.id)!;
    const f = t.features;

    let sx = 0, sy = 0, tw = 0;
    poles.forEach(p => { const w = emotions[p.name] ?? 0; sx += p.x * w; sy += p.y * w; tw += w; });
    const ix = tw > 0 ? sx / tw : 0;
    const iy = tw > 0 ? sy / tw : 0;

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
      x:  ix + (Math.random() - 0.5) * 8,
      y:  iy + (Math.random() - 0.5) * 8,
      vx: 0, vy: 0,
      rx: 0, ry: 0,
      floatPhase: (t.id * 2.399) % (Math.PI * 2),  // golden-angle spread
      radius: NODE_MIN_R + energy * (NODE_MAX_R - NODE_MIN_R),
      colorRgb: valenceRgb(valence),
      bpmNorm,
    };
  });
}

// ─── Physics step ─────────────────────────────────────────────────────────────

function stepSimulation(nodes: GraphNode[], poles: Pole[]): boolean {
  let totalKE = 0;
  const damp = (nodes as any)._dampingOverride ?? DAMPING;
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

    n.vx = (n.vx + fx) * damp;
    n.vy = (n.vy + fy) * damp;
    n.x += n.vx;
    n.y += n.vy;
    totalKE += Math.abs(n.vx) + Math.abs(n.vy);
  }
  return totalKE / Math.max(1, nodes.length) > 0.03;
}

// ─── Persistent similarity edges ──────────────────────────────────────────────

const EDGE_TOP_K = 3;

function buildEdges(nodes: GraphNode[], dims: string[]): Edge[] {
  const edges: Edge[] = [];
  const seen = new Set<string>();
  for (const node of nodes) {
    const sims = nodes
      .filter(n => n !== node)
      .map(n => ({ n, s: cosineSim(node.emotions, n.emotions, dims) }))
      .sort((a, b) => b.s - a.s)
      .slice(0, EDGE_TOP_K);
    for (const { n, s } of sims) {
      const key = [Math.min(node.id, n.id), Math.max(node.id, n.id)].join('-');
      if (!seen.has(key)) { seen.add(key); edges.push({ a: node, b: n, weight: s }); }
    }
  }
  return edges;
}

// ─── KNN terrain canvas ───────────────────────────────────────────────────────
// Pre-renders a soft colour blend via inverse-distance weighting across poles.
// Each pixel = weighted mean of pole colours, weight = 1 / d^2.2
// Gives a smooth Voronoi-like "emotion territory" background.

const TERRAIN_PIXEL = 256;
const TERRAIN_WORLD = 700;  // half-extent; poles sit at r≈310

function buildTerrain(poles: Pole[]): HTMLCanvasElement {
  const offscreen = document.createElement('canvas');
  offscreen.width = offscreen.height = TERRAIN_PIXEL;
  const ctx = offscreen.getContext('2d')!;
  const imgData = ctx.createImageData(TERRAIN_PIXEL, TERRAIN_PIXEL);
  const data = imgData.data;
  for (let py = 0; py < TERRAIN_PIXEL; py++) {
    for (let px = 0; px < TERRAIN_PIXEL; px++) {
      const wx = ((px / TERRAIN_PIXEL) - 0.5) * TERRAIN_WORLD * 2;
      const wy = ((py / TERRAIN_PIXEL) - 0.5) * TERRAIN_WORLD * 2;
      let totalW = 0, r = 0, g = 0, b = 0;
      for (const p of poles) {
        const dx = wx - p.x, dy = wy - p.y;
        const w  = 1 / (Math.pow(Math.sqrt(dx*dx + dy*dy), 2.2) + 15);
        totalW += w; r += p.rgb[0] * w; g += p.rgb[1] * w; b += p.rgb[2] * w;
      }
      const idx = (py * TERRAIN_PIXEL + px) * 4;
      data[idx] = r / totalW; data[idx+1] = g / totalW;
      data[idx+2] = b / totalW; data[idx+3] = 30;
    }
  }
  ctx.putImageData(imgData, 0, 0);
  return offscreen;
}

// ─── Component ────────────────────────────────────────────────────────────────

export default function EmotionGraph() {
  const canvasRef    = useRef<HTMLCanvasElement | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const [loading, setLoading]     = useState(true);
  const [error, setError]         = useState<string | null>(null);
  const [nodeCount, setCount]     = useState(0);
  const [selected, setSelected]   = useState<GraphNode | null>(null);
  const [selectedLabel, setSelectedLabel] = useState<{ label_path: string[]; auto_label_path: string[] } | null>(null);
  const [filterEmotion, setFE]    = useState<string | null>(null);
  const [threshold, setThresh]    = useState(0.5);
  const [settled, setSettled]     = useState(false);
  const [showEdges, setShowEdges] = useState(true);

  const nodesRef    = useRef<GraphNode[]>([]);
  const polesRef    = useRef<Pole[]>([]);
  const edgesRef    = useRef<Edge[]>([]);
  const terrainRef  = useRef<HTMLCanvasElement | null>(null);
  const tf          = useRef({ x: 0, y: 0, scale: 1 });
  const hoveredRef  = useRef<GraphNode | null>(null);
  const selectedRef = useRef<GraphNode | null>(null);
  const filterRef   = useRef<{ emotion: string | null; threshold: number }>({ emotion: null, threshold: 0.5 });
  const dimsRef     = useRef<string[]>([]);
  const showEdgesRef= useRef(true);
  const simActive   = useRef(false);
  const floatActive = useRef(false);
  const revealStartRef = useRef(0);
  const animRef     = useRef(0);
  const drag        = useRef({ on: false, sx: 0, sy: 0, ox: 0, oy: 0 });

  useEffect(() => { filterRef.current   = { emotion: filterEmotion, threshold }; }, [filterEmotion, threshold]);
  useEffect(() => { selectedRef.current = selected; }, [selected]);
  useEffect(() => { showEdgesRef.current = showEdges; }, [showEdges]);

  // Fetch label whenever a node is selected
  useEffect(() => {
    if (!selected) { setSelectedLabel(null); return; }
    axios.get(`${ANALYTICS}/api/v1/tracks/${selected.id}/label`)
      .then(r => setSelectedLabel(r.data))
      .catch(() => setSelectedLabel(null));
  }, [selected?.id]);

  // ── Fetch data ─────────────────────────────────────────────────────────────
  useEffect(() => {
    let dead = false;
    (async () => {
      try {
        const [trRes, emRes] = await Promise.all([
          axios.get(`${ANALYTICS}/api/v1/tracks`,   { params: { status: 'completed', limit: 5000 }, timeout: 20000 }),
          axios.get(`${ANALYTICS}/api/v1/emotions`,  { params: { limit: 5000 }, timeout: 20000 }),
        ]);
        if (dead) return;
        const tracks: RawTrack[]   = trRes.data?.tracks   ?? [];
        const rawEm: RawEmotions[] = emRes.data?.emotions ?? [];
        const emotionsMap = new Map<number, Record<string, number>>();
        const dims = new Set<string>();
        rawEm.forEach(e => {
          const m: Record<string, number> = {};
          Object.entries(e).forEach(([k, v]) => {
            if (k !== 'track_id' && k !== 'computed_at' && typeof v === 'number') { m[k] = v; dims.add(k); }
          });
          emotionsMap.set(e.track_id as number, m);
        });
        const keys = dims.size > 0 ? [...dims] : Object.keys(KNOWN_PALETTE);
        dimsRef.current = keys;
        const poles = buildPoles(keys);
        const nodes = buildNodes(tracks, emotionsMap, poles);
        nodes.forEach(n => { n.rx = n.x; n.ry = n.y; n.vx = 0; n.vy = 0; });
        polesRef.current   = poles;
        nodesRef.current   = nodes;
        edgesRef.current   = buildEdges(nodes, keys);
        terrainRef.current = buildTerrain(poles);
        floatActive.current = false;
        simActive.current  = false;
        revealStartRef.current = performance.now();
        setSettled(false);
        const cont = containerRef.current;
        if (cont) tf.current = { x: cont.clientWidth / 2, y: cont.clientHeight / 2, scale: 1 };
        setCount(nodes.length);
        setError(null);
      } catch (e: unknown) {
        if (!dead) setError(e instanceof Error ? e.message : 'Failed to load graph data');
      } finally {
        if (!dead) setLoading(false);
      }
    })();
    return () => { dead = true; };
  }, []);

  // ── Render loop ─────────────────────────────────────────────────────────────
  const startLoop = useCallback((el: HTMLCanvasElement | null) => {
    canvasRef.current = el;
    cancelAnimationFrame(animRef.current);
    if (!el) return;

    let lastRender = 0;
    const tick = (ts: number) => {
      // Float mode at 30fps; reveal and hover at full 60fps
      const slowFrame = floatActive.current && !hoveredRef.current;
      if (slowFrame && ts - lastRender < 33) { animRef.current = requestAnimationFrame(tick); return; }
      lastRender = ts;
      const W = el.parentElement?.clientWidth  ?? 0;
      const H = el.parentElement?.clientHeight ?? 0;
      if (el.width !== W) el.width = W;
      if (el.height !== H) el.height = H;
      if (W === 0 || H === 0) { animRef.current = requestAnimationFrame(tick); return; }
      const ctx = el.getContext('2d');
      if (!ctx) { animRef.current = requestAnimationFrame(tick); return; }

      const nodes  = nodesRef.current;
      const poles  = polesRef.current;
      const edges  = edgesRef.current;

      if (tf.current.x === 0 && tf.current.y === 0 && nodes.length > 0) {
        tf.current = { x: W / 2, y: H / 2, scale: 1 };
      }
      const t = tf.current;

      const revealAge = revealStartRef.current > 0 ? ts - revealStartRef.current : 0;
      if (!floatActive.current && revealAge > 2400) {
        floatActive.current = true;
        setSettled(true);
      }

      // Float mode: each bubble gently drifts on its own Lissajous path
      if (floatActive.current) {
        const ft = ts * 0.00025;
        for (const node of nodes) {
          node.x = node.rx + Math.sin(ft       + node.floatPhase) * 5;
          node.y = node.ry + Math.cos(ft * 0.73 + node.floatPhase * 1.3) * 5;
        }
      }

      // Background
      ctx.fillStyle = '#080e1a';
      ctx.fillRect(0, 0, W, H);
      const bgGrd = ctx.createRadialGradient(W/2, H/2, 0, W/2, H/2, Math.min(W,H)*0.75);
      bgGrd.addColorStop(0, 'rgba(25,35,65,0.6)');
      bgGrd.addColorStop(1, 'rgba(8,14,26,0)');
      ctx.fillStyle = bgGrd;
      ctx.fillRect(0, 0, W, H);

      ctx.save();
      ctx.translate(t.x, t.y);
      ctx.scale(t.scale, t.scale);

      // Layer 1: KNN terrain
      const terrain = terrainRef.current;
      if (terrain) {
        ctx.save();
        ctx.globalAlpha = Math.min(1, revealAge / 900);
        ctx.drawImage(terrain, -TERRAIN_WORLD, -TERRAIN_WORLD, TERRAIN_WORLD*2, TERRAIN_WORLD*2);
        ctx.restore();
      }

      // Layer 2: Nebula glows
      for (const p of poles) {
        const [r, g, b] = p.rgb;
        const grd = ctx.createRadialGradient(p.x, p.y, 0, p.x, p.y, NEBULA_R);
        grd.addColorStop(0,    `rgba(${r},${g},${b},0.11)`);
        grd.addColorStop(0.45, `rgba(${r},${g},${b},0.04)`);
        grd.addColorStop(1,    `rgba(${r},${g},${b},0)`);
        ctx.save();
        ctx.globalAlpha = Math.min(1, revealAge / 1100);
        ctx.fillStyle = grd;
        ctx.beginPath(); ctx.arc(p.x, p.y, NEBULA_R, 0, Math.PI*2); ctx.fill();
        ctx.restore();
      }

      const hov  = hoveredRef.current;
      const sel  = selectedRef.current;
      const flt  = filterRef.current;
      const dims = dimsRef.current;

      // Layer 3: Persistent similarity edges (skip for large graphs — too cluttered)
      if (showEdgesRef.current && edges.length > 0 && nodes.length <= 250) {
        ctx.setLineDash([]);
        ctx.lineWidth = 0.8;
        for (const edge of edges) {
          const fade = Math.min(1, Math.max(0, (revealAge - 700) / 900));
          ctx.strokeStyle = `rgba(148,163,184,${(edge.weight * 0.18 * fade).toFixed(3)})`;
          ctx.beginPath(); ctx.moveTo(edge.a.x, edge.a.y); ctx.lineTo(edge.b.x, edge.b.y); ctx.stroke();
        }
      }

      // Layer 4: Hover similarity threads (bright dashed, top-6)
      if (hov && nodes.length > 1) {
        const sims = nodes
          .filter(n => n !== hov)
          .map(n => ({ n, s: cosineSim(hov.emotions, n.emotions, dims) }))
          .sort((a, b) => b.s - a.s).slice(0, 6);
        ctx.setLineDash([4, 6]); ctx.lineWidth = 1.2;
        for (const { n, s } of sims) {
          ctx.strokeStyle = `rgba(148,163,184,${(s * 0.8).toFixed(2)})`;
          ctx.beginPath(); ctx.moveTo(hov.x, hov.y); ctx.lineTo(n.x, n.y); ctx.stroke();
        }
        ctx.setLineDash([]);
      }

      // Layer 5: Song nodes — bubble style (halo + body + specular, no shadowBlur)
      for (let index = 0; index < nodes.length; index++) {
        const node = nodes[index];
        const isHov  = node === hov;
        const isSel  = node === sel;
        const emVal  = flt.emotion ? (node.emotions[flt.emotion] ?? 0) : 1;
        const dimmed = flt.emotion !== null && emVal < flt.threshold;
        const [r, g, b] = node.colorRgb;
        const nodeDelay = Math.min(index * 2, 900);
        const nodeT = Math.max(0, Math.min(1, (revealAge - nodeDelay) / 1400));
        const eased = 1 - Math.pow(1 - nodeT, 3);
        if (eased <= 0) continue;
        const radius = (node.radius + (isHov || isSel ? 2 : 0)) * (0.35 + eased * 0.65);

        ctx.save();

        // Soft outer glow / halo
        ctx.globalAlpha = (dimmed ? 0.03 : (isHov ? 0.22 : 0.11)) * eased;
        ctx.fillStyle = `rgb(${r},${g},${b})`;
        ctx.beginPath(); ctx.arc(node.x, node.y, radius * 2.8, 0, Math.PI*2); ctx.fill();

        // Main bubble body
        ctx.globalAlpha = (dimmed ? 0.07 : 0.86) * eased;
        ctx.fillStyle = `rgb(${r},${g},${b})`;
        ctx.beginPath(); ctx.arc(node.x, node.y, radius, 0, Math.PI*2); ctx.fill();

        // Selected ring
        if (isSel) {
          ctx.globalAlpha = 1;
          ctx.strokeStyle = '#ffffff'; ctx.lineWidth = 1.5; ctx.stroke();
        }

        // Hover ring
        if (isHov && !isSel) {
          ctx.globalAlpha = 0.55;
          ctx.strokeStyle = `rgb(${r},${g},${b})`; ctx.lineWidth = 2.5;
          ctx.beginPath(); ctx.arc(node.x, node.y, radius + 5, 0, Math.PI*2); ctx.stroke();
        }

        // Specular glint (top-left highlight gives 3-D bubble feel)
        if (!dimmed) {
          ctx.globalAlpha = 0.30 * eased;
          ctx.fillStyle = '#ffffff';
          ctx.beginPath();
          ctx.arc(node.x - radius*0.28, node.y - radius*0.30, radius*0.40, 0, Math.PI*2);
          ctx.fill();
        }

        ctx.restore();
      }

      // Layer 6: Emotion poles
      for (const p of poles) {
        const [r, g, b] = p.rgb;
        ctx.save();
        ctx.strokeStyle = `rgba(${r},${g},${b},0.3)`; ctx.lineWidth = 1;
        ctx.beginPath(); ctx.arc(p.x, p.y, 30, 0, Math.PI*2); ctx.stroke();
        ctx.shadowColor = p.color; ctx.shadowBlur = 35; ctx.fillStyle = p.color;
        ctx.beginPath(); ctx.arc(p.x, p.y, 20, 0, Math.PI*2); ctx.fill();
        ctx.shadowBlur = 0; ctx.fillStyle = 'rgba(255,255,255,0.8)';
        ctx.beginPath(); ctx.arc(p.x, p.y, 5, 0, Math.PI*2); ctx.fill();
        ctx.restore();
        ctx.font = 'bold 11px system-ui,sans-serif'; ctx.textAlign = 'center';
        ctx.fillStyle = '#e2e8f0'; ctx.shadowBlur = 0;
        ctx.fillText(p.name.charAt(0).toUpperCase() + p.name.slice(1), p.x, p.y + 40);
      }

      // Hover tooltip
      if (hov) {
        ctx.font = '12px system-ui,sans-serif';
        const tw = ctx.measureText(hov.shortName).width, pad = 8, bh = 22, bw = tw + pad*2;
        const bx = hov.x - bw/2, by = hov.y - hov.radius - bh - 8;
        ctx.fillStyle = 'rgba(8,14,26,0.9)'; drawRoundRect(ctx, bx, by, bw, bh, 5); ctx.fill();
        ctx.fillStyle = '#f1f5f9'; ctx.textAlign = 'center'; ctx.fillText(hov.shortName, hov.x, by + 15);
      }

      ctx.restore();
      animRef.current = requestAnimationFrame(tick);
    };
    animRef.current = requestAnimationFrame(tick);
  }, []);

  // ── Non-passive wheel ──────────────────────────────────────────────────────
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const rect = canvas.getBoundingClientRect(), mx = e.clientX - rect.left, my = e.clientY - rect.top;
      const factor = e.deltaY < 0 ? 1.12 : 0.9, t = tf.current;
      const ns = Math.max(0.06, Math.min(14, t.scale * factor));
      tf.current = { scale: ns, x: mx - (mx - t.x)*(ns/t.scale), y: my - (my - t.y)*(ns/t.scale) };
    };
    canvas.addEventListener('wheel', onWheel, { passive: false });
    return () => canvas.removeEventListener('wheel', onWheel);
  }, []);

  // ── Interaction ────────────────────────────────────────────────────────────
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
    floatActive.current = false;
    simActive.current = false;
    revealStartRef.current = performance.now();
    setSettled(false);
    nodesRef.current.forEach(n => { n.x = n.rx; n.y = n.ry; });
  }, []);

  const poles = polesRef.current;

  // ── UI ─────────────────────────────────────────────────────────────────────
  return (
    <div className="relative flex h-full bg-[#080e1a] overflow-hidden">

      {loading && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-[#080e1a]">
          <div className="text-center space-y-3">
            <div className="w-10 h-10 border-2 border-violet-500 border-t-transparent rounded-full animate-spin mx-auto" />
            <p className="text-slate-400">Mapping emotion space…</p>
            <p className="text-slate-600 text-sm">Fetching tracks &amp; computing positions</p>
          </div>
        </div>
      )}
      {error && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-[#080e1a]">
          <div className="text-center">
            <p className="text-red-400 text-lg mb-2">Graph unavailable</p>
            <p className="text-slate-500 text-sm">{error}</p>
            <p className="text-slate-600 text-xs mt-2">Is the analytics API running on port 8080?</p>
          </div>
        </div>
      )}

      {/* ── Left sidebar ────────────────────────────────────────────────── */}
      <div className="w-52 bg-[#0d1424] border-r border-slate-800 flex flex-col flex-shrink-0 overflow-y-auto">
        <div className="px-4 py-3 border-b border-slate-800">
          <h3 className="text-slate-400 font-bold text-xs uppercase tracking-widest">Emotion Filter</h3>
        </div>
        <div className="p-2 space-y-0.5 flex-1">
          <button onClick={() => setFE(null)}
            className={`w-full text-left px-3 py-1.5 rounded text-xs transition ${filterEmotion === null ? 'bg-slate-700 text-white' : 'text-slate-500 hover:text-white hover:bg-slate-800'}`}>
            All ({nodeCount})
          </button>
          {poles.map(p => (
            <button key={p.name} onClick={() => setFE(filterEmotion === p.name ? null : p.name)}
              className={`w-full text-left px-3 py-1.5 rounded text-xs transition flex items-center gap-2 ${filterEmotion === p.name ? 'bg-slate-700 text-white' : 'text-slate-500 hover:text-white hover:bg-slate-800'}`}>
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
              onChange={e => setThresh(Number(e.target.value) / 100)} className="w-full accent-violet-500" />
            <p className="text-slate-700 text-xs">Dim below threshold</p>
          </div>
        )}
        <div className="p-2 border-t border-slate-800 space-y-0.5">
          <button onClick={() => setShowEdges(v => !v)}
            className="w-full text-left px-3 py-1.5 rounded text-xs text-slate-500 hover:text-white hover:bg-slate-800 transition flex items-center gap-2">
            <span className={`w-2 h-2 rounded-full flex-shrink-0 ${showEdges ? 'bg-slate-400' : 'bg-slate-700'}`} />
            {showEdges ? 'Hide' : 'Show'} connections
          </button>
          <button onClick={resetView}
            className="w-full text-left px-3 py-1.5 rounded text-xs text-slate-500 hover:text-white hover:bg-slate-800 transition">
            ⊙&nbsp; Reset view
          </button>
          <button onClick={rerunSim}
            className="w-full text-left px-3 py-1.5 rounded text-xs text-slate-500 hover:text-white hover:bg-slate-800 transition">
            ✦&nbsp; Replay reveal
          </button>
        </div>
        <div className="p-3 border-t border-slate-800">
          <p className="text-slate-700 text-xs font-semibold mb-1 uppercase tracking-wider">Legend</p>
          <div className="text-slate-700 text-xs space-y-0.5">
            <div>Position → emotion mix</div>
            <div>Territory → KNN zones</div>
            <div>Size → energy</div>
            <div>Colour → valence</div>
            <div>Ring → BPM speed</div>
            <div>Glow → intensity</div>
            <div>Lines → similar songs</div>
            <div className="text-slate-800">─── hover ───</div>
            <div>Dashes → top-6 similar</div>
          </div>
        </div>
      </div>

      {/* ── Canvas area ──────────────────────────────────────────────────── */}
      <div ref={containerRef} className="flex-1 relative overflow-hidden">
        <canvas ref={startLoop} className="absolute inset-0 w-full h-full"
          style={{ cursor: 'grab', display: 'block' }}
          onMouseMove={onMouseMove} onMouseDown={onMouseDown}
          onMouseUp={onMouseUp} onMouseLeave={onMouseLeave} />
        {!settled && nodeCount > 0 && (
          <div className="absolute top-3 right-4 bg-slate-900/70 text-slate-500 text-xs px-2 py-1 rounded-full backdrop-blur">
            Appearing…
          </div>
        )}
        <div className="absolute bottom-3 right-4 text-slate-700 text-xs pointer-events-none select-none">
          Scroll = zoom · Drag = pan · Click = details
        </div>
      </div>

      {/* ── Right sidebar: selected node ─────────────────────────────────── */}
      {selected && (
        <div className="w-68 min-w-64 bg-[#0d1424] border-l border-slate-800 flex flex-col flex-shrink-0 overflow-y-auto">
          <div className="px-4 py-3 border-b border-slate-800 flex justify-between items-start gap-2">
            <p className="text-white text-sm font-semibold leading-snug">{selected.shortName}</p>
            <button onClick={() => setSelected(null)}
              className="text-slate-600 hover:text-white text-xl leading-none flex-shrink-0">×</button>
          </div>
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
          {/* ── Curated label ───────────────────────────────────────────── */}
          {(selectedLabel?.label_path?.length || selectedLabel?.auto_label_path?.length) ? (
            <div className="px-4 py-3 border-b border-slate-800">
              <p className="text-slate-600 text-xs uppercase tracking-widest mb-2">Label</p>
              {selectedLabel.label_path?.length > 0 && (
                <div className="flex flex-wrap gap-1 mb-1">
                  {selectedLabel.label_path.map((seg, i) => (
                    <span key={i} className="text-xs bg-indigo-900/60 text-indigo-300 rounded px-2 py-0.5">{seg}</span>
                  ))}
                </div>
              )}
              {selectedLabel.auto_label_path?.length > 0 && selectedLabel.label_path?.length === 0 && (
                <div className="flex flex-wrap gap-1">
                  {selectedLabel.auto_label_path.map((seg, i) => (
                    <span key={i} className="text-xs bg-slate-700/60 text-slate-400 rounded px-2 py-0.5 italic">{seg}</span>
                  ))}
                  <span className="text-slate-600 text-xs self-center">auto</span>
                </div>
              )}
            </div>
          ) : null}
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
                  else if (typeof v === 'number' && v >= 0 && v <= 1) display = `${(v * 100).toFixed(0)}%`;
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



















