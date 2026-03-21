import { describe, it, expect } from 'vitest';
import { computeZoneSuggestions, generateDescriptiveSuggestions } from './labelSuggestions';

// ─── computeZoneSuggestions ───────────────────────────────────────────────────

describe('computeZoneSuggestions', () => {
  it('returns all 8 zones', () => {
    const results = computeZoneSuggestions({});
    expect(results).toHaveLength(8);
    const zones = results.map(r => r.zone);
    expect(zones).toContain('euphoria');
    expect(zones).toContain('power');
    expect(zones).toContain('serenity');
    expect(zones).toContain('melancholy');
    expect(zones).toContain('nostalgia');
    expect(zones).toContain('sensuality');
    expect(zones).toContain('tension');
    expect(zones).toContain('aggression');
  });

  it('returns results sorted descending by score', () => {
    const results = computeZoneSuggestions({ energy: 0.9, valence: 0.9, danceability: 0.95 });
    for (let i = 1; i < results.length; i++) {
      expect(results[i - 1].score).toBeGreaterThanOrEqual(results[i].score);
    }
  });

  it('all scores are clamped to [0, 1]', () => {
    const results = computeZoneSuggestions({ energy: 1, valence: 1, danceability: 1, acousticness: 1 });
    for (const r of results) {
      expect(r.score).toBeGreaterThanOrEqual(0);
      expect(r.score).toBeLessThanOrEqual(1);
    }
  });

  it('high energy + high valence + high dance → euphoria first', () => {
    const results = computeZoneSuggestions({ energy: 0.95, valence: 0.95, danceability: 0.95, acousticness: 0.05 });
    expect(results[0].zone).toBe('euphoria');
  });

  it('low energy + high acousticness → serenity near the top', () => {
    const results = computeZoneSuggestions({ energy: 0.1, acousticness: 0.9, danceability: 0.1, valence: 0.6 });
    const top3 = results.slice(0, 3).map(r => r.zone);
    expect(top3).toContain('serenity');
  });

  it('low valence + high acousticness + low energy → melancholy or nostalgia near top', () => {
    const results = computeZoneSuggestions({ energy: 0.15, valence: 0.1, acousticness: 0.85, danceability: 0.2 });
    const top3 = results.slice(0, 3).map(r => r.zone);
    const hasMelOrNos = top3.includes('melancholy') || top3.includes('nostalgia');
    expect(hasMelOrNos).toBe(true);
  });

  it('high energy + low valence → tension or aggression near top', () => {
    const results = computeZoneSuggestions({ energy: 0.95, valence: 0.05, danceability: 0.5, acousticness: 0.05 });
    const top3 = results.slice(0, 3).map(r => r.zone);
    const hasTenOrAgg = top3.includes('tension') || top3.includes('aggression');
    expect(hasTenOrAgg).toBe(true);
  });

  it('handles undefined features with defaults (no crash)', () => {
    expect(() => computeZoneSuggestions({})).not.toThrow();
    const results = computeZoneSuggestions({});
    expect(results).toHaveLength(8);
  });

  it('high speechiness contributes to aggression score', () => {
    const withSpeech = computeZoneSuggestions({ energy: 0.8, speechiness: 0.9, valence: 0.2 });
    const withoutSpeech = computeZoneSuggestions({ energy: 0.8, speechiness: 0.0, valence: 0.2 });
    const aggWith = withSpeech.find(r => r.zone === 'aggression')!.score;
    const aggWithout = withoutSpeech.find(r => r.zone === 'aggression')!.score;
    expect(aggWith).toBeGreaterThan(aggWithout);
  });
});

// ─── generateDescriptiveSuggestions ──────────────────────────────────────────

describe('generateDescriptiveSuggestions', () => {
  it('always returns at least one label (tempo is always picked)', () => {
    const labels = generateDescriptiveSuggestions({});
    expect(labels.length).toBeGreaterThanOrEqual(1);
  });

  it('deduplicates repeated labels', () => {
    const labels = generateDescriptiveSuggestions({ bpm: 120, energy: 0.9, valence: 0.9, danceability: 0.9 });
    const unique = new Set(labels);
    expect(labels.length).toBe(unique.size);
  });

  // ── Tempo rules ──────────────────────────────────────────────────────────

  it('bpm < 75 → slow burn', () => {
    expect(generateDescriptiveSuggestions({ bpm: 60 })).toContain('slow burn');
  });

  it('75 ≤ bpm < 95 → downtempo', () => {
    expect(generateDescriptiveSuggestions({ bpm: 85 })).toContain('downtempo');
  });

  it('95 ≤ bpm < 115 → mid-tempo', () => {
    expect(generateDescriptiveSuggestions({ bpm: 105 })).toContain('mid-tempo');
  });

  it('115 ≤ bpm < 125 → house tempo', () => {
    expect(generateDescriptiveSuggestions({ bpm: 120 })).toContain('house tempo');
  });

  it('125 ≤ bpm < 135 → peak time', () => {
    expect(generateDescriptiveSuggestions({ bpm: 130 })).toContain('peak time');
  });

  it('135 ≤ bpm < 150 → techno', () => {
    expect(generateDescriptiveSuggestions({ bpm: 140 })).toContain('techno');
  });

  it('bpm ≥ 150 → hard & fast', () => {
    expect(generateDescriptiveSuggestions({ bpm: 160 })).toContain('hard & fast');
  });

  // ── Energy + texture rules ───────────────────────────────────────────────

  it('high energy + low acousticness → club ready', () => {
    const labels = generateDescriptiveSuggestions({ energy: 0.9, acousticness: 0.1 });
    expect(labels).toContain('club ready');
  });

  it('low energy + high acousticness → acoustic chill', () => {
    const labels = generateDescriptiveSuggestions({ energy: 0.2, acousticness: 0.7 });
    expect(labels).toContain('acoustic chill');
  });

  it('low energy + low acousticness → dark ambient', () => {
    const labels = generateDescriptiveSuggestions({ energy: 0.2, acousticness: 0.2 });
    expect(labels).toContain('dark ambient');
  });

  it('high energy + fast bpm + low valence → dark dance floor', () => {
    const labels = generateDescriptiveSuggestions({ energy: 0.8, bpm: 135, valence: 0.3 });
    expect(labels).toContain('dark dance floor');
  });

  it('high energy + fast bpm + high valence → euphoric peak', () => {
    const labels = generateDescriptiveSuggestions({ energy: 0.8, bpm: 135, valence: 0.8 });
    expect(labels).toContain('euphoric peak');
  });

  it('high energy + very fast bpm → rave energy', () => {
    const labels = generateDescriptiveSuggestions({ energy: 0.7, bpm: 145 });
    expect(labels).toContain('rave energy');
  });

  // ── Mood rules ───────────────────────────────────────────────────────────

  it('high valence → feel good', () => {
    expect(generateDescriptiveSuggestions({ valence: 0.9 })).toContain('feel good');
  });

  it('very low valence → deep & dark', () => {
    expect(generateDescriptiveSuggestions({ valence: 0.1 })).toContain('deep & dark');
  });

  it('high valence + high energy → uplifting', () => {
    expect(generateDescriptiveSuggestions({ valence: 0.7, energy: 0.7 })).toContain('uplifting');
  });

  it('low valence + high energy → bittersweet driving', () => {
    expect(generateDescriptiveSuggestions({ valence: 0.3, energy: 0.6 })).toContain('bittersweet driving');
  });

  it('low valence + low energy → introspective', () => {
    expect(generateDescriptiveSuggestions({ valence: 0.3, energy: 0.3 })).toContain('introspective');
  });

  // ── Danceability rules ───────────────────────────────────────────────────

  it('very high danceability → floor filler', () => {
    expect(generateDescriptiveSuggestions({ danceability: 0.9 })).toContain('floor filler');
  });

  it('low dance + low energy → listening, not dancing', () => {
    expect(generateDescriptiveSuggestions({ danceability: 0.2, energy: 0.3 })).toContain('listening, not dancing');
  });

  it('high dance + low energy → subtle groove', () => {
    expect(generateDescriptiveSuggestions({ danceability: 0.8, energy: 0.4 })).toContain('subtle groove');
  });

  it('high dance + house tempo bpm → deep house groove', () => {
    expect(generateDescriptiveSuggestions({ danceability: 0.8, bpm: 125, energy: 0.4 })).toContain('deep house groove');
  });

  // ── Organic / live rules ─────────────────────────────────────────────────

  it('very high acousticness → fully acoustic', () => {
    expect(generateDescriptiveSuggestions({ acousticness: 0.9 })).toContain('fully acoustic');
  });

  it('high liveness → live recording energy', () => {
    expect(generateDescriptiveSuggestions({ liveness: 0.7 })).toContain('live recording energy');
  });

  it('moderate liveness + moderate acousticness → live feel', () => {
    expect(generateDescriptiveSuggestions({ liveness: 0.5, acousticness: 0.5 })).toContain('live feel');
  });

  // ── Vocal rules ──────────────────────────────────────────────────────────

  it('high speechiness → vocal heavy', () => {
    expect(generateDescriptiveSuggestions({ speechiness: 0.7 })).toContain('vocal heavy');
  });

  it('very low speechiness → pure instrumental', () => {
    expect(generateDescriptiveSuggestions({ speechiness: 0.05 })).toContain('pure instrumental');
  });

  it('moderate speechiness → lyrical', () => {
    expect(generateDescriptiveSuggestions({ speechiness: 0.4 })).toContain('lyrical');
  });

  // ── Set context rules ────────────────────────────────────────────────────

  it('low energy + low danceability → opening / closing', () => {
    expect(generateDescriptiveSuggestions({ energy: 0.3, danceability: 0.3 })).toContain('opening / closing');
  });

  it('high energy + high danceability → main set', () => {
    expect(generateDescriptiveSuggestions({ energy: 0.8, danceability: 0.8 })).toContain('main set');
  });

  it('low energy + positive valence + moderate acousticness → sunset / after hours', () => {
    expect(generateDescriptiveSuggestions({ energy: 0.4, valence: 0.6, acousticness: 0.4 })).toContain('sunset / after hours');
  });

  it('very low energy + low valence → late night', () => {
    expect(generateDescriptiveSuggestions({ energy: 0.25, valence: 0.35 })).toContain('late night');
  });

  // ── Combined / integration ───────────────────────────────────────────────

  it('techno track profile: bpm=142, energy=0.9, valence=0.3, dance=0.8, acoustic=0.1', () => {
    const labels = generateDescriptiveSuggestions({
      bpm: 142, energy: 0.9, valence: 0.3, danceability: 0.8, acousticness: 0.1,
    });
    expect(labels).toContain('techno');
    expect(labels).toContain('club ready');
    expect(labels).toContain('rave energy');
    expect(labels).toContain('main set');
  });

  it('ambient/chill profile: bpm=70, energy=0.15, valence=0.4, acoustic=0.85, dance=0.2', () => {
    const labels = generateDescriptiveSuggestions({
      bpm: 70, energy: 0.15, valence: 0.4, acousticness: 0.85, danceability: 0.2,
    });
    expect(labels).toContain('slow burn');
    expect(labels).toContain('acoustic chill');
    expect(labels).toContain('fully acoustic');
    expect(labels).toContain('opening / closing');
  });

  it('handles all undefined features (defaults only)', () => {
    expect(() => generateDescriptiveSuggestions({})).not.toThrow();
    // With all defaults: bpm=120 → house tempo, energy=0.5, valence=0.5 — no extreme rules fire
    const labels = generateDescriptiveSuggestions({});
    expect(labels).toContain('house tempo');
  });
});
