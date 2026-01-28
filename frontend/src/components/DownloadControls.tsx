import { useState } from 'react';
import apiService from '../services/api';

export const DownloadControls = () => {
  const [loading, setLoading] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error', text: string } | null>(null);

  const handleDownloadAll = async () => {
    setLoading('all');
    setMessage(null);
    try {
      const response = await apiService.downloadAll();
      setMessage({ type: 'success', text: `Download started! Job ID: ${response.data.job_id}` });
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
      const response = await apiService.removeDuplicates();
      setMessage({ type: 'success', text: `Deduplication started! Job ID: ${response.data.job_id}` });
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
      setMessage({ type: 'success', text: `Matching started! Job ID: ${response.data.job_id}` });
    } catch (error: any) {
      setMessage({ type: 'error', text: error.response?.data?.error || 'Failed to start matching' });
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
      </div>
    </div>
  );
};
