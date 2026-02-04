import { useState, useEffect, useCallback } from 'react';
import { apiService } from '../services/api';
import type { TrackInfo, TrackScanStatus } from '../types';
import AudioAnalysisDisplay from './AudioAnalysisDisplay';

export default function TrackBrowser() {
  const [tracks, setTracks] = useState<TrackInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [perPage] = useState(50);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [search, setSearch] = useState('');
  const [sortBy, setSortBy] = useState('mod_time');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
  const [currentTrack, setCurrentTrack] = useState<string | null>(null);
  const [source, setSource] = useState<'downloads' | 'nas'>('downloads');
  const [scanStatus, setScanStatus] = useState<TrackScanStatus | null>(null);
  const [analyzing, setAnalyzing] = useState<Set<string>>(new Set());
  const [expandedTrack, setExpandedTrack] = useState<string | null>(null);
  const [nasPath, setNasPath] = useState<string>(() => {
    return localStorage.getItem('nasPath') || '/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks';
  });

  const loadTracks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Send simple keywords 'downloads' or 'nas' - backend will resolve to correct paths
      const downloadsDir = source === 'nas' ? 'nas' : 'downloads';
      
      const response = await apiService.getTracks({
        page,
        per_page: perPage,
        search,
        sort_by: sortBy,
        sort_order: sortOrder,
        downloads_dir: downloadsDir,
        async: source === 'nas',
      });
      setTracks(response.data.tracks || []);
      setTotal(response.data.total);
      setTotalPages(response.data.total_pages);
      if (response.data.in_progress) {
        setScanStatus({
          in_progress: true,
          scanned: response.data.scanned ?? 0,
          total: 0,
          last_updated: '',
        });
      } else {
        setScanStatus(null);
      }
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load tracks');
    } finally {
      setLoading(false);
    }
  }, [nasPath, page, perPage, search, sortBy, sortOrder, source]);

  useEffect(() => {
    localStorage.setItem('nasPath', nasPath);
  }, [nasPath]);

  useEffect(() => {
    loadTracks();
  }, [loadTracks]);

  useEffect(() => {
    if (source !== 'nas' || !scanStatus?.in_progress) {
      return;
    }

    const interval = setInterval(async () => {
      try {
        const response = await apiService.getTrackScanStatus({
          downloads_dir: 'nas',
        });
        setScanStatus(response.data);
        if (!response.data.in_progress) {
          loadTracks();
        }
      } catch {
        // Ignore polling errors
      }
    }, 1500);

    return () => clearInterval(interval);
  }, [loadTracks, nasPath, scanStatus?.in_progress, source]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    loadTracks();
  };

  const formatDuration = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024 * 1024) {
      return `${(bytes / 1024).toFixed(1)} KB`;
    }
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const handleSort = (field: string) => {
    if (sortBy === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(field);
      setSortOrder('asc');
    }
  };

  const handleAnalyzeTrack = async (track: TrackInfo) => {
    try {
      setAnalyzing(prev => new Set(prev).add(track.file_path));
      const response = await apiService.analyzeTrack(track.file_path);
      const jobId = response.data.job.id;

      // Poll job status
      const pollInterval = setInterval(async () => {
        try {
          const jobResponse = await apiService.getJob(jobId);
          const job = jobResponse.data;
          
          if (job.status === 'completed') {
            clearInterval(pollInterval);
            // Update the track with analysis data
            setTracks(prevTracks =>
              prevTracks.map(t =>
                t.file_path === track.file_path
                  ? { ...t, analysis: job.result.analysis }
                  : t
              )
            );
            setAnalyzing(prev => {
              const newSet = new Set(prev);
              newSet.delete(track.file_path);
              return newSet;
            });
            setExpandedTrack(track.file_path); // Auto-expand to show analysis
          } else if (job.status === 'failed') {
            clearInterval(pollInterval);
            setAnalyzing(prev => {
              const newSet = new Set(prev);
              newSet.delete(track.file_path);
              return newSet;
            });
            setError(`Analysis failed: ${job.error}`);
          }
        } catch (err) {
          clearInterval(pollInterval);
          setAnalyzing(prev => {
            const newSet = new Set(prev);
            newSet.delete(track.file_path);
            return newSet;
          });
        }
      }, 1000);

    } catch (err: any) {
      setAnalyzing(prev => {
        const newSet = new Set(prev);
        newSet.delete(track.file_path);
        return newSet;
      });
      setError(err.response?.data?.error || 'Failed to analyze track');
    }
  };

  return (
    <div className="bg-slate-800 rounded-lg shadow-lg p-6">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-white">Track Library</h2>
        
        {/* Source Selector */}
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => {
              setSource('downloads');
              setPage(1);
            }}
            className={`px-4 py-2 rounded-lg transition ${
              source === 'downloads'
                ? 'bg-blue-600 text-white'
                : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
            }`}
          >
            Local Downloads
          </button>
          <button
            type="button"
            onClick={() => {
              setSource('nas');
              setPage(1);
            }}
            className={`px-4 py-2 rounded-lg transition ${
              source === 'nas'
                ? 'bg-blue-600 text-white'
                : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
            }`}
          >
            NAS Storage
          </button>
        </div>
      </div>

      {source === 'nas' && (
        <div className="mb-4">
          <label className="block text-sm text-slate-400 mb-2">NAS path</label>
          <div className="flex gap-2">
            <input
              type="text"
              value={nasPath}
              onChange={(e) => setNasPath(e.target.value)}
              placeholder="/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks"
              className="flex-1 px-4 py-2 bg-slate-700 text-white rounded-lg border border-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              type="button"
              onClick={() => {
                setPage(1);
                loadTracks();
              }}
              className="px-4 py-2 bg-slate-600 text-white rounded-lg hover:bg-slate-700 transition"
            >
              Load
            </button>
          </div>
        </div>
      )}

      {/* Search Bar */}
      <form onSubmit={handleSearch} className="mb-6">
        <div className="flex gap-2">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by title, artist, album, or filename..."
            className="flex-1 px-4 py-2 bg-slate-700 text-white rounded-lg border border-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            type="submit"
            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
          >
            Search
          </button>
          {search && (
            <button
              type="button"
              onClick={() => {
                setSearch('');
                setPage(1);
                setTimeout(loadTracks, 0);
              }}
              className="px-4 py-2 bg-slate-600 text-white rounded-lg hover:bg-slate-700 transition"
            >
              Clear
            </button>
          )}
        </div>
      </form>

      {/* Stats */}
      <div className="mb-4 text-slate-400 text-sm">
        Showing {tracks.length} of {total} tracks
      </div>

      {/* Scan Status */}
      {scanStatus?.in_progress && (
        <div className="mb-4 p-3 bg-slate-700/60 border border-slate-600 rounded-lg text-slate-200 text-sm">
          Scanning NAS… {scanStatus.scanned} tracks loaded so far.
        </div>
      )}

      {/* Error Message */}
      {error && (
        <div className="mb-4 p-4 bg-red-900/50 border border-red-700 rounded-lg text-red-200">
          {error}
        </div>
      )}

      {/* Loading State */}
      {loading && (
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto"></div>
          <p className="text-slate-400 mt-4">Loading tracks...</p>
        </div>
      )}

      {/* Track Table */}
      {!loading && tracks.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-slate-300">
            <thead className="bg-slate-700 text-slate-200">
              <tr>
                <th className="px-4 py-3 cursor-pointer hover:bg-slate-600" onClick={() => handleSort('title')}>
                  Title {sortBy === 'title' && (sortOrder === 'asc' ? '↑' : '↓')}
                </th>
                <th className="px-4 py-3 cursor-pointer hover:bg-slate-600" onClick={() => handleSort('artist')}>
                  Artist {sortBy === 'artist' && (sortOrder === 'asc' ? '↑' : '↓')}
                </th>
                <th className="px-4 py-3">Album</th>
                <th className="px-4 py-3 cursor-pointer hover:bg-slate-600" onClick={() => handleSort('duration')}>
                  Duration {sortBy === 'duration' && (sortOrder === 'asc' ? '↑' : '↓')}
                </th>
                <th className="px-4 py-3 cursor-pointer hover:bg-slate-600" onClick={() => handleSort('size')}>
                  Size {sortBy === 'size' && (sortOrder === 'asc' ? '↑' : '↓')}
                </th>
                <th className="px-4 py-3 cursor-pointer hover:bg-slate-600" onClick={() => handleSort('mod_time')}>
                  Date {sortBy === 'mod_time' && (sortOrder === 'asc' ? '↑' : '↓')}
                </th>
                <th className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {tracks.map((track, index) => (
                <>
                  <tr
                    key={`${track.file_path}-${index}`}
                    className="border-b border-slate-700 hover:bg-slate-700/50 transition"
                  >
                    <td className="px-4 py-3">
                      <div className="font-medium">{track.title || track.filename}</div>
                    </td>
                    <td className="px-4 py-3">{track.artist || '-'}</td>
                    <td className="px-4 py-3 text-slate-400">{track.album || '-'}</td>
                    <td className="px-4 py-3">{track.duration ? formatDuration(track.duration) : '-'}</td>
                    <td className="px-4 py-3 text-slate-400">{formatFileSize(track.file_size)}</td>
                    <td className="px-4 py-3 text-slate-400">{track.mod_time.split(' ')[0]}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button
                          onClick={() => setCurrentTrack(track.file_path)}
                          className="text-blue-400 hover:text-blue-300 transition"
                          title="Play"
                        >
                          ▶
                        </button>
                        <button
                          onClick={() => handleAnalyzeTrack(track)}
                          disabled={analyzing.has(track.file_path)}
                          className={`px-2 py-1 text-xs rounded transition ${
                            analyzing.has(track.file_path)
                              ? 'bg-slate-600 text-slate-400 cursor-not-allowed'
                              : track.analysis
                              ? 'bg-green-600 hover:bg-green-700 text-white'
                              : 'bg-purple-600 hover:bg-purple-700 text-white'
                          }`}
                          title={track.analysis ? 'Re-analyze' : 'Analyze audio'}
                        >
                          {analyzing.has(track.file_path) ? '...' : track.analysis ? '✓' : '🎵'}
                        </button>
                        {track.analysis && (
                          <button
                            onClick={() => setExpandedTrack(expandedTrack === track.file_path ? null : track.file_path)}
                            className="text-slate-400 hover:text-slate-300 transition text-xs"
                            title={expandedTrack === track.file_path ? 'Hide analysis' : 'Show analysis'}
                          >
                            {expandedTrack === track.file_path ? '▲' : '▼'}
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                  {/* Analysis Display Row */}
                  {track.analysis && expandedTrack === track.file_path && (
                    <tr key={`${track.file_path}-analysis`} className="bg-slate-700/30">
                      <td colSpan={7} className="px-4 py-4">
                        <AudioAnalysisDisplay analysis={track.analysis} />
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* No Results */}
      {!loading && tracks.length === 0 && !scanStatus?.in_progress && (
        <div className="text-center py-12 text-slate-400">
          {search ? 'No tracks found matching your search.' : 'No tracks available.'}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="mt-6 flex justify-center gap-2">
          <button
            onClick={() => setPage(Math.max(1, page - 1))}
            disabled={page === 1}
            className="px-4 py-2 bg-slate-700 text-white rounded-lg hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed transition"
          >
            Previous
          </button>
          <span className="px-4 py-2 text-slate-300">
            Page {page} of {totalPages}
          </span>
          <button
            onClick={() => setPage(Math.min(totalPages, page + 1))}
            disabled={page === totalPages}
            className="px-4 py-2 bg-slate-700 text-white rounded-lg hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed transition"
          >
            Next
          </button>
        </div>
      )}

      {/* Audio Player */}
      {currentTrack && (
        <div className="fixed bottom-0 left-0 right-0 bg-slate-900 border-t border-slate-700 p-4 shadow-lg">
          <div className="max-w-7xl mx-auto flex items-center gap-4">
            <button
              onClick={() => setCurrentTrack(null)}
              className="text-slate-400 hover:text-white transition"
            >
              ✕
            </button>
            <audio
              controls
              autoPlay
              src={apiService.getStreamUrl(currentTrack)}
              className="flex-1"
            />
          </div>
        </div>
      )}
    </div>
  );
}
