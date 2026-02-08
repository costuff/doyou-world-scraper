import { useEffect, useState } from 'react';
import apiService from '../services/api';
import type { VersionInfo } from '../types';

export function Footer() {
  const [version, setVersion] = useState<VersionInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchVersion = async () => {
      try {
        const response = await apiService.getVersion();
        setVersion(response.data);
        setError(null);
      } catch (err) {
        console.error('Failed to fetch version:', err);
        setError('Failed to load version info');
      } finally {
        setLoading(false);
      }
    };

    fetchVersion();
  }, []);

  return (
    <footer className="mt-12 py-6 border-t border-slate-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-center space-y-2">
          <p className="text-center text-slate-500 text-sm">
            DoYou World Radio Track Scraper & Downloader
          </p>
          
          {loading && (
            <p className="text-xs text-slate-600">Loading version info...</p>
          )}
          
          {error && (
            <p className="text-xs text-red-500">{error}</p>
          )}
          
          {version && !loading && (
            <div className="flex items-center gap-4 text-xs text-slate-600">
              <span>Backend: <span className="text-slate-400 font-mono">{version.backend}</span></span>
              <span className="text-slate-700">•</span>
              <span>Frontend: <span className="text-slate-400 font-mono">{version.frontend}</span></span>
              
              {version.commit && (
                <>
                  <span className="text-slate-700">•</span>
                  <span>
                    Commit: <span className="text-slate-400 font-mono">{version.commit.slice(0, 7)}</span>
                  </span>
                </>
              )}
              
              {version.build_time && (
                <>
                  <span className="text-slate-700">•</span>
                  <span className="text-slate-400">{new Date(version.build_time).toLocaleDateString()}</span>
                </>
              )}
            </div>
          )}
        </div>
      </div>
    </footer>
  );
}
