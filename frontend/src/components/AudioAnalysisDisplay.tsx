import type { AudioAnalysis } from '../types';

interface Props {
  analysis: AudioAnalysis;
}

export default function AudioAnalysisDisplay({ analysis }: Props) {
  const features = [
    { name: 'Energy', value: analysis.energy, color: 'bg-red-500' },
    { name: 'Danceability', value: analysis.danceability, color: 'bg-purple-500' },
    { name: 'Valence', value: analysis.valence, color: 'bg-yellow-500' },
    { name: 'Acousticness', value: analysis.acousticness, color: 'bg-green-500' },
    { name: 'Melodicness', value: analysis.melodicness, color: 'bg-blue-500' },
    { name: 'Instrumentalness', value: analysis.instrumentalness, color: 'bg-indigo-500' },
  ];

  return (
    <div className="mt-2 p-3 bg-slate-800 rounded-lg space-y-2">
      <div className="flex items-center gap-4 text-sm text-slate-300">
        <div>
          <span className="text-slate-400">BPM:</span> <span className="font-mono font-bold text-white">{analysis.bpm.toFixed(1)}</span>
        </div>
        <div>
          <span className="text-slate-400">Key:</span> <span className="font-mono font-bold text-white">{analysis.key}</span>
        </div>
        <div>
          <span className="text-slate-400">Duration:</span> <span className="font-mono">{Math.floor(analysis.duration_seconds / 60)}:{(Math.floor(analysis.duration_seconds) % 60).toString().padStart(2, '0')}</span>
        </div>
      </div>
      
      <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
        {features.map(({ name, value, color }) => (
          <div key={name} className="space-y-1">
            <div className="flex justify-between text-xs text-slate-400">
              <span>{name}</span>
              <span className="font-mono">{(value * 100).toFixed(0)}%</span>
            </div>
            <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
              <div
                className={`h-full ${color} transition-all duration-300`}
                style={{ width: `${value * 100}%` }}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
