import { useState } from 'react';
import { useWebSocket } from './hooks/useWebSocket';
import { ConnectionStatus } from './components/ConnectionStatus';
import { StatsPanel } from './components/StatsPanel';
import { ScrapeControls } from './components/ScrapeControls';
import { DownloadControls } from './components/DownloadControls';
import { JobsList } from './components/JobsList';
import { ErrorBoundary } from './components/ErrorBoundary';
import TrackBrowser from './components/TrackBrowser';

type View = 'dashboard' | 'tracks';

function App() {
  const { isConnected, jobs } = useWebSocket();
  const [currentView, setCurrentView] = useState<View>('dashboard');

  return (
    <ErrorBoundary>
      <div className="min-h-screen bg-slate-900">
        {/* Header */}
        <header className="bg-slate-800 border-b border-slate-700 shadow-lg">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex justify-between items-center">
            <div className="flex items-center gap-8">
              <h1 className="text-3xl font-bold text-white">DoYou World Scraper</h1>
              <nav className="flex gap-2">
                <button
                  onClick={() => setCurrentView('dashboard')}
                  className={`px-4 py-2 rounded-lg transition ${
                    currentView === 'dashboard'
                      ? 'bg-blue-600 text-white'
                      : 'text-slate-300 hover:text-white hover:bg-slate-700'
                  }`}
                >
                  Dashboard
                </button>
                <button
                  onClick={() => setCurrentView('tracks')}
                  className={`px-4 py-2 rounded-lg transition ${
                    currentView === 'tracks'
                      ? 'bg-blue-600 text-white'
                      : 'text-slate-300 hover:text-white hover:bg-slate-700'
                  }`}
                >
                  Browse Tracks
                </button>
              </nav>
            </div>
            <ConnectionStatus isConnected={isConnected} />
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {currentView === 'dashboard' && (
          <div className="space-y-8">
            {/* Stats */}
            <StatsPanel />

            {/* Controls Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
              <ScrapeControls />
              <DownloadControls />
            </div>

            {/* Jobs List */}
            <JobsList jobs={jobs} />
          </div>
        )}

        {currentView === 'tracks' && <TrackBrowser />}
      </main>

      {/* Footer */}
      <footer className="mt-12 py-6 border-t border-slate-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <p className="text-center text-slate-500 text-sm">
            DoYou World Radio Track Scraper & Downloader
          </p>
        </div>
      </footer>
      </div>
    </ErrorBoundary>
  );
}

export default App;
