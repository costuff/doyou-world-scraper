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
  homeX: number; homeY: number; // balanced anchor position
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

interface PhysicsTuning {
  radiusScale: number;
  radiusBase: number;
  radialPush: number;
  wakePush: number;
  nearPush: number;
  springBack: number;
  settleDamping: number;
  collisionPush: number;
  maxVelocity: number;
}

const DEFAULT_PHYSICS: PhysicsTuning = {
  radiusScale: 17.5,
  radiusBase: 260,
  radialPush: 1.2,
  wakePush: 1.2,
  nearPush: 0.45,
  springBack: 0.032,
  settleDamping: 0.98,
  collisionPush: 0.14,
  maxVelocity: 14,
};

const FLUIDITY_MIN: PhysicsTuning = {
  radiusScale: 9,
  radiusBase: 220,
  radialPush: 0.4,
  wakePush: 0.2,
  nearPush: 0.28,
  springBack: 0.052,
  settleDamping: 0.963,
  collisionPush: 0.07,
  maxVelocity: 10,
};

const FLUIDITY_MAX: PhysicsTuning = {
  radiusScale: 26,
  radiusBase: 300,
  radialPush: 2.0,
  wakePush: 2.2,
  nearPush: 0.62,
  springBack: 0.012,
  settleDamping: 0.997,
  collisionPush: 0.21,
  maxVelocity: 18,
};

function fluidityToPhysics(level: number): PhysicsTuning {
  const t = Math.max(0, Math.min(1, level / 100));
  const lerp = (a: number, b: number) => a + (b - a) * t;
  return {
    radiusScale: lerp(FLUIDITY_MIN.radiusScale, FLUIDITY_MAX.radiusScale),
    radiusBase: lerp(FLUIDITY_MIN.radiusBase, FLUIDITY_MAX.radiusBase),
    radialPush: lerp(FLUIDITY_MIN.radialPush, FLUIDITY_MAX.radialPush),
    wakePush: lerp(FLUIDITY_MIN.wakePush, FLUIDITY_MAX.wakePush),
    nearPush: lerp(FLUIDITY_MIN.nearPush, FLUIDITY_MAX.nearPush),
    springBack: lerp(FLUIDITY_MIN.springBack, FLUIDITY_MAX.springBack),
    settleDamping: lerp(FLUIDITY_MIN.settleDamping, FLUIDITY_MAX.settleDamping),
    collisionPush: lerp(FLUIDITY_MIN.collisionPush, FLUIDITY_MAX.collisionPush),
    maxVelocity: lerp(FLUIDITY_MIN.maxVelocity, FLUIDITY_MAX.maxVelocity),
  };
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
      homeX: 0, homeY: 0,
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
    node.homeX = node.x;
    node.homeY = node.y;
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
  tuning: PhysicsTuning,
  viewScale: number = 1,
  bowStrength: number = 1,
  dragVx: number = 0,
  dragVy: number = 0,
): void {
  const dragged = nodes[draggedIndex];
  // Drag flow field: strong near cursor, fades smoothly with distance.
  const bowRadius = Math.max(dragged.radius * tuning.radiusScale, tuning.radiusBase / Math.max(viewScale, 0.07));
  const dragSpeed = Math.sqrt(dragVx * dragVx + dragVy * dragVy);
  const invDrag = dragSpeed > 0.0001 ? 1 / dragSpeed : 0;
  const dirX = dragVx * invDrag;
  const dirY = dragVy * invDrag;

  for (let i = 0; i < nodes.length; i++) {
    if (i === draggedIndex) continue;
    const node = nodes[i];
    const dx = node.rx - dragged.rx;
    const dy = node.ry - dragged.ry;
    const d = Math.sqrt(dx * dx + dy * dy) || 0.01;
    if (d < bowRadius) {
      const tVal = 1 - d / bowRadius;
      const smooth = tVal * tVal * (3 - 2 * tVal);
      const radialImpulse = smooth * (tuning.radialPush + dragSpeed * 1.1) * bowStrength;
      const wakeImpulse = smooth * (tuning.wakePush + dragSpeed * 0.95);
      node.vx += (dx / d) * radialImpulse + dirX * wakeImpulse;
      node.vy += (dy / d) * radialImpulse + dirY * wakeImpulse;

      // Hard separation very close to the dragged node.
      const minD = dragged.radius + node.radius + 20;
      if (d < minD) {
        const push = (minD - d) * tuning.nearPush;
        node.vx += (dx / d) * push;
        node.vy += (dy / d) * push;
      }

      // Keep speeds bounded to avoid jitter spikes.
      const v = Math.sqrt(node.vx * node.vx + node.vy * node.vy);
      const maxV = tuning.maxVelocity;
      if (v > maxV) {
        const s = maxV / v;
        node.vx *= s;
        node.vy *= s;
      }
    }
  }

  // Soft overlap separation via velocity (not hard position snapping)
  for (let iter = 0; iter < 1; iter++) {
    for (let i = 0; i < nodes.length; i++) {
      if (i === draggedIndex) continue;
      for (let j = i + 1; j < nodes.length; j++) {
        if (j === draggedIndex) continue;
        const a = nodes[i], b = nodes[j];
        const dx = a.rx - b.rx, dy = a.ry - b.ry;
        const d2 = dx * dx + dy * dy;
        if (d2 < 0.01) { a.vx += 0.2; b.vx -= 0.2; continue; }
        const minD = a.radius + b.radius + 18;
        if (d2 < minD * minD) {
          const d = Math.sqrt(d2);
          const push = (minD - d) * 0.08;
          const nx = dx / d, ny = dy / d;
          a.vx += nx * push; a.vy += ny * push;
          b.vx -= nx * push; b.vy -= ny * push;
        }
      }
    }
  }
}

function stepLayoutPhysics(nodes: GraphNode[], draggedIndex: number, tuning: PhysicsTuning): void {
  // 1) Weak spring to balanced anchor positions (keeps graph coherent after release)
  for (let i = 0; i < nodes.length; i++) {
    if (i === draggedIndex) continue;
    const n = nodes[i];
    n.vx += (n.homeX - n.rx) * tuning.springBack;
    n.vy += (n.homeY - n.ry) * tuning.springBack;
  }

  // 2) Neighbour collision response as smooth velocity impulses
  for (let i = 0; i < nodes.length; i++) {
    for (let j = i + 1; j < nodes.length; j++) {
      if (i === draggedIndex || j === draggedIndex) continue;
      const a = nodes[i], b = nodes[j];
      const dx = a.rx - b.rx;
      const dy = a.ry - b.ry;
      const d2 = dx * dx + dy * dy;
      if (d2 < 0.0001) {
        a.vx += 0.05;
        b.vx -= 0.05;
        continue;
      }
      const minD = a.radius + b.radius + 18;
      if (d2 < minD * minD) {
        const d = Math.sqrt(d2);
        const nx = dx / d;
        const ny = dy / d;
        const push = (minD - d) * tuning.collisionPush;
        a.vx += nx * push; a.vy += ny * push;
        b.vx -= nx * push; b.vy -= ny * push;
      }
    }
  }

  // 3) Integrate with damping (smooth deceleration)
  for (let i = 0; i < nodes.length; i++) {
    if (i === draggedIndex) continue;
    const n = nodes[i];
    n.vx *= tuning.settleDamping;
    n.vy *= tuning.settleDamping;
    if (Math.abs(n.vx) < 0.005) n.vx = 0;
    if (Math.abs(n.vy) < 0.005) n.vy = 0;
    n.rx += n.vx;
    n.ry += n.vy;
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
  const [fluidity, setFluidity]   = useState(50);
  const [physics, setPhysics]     = useState<PhysicsTuning>(DEFAULT_PHYSICS);

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
  const dragVelRef     = useRef({ x: 0, y: 0 });
  const physicsRef     = useRef<PhysicsTuning>(DEFAULT_PHYSICS);

  useEffect(() => { filterRef.current    = { emotion: filterEmotion, threshold }; }, [filterEmotion, threshold]);
  useEffect(() => { focusedRef.current   = focused; }, [focused]);
  useEffect(() => { showEdgesRef.current = showEdges; }, [showEdges]);
  useEffect(() => { physicsRef.current   = physics; }, [physics]);
  useEffect(() => { setPhysics(fluidityToPhysics(fluidity)); }, [fluidity]);

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

      // ── Drag + layout physics ──
      const isDraggingNode = drag.current.nodeOn && drag.current.nodeIdx >= 0 && !!cursorWorldRef.current;
      const dragIdx = isDraggingNode ? drag.current.nodeIdx : -1;

      if (isDraggingNode) {
        const dragged = nodes[dragIdx];
        const cur = cursorWorldRef.current!;
        if (dragged) {
          dragged.rx = cur.x;
          dragged.ry = cur.y;
          dragged.vx = 0;
          dragged.vy = 0;
          relaxLayoutAround(nodes, dragIdx, physicsRef.current, t.scale, 1, dragVelRef.current.x, dragVelRef.current.y);
        }
      }

      stepLayoutPhysics(nodes, dragIdx, physicsRef.current);

      // Float mode overlays only when node is near-rest, so active flow remains visible.
      const ft = ts * 0.00025;
      for (let i = 0; i < nodes.length; i++) {
        const node = nodes[i];
        const moving = Math.abs(node.vx) > 0.01 || Math.abs(node.vy) > 0.01;
        if (floatActive.current && !isDraggingNode && !moving) {
          node.x = node.rx + Math.sin(ft + node.floatPhase) * 5;
          node.y = node.ry + Math.cos(ft * 0.73 + node.floatPhase * 1.3) * 5;
        } else {
          node.x = node.rx;
          node.y = node.ry;
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
      const dx = w.x - d.lastX;
      const dy = w.y - d.lastY;
      dragVelRef.current.x = dragVelRef.current.x * 0.65 + dx * 0.35;
      dragVelRef.current.y = dragVelRef.current.y * 0.65 + dy * 0.35;
      drag.current.lastX = w.x;
      drag.current.lastY = w.y;
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
      dragVelRef.current = { x: 0, y: 0 };
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

    // Always persist dragged node at release point (independent of moved threshold).
    if (wasNodeDrag && nodeIdx >= 0) {
      const dropped = nodesRef.current[nodeIdx];
      if (dropped) {
        const w = toWorld(e.clientX, e.clientY);
        dropped.x = w.x;
        dropped.y = w.y;
        dropped.rx = w.x;
        dropped.ry = w.y;
        dropped.homeX = w.x;
        dropped.homeY = w.y;
      }
    }

    cursorWorldRef.current = null;
    dragVelRef.current = { x: 0, y: 0 };
    drag.current.on = false;
    drag.current.nodeOn = false;
    canvasRef.current!.style.cursor = 'grab';

    // If we were dragging a node, resolve overlaps once and commit positions
    if (wasNodeDrag && moved && nodeIdx >= 0) {
      // Velocities decay naturally via damping — no burst, no snap.
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
    const d = drag.current;
    if (d.nodeOn && d.nodeIdx >= 0) {
      const dropped = nodesRef.current[d.nodeIdx];
      if (dropped) {
        // Leaving canvas while dragging should still persist the drop position.
        dropped.homeX = dropped.rx;
        dropped.homeY = dropped.ry;
      }
    }
    // Natural release on leave: keep current velocities so nodes decelerate smoothly.
    // Do not snap rx/ry or zero vx/vy here.
    cursorWorldRef.current = null;
    dragVelRef.current = { x: 0, y: 0 };
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
        <div className="p-2 space-y-0.5 max-h-52 overflow-y-auto">
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
        <div className="p-3 border-t border-slate-800 space-y-3">
          <div className="flex items-center justify-between">
            <p className="text-slate-400 font-bold text-xs uppercase tracking-widest">Fluidity (Live)</p>
            <button
              onClick={() => setFluidity(50)}
              className="text-[10px] text-slate-600 hover:text-slate-300 transition"
            >
              Reset
            </button>
          </div>

          <div className="space-y-1">
            <div className="flex justify-between text-[11px] text-slate-500">
              <span>Barely fluid</span>
              <span className="text-slate-300 font-mono">{fluidity}%</span>
              <span>Extra fluid</span>
            </div>
            <input
              type="range"
              min={0}
              max={100}
              step={1}
              value={fluidity}
              onChange={e => setFluidity(Number(e.target.value))}
              className="w-full accent-violet-500"
            />
          </div>
          <p className="text-slate-700 text-[11px] leading-4">
            Default at load is your tuned baseline. This single bar scales reach, push, flow, return, slowdown, and spacing together.
          </p>
        </div>
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



















