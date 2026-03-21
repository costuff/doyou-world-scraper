/**
 * labelSuggestions.ts
 *
 * Pure functions for computing label suggestions from audio features.
 * Extracted from CurateTab.tsx so they can be independently unit-tested.
 */

export interface AudioFeatures {
  bpm?: number;
  key?: string;
  mode?: string;
  energy?: number;
  valence?: number;
  danceability?: number;
  acousticness?: number;
  speechiness?: number;
  liveness?: number;
  duration_seconds?: number;
}

/**
 * Mirrors the Go ComputeEmotions formula.
 * Returns all 8 emotion zones ranked by score (highest first).
 */
export function computeZoneSuggestions(f: AudioFeatures): { zone: string; score: number }[] {
  const clamp = (v: number) => Math.max(0, Math.min(1, v));
  const p = (x: number | undefined, d: number) => x ?? d;

  const energy   = p(f.energy,       0.5);
  const valence  = p(f.valence,      0.5);
  const dance    = p(f.danceability, 0.5);
  const acoustic = p(f.acousticness, 0.5);
  const speech   = p(f.speechiness,  0.3);
  const bpmNorm  = f.bpm != null ? Math.max(0, Math.min(1, (f.bpm - 60) / 140)) : 0.5;

  return [
    { zone: 'euphoria',   score: clamp(0.4 * valence + 0.3 * dance    + 0.3 * energy) },
    { zone: 'power',      score: clamp(0.5 * energy  + 0.3 * (1 - acoustic) + 0.2 * bpmNorm) },
    { zone: 'serenity',   score: clamp(0.5 * (1 - energy) + 0.3 * acoustic + 0.2 * (1 - dance)) },
    { zone: 'melancholy', score: clamp(0.4 * (1 - valence) + 0.3 * acoustic + 0.3 * (1 - energy)) },
    { zone: 'nostalgia',  score: clamp(0.4 * acoustic + 0.3 * (1 - valence) + 0.3 * (1 - bpmNorm)) },
    { zone: 'sensuality', score: clamp(0.4 * (1 - energy) + 0.3 * (1 - speech) + 0.3 * valence) },
    { zone: 'tension',    score: clamp(0.5 * energy + 0.3 * (1 - valence) + 0.2 * (1 - acoustic)) },
    { zone: 'aggression', score: clamp(0.5 * energy + 0.3 * speech + 0.2 * (1 - valence)) },
  ].sort((a, b) => b.score - a.score);
}

/**
 * Returns descriptive, genre/mood labels derived from feature combination rules.
 * Each returned string is a standalone label suggestion (not an emotion zone).
 */
export function generateDescriptiveSuggestions(f: AudioFeatures): string[] {
  const p = (x: number | undefined, d: number) => x ?? d;
  const energy   = p(f.energy,       0.5);
  const valence  = p(f.valence,      0.5);
  const dance    = p(f.danceability, 0.5);
  const acoustic = p(f.acousticness, 0.5);
  const speech   = p(f.speechiness,  0.3);
  const liveness = p(f.liveness,     0.2);
  const bpm      = f.bpm ?? 120;
  const labels: string[] = [];

  // ── Tempo-based ──────────────────────────────────────────────────────────
  if (bpm < 75) labels.push('slow burn');
  else if (bpm < 95) labels.push('downtempo');
  else if (bpm < 115) labels.push('mid-tempo');
  else if (bpm < 125) labels.push('house tempo');
  else if (bpm < 135) labels.push('peak time');
  else if (bpm < 150) labels.push('techno');
  else labels.push('hard & fast');

  // ── Energy + texture ─────────────────────────────────────────────────────
  if (energy > 0.8 && acoustic < 0.2) labels.push('club ready');
  if (energy < 0.3 && acoustic > 0.5) labels.push('acoustic chill');
  if (energy < 0.3 && acoustic < 0.3) labels.push('dark ambient');
  if (energy < 0.4 && acoustic > 0.6) labels.push('organic texture');
  if (energy > 0.7 && bpm > 128 && valence < 0.4) labels.push('dark dance floor');
  if (energy > 0.7 && bpm > 128 && valence > 0.6) labels.push('euphoric peak');
  if (energy > 0.6 && bpm > 140) labels.push('rave energy');
  if (energy > 0.5 && acoustic > 0.5) labels.push('warm live feel');

  // ── Mood ─────────────────────────────────────────────────────────────────
  if (valence > 0.8) labels.push('feel good');
  if (valence < 0.2) labels.push('deep & dark');
  if (valence > 0.6 && energy > 0.6) labels.push('uplifting');
  if (valence < 0.35 && energy > 0.55) labels.push('bittersweet driving');
  if (valence < 0.35 && energy < 0.4) labels.push('introspective');

  // ── Danceability ─────────────────────────────────────────────────────────
  if (dance > 0.85) labels.push('floor filler');
  if (dance < 0.3 && energy < 0.4) labels.push('listening, not dancing');
  if (dance > 0.7 && energy < 0.5) labels.push('subtle groove');
  if (dance > 0.7 && bpm > 120 && bpm < 130) labels.push('deep house groove');

  // ── Organic / live ───────────────────────────────────────────────────────
  if (acoustic > 0.8) labels.push('fully acoustic');
  if (liveness > 0.6) labels.push('live recording energy');
  if (liveness > 0.4 && acoustic > 0.4) labels.push('live feel');

  // ── Vocal presence ───────────────────────────────────────────────────────
  if (speech > 0.55) labels.push('vocal heavy');
  if (speech < 0.1) labels.push('pure instrumental');
  if (speech > 0.3 && speech < 0.55) labels.push('lyrical');

  // ── Set context ──────────────────────────────────────────────────────────
  if (energy < 0.4 && dance < 0.45) labels.push('opening / closing');
  if (energy > 0.75 && dance > 0.75) labels.push('main set');
  if (energy < 0.5 && valence > 0.5 && acoustic > 0.3) labels.push('sunset / after hours');
  if (energy < 0.35 && valence < 0.45) labels.push('late night');

  return [...new Set(labels)]; // deduplicate
}
