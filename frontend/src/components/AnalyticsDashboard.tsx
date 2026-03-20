import { useEffect, useState } from 'react';
import axios from 'axios';
import apiService from '../services/api';
import type { AnalyzedTrack, AudioAnalysis } from '../types';

const ANALYTICS_API_URL = import.meta.env.VITE_ANALYTICS_API_URL || '/analytics-api';

interface EmotionDistribution {
  dimension: string;
  min: number;
  max: number;
  avg: number;
  count: number;
}

export const AnalyticsDashboard = () => {
  const [tracks, setTracks] = useState<AnalyzedTrack[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedTracks, setSelectedTracks] = useState<Set<string>>(new Set());
  const [currentTrack, setCurrentTrack] = useState<string | null>(null);
  const [audioElement] = useState(new Audio());
  const [emotionDistribution, setEmotionDistribution] = useState<EmotionDistribution[]>([]);

  const fetchAnalyzedTracks = async () => {
    try {
      setLoading(true);
      const [tracksResponse, emotionsResponse] = await Promise.allSettled([
        apiService.getAnalyzedTracks(),
        axios.get(`${ANALYTICS_API_URL}/api/v1/emotions/distribution`, { timeout: 5000 }),
      ]);

      if (tracksResponse.status === 'fulfilled' && tracksResponse.value?.data) {
        const apiTracks = tracksResponse.value.data.tracks || [];
        console.log(`Loaded ${apiTracks.length} of ${tracksResponse.value.data.total} analyzed tracks`);
        setTracks(
          apiTracks.map((track: any) => ({
            file_path: track.file_path,
            file_name: track.file_name,
            analysis: track.features,
          }))
        );
        setError(null);
      } else if (tracksResponse.status === 'rejected') {
        const err = tracksResponse.reason;
        setError(err?.message || 'Failed to load analyzed tracks from analytics service');
      }

      if (emotionsResponse.status === 'fulfilled') {
        setEmotionDistribution(emotionsResponse.value.data?.distribution || []);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAnalyzedTracks();
    return () => {
      audioElement.pause();
      audioElement.src = '';
    };
  }, []);

  const handleSelectAll = () => {
    if (selectedTracks.size === tracks.length) {
      setSelectedTracks(new Set());
    } else {
      setSelectedTracks(new Set(tracks.map(t => t.file_path)));
    }
  };

  const handleSelectTrack = (filePath: string) => {
    const newSelected = new Set(selectedTracks);
    if (newSelected.has(filePath)) {
      newSelected.delete(filePath);
    } else {
      newSelected.add(filePath);
    }
    setSelectedTracks(newSelected);
  };

  const handlePlayTrack = (track: AnalyzedTrack) => {
    if (currentTrack === track.file_path) {
      // Toggle play/pause
      if (audioElement.paused) {
        audioElement.play();
      } else {
        audioElement.pause();
      }
    } else {
      // Play new track
      const streamUrl = apiService.getStreamUrl(track.file_path);
      audioElement.src = streamUrl;
      audioElement.play();
      setCurrentTrack(track.file_path);
    }
  };

  const handleStopAudio = () => {
    audioElement.pause();
    audioElement.currentTime = 0;
    setCurrentTrack(null);
  };

  const handleExportSelected = () => {
    const selectedData = tracks.filter(t => selectedTracks.has(t.file_path));
    const csv = [
      'File Name,BPM,Key,Energy,Danceability,Valence,Acousticness',
      ...selectedData.map(t => 
        `${t.file_name},${t.analysis.bpm.toFixed(2)},${t.analysis.key},${t.analysis.energy.toFixed(3)},${t.analysis.danceability.toFixed(3)},${t.analysis.valence.toFixed(3)},${t.analysis.acousticness.toFixed(3)}`
      )
    ].join('\n');
    
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `analyzed-tracks-${new Date().toISOString().split('T')[0]}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (loading) {
    return (
      <div className="space-y-8">
        <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
          <h2 className="text-2xl font-bold text-white mb-4">Music Library Analytics</h2>
          <p className="text-slate-400">Loading analyzed tracks...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-8">
        <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
          <h2 className="text-2xl font-bold text-white mb-4">Music Library Analytics</h2>
          <p className="text-red-400">{error}</p>
          <button
            onClick={fetchAnalyzedTracks}
            className="mt-4 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (tracks.length === 0) {
    return (
      <div className="space-y-8">
        <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
          <h2 className="text-2xl font-bold text-white mb-4">Music Library Analytics</h2>
          <p className="text-slate-400">No analyzed tracks found. Run audio analysis on your tracks first.</p>
        </div>
      </div>
    );
  }

  // Calculate statistics
  const stats = calculateStats(tracks.map(t => t.analysis));
  const bpmDistribution = calculateBPMDistribution(tracks.map(t => t.analysis));
  const keyDistribution = calculateKeyDistribution(tracks.map(t => t.analysis));

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
        <div className="flex justify-between items-center">
          <h2 className="text-2xl font-bold text-white">Music Library Analytics</h2>
          <button
            onClick={fetchAnalyzedTracks}
            className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded transition-colors"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Total Tracks" value={tracks.length.toString()} />
        <StatCard label="Avg BPM" value={stats.avgBPM.toFixed(1)} />
        <StatCard label="Avg Energy" value={(stats.avgEnergy * 100).toFixed(0) + '%'} />
        <StatCard label="Avg Danceability" value={(stats.avgDanceability * 100).toFixed(0) + '%'} />
      </div>

      {/* Emotion Profile */}
      {emotionDistribution.length > 0 && (
        <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
          <h3 className="text-xl font-bold text-white mb-1">Emotion Profile</h3>
          <p className="text-slate-400 text-sm mb-4">
            Average intensities across {emotionDistribution[0]?.count ?? 0} analyzed tracks
          </p>
          <div className="flex flex-col lg:flex-row gap-8 items-center">
            <EmotionRadar data={emotionDistribution} />
            <EmotionBars data={emotionDistribution} />
          </div>
        </div>
      )}

      {/* Visualizations Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* BPM Distribution */}
        <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
          <h3 className="text-xl font-bold text-white mb-4">BPM Distribution</h3>
          <BPMHistogram data={bpmDistribution} />
        </div>

        {/* Key Distribution */}
        <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
          <h3 className="text-xl font-bold text-white mb-4">Key Distribution</h3>
          <KeyDistribution data={keyDistribution} />
        </div>

        {/* Energy vs Danceability */}
        <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
          <h3 className="text-xl font-bold text-white mb-4">Energy vs Danceability</h3>
          <ScatterPlot tracks={tracks} />
        </div>

        {/* Audio Features Summary */}
        <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
          <h3 className="text-xl font-bold text-white mb-4">Average Audio Features</h3>
          <FeaturesBars stats={stats} />
        </div>
      </div>

      {/* Track List */}
      <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-xl font-bold text-white">Analyzed Tracks</h3>
          {selectedTracks.size > 0 && (
            <div className="flex gap-2 items-center">
              <span className="text-slate-400 text-sm">{selectedTracks.size} selected</span>
              <button
                onClick={handleExportSelected}
                className="px-3 py-1 bg-green-600 hover:bg-green-700 text-white text-sm rounded transition-colors"
              >
                Export CSV
              </button>
              <button
                onClick={() => setSelectedTracks(new Set())}
                className="px-3 py-1 bg-slate-600 hover:bg-slate-700 text-white text-sm rounded transition-colors"
              >
                Clear Selection
              </button>
            </div>
          )}
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="border-b border-slate-700">
              <tr className="text-slate-400">
                <th className="text-left py-2 px-2">
                  <input
                    type="checkbox"
                    checked={selectedTracks.size === tracks.length && tracks.length > 0}
                    onChange={handleSelectAll}
                    className="cursor-pointer"
                  />
                </th>
                <th className="text-left py-2 px-2">Play</th>
                <th className="text-left py-2 px-2">Track</th>
                <th className="text-center py-2 px-2">BPM</th>
                <th className="text-center py-2 px-2">Key</th>
                <th className="text-center py-2 px-2">Energy</th>
                <th className="text-center py-2 px-2">Dance</th>
                <th className="text-center py-2 px-2">Valence</th>
              </tr>
            </thead>
            <tbody className="text-slate-300">
              {tracks.slice(0, 50).map((track, idx) => (
                <tr
                  key={idx}
                  className={`border-b border-slate-700 hover:bg-slate-700/50 ${
                    selectedTracks.has(track.file_path) ? 'bg-slate-700/30' : ''
                  }`}
                >
                  <td className="py-2 px-2">
                    <input
                      type="checkbox"
                      checked={selectedTracks.has(track.file_path)}
                      onChange={() => handleSelectTrack(track.file_path)}
                      className="cursor-pointer"
                    />
                  </td>
                  <td className="py-2 px-2">
                    <button
                      onClick={() => handlePlayTrack(track)}
                      className={`text-lg hover:scale-110 transition-transform ${
                        currentTrack === track.file_path ? 'text-green-400' : 'text-slate-400 hover:text-white'
                      }`}
                      title="Play preview"
                    >
                      {currentTrack === track.file_path && !audioElement.paused ? '⏸' : '▶'}
                    </button>
                  </td>
                  <td className="py-2 px-2 truncate max-w-xs">{track.file_name}</td>
                  <td className="py-2 px-2 text-center">{track.analysis.bpm.toFixed(0)}</td>
                  <td className="py-2 px-2 text-center">{track.analysis.key}</td>
                  <td className="py-2 px-2 text-center">{(track.analysis.energy * 100).toFixed(0)}%</td>
                  <td className="py-2 px-2 text-center">{(track.analysis.danceability * 100).toFixed(0)}%</td>
                  <td className="py-2 px-2 text-center">{(track.analysis.valence * 100).toFixed(0)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
          {tracks.length > 50 && (
            <p className="text-slate-400 text-sm mt-4">Showing first 50 tracks of {tracks.length}</p>
          )}
        </div>
      </div>

      {/* Audio Player Bar */}
      {currentTrack && (
        <div className="fixed bottom-0 left-0 right-0 bg-slate-800 border-t border-slate-700 p-4 shadow-lg z-50">
          <div className="max-w-7xl mx-auto flex items-center justify-between">
            <div className="flex items-center gap-4 flex-1">
              <button
                onClick={handleStopAudio}
                className="text-slate-400 hover:text-white transition-colors"
              >
                ⏹
              </button>
              <div className="text-slate-300 truncate">
                <span className="text-sm text-slate-400">Now Playing: </span>
                {tracks.find(t => t.file_path === currentTrack)?.file_name || 'Unknown'}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// Emotion Charts

const EMOTION_COLORS: Record<string, string> = {
  euphoria:   '#a78bfa',
  tension:    '#f87171',
  melancholy: '#60a5fa',
  aggression: '#fb923c',
  serenity:   '#34d399',
  nostalgia:  '#fbbf24',
  sensuality: '#f472b6',
  power:      '#e879f9',
};

const EmotionRadar = ({ data }: { data: EmotionDistribution[] }) => {
  const SIZE = 260, CX = 130, CY = 130, R = 100;
  const n = data.length;
  if (n === 0) return null;

  const pt = (v: number, i: number) => {
    const a = (i * 2 * Math.PI) / n - Math.PI / 2;
    return { x: CX + R * v * Math.cos(a), y: CY + R * v * Math.sin(a) };
  };
  const poly = (vals: number[]) =>
    vals.map((v, i) => pt(v, i)).map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ') + ' Z';

  return (
    <div className="flex-shrink-0">
      <svg width={SIZE} height={SIZE} viewBox={`0 0 ${SIZE} ${SIZE}`}>
        {[0.25, 0.5, 0.75, 1].map(lv => (
          <path key={lv} d={poly(data.map(() => lv))} fill="none" stroke="#475569" strokeWidth={1} strokeDasharray={lv < 1 ? '3,3' : undefined} />
        ))}
        {data.map((_, i) => { const t = pt(1, i); return <line key={i} x1={CX} y1={CY} x2={t.x} y2={t.y} stroke="#475569" strokeWidth={1} />; })}
        <path d={poly(data.map(d => d.avg))} fill="rgba(139,92,246,0.2)" stroke="#8b5cf6" strokeWidth={2} />
        {data.map((d, i) => { const p = pt(d.avg, i); return <circle key={i} cx={p.x} cy={p.y} r={5} fill={EMOTION_COLORS[d.dimension] ?? '#8b5cf6'}><title>{d.dimension}: {(d.avg * 100).toFixed(0)}%</title></circle>; })}
        {data.map((d, i) => { const p = pt(1.3, i); return <text key={i} x={p.x} y={p.y} textAnchor="middle" dominantBaseline="middle" fontSize={10} fill="#94a3b8">{d.dimension}</text>; })}
      </svg>
    </div>
  );
};

const EmotionBars = ({ data }: { data: EmotionDistribution[] }) => (
  <div className="flex-1 space-y-3 w-full min-w-0">
    {data.map(d => {
      const color = EMOTION_COLORS[d.dimension] ?? '#8b5cf6';
      return (
        <div key={d.dimension}>
          <div className="flex justify-between text-xs mb-1">
            <span className="text-slate-300 capitalize font-medium">{d.dimension}</span>
            <span className="text-slate-400">{(d.avg * 100).toFixed(0)}% avg</span>
          </div>
          <div className="relative h-5 bg-slate-700 rounded-full overflow-hidden">
            <div className="absolute h-full rounded-full opacity-30" style={{ left: `${d.min * 100}%`, width: `${Math.max((d.max - d.min) * 100, 1)}%`, backgroundColor: color }} />
            <div className="absolute top-0 bottom-0 w-1 rounded-full" style={{ left: `${d.avg * 100}%`, backgroundColor: color }} />
          </div>
          <div className="flex justify-between text-xs text-slate-500 mt-0.5">
            <span>min {(d.min * 100).toFixed(0)}%</span>
            <span>max {(d.max * 100).toFixed(0)}%</span>
          </div>
        </div>
      );
    })}
  </div>
);

// Helper Components

const StatCard = ({ label, value }: { label: string; value: string }) => (
  <div className="bg-slate-800 rounded-lg p-4 border border-slate-700">
    <p className="text-slate-400 text-sm">{label}</p>
    <p className="text-white text-2xl font-bold mt-1">{value}</p>
  </div>
);

const BPMHistogram = ({ data }: { data: Array<{ range: string; count: number }> }) => {
  const maxCount = Math.max(...data.map(d => d.count));
  
  return (
    <div className="space-y-2">
      {data.map((item, idx) => (
        <div key={idx} className="flex items-center gap-2">
          <span className="text-slate-400 text-sm w-20">{item.range}</span>
          <div className="flex-1 bg-slate-700 rounded-full h-6 overflow-hidden">
            <div
              className="bg-blue-500 h-full rounded-full transition-all duration-500"
              style={{ width: `${(item.count / maxCount) * 100}%` }}
            />
          </div>
          <span className="text-slate-300 text-sm w-8 text-right">{item.count}</span>
        </div>
      ))}
    </div>
  );
};

const KeyDistribution = ({ data }: { data: Array<{ key: string; count: number }> }) => {
  const maxCount = Math.max(...data.map(d => d.count));
  
  return (
    <div className="space-y-2">
      {data.slice(0, 12).map((item, idx) => (
        <div key={idx} className="flex items-center gap-2">
          <span className="text-slate-400 text-sm w-16">{item.key}</span>
          <div className="flex-1 bg-slate-700 rounded-full h-6 overflow-hidden">
            <div
              className="bg-purple-500 h-full rounded-full transition-all duration-500"
              style={{ width: `${(item.count / maxCount) * 100}%` }}
            />
          </div>
          <span className="text-slate-300 text-sm w-8 text-right">{item.count}</span>
        </div>
      ))}
    </div>
  );
};

const ScatterPlot = ({ tracks }: { tracks: AnalyzedTrack[] }) => {
  return (
    <div className="relative h-64 bg-slate-900 rounded-lg p-4">
      <div className="absolute inset-0 flex items-end justify-start p-4">
        {/* Grid lines */}
        <div className="absolute inset-4 grid grid-cols-10 grid-rows-10 opacity-20">
          {[...Array(10)].map((_, i) => (
            <div key={i} className="border-r border-b border-slate-600" />
          ))}
        </div>
        
        {/* Scatter points */}
        {tracks.map((track, idx) => (
          <div
            key={idx}
            className="absolute w-2 h-2 bg-green-400 rounded-full opacity-75 hover:opacity-100 hover:scale-150 transition-all cursor-pointer"
            style={{
              left: `${track.analysis.danceability * 90 + 5}%`,
              bottom: `${track.analysis.energy * 90 + 5}%`,
            }}
            title={`${track.file_name}\nEnergy: ${(track.analysis.energy * 100).toFixed(0)}%\nDanceability: ${(track.analysis.danceability * 100).toFixed(0)}%`}
          />
        ))}
      </div>
      
      {/* Axis labels */}
      <div className="absolute bottom-0 left-0 right-0 text-center text-slate-400 text-xs pb-1">
        Danceability →
      </div>
      <div className="absolute top-0 bottom-0 left-0 flex items-center pl-1">
        <div className="text-slate-400 text-xs transform -rotate-90 whitespace-nowrap">
          Energy →
        </div>
      </div>
    </div>
  );
};

const FeaturesBars = ({ stats }: { stats: ReturnType<typeof calculateStats> }) => {
  const features = [
    { label: 'Energy', value: stats.avgEnergy, color: 'bg-red-500' },
    { label: 'Danceability', value: stats.avgDanceability, color: 'bg-green-500' },
    { label: 'Valence', value: stats.avgValence, color: 'bg-yellow-500' },
    { label: 'Acousticness', value: stats.avgAcousticness, color: 'bg-blue-500' },
  ];

  return (
    <div className="space-y-3">
      {features.map((feature, idx) => (
        <div key={idx}>
          <div className="flex justify-between text-sm mb-1">
            <span className="text-slate-400">{feature.label}</span>
            <span className="text-slate-300">{(feature.value * 100).toFixed(0)}%</span>
          </div>
          <div className="bg-slate-700 rounded-full h-4 overflow-hidden">
            <div
              className={`${feature.color} h-full rounded-full transition-all duration-500`}
              style={{ width: `${feature.value * 100}%` }}
            />
          </div>
        </div>
      ))}
    </div>
  );
};

// Helper Functions

function calculateStats(analyses: AudioAnalysis[]) {
  const count = analyses.length;
  
  return {
    avgBPM: analyses.reduce((sum, a) => sum + a.bpm, 0) / count,
    avgEnergy: analyses.reduce((sum, a) => sum + a.energy, 0) / count,
    avgDanceability: analyses.reduce((sum, a) => sum + a.danceability, 0) / count,
    avgValence: analyses.reduce((sum, a) => sum + a.valence, 0) / count,
    avgAcousticness: analyses.reduce((sum, a) => sum + a.acousticness, 0) / count,
  };
}

function calculateBPMDistribution(analyses: AudioAnalysis[]) {
  const ranges = [
    { min: 0, max: 80, label: '0-80' },
    { min: 80, max: 100, label: '80-100' },
    { min: 100, max: 120, label: '100-120' },
    { min: 120, max: 140, label: '120-140' },
    { min: 140, max: 160, label: '140-160' },
    { min: 160, max: 180, label: '160-180' },
    { min: 180, max: Infinity, label: '180+' },
  ];

  return ranges.map(range => ({
    range: range.label,
    count: analyses.filter(a => a.bpm >= range.min && a.bpm < range.max).length,
  }));
}

function calculateKeyDistribution(analyses: AudioAnalysis[]) {
  const keyCounts: Record<string, number> = {};
  
  analyses.forEach(a => {
    const key = a.key || 'Unknown';
    keyCounts[key] = (keyCounts[key] || 0) + 1;
  });

  return Object.entries(keyCounts)
    .map(([key, count]) => ({ key, count }))
    .sort((a, b) => b.count - a.count);
}

export default AnalyticsDashboard;
