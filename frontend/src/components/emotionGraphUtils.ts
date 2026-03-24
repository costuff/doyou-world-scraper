/**
 * Pure utility functions extracted from EmotionGraph.tsx.
 * Kept in a separate file so they can be unit-tested without a DOM/canvas.
 */

export const POLE_ORBIT = 310;
export const NEBULA_R   = 215;
export const ATTRACT    = 0.007;
export const REPEL      = 480;
export const CENTER_PULL = 0.00015;
export const DAMPING    = 0.94;
export const NODE_MIN_R = 4;
export const NODE_MAX_R = 14;

export const KNOWN_PALETTE: Record<string, [string, [number, number, number]]> = {
  euphoria:   ['#fbbf24', [251, 191,  36]],
  tension:    ['#ef4444', [239,  68,  68]],
  melancholy: ['#3b82f6', [ 59, 130, 246]],
  aggression: ['#f97316', [249, 115,  22]],
  serenity:   ['#10b981', [ 16, 185, 129]],
  nostalgia:  ['#d97706', [217, 119,   6]],
  sensuality: ['#ec4899', [236,  72, 153]],
  power:      ['#8b5cf6', [139,  92, 246]],
};

export const clamp01 = (v: number) => Math.max(0, Math.min(1, v));
export const lerp    = (a: number, b: number, t: number) => a + (b - a) * clamp01(t);

export function valenceRgb(v: number): [number, number, number] {
  return [
    Math.round(lerp(59,  251, v)),
    Math.round(lerp(130, 191, v)),
    Math.round(lerp(246,  36, v)),
  ];
}

export function cosineSim(
  a: Record<string, number>,
  b: Record<string, number>,
  dims: string[],
): number {
  let dot = 0, magA = 0, magB = 0;
  for (const d of dims) {
    const av = a[d] ?? 0, bv = b[d] ?? 0;
    dot += av * bv; magA += av * av; magB += bv * bv;
  }
  const denom = Math.sqrt(magA * magB);
  return denom > 0 ? dot / denom : 0;
}

export interface Pole {
  name: string;
  x: number;
  y: number;
  color: string;
  rgb: [number, number, number];
}

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
  const bv = Math.round(hue2rgb(h - 1/3) * 255);
  return [`rgb(${r},${g},${bv})`, [r, g, bv]];
}

export function buildPoles(emotionKeys: string[]): Pole[] {
  return emotionKeys.map((name, i) => {
    const angle = (i / emotionKeys.length) * 2 * Math.PI - Math.PI / 2;
    const [color, rgb] = getPoleColor(name, i);
    return { name, x: POLE_ORBIT * Math.cos(angle), y: POLE_ORBIT * Math.sin(angle), color, rgb };
  });
}
