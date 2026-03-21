/**
 * CurateTab — AI taxonomy labeling + active-learning curation interface.
 *
 * Left panel: queue of completed tracks with auto-proposed emotion labels.
 * Right panel: taxonomy tree + curation actions for the selected track.
 *
 * Workflow:
 *  1. Load completed tracks (GET /analytics-api/api/v1/tracks?status=completed)
 *  2. For each track, fetch its auto label (GET /analytics-api/api/v1/tracks/:id/label)
 *  3. Display taxonomy tree (GET /analytics-api/api/v1/taxonomy)
 *  4. User accepts the auto label, adjusts it, or adds a new node
 *  5. Save via POST /analytics-api/api/v1/tracks/:id/label
 */

import { useState, useEffect, useCallback } from 'react';
import axios from 'axios';

const ANALYTICS = import.meta.env.VITE_ANALYTICS_API_URL || '/analytics-api';

// ─── Types ────────────────────────────────────────────────────────────────────

interface Track {
  id: number;
  file_name: string;
  analysis_status: string;
  audio_features?: {
    bpm?: number;
    key?: string;
    mode?: string;
    energy?: number;
    valence?: number;
    danceability?: number;
    duration_seconds?: number;
  };
}

interface TrackLabel {
  track_id: number;
  label_path: string[];
  auto_label_path: string[];
  curated_by?: string;
  notes?: string;
}

interface TaxonomyNode {
  id: number;
  parent_id?: number;
  name: string;
  description?: string;
  source: 'auto' | 'user';
  children?: TaxonomyNode[];
}

const ZONE_COLORS: Record<string, string> = {
  euphoria: '#fbbf24', power: '#f97316', aggression: '#dc2626',
  tension: '#ef4444', melancholy: '#6366f1', nostalgia: '#a78bfa',
  serenity: '#34d399', sensuality: '#f472b6',
};

// ─── Component ────────────────────────────────────────────────────────────────

export default function CurateTab() {
  const [tracks, setTracks]         = useState<Track[]>([]);
  const [labels, setLabels]         = useState<Record<number, TrackLabel>>({});
  const [taxonomy, setTaxonomy]     = useState<TaxonomyNode[]>([]);
  const [selected, setSelected]     = useState<Track | null>(null);
  const [curLabel, setCurLabel]     = useState<TrackLabel | null>(null);
  const [pickedPath, setPickedPath] = useState<string[]>([]);
  const [notes, setNotes]           = useState('');
  const [saving, setSaving]         = useState(false);
  const [saved, setSaved]           = useState<Record<number, boolean>>({});
  const [loading, setLoading]       = useState(true);
  const [error, setError]           = useState<string | null>(null);
  const [newNodeName, setNewNodeName] = useState('');
  const [addingTo, setAddingTo]     = useState<TaxonomyNode | null>(null);

  // ── Load tracks + taxonomy on mount ─────────────────────────────────────────
  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const [tracksRes, taxRes] = await Promise.all([
          axios.get(`${ANALYTICS}/api/v1/tracks?status=completed&limit=100`),
          axios.get(`${ANALYTICS}/api/v1/taxonomy`),
        ]);
        if (cancelled) return;
        setTracks(tracksRes.data.tracks ?? []);
        setTaxonomy(taxRes.data.nodes ?? []);
        setLoading(false);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load');
          setLoading(false);
        }
      }
    }
    load();
    return () => { cancelled = true; };
  }, []);

  // ── Select a track → load its label ─────────────────────────────────────────
  const selectTrack = useCallback(async (track: Track) => {
    setSelected(track);
    setCurLabel(null);
    setPickedPath([]);
    setNotes('');

    // Check cache
    if (labels[track.id]) {
      const lbl = labels[track.id];
      setCurLabel(lbl);
      setPickedPath(lbl.label_path.length ? lbl.label_path : lbl.auto_label_path);
      return;
    }

    try {
      const { data } = await axios.get<TrackLabel>(`${ANALYTICS}/api/v1/tracks/${track.id}/label`);
      setLabels(prev => ({ ...prev, [track.id]: data }));
      setCurLabel(data);
      setPickedPath(data.label_path.length ? data.label_path : data.auto_label_path);
    } catch {
      // No label yet — show empty
      const empty: TrackLabel = { track_id: track.id, label_path: [], auto_label_path: [] };
      setCurLabel(empty);
      setPickedPath([]);
    }
  }, [labels]);

  // ── Save label ───────────────────────────────────────────────────────────────
  const saveLabel = useCallback(async () => {
    if (!selected) return;
    setSaving(true);
    try {
      const { data } = await axios.post<TrackLabel>(
        `${ANALYTICS}/api/v1/tracks/${selected.id}/label`,
        { label_path: pickedPath, curated_by: 'user', notes },
      );
      setLabels(prev => ({ ...prev, [selected.id]: data }));
      setCurLabel(data);
      setSaved(prev => ({ ...prev, [selected.id]: true }));
    } catch (err) {
      alert('Failed to save: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      setSaving(false);
    }
  }, [selected, pickedPath, notes]);

  // ── Add new taxonomy node ────────────────────────────────────────────────────
  const addNode = useCallback(async () => {
    const name = newNodeName.trim();
    if (!name) return;
    try {
      const { data } = await axios.post<TaxonomyNode>(`${ANALYTICS}/api/v1/taxonomy/nodes`, {
        parent_id: addingTo?.id,
        name,
        source: 'user',
      });
      setTaxonomy(prev => {
        if (!addingTo) return [...prev, data];
        const inject = (nodes: TaxonomyNode[]): TaxonomyNode[] =>
          nodes.map(n => n.id === addingTo.id
            ? { ...n, children: [...(n.children ?? []), data] }
            : { ...n, children: n.children ? inject(n.children) : undefined }
          );
        return inject(prev);
      });
      setNewNodeName('');
      setAddingTo(null);
    } catch (err) {
      alert('Failed to add node: ' + (err instanceof Error ? err.message : String(err)));
    }
  }, [newNodeName, addingTo]);

  // ── UI helpers ───────────────────────────────────────────────────────────────
  const displayName = (track: Track) =>
    track.file_name
      .replace(/\s*\[[\w-]+\](\.[^.]+)?$/, '')
      .replace(/\.[^.]+$/, '')
      .trim() || track.file_name;

  const stats = {
    total: tracks.length,
    curated: Object.keys(saved).length,
    remaining: tracks.length - Object.keys(saved).length,
  };

  // ─── Render ─────────────────────────────────────────────────────────────────
  if (loading) return (
    <div className="flex items-center justify-center h-96 text-slate-400 text-sm animate-pulse">
      Loading tracks and taxonomy…
    </div>
  );

  if (error) return (
    <div className="flex items-center justify-center h-96 text-red-400 text-sm">
      Analytics API unavailable: {error}
    </div>
  );

  return (
    <div className="flex gap-6 h-full max-h-[calc(100vh-130px)]">
      {/* ── Left: track queue ── */}
      <div className="w-80 flex-shrink-0 flex flex-col">
        <div className="bg-slate-800 rounded-xl p-4 mb-3 flex gap-4 text-center">
          {[
            ['Total', stats.total],
            ['Curated', stats.curated],
            ['Remaining', stats.remaining],
          ].map(([label, n]) => (
            <div key={label as string} className="flex-1">
              <div className="text-lg font-bold text-white">{n}</div>
              <div className="text-xs text-slate-400">{label}</div>
            </div>
          ))}
        </div>

        <div className="bg-slate-800 rounded-xl flex-1 overflow-y-auto">
          <div className="p-3 border-b border-slate-700">
            <p className="text-xs font-bold text-slate-400 uppercase tracking-widest">Track Queue</p>
          </div>
          {tracks.length === 0 ? (
            <p className="text-slate-500 text-sm text-center py-8">No analyzed tracks found.</p>
          ) : tracks.map(track => {
            const lbl = labels[track.id];
            const isSaved = saved[track.id];
            const autoZone = lbl?.auto_label_path?.[0];
            const hasUserLabel = lbl && lbl.label_path.length > 0;
            return (
              <button key={track.id}
                onClick={() => selectTrack(track)}
                className={`w-full text-left flex items-start gap-3 px-3 py-2.5 border-b border-slate-700/50 transition text-sm ${
                  selected?.id === track.id ? 'bg-slate-700' : 'hover:bg-slate-700/50'
                }`}
              >
                <div className="flex-1 min-w-0">
                  <p className="text-white truncate text-xs font-medium">{displayName(track)}</p>
                  <div className="flex items-center gap-1.5 mt-0.5">
                    {autoZone && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full font-medium"
                        style={{ background: ZONE_COLORS[autoZone] ?? '#475569', color: '#fff' }}>
                        {autoZone}
                      </span>
                    )}
                    {hasUserLabel && (
                      <span className="text-[10px] text-green-400">✓ curated</span>
                    )}
                  </div>
                </div>
                {isSaved && <span className="text-green-400 text-xs flex-shrink-0 mt-0.5">✓</span>}
              </button>
            );
          })}
        </div>
      </div>

      {/* ── Right: curation panel ── */}
      <div className="flex-1 min-w-0 overflow-y-auto">
        {!selected ? (
          <div className="flex flex-col items-center justify-center h-full text-center gap-3 text-slate-500">
            <div className="text-4xl">🎵</div>
            <p className="text-sm">Select a track from the queue to curate its label.</p>
            <p className="text-xs max-w-xs">
              The system automatically proposes a label based on the dominant emotion zone.
              You can accept it, adjust it, or assign a more specific sub-category.
            </p>
          </div>
        ) : (
          <div className="space-y-5">
            {/* Track info */}
            <div className="bg-slate-800 rounded-xl p-5">
              <h2 className="font-bold text-white text-base mb-1 leading-tight">
                {displayName(selected)}
              </h2>
              <div className="flex flex-wrap gap-3 mt-2 text-xs text-slate-400">
                {selected.audio_features?.bpm != null && (
                  <span>BPM: <span className="text-slate-200 font-mono">{selected.audio_features.bpm.toFixed(0)}</span></span>
                )}
                {selected.audio_features?.key && (
                  <span>Key: <span className="text-slate-200 font-mono">{selected.audio_features.key} {selected.audio_features.mode}</span></span>
                )}
                {selected.audio_features?.energy != null && (
                  <span>Energy: <span className="text-slate-200 font-mono">{(selected.audio_features.energy * 100).toFixed(0)}%</span></span>
                )}
                {selected.audio_features?.valence != null && (
                  <span>Valence: <span className="text-slate-200 font-mono">{(selected.audio_features.valence * 100).toFixed(0)}%</span></span>
                )}
              </div>
            </div>

            {/* Current label */}
            <div className="bg-slate-800 rounded-xl p-5">
              <p className="text-xs font-bold text-slate-400 uppercase tracking-widest mb-3">Label Path</p>
              <div className="flex items-center gap-1.5 flex-wrap mb-3">
                {pickedPath.length === 0 ? (
                  <span className="text-slate-500 text-sm italic">No label assigned — select from the tree below</span>
                ) : (
                  pickedPath.map((seg, i) => (
                    <span key={i} className="flex items-center gap-1">
                      {i > 0 && <span className="text-slate-600">/</span>}
                      <span className="text-sm px-2 py-0.5 rounded-md"
                        style={{
                          background: i === 0 && ZONE_COLORS[seg] ? ZONE_COLORS[seg] + '33' : 'rgba(71,85,105,0.6)',
                          color: i === 0 && ZONE_COLORS[seg] ? ZONE_COLORS[seg] : '#e2e8f0',
                          border: `1px solid ${i === 0 && ZONE_COLORS[seg] ? ZONE_COLORS[seg] + '66' : 'rgba(71,85,105,0.8)'}`,
                        }}>
                        {seg}
                      </span>
                      <button onClick={() => setPickedPath(pickedPath.slice(0, i))}
                        className="text-slate-600 hover:text-red-400 text-xs transition" title="Remove from here">×</button>
                    </span>
                  ))
                )}
              </div>

              {curLabel?.auto_label_path && curLabel.auto_label_path.length > 0 && (
                <div className="text-xs text-slate-500 mb-3">
                  Auto-proposed: <span className="text-slate-300">{curLabel.auto_label_path.join(' / ')}</span>
                  <button
                    onClick={() => setPickedPath(curLabel.auto_label_path)}
                    className="ml-2 text-blue-400 hover:text-blue-300 transition underline underline-offset-2">
                    accept
                  </button>
                </div>
              )}

              <div>
                <label className="text-xs text-slate-400">Notes (optional)</label>
                <textarea
                  value={notes}
                  onChange={e => setNotes(e.target.value)}
                  className="w-full mt-1 bg-slate-700 text-slate-200 text-xs rounded-lg p-2 resize-none outline-none focus:ring-1 focus:ring-blue-500"
                  rows={2}
                  placeholder="e.g. melancholic slow burn, good for late night sets"
                />
              </div>

              <div className="flex gap-3 mt-3">
                <button onClick={saveLabel} disabled={saving || pickedPath.length === 0}
                  className="px-4 py-2 text-sm bg-green-600 hover:bg-green-500 disabled:opacity-50 rounded-lg font-semibold transition">
                  {saving ? 'Saving…' : '✓ Save Label'}
                </button>
                {saved[selected.id] && (
                  <span className="text-green-400 text-sm self-center">Saved!</span>
                )}
              </div>
            </div>

            {/* Taxonomy tree */}
            <div className="bg-slate-800 rounded-xl p-5">
              <p className="text-xs font-bold text-slate-400 uppercase tracking-widest mb-3">Taxonomy Tree</p>
              <p className="text-xs text-slate-500 mb-3">Click a node to append it to the label path.</p>

              <TaxonomyTree
                nodes={taxonomy}
                pickedPath={pickedPath}
                onPick={(node) => {
                  // Build the full path from root to this node
                  const path = buildPathTo(taxonomy, node.id);
                  if (path) setPickedPath(path.map(n => n.name));
                }}
                onAddChild={(node) => { setAddingTo(node); setNewNodeName(''); }}
              />

              {/* Add root node */}
              <div className="mt-3 pt-3 border-t border-slate-700">
                <p className="text-xs text-slate-400 mb-2">Add root-level node</p>
                <div className="flex gap-2">
                  <input
                    value={addingTo === null ? newNodeName : ''}
                    onChange={e => { setAddingTo(null); setNewNodeName(e.target.value); }}
                    onKeyDown={e => e.key === 'Enter' && addNode()}
                    placeholder="e.g. summer vibes"
                    className="flex-1 bg-slate-700 text-sm text-slate-200 rounded-lg px-3 py-1.5 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  <button onClick={addNode}
                    className="px-3 py-1.5 text-sm bg-blue-600 hover:bg-blue-500 rounded-lg transition">
                    Add
                  </button>
                </div>
              </div>

              {/* Add child node inline */}
              {addingTo && (
                <div className="mt-2 p-3 bg-slate-700 rounded-lg">
                  <p className="text-xs text-slate-400 mb-1.5">
                    Adding child of <span className="text-white font-semibold">{addingTo.name}</span>:
                  </p>
                  <div className="flex gap-2">
                    <input
                      value={newNodeName}
                      onChange={e => setNewNodeName(e.target.value)}
                      onKeyDown={e => e.key === 'Enter' && addNode()}
                      autoFocus
                      placeholder="e.g. rainy melancholy"
                      className="flex-1 bg-slate-600 text-sm text-slate-200 rounded-lg px-3 py-1.5 outline-none focus:ring-1 focus:ring-blue-500"
                    />
                    <button onClick={addNode}
                      className="px-3 py-1.5 text-sm bg-blue-600 hover:bg-blue-500 rounded-lg transition">
                      Add
                    </button>
                    <button onClick={() => { setAddingTo(null); setNewNodeName(''); }}
                      className="px-3 py-1.5 text-sm bg-slate-600 hover:bg-slate-500 rounded-lg transition">
                      ×
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Taxonomy Tree ────────────────────────────────────────────────────────────

function TaxonomyTree({
  nodes, pickedPath, onPick, onAddChild,
}: {
  nodes: TaxonomyNode[];
  pickedPath: string[];
  onPick: (node: TaxonomyNode) => void;
  onAddChild: (node: TaxonomyNode) => void;
}) {
  if (nodes.length === 0) return <p className="text-slate-500 text-xs">No nodes yet.</p>;
  return (
    <div className="space-y-1">
      {nodes.map(n => (
        <TaxonomyNodeRow key={n.id} node={n} depth={0} pickedPath={pickedPath} onPick={onPick} onAddChild={onAddChild} />
      ))}
    </div>
  );
}

function TaxonomyNodeRow({
  node, depth, pickedPath, onPick, onAddChild,
}: {
  node: TaxonomyNode;
  depth: number;
  pickedPath: string[];
  onPick: (node: TaxonomyNode) => void;
  onAddChild: (node: TaxonomyNode) => void;
}) {
  const [expanded, setExpanded] = useState(depth === 0);
  const isInPath = pickedPath.includes(node.name);
  const color = depth === 0 ? (ZONE_COLORS[node.name] ?? '#60a5fa') : '#94a3b8';

  return (
    <div>
      <div
        className="flex items-center gap-1.5 group"
        style={{ paddingLeft: `${depth * 16}px` }}
      >
        {(node.children?.length ?? 0) > 0 ? (
          <button onClick={() => setExpanded(e => !e)}
            className="text-slate-500 hover:text-slate-300 text-xs w-4 flex-shrink-0 transition">
            {expanded ? '▾' : '▸'}
          </button>
        ) : (
          <span className="w-4 flex-shrink-0" />
        )}

        <button
          onClick={() => onPick(node)}
          title={node.description}
          className={`flex-1 text-left text-sm py-0.5 rounded transition ${
            isInPath ? 'font-semibold' : 'text-slate-300 hover:text-white'
          }`}
          style={{ color: isInPath ? color : undefined }}
        >
          {depth === 0 && (
            <span className="inline-block w-2.5 h-2.5 rounded-full mr-1.5 align-middle"
              style={{ background: color }} />
          )}
          {node.name}
          {node.source === 'user' && (
            <span className="ml-1.5 text-[10px] text-slate-500 align-middle">user</span>
          )}
        </button>

        <button
          onClick={() => onAddChild(node)}
          title="Add child node"
          className="opacity-0 group-hover:opacity-100 text-slate-500 hover:text-blue-400 text-xs transition px-1">
          +
        </button>
      </div>

      {expanded && (node.children?.length ?? 0) > 0 && (
        <div>
          {node.children!.map(child => (
            <TaxonomyNodeRow key={child.id} node={child} depth={depth + 1}
              pickedPath={pickedPath} onPick={onPick} onAddChild={onAddChild} />
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function buildPathTo(nodes: TaxonomyNode[], targetId: number): TaxonomyNode[] | null {
  for (const n of nodes) {
    if (n.id === targetId) return [n];
    if (n.children) {
      const sub = buildPathTo(n.children, targetId);
      if (sub) return [n, ...sub];
    }
  }
  return null;
}
