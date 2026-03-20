import { describe, it, expect } from 'vitest';
import {
  POLE_ORBIT,
  NEBULA_R,
  ATTRACT,
  REPEL,
  CENTER_PULL,
  DAMPING,
  NODE_MIN_R,
  NODE_MAX_R,
  KNOWN_PALETTE,
  clamp01,
  lerp,
  valenceRgb,
  cosineSim,
  buildPoles,
} from './emotionGraphUtils';

// ─── Constants sanity checks ──────────────────────────────────────────────────

describe('physics constants', () => {
  it('POLE_ORBIT > NEBULA_R so nebula stays inside pole circle', () => {
    expect(POLE_ORBIT).toBeGreaterThan(NEBULA_R);
  });

  it('ATTRACT is a small positive multiplier', () => {
    expect(ATTRACT).toBeGreaterThan(0);
    expect(ATTRACT).toBeLessThan(1);
  });

  it('REPEL is a large positive force', () => {
    expect(REPEL).toBeGreaterThan(100);
  });

  it('CENTER_PULL is small (much less than ATTRACT)', () => {
    expect(CENTER_PULL).toBeGreaterThan(0);
    expect(CENTER_PULL).toBeLessThan(ATTRACT);
  });

  it('DAMPING is in (0, 1) for stable settling', () => {
    expect(DAMPING).toBeGreaterThan(0);
    expect(DAMPING).toBeLessThan(1);
  });

  it('NODE_MIN_R < NODE_MAX_R', () => {
    expect(NODE_MIN_R).toBeLessThan(NODE_MAX_R);
    expect(NODE_MIN_R).toBeGreaterThan(0);
  });

  it('KNOWN_PALETTE has 8 entries', () => {
    expect(Object.keys(KNOWN_PALETTE)).toHaveLength(8);
  });
});

// ─── clamp01 ─────────────────────────────────────────────────────────────────

describe('clamp01', () => {
  it('passes through values inside [0, 1]', () => {
    expect(clamp01(0)).toBe(0);
    expect(clamp01(0.5)).toBe(0.5);
    expect(clamp01(1)).toBe(1);
  });

  it('clamps negative values to 0', () => {
    expect(clamp01(-0.1)).toBe(0);
    expect(clamp01(-100)).toBe(0);
  });

  it('clamps values above 1 to 1', () => {
    expect(clamp01(1.1)).toBe(1);
    expect(clamp01(999)).toBe(1);
  });
});

// ─── lerp ────────────────────────────────────────────────────────────────────

describe('lerp', () => {
  it('returns a at t=0', () => {
    expect(lerp(10, 20, 0)).toBe(10);
  });

  it('returns b at t=1', () => {
    expect(lerp(10, 20, 1)).toBe(20);
  });

  it('interpolates midpoint correctly', () => {
    expect(lerp(0, 100, 0.5)).toBe(50);
  });

  it('works with negative ranges', () => {
    expect(lerp(-10, 10, 0.5)).toBe(0);
  });

  it('clamps t below 0', () => {
    expect(lerp(10, 20, -1)).toBe(10); // t clamped to 0 → returns a
  });

  it('clamps t above 1', () => {
    expect(lerp(10, 20, 2)).toBe(20); // t clamped to 1 → returns b
  });
});

// ─── valenceRgb ──────────────────────────────────────────────────────────────

describe('valenceRgb', () => {
  it('returns cold-blue [59,130,246] at valence=0', () => {
    const [r, g, b] = valenceRgb(0);
    expect(r).toBe(59);
    expect(g).toBe(130);
    expect(b).toBe(246);
  });

  it('returns warm-gold [251,191,36] at valence=1', () => {
    const [r, g, b] = valenceRgb(1);
    expect(r).toBe(251);
    expect(g).toBe(191);
    expect(b).toBe(36);
  });

  it('returns a value in [0,255] for every channel at midpoint', () => {
    const rgb = valenceRgb(0.5);
    for (const channel of rgb) {
      expect(channel).toBeGreaterThanOrEqual(0);
      expect(channel).toBeLessThanOrEqual(255);
    }
  });

  it('returns integers', () => {
    const rgb = valenceRgb(0.37);
    for (const channel of rgb) {
      expect(Number.isInteger(channel)).toBe(true);
    }
  });

  it('r and g increase as valence increases (warm shift)', () => {
    const [r0] = valenceRgb(0);
    const [r1] = valenceRgb(1);
    expect(r1).toBeGreaterThan(r0);
  });

  it('b decreases as valence increases (less blue)', () => {
    const [,, b0] = valenceRgb(0);
    const [,, b1] = valenceRgb(1);
    expect(b1).toBeLessThan(b0);
  });
});

// ─── cosineSim ───────────────────────────────────────────────────────────────

describe('cosineSim', () => {
  const dims = ['a', 'b', 'c'];

  it('returns 1.0 for identical vectors', () => {
    const v = { a: 1, b: 2, c: 3 };
    expect(cosineSim(v, v, dims)).toBeCloseTo(1.0);
  });

  it('returns 0.0 for orthogonal vectors', () => {
    const u = { a: 1, b: 0, c: 0 };
    const v = { a: 0, b: 1, c: 0 };
    expect(cosineSim(u, v, dims)).toBeCloseTo(0.0);
  });

  it('returns 0.0 when one vector is zero', () => {
    const u = { a: 1, b: 2, c: 3 };
    const zero = { a: 0, b: 0, c: 0 };
    expect(cosineSim(u, zero, dims)).toBe(0);
  });

  it('returns 0.0 when both vectors are zero', () => {
    const zero = { a: 0, b: 0, c: 0 };
    expect(cosineSim(zero, zero, dims)).toBe(0);
  });

  it('returns same value when a and b are swapped (symmetry)', () => {
    const u = { a: 0.8, b: 0.2, c: 0.5 };
    const v = { a: 0.3, b: 0.9, c: 0.1 };
    expect(cosineSim(u, v, dims)).toBeCloseTo(cosineSim(v, u, dims));
  });

  it('treats missing keys as 0', () => {
    const u = { a: 1, b: 0, c: 0 };
    const v = {}; // all dims missing → zero vector
    expect(cosineSim(u, v, dims)).toBe(0);
  });

  it('result is in [-1, 1] for any non-negative emotion values', () => {
    const u = { a: 0.9, b: 0.1, c: 0.0 };
    const v = { a: 0.2, b: 0.7, c: 0.1 };
    const sim = cosineSim(u, v, dims);
    expect(sim).toBeGreaterThanOrEqual(-1);
    expect(sim).toBeLessThanOrEqual(1);
  });

  it('ignores extra keys on the objects that are not in dims', () => {
    const u = { a: 1, b: 0, c: 0, extra: 999 };
    const v = { a: 1, b: 0, c: 0 };
    expect(cosineSim(u, v, dims)).toBeCloseTo(1.0);
  });
});

// ─── buildPoles ──────────────────────────────────────────────────────────────

describe('buildPoles', () => {
  const emotions = ['euphoria', 'tension', 'melancholy', 'aggression', 'serenity', 'nostalgia', 'sensuality', 'power'];

  it('returns the same number of poles as input keys', () => {
    const poles = buildPoles(emotions);
    expect(poles).toHaveLength(emotions.length);
  });

  it('each pole name matches input key', () => {
    const poles = buildPoles(emotions);
    poles.forEach((p, i) => expect(p.name).toBe(emotions[i]));
  });

  it('poles are arranged on a circle of radius POLE_ORBIT', () => {
    const poles = buildPoles(emotions);
    for (const p of poles) {
      const dist = Math.sqrt(p.x * p.x + p.y * p.y);
      expect(dist).toBeCloseTo(POLE_ORBIT, 6);
    }
  });

  it('first pole is at angle -π/2 (top of circle)', () => {
    const [first] = buildPoles(emotions);
    expect(first.x).toBeCloseTo(0, 5);
    expect(first.y).toBeCloseTo(-POLE_ORBIT, 5);
  });

  it('uses KNOWN_PALETTE colors for recognised emotion names', () => {
    const poles = buildPoles(emotions);
    const euphoria = poles.find(p => p.name === 'euphoria')!;
    expect(euphoria.color).toBe(KNOWN_PALETTE['euphoria'][0]);
    expect(euphoria.rgb).toEqual(KNOWN_PALETTE['euphoria'][1]);
  });

  it('generates a colour for unknown emotion names without throwing', () => {
    const poles = buildPoles(['unknown_emotion']);
    expect(poles[0].color).toBeTruthy();
    expect(poles[0].rgb).toHaveLength(3);
  });

  it('handles a single pole at the top of the circle', () => {
    const [p] = buildPoles(['solo']);
    expect(p.x).toBeCloseTo(0, 5);
    expect(p.y).toBeCloseTo(-POLE_ORBIT, 5);
  });

  it('handles empty input', () => {
    expect(buildPoles([])).toEqual([]);
  });
});
