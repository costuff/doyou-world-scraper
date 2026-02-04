import { useState } from 'react';
import apiService from '../services/api';

export const DownloadControls = () => {
  const [loading, setLoading] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error', text: string } | null>(null);
  const [nasPath, setNasPath] = useState<string>(() => {
    return localStorage.getItem('nasPath') || '';
  });

  const saveNasPath = (path: string) => {
    setNasPath(path);
    localStorage.setItem('nasPath', path);
  };

  const handleDownloadAll = async () => {
    setLoading('all');
    setMessage(null);
    try {
      const response = await apiService.downloadAll();
      setMessage({ type: 'success', text: `Download started! Job ID: ${response.data.job.id}` });
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Failed to start download' });
    } finally {
      setLoading(null);
    }
  };

  const handleRemoveDuplicates = async () => {
    setLoading('duplicates');
    setMessage(null);
    try {
      const response = await apiService.removeDuplicates(nasPath || 'skip');
      setMessage({ type: 'success', text: `Deduplication started! Job ID: ${response.data.job.id}` });
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Failed to start deduplication' });
    } finally {
      setLoading(null);
    }
  };

  const handleMatchDownloads = async () => {
    setLoading('match');
    setMessage(null);
    try {
      const response = await apiService.matchDownloads();
      setMessage({ type: 'success', text: `Matching started! Job ID: ${response.data.job.id}` });
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Failed to start matching' });
    } finally {
      setLoading(null);
    }
  };

  const handleMoveToNAS = async () => {
    if (!nasPath) {
      setMessage({ type: 'error', text: 'Please enter a NAS path first' });
      return;
    }
    
    setLoading('move');
    setMessage(null);
    try {
      const response = await apiService.moveToNAS(nasPath);
      setMessage({ type: 'success', text: `Move to NAS started! Job ID: ${response.data.job.id}` });
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Failed to start move' });
    } finally {
      setLoading(null);
    }
  };

  const handleClearCSVs = async () => {
    if (!window.confirm('Are you sure you want to delete all CSV files? This cannot be undone.')) {
      return;
    }
    
    setLoading('clear-csv');
    setMessage(null);
    try {
      const response = await apiService.clearCSVs();
      setMessage({ type: 'success', text: `Clear CSVs started! Job ID: ${response.data.job.id}` });
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Failed to clear CSVs' });
    } finally {
      setLoading(null);
    }
  };

  const handleClearDownloads = async () => {
    if (!window.confirm('Are you sure you want to delete ALL downloaded MP3 files? This cannot be undone!')) {
      return;
    }
    
    setLoading('clear-downloads');
    setMessage(null);
    try {
      const response = await apiService.clearDownloads();
      setMessage({ type: 'success', text: `Clear downloads started! Job ID: ${response.data.job.id}` });
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Failed to clear downloads' });
    } finally {
      setLoading(null);
    }
  };

  return (
    <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
      <h2 className="text-2xl font-bold text-white mb-6">Downloads & Utilities</h2>
      
      {message && (
        <div className={`mb-4 p-3 rounded ${message.type === 'success' ? 'bg-green-900/20 border border-green-500 text-green-300' : 'bg-red-900/20 border border-red-500 text-red-300'}`}>
          {message.text}
        </div>
      )}

      {/* NAS Path Configuration */}
      <div className="mb-6 p-4 bg-slate-700/50 rounded-lg border border-slate-600">
        <label className="block text-sm font-medium text-slate-300 mb-2">
          NAS Path (for deduplication & move operations)
        </label>
        <input
          type="text"
          value={nasPath}
          onChange={(e) => saveNasPath(e.target.value)}
          placeholder="e.g., /Volumes/MyNAS/Music or leave empty to skip"
          className="w-full px-3 py-2 bg-slate-800 text-white rounded border border-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
        />
        <p className="mt-1 text-xs text-slate-400">
          Leave empty to scan downloads folder only. Use 'skip' to explicitly skip NAS.
        </p>
      </div>

      <div className="space-y-3">
        <button
          onClick={handleDownloadAll}
          disabled={!!loading}
          className="w-full px-4 py-3 bg-purple-600 hover:bg-purple-700 disabled:bg-slate-600 text-white rounded font-medium transition-colors text-left"
        >
          <div className="flex justify-between items-center">
            <span>{loading === 'all' ? 'Downloading...' : 'Download All CSVs'}</span>
            <span className="text-sm opacity-70">Process all track listings</span>
          </div>
        </button>

        <button
          onClick={handleRemoveDuplicates}
          disabled={!!loading}
          className="w-full px-4 py-3 bg-orange-600 hover:bg-orange-700 disabled:bg-slate-600 text-white rounded font-medium transition-colors text-left"
        >
          <div className="flex justify-between items-center">
            <span>{loading === 'duplicates' ? 'Processing...' : 'Remove Duplicates'}</span>
            <span className="text-sm opacity-70">Clean up duplicate files</span>
          </div>
        </button>

        <button
          onClick={handleMatchDownloads}
          disabled={!!loading}
          className="w-full px-4 py-3 bg-teal-600 hover:bg-teal-700 disabled:bg-slate-600 text-white rounded font-medium transition-colors text-left"
        >
          <div className="flex justify-between items-center">
            <span>{loading === 'match' ? 'Matching...' : 'Match Downloads to CSVs'}</span>
            <span className="text-sm opacity-70">Find missing tracks</span>
          </div>
        </button>

        <button
          onClick={handleMoveToNAS}
          disabled={!!loading}
          className="w-full px-4 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white rounded font-medium transition-colors text-left"
        >
          <div className="flex justify-between items-center">
            <span>{loading === 'move' ? 'Moving...' : 'Move to NAS'}</span>
            <span className="text-sm opacity-70">Transfer files to network storage</span>
          </div>
        </button>

        <button
          onClick={handleClearCSVs}
          disabled={!!loading}
          className="w-full px-4 py-3 bg-red-600 hover:bg-red-700 disabled:bg-slate-600 text-white rounded font-medium transition-colors text-left"
        >
          <div className="flex justify-between items-center">
            <span>{loading === 'clear-csv' ? 'Clearing...' : 'Clear All CSVs'}</span>
            <span className="text-sm opacity-70">Delete all scraped CSV files</span>
          </div>
        </button>

        <button
          onClick={handleClearDownloads}
          disabled={!!loading}
          className="w-full px-4 py-3 bg-red-700 hover:bg-red-800 disabled:bg-slate-600 text-white rounded font-medium transition-colors text-left border-2 border-red-400"
        >
          <div className="flex justify-between items-center">
            <span>{loading === 'clear-downloads' ? 'Deleting...' : 'Clear All Downloads'}</span>
            <span className="text-sm opacity-70">⚠️ Delete ALL MP3 files</span>
          </div>
        </button>
      </div>
    </div>
  );
};
