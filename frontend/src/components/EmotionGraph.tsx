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
  NODE_MIN_R, NODE_MAX_R, KNOWN_PALETTE,
  clamp01, valenceRgb, cosineSim, buildPoles,
  type Pole,
} from './emotionGraphUtils';

const ANALYTICS = import.meta.env.VITE_ANALYTICS_API_URL || '/analytics-api';

// Poles hidden from the canvas and sidebar until a UX is designed for them
const HIDDEN_POLES = new Set(['tension', 'power', 'melancholy']);

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
  _poles: Pole[],
): GraphNode[] {
  return tracks.filter(t => emotionsMap.has(t.id) && t.features != null).map((t) => {
    const emotions = emotionsMap.get(t.id)!;
    const f = t.features;

    const energy  = f.energy  ?? 0.5;
    const valence = f.valence ?? 0.5;
    const bpmNorm = f.bpm != null ? clamp01(f.bpm / 200) : 0.5;

    // Fully random seed — no feature bias. Position carries no meaning at this stage.
    // TODO: add "Group by label" button that re-seeds by label clusters on demand.
    // Use golden-angle spiral to guarantee even coverage with no centre clumping.
    const i = tracks.indexOf(t);
    const goldenAngle = 2.399963; // radians
    const r = Math.sqrt(i + 1) * 38 + (Math.random() - 0.5) * 80;
    const theta = i * goldenAngle + Math.random() * 0.9;
    const ix = Math.cos(theta) * r;
    const iy = Math.sin(theta) * r;

    const raw = t.file_name;
    const clean = raw.replace(/\s*\[[\w-]+\](\.\w+)?$/, '').replace(/\.\w+$/, '').trim() || raw;
    const shortName = clean.length > 36 ? clean.slice(0, 35) + '\u2026' : clean;

    return {
      id: t.id,
      name: raw,
      shortName,
      features: f,
      emotions,
      x:  ix,
      y:  iy,
      vx: 0, vy: 0,
      rx: 0, ry: 0,
      floatPhase: (t.id * 2.399) % (Math.PI * 2),
      radius: NODE_MIN_R + energy * (NODE_MAX_R - NODE_MIN_R),
      colorRgb: valenceRgb(valence),
      bpmNorm,
    };
  });
}

function resolveLayout(nodes: GraphNode[]): void {
  const anchors = nodes.map(node => ({ x: node.x, y: node.y }));

  // ── Seed: keep initial random world-space positions as-is
  // Pure chaos means each node starts somewhere random, then collision relaxation
  // naturally separates and organizes them. No poles, no structure imposed.
  // (anchors are already set from buildNodes random seeding)

  // ── Phase 1: 250 iterations of pure symmetric collision — zero attraction.
  // Symmetric push (both nodes move) is far more stable than one-sided.
  // Gap = 22px so body + halo never overlap: avg halo radius is 9×2.8 = 25px,
  // so two average halos clear with 22px body gap.
  for (let iter = 0; iter < 250; iter++) {
    for (let i = 0; i < nodes.length; i++) {
      const a = nodes[i];
      for (let j = i + 1; j < nodes.length; j++) {
        const b = nodes[j];
        const dx = a.x - b.x;
        const dy = a.y - b.y;
        const d2 = dx * dx + dy * dy;
        if (d2 < 0.0001) { a.x += 1.5; b.x -= 1.5; continue; }
        const minD = a.radius + b.radius + 22;
        if (d2 < minD * minD) {
          const d = Math.sqrt(d2);
          const push = (minD - d) * 0.5;
          const nx = dx / d, ny = dy / d;
          a.x += nx * push; a.y += ny * push;
          b.x -= nx * push; b.y -= ny * push;
        }
      }
    }
  }

  // ── Phase 2: 60 iterations of light attraction back to starting position + collision.
  // Gently pulls nodes toward their initial random positions (no poles, just settling).
  for (let iter = 0; iter < 60; iter++) {
    for (let i = 0; i < nodes.length; i++) {
      const a = nodes[i];
      a.x += (anchors[i].x - a.x) * 0.012;
      a.y += (anchors[i].y - a.y) * 0.012;
      for (let j = i + 1; j < nodes.length; j++) {
        const b = nodes[j];
        const dx = a.x - b.x;
        const dy = a.y - b.y;
        const d2 = dx * dx + dy * dy;
        if (d2 < 0.0001) { a.x += 0.8; b.x -= 0.8; continue; }
        const minD = a.radius + b.radius + 22;
        if (d2 < minD * minD) {
          const d = Math.sqrt(d2);
          const push = (minD - d) * 0.5;
          const nx = dx / d, ny = dy / d;
          a.x += nx * push; a.y += ny * push;
          b.x -= nx * push; b.y -= ny * push;
        }
      }
    }
  }

  nodes.forEach(node => {
    node.rx = node.x;
    node.ry = node.y;
    node.vx = 0;
    node.vy = 0;
  });
}

// Relax layout around a specific node (for interactive dragging)
// viewScale: current canvas zoom — used to compute a world-space push radius
// that stays proportional at any zoom level.
function relaxLayoutAround(
  nodes: GraphNode[],
  draggedIndex: number,
  iterations: number = 6,
  viewScale: number = 1,
  bowStrength: number = 1,
  updateRestPos: boolean = false,
): void {
  const dragged = nodes[draggedIndex];
  // Bow radius scales with zoom so it's visible at any zoom level
  const bowRadius = Math.max(dragged.radius * 5, 120 / Math.max(viewScale, 0.07));

  for (let iter = 0; iter < iterations; iter++) {
    for (let i = 0; i < nodes.length; i++) {
      const node = nodes[i];
      if (i === draggedIndex) continue;

      const dxd = node.x - dragged.x;
      const dyd = node.y - dragged.y;
      const distToDragged = Math.sqrt(dxd * dxd + dyd * dyd) || 0.01;

      // Bow-wave push from dragged node
      if (distToDragged < bowRadius) {
        const t = 1 - distToDragged / bowRadius;
        const smooth = t * t * (3 - 2 * t);
        const strength = smooth * bowRadius * 0.03 * bowStrength;
        node.x += (dxd / distToDragged) * strength;
        node.y += (dyd / distToDragged) * strength;
      }

      // Overlap repulsion with all other non-dragged nodes
      for (let j = 0; j < nodes.length; j++) {
        if (j === i || j === draggedIndex) continue;
        const other = nodes[j];
        const dx = node.x - other.x;
        const dy = node.y - other.y;
        const d2 = dx * dx + dy * dy;
        if (d2 < 0.01) { node.x += 1; continue; }
        const minD = node.radius + other.radius + 20;
        if (d2 < minD * minD) {
          const d = Math.sqrt(d2);
          const push = (minD - d) * 0.4;
          node.x += (dx / d) * push;
          node.y += (dy / d) * push;
        }
      }

      if (updateRestPos) {
        node.rx = node.x;
        node.ry = node.y;
      }
    }
  }
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

// TODO: "Group by label" button — when clicked, re-seed node positions by their
// curated label clusters (similar labels → same region) so the graph becomes a
// spatial label map. Re-run collision relaxation after repositioning.

// ─── Component ────────────────────────────────────────────────────────────────

export default function EmotionGraph() {
  const canvasRef    = useRef<HTMLCanvasElement | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const [loading, setLoading]     = useState(true);
  const [error, setError]         = useState<string | null>(null);
  const [nodeCount, setCount]     = useState(0);
  // Multi-select: focused = node shown in sidebar detail; selectedIds = all highlighted nodes
  const [focused, setFocused]     = useState<GraphNode | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [selectedLabel, setSelectedLabel] = useState<{ label_path: string[]; auto_label_path: string[] } | null>(null);
  const [filterEmotion, setFE]    = useState<string | null>(null);
  const [threshold, setThresh]    = useState(0.5);
  const [settled, setSettled]     = useState(false);
  const [showEdges, setShowEdges] = useState(true);

  const nodesRef       = useRef<GraphNode[]>([]);
  const polesRef       = useRef<Pole[]>([]);
  const edgesRef       = useRef<Edge[]>([]);
  const tf             = useRef({ x: 0, y: 0, scale: 1 });
  const hoveredRef     = useRef<GraphNode | null>(null);
  const focusedRef     = useRef<GraphNode | null>(null);
  const selectedIdsRef = useRef<Set<number>>(new Set());
  const filterRef      = useRef<{ emotion: string | null; threshold: number }>({ emotion: null, threshold: 0.5 });
  const dimsRef        = useRef<string[]>([]);
  const showEdgesRef   = useRef(true);
  const simActive      = useRef(false);
  const floatActive    = useRef(false);
  const revealStartRef = useRef(0);
  const animRef        = useRef(0);
  const drag           = useRef({ on: false, sx: 0, sy: 0, ox: 0, oy: 0, nodeOn: false, nodeIdx: -1, lastX: 0, lastY: 0 });
  const cursorWorldRef = useRef<{ x: number; y: number } | null>(null);

  useEffect(() => { filterRef.current    = { emotion: filterEmotion, threshold }; }, [filterEmotion, threshold]);
  useEffect(() => { focusedRef.current   = focused; }, [focused]);
  useEffect(() => { showEdgesRef.current = showEdges; }, [showEdges]);

  // Fetch label whenever focused node changes
  useEffect(() => {
    if (!focused) { setSelectedLabel(null); return; }
    axios.get(`${ANALYTICS}/api/v1/tracks/${focused.id}/label`)
      .then(r => setSelectedLabel(r.data))
      .catch(() => setSelectedLabel(null));
  }, [focused?.id]);

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
        resolveLayout(nodes);
        polesRef.current   = poles;
        nodesRef.current   = nodes;
        edgesRef.current   = buildEdges(nodes, keys);
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

      // ── Drag physics — in render loop so it runs after float overwrites x/y ──
      if (drag.current.nodeOn && drag.current.nodeIdx >= 0 && cursorWorldRef.current) {
        const dragIdx = drag.current.nodeIdx;
        const dragged = nodes[dragIdx];
        if (dragged) {
          const cur = cursorWorldRef.current;
          dragged.x  = cur.x; dragged.y  = cur.y;
          dragged.rx = cur.x; dragged.ry = cur.y;
          // 3 iters per frame: nodes part around the dragged node, rx/ry NOT updated so they spring back when released
          relaxLayoutAround(nodes, dragIdx, 3, t.scale, 1, false);
        }
      }

      // ── Background: dark base ──────────────────────────────────────────
      ctx.fillStyle = '#080e1a';
      ctx.fillRect(0, 0, W, H);

      // ── Screen-space emotion blobs — fill entire canvas at any zoom/pan ──
      // Each visible pole projects to a screen position and radiates a colour
      // blob large enough to always bleed past the screen edges.
      const bgFade = Math.min(1, revealAge / 1200);
      if (bgFade > 0) {
        for (const p of poles) {
          if (HIDDEN_POLES.has(p.name)) continue;
          const [r, g, b] = p.rgb;
          const sx = t.x + p.x * t.scale;
          const sy = t.y + p.y * t.scale;
          const blobR = Math.max(W, H) * 1.6;
          const grd = ctx.createRadialGradient(sx, sy, 0, sx, sy, blobR);
          grd.addColorStop(0,    `rgba(${r},${g},${b},${(0.14 * bgFade).toFixed(3)})`);
          grd.addColorStop(0.4,  `rgba(${r},${g},${b},${(0.05 * bgFade).toFixed(3)})`);
          grd.addColorStop(1,    `rgba(${r},${g},${b},0)`);
          ctx.fillStyle = grd;
          ctx.fillRect(0, 0, W, H);
        }
      }

      ctx.save();
      ctx.translate(t.x, t.y);
      ctx.scale(t.scale, t.scale);

      const hov  = hoveredRef.current;
      const flt  = filterRef.current;
      const dims = dimsRef.current;
      const selIds = selectedIdsRef.current;

      // Layer 3: Persistent similarity edges — always visible
      if (showEdgesRef.current && edges.length > 0) {
        ctx.setLineDash([]);
        for (const edge of edges) {
          const fade = Math.min(1, Math.max(0, (revealAge - 700) / 900));
          const bothSelected = selIds.has(edge.a.id) && selIds.has(edge.b.id);
          if (bothSelected) {
            ctx.lineWidth = 2;
            ctx.strokeStyle = `rgba(168,85,247,${(0.75 * fade).toFixed(3)})`;
          } else {
            ctx.lineWidth = 0.7;
            ctx.strokeStyle = `rgba(148,163,184,${(edge.weight * 0.15 * fade).toFixed(3)})`;
          }
          ctx.beginPath(); ctx.moveTo(edge.a.x, edge.a.y); ctx.lineTo(edge.b.x, edge.b.y); ctx.stroke();
        }
      }

      // Layer 3.5: Direct connection lines between ALL selected nodes (not just KNN neighbours)
      if (selIds.size >= 2) {
        const selNodes = nodes.filter((n: GraphNode) => selIds.has(n.id));
        ctx.setLineDash([3, 5]);
        ctx.lineWidth = 1.2;
        ctx.strokeStyle = 'rgba(168,85,247,0.35)';
        for (let i = 0; i < selNodes.length; i++) {
          for (let j = i + 1; j < selNodes.length; j++) {
            ctx.beginPath();
            ctx.moveTo(selNodes[i].x, selNodes[i].y);
            ctx.lineTo(selNodes[j].x, selNodes[j].y);
            ctx.stroke();
          }
        }
        ctx.setLineDash([]);
      }

      // Layer 4: Hover similarity threads (bright dashed, top-6)
      if (hov && nodes.length > 1) {
        const sims = nodes
          .filter((n: GraphNode) => n !== hov)
          .map((n: GraphNode) => ({ n, s: cosineSim(hov.emotions, n.emotions, dims) }))
          .sort((a: any, b: any) => b.s - a.s).slice(0, 6);
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
        const isSel  = selIds.has(node.id);
        const emVal  = flt.emotion ? (node.emotions[flt.emotion] ?? 0) : 1;
        const dimmed = flt.emotion !== null && emVal < flt.threshold && !isSel;
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

      // Layer 6: (poles removed — no emotion nodes displayed)

      // Hover tooltip
      if (hov) {
        ctx.font = '12px system-ui,sans-serif';
        const tw = ctx.measureText(hov.shortName).width, pad = 8, bh = 22, bw = tw + pad*2;
        const bx = hov.x - bw/2, by = hov.y - hov.radius - bh - 8;
        ctx.fillStyle = 'rgba(8,14,26,0.9)'; drawRoundRect(ctx, bx, by, bw, bh, 5); ctx.fill();
        ctx.fillStyle = '#f1f5f9'; ctx.textAlign = 'center'; ctx.fillText(hov.shortName, hov.x, by + 15);
      }

      // Persistent labels for selected nodes (up to 18 to reduce clutter)
      const selectedNodes = nodes.filter((n: GraphNode) => selIds.has(n.id)).slice(0, 18);
      for (const node of selectedNodes) {
        if (hov && hov.id === node.id) continue; // hover tooltip already shown
        const label = node.shortName;
        ctx.font = '11px system-ui,sans-serif';
        const tw = ctx.measureText(label).width;
        const pad = 6;
        const bh = 18;
        const bw = tw + pad * 2;
        const by = node.y - node.radius - bh - 7;
        const bx = node.x - bw / 2;
        ctx.fillStyle = 'rgba(15,23,42,0.88)';
        drawRoundRect(ctx, bx, by, bw, bh, 4);
        ctx.fill();
        ctx.strokeStyle = 'rgba(255,255,255,0.45)';
        ctx.lineWidth = 0.8;
        ctx.stroke();
        ctx.fillStyle = '#f8fafc';
        ctx.textAlign = 'center';
        ctx.fillText(label, node.x, by + 12.5);
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
    const d = drag.current;
    const w = toWorld(e.clientX, e.clientY);

    if (d.on && d.nodeOn) {
      cursorWorldRef.current = w;
      return;
    }

    if (d.on) {
      tf.current = { ...tf.current, x: d.ox + e.clientX - d.sx, y: d.oy + e.clientY - d.sy };
      return;
    }

    hoveredRef.current = hitTest(w.x, w.y);
    canvasRef.current!.style.cursor = 'grab';
  }, [toWorld, hitTest]);

  const onMouseDown = useCallback((e: React.MouseEvent<HTMLCanvasElement>) => {
    const w = toWorld(e.clientX, e.clientY);
    const node = hitTest(w.x, w.y);
    
    if (e.button === 0 && node) {
      // Left-click on node: drag the node itself
      cursorWorldRef.current = w;
      drag.current = { 
        on: true, sx: e.clientX, sy: e.clientY, ox: 0, oy: 0,
        nodeOn: true, nodeIdx: nodesRef.current.indexOf(node), lastX: w.x, lastY: w.y 
      };
      canvasRef.current!.style.cursor = 'grabbing';
    } else if (e.button === 0) {
      // Left-click on empty space: pan
      drag.current = { 
        on: true, sx: e.clientX, sy: e.clientY, ox: tf.current.x, oy: tf.current.y,
        nodeOn: false, nodeIdx: -1, lastX: 0, lastY: 0
      };
      canvasRef.current!.style.cursor = 'grabbing';
    }
  }, [toWorld, hitTest]);

  const onMouseUp = useCallback((e: React.MouseEvent<HTMLCanvasElement>) => {
    const d = drag.current;
    const moved = Math.abs(e.clientX - d.sx) > 4 || Math.abs(e.clientY - d.sy) > 4;
    const wasNodeDrag = d.nodeOn;
    const nodeIdx = d.nodeIdx;

    cursorWorldRef.current = null;
    drag.current.on = false;
    drag.current.nodeOn = false;
    canvasRef.current!.style.cursor = 'grab';

    // If we were dragging a node, resolve overlaps once and commit positions
    if (wasNodeDrag && moved && nodeIdx >= 0) {
      const allNodes = nodesRef.current;
      // One clean burst of pure collision separation — no bow-wave, no ongoing flow
      for (let iter = 0; iter < 35; iter++) {
        for (let i = 0; i < allNodes.length; i++) {
          if (i === nodeIdx) continue;
          for (let j = i + 1; j < allNodes.length; j++) {
            if (j === nodeIdx) continue;
            const a = allNodes[i], b = allNodes[j];
            const dx = a.x - b.x, dy = a.y - b.y;
            const d2 = dx * dx + dy * dy;
            if (d2 < 0.01) { a.x += 1; b.x -= 1; continue; }
            const minD = a.radius + b.radius + 20;
            if (d2 < minD * minD) {
              const d = Math.sqrt(d2);
              const push = (minD - d) * 0.5;
              const nx = dx / d, ny = dy / d;
              a.x += nx * push; a.y += ny * push;
              b.x -= nx * push; b.y -= ny * push;
            }
          }
        }
      }
      // Commit all final positions as new rest positions
      allNodes.forEach(n => { n.rx = n.x; n.ry = n.y; });
      return;
    }

    if (!moved) {
      // Click (no movement) — handle selection
      const w = toWorld(e.clientX, e.clientY);
      const node = wasNodeDrag ? nodesRef.current[nodeIdx] : hitTest(w.x, w.y);

      if (node) {
        // Toggle this node in the selection set
        const next = new Set(selectedIdsRef.current);
        if (next.has(node.id)) {
          next.delete(node.id);
          // If we deselected the focused node, focus another selected one
          if (focusedRef.current?.id === node.id) {
            const remaining = nodesRef.current.filter((n: GraphNode) => next.has(n.id));
            const nxt = remaining[remaining.length - 1] ?? null;
            focusedRef.current = nxt;
            setFocused(nxt);
          }
        } else {
          next.add(node.id);
          focusedRef.current = node;
          setFocused(node);
        }
        selectedIdsRef.current = next;
        setSelectedIds(new Set(next));
      }
    }
  }, [toWorld, hitTest]);

  const onMouseLeave = useCallback(() => {
    if (drag.current.nodeOn) {
      nodesRef.current.forEach(n => { n.rx = n.x; n.ry = n.y; });
    }
    cursorWorldRef.current = null;
    drag.current.on = false;
    drag.current.nodeOn = false;
    hoveredRef.current = null;
  }, []);

  const resetView = useCallback(() => {
    const c = containerRef.current;
    if (c) tf.current = { x: c.clientWidth / 2, y: c.clientHeight / 2, scale: 1 };
  }, []);

  const rerunSim = useCallback(() => {
    floatActive.current = false;
    simActive.current = false;
    revealStartRef.current = performance.now();
    setSettled(false);
    nodesRef.current.forEach((n: GraphNode) => { n.x = n.rx; n.y = n.ry; });
  }, []);

  const clearSelection = useCallback(() => {
    selectedIdsRef.current = new Set();
    setSelectedIds(new Set());
    focusedRef.current = null;
    setFocused(null);
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
          {poles.filter(p => !HIDDEN_POLES.has(p.name)).map(p => (
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
          Scroll = zoom · Drag canvas = pan · Drag node = move · Click = select/deselect
        </div>
      </div>

      {/* ── Right sidebar: selected node(s) ──────────────────────────────── */}
      {(focused || selectedIds.size > 0) && (
        <div className="w-68 min-w-64 bg-[#0d1424] border-l border-slate-800 flex flex-col flex-shrink-0 overflow-y-auto">
          {/* Selection header */}
          <div className="px-4 py-2.5 border-b border-slate-800 flex items-center justify-between gap-2">
            <span className="text-slate-500 text-xs">
              {selectedIds.size === 1 ? '1 track selected' : `${selectedIds.size} tracks selected`}
            </span>
            <button onClick={clearSelection}
              className="text-slate-600 hover:text-slate-300 text-xs transition">Clear all</button>
          </div>
          {/* Multi-select track list (when more than 1) */}
          {selectedIds.size > 1 && (
            <div className="border-b border-slate-800 max-h-36 overflow-y-auto">
              {nodesRef.current.filter(n => selectedIds.has(n.id)).map(n => (
                <button key={n.id}
                  onClick={() => { focusedRef.current = n; setFocused(n); }}
                  className={`w-full text-left px-3 py-1.5 text-xs transition flex items-center gap-2 ${
                    focused?.id === n.id ? 'bg-slate-700/60 text-white' : 'text-slate-400 hover:text-white hover:bg-slate-800/60'
                  }`}>
                  <span className="w-1.5 h-1.5 rounded-full flex-shrink-0 bg-white/40" />
                  <span className="truncate">{n.shortName}</span>
                </button>
              ))}
            </div>
          )}
          {focused && (
            <>
          <div className="px-4 py-3 border-b border-slate-800 flex justify-between items-start gap-2">
            <p className="text-white text-sm font-semibold leading-snug">{focused.shortName}</p>
          </div>
          <div className="px-4 py-3 border-b border-slate-800">
            <p className="text-slate-600 text-xs uppercase tracking-widest mb-3">Emotions</p>
            <div className="space-y-2">
              {poles.map(p => {
                const v = focused.emotions[p.name] ?? 0;
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
              {Object.entries(focused.features)
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
            </>
          )}
        </div>
      )}
    </div>
  );
}



















