import { useWebSocket } from './hooks/useWebSocket';
import { ConnectionStatus } from './components/ConnectionStatus';
import { StatsPanel } from './components/StatsPanel';
import { ScrapeControls } from './components/ScrapeControls';
import { DownloadControls } from './components/DownloadControls';
import { JobsList } from './components/JobsList';
import { ErrorBoundary } from './components/ErrorBoundary';

function App() {
  const { isConnected, jobs } = useWebSocket();

  return (
    <ErrorBoundary>
      <div className="min-h-screen bg-slate-900">
        {/* Header */}
        <header className="bg-slate-800 border-b border-slate-700 shadow-lg">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex justify-between items-center">
            <h1 className="text-3xl font-bold text-white">DoYou World Scraper</h1>
            <ConnectionStatus isConnected={isConnected} />
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
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
