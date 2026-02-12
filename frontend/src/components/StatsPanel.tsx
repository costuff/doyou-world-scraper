import { useEffect, useState } from 'react';
import apiService from '../services/api';
import type { Stats, AnalyticsStats } from '../types';

export const StatsPanel = () => {
  const [stats, setStats] = useState<Stats | null>(null);
  const [analyticsStats, setAnalyticsStats] = useState<AnalyticsStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = async () => {
    try {
      const [scraperResponse, analyticsResponse] = await Promise.all([
        apiService.getStats(),
        apiService.getAnalyticsDetailed(),
      ]);
      
      if (scraperResponse?.data) {
        setStats(scraperResponse.data);
        setError(null);
      } else {
        throw new Error('Invalid response format');
      }
      
      if (analyticsResponse?.data) {
        setAnalyticsStats(analyticsResponse.data);
      }
    } catch (err: any) {
      console.error('Failed to fetch stats:', err);
      setError(err.response?.data?.error || err.message || 'Failed to load statistics');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
        <h2 className="text-2xl font-bold text-white mb-6">Statistics</h2>
        <p className="text-slate-400">Loading...</p>
      </div>
    );
  }

  if (error || !stats) {
    return (
      <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
        <h2 className="text-2xl font-bold text-white mb-6">Statistics</h2>
        <p className="text-red-400">{error || 'No data available'}</p>
      </div>
    );
  }

  return (
    <div className="bg-slate-800 rounded-lg p-6 shadow-lg border border-slate-700">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-white">Statistics</h2>
        <button
          onClick={fetchStats}
          className="px-3 py-1 text-sm bg-slate-700 hover:bg-slate-600 text-white rounded transition-colors"
        >
          Refresh
        </button>
      </div>

      <div className="space-y-6">
        <div>
          <h3 className="text-sm font-semibold text-slate-400 mb-3 uppercase tracking-wide">Scraper Statistics</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <StatCard
              label="Total CSVs"
              value={(stats.total_csvs ?? 0).toLocaleString()}
              subtitle="Track listings"
            />
            <StatCard
              label="Total Tracks Scraped"
              value={(stats.total_tracks_scraped ?? 0).toLocaleString()}
              subtitle="All time"
            />
            <StatCard
              label="Downloads"
              value={(stats.total_downloads ?? 0).toLocaleString()}
              subtitle={`${(stats.downloads_size_gb ?? 0).toFixed(2)} GB`}
            />
            <StatCard
              label="NAS Storage"
              value={(stats.nas_downloads ?? 0).toLocaleString()}
              subtitle={`${(stats.nas_size_gb ?? 0).toFixed(2)} GB`}
            />
          </div>
        </div>

        {analyticsStats && (
          <div>
            <h3 className="text-sm font-semibold text-slate-400 mb-3 uppercase tracking-wide">Audio Analysis Progress</h3>
            <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
              <StatCard
                label="Total Tracks"
                value={analyticsStats.total.toLocaleString()}
                subtitle="Imported"
                color="blue"
              />
              <StatCard
                label="Completed"
                value={analyticsStats.completed.toLocaleString()}
                subtitle={analyticsStats.completion_percentage ? `${analyticsStats.completion_percentage}%` : ''}
                color="green"
              />
              <StatCard
                label="Pending"
                value={analyticsStats.pending.toLocaleString()}
                subtitle="Waiting"
                color="yellow"
              />
              <StatCard
                label="Analyzing"
                value={analyticsStats.analyzing.toLocaleString()}
                subtitle="In progress"
                color="cyan"
              />
              <StatCard
                label="Failed"
                value={analyticsStats.failed.toLocaleString()}
                subtitle="Errors"
                color="red"
              />
            </div>
            {analyticsStats.tracks_per_hour && (
              <div className="mt-3 text-sm text-slate-400">
                Processing Rate: <span className="text-cyan-400 font-medium">{analyticsStats.tracks_per_hour} tracks/hr</span>
                {analyticsStats.avg_processing_time_sec && (
                  <span className="ml-4">
                    Avg Time: <span className="text-cyan-400 font-medium">{analyticsStats.avg_processing_time_sec}s per track</span>
                  </span>
                )}
              </div>
            )}
          </div>
        )}
      </div>

      {stats.earliest_date && stats.latest_date && (
        <div className="mt-4 pt-4 border-t border-slate-700">
          <p className="text-sm text-slate-400">
            Date Range: <span className="text-white font-medium">{stats.earliest_date}</span>
            {' to '}
            <span className="text-white font-medium">{stats.latest_date}</span>
          </p>
        </div>
      )}
    </div>
  );
};

const StatCard = ({ 
  label, 
  value, 
  subtitle, 
  color = 'white' 
}: { 
  label: string; 
  value: string; 
  subtitle: string;
  color?: 'white' | 'blue' | 'green' | 'yellow' | 'cyan' | 'red';
}) => {
  const colorClasses = {
    white: 'text-white',
    blue: 'text-blue-400',
    green: 'text-green-400',
    yellow: 'text-yellow-400',
    cyan: 'text-cyan-400',
    red: 'text-red-400',
  };
  
  return (
    <div className="bg-slate-900 rounded-lg p-4">
      <p className="text-sm text-slate-400 mb-1">{label}</p>
      <p className={`text-2xl font-bold ${colorClasses[color]} mb-1`}>{value}</p>
      <p className="text-xs text-slate-500">{subtitle}</p>
    </div>
  );
};
