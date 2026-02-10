# DoYou World Scraper - Frontend

React + TypeScript frontend for the DoYou World Radio track scraper.

## Features

- 🎵 **Scraping Controls**: Scrape tracks by date, month, or year
- 📥 **Download Manager**: Download tracks from scraped CSVs
- 📊 **Real-time Statistics**: Track counts, storage usage, date ranges
- 🔄 **Live Job Monitoring**: WebSocket-powered real-time progress updates
- 🧹 **Utilities**: Remove duplicates, match downloads to CSVs
- 🎨 **Dark Theme**: Modern, responsive UI with Tailwind CSS

## Getting Started

### Prerequisites

- Node.js 18+ and npm
- Backend API running on port 3000

### Installation

```bash
cd frontend
npm install
```

### Development

```bash
npm run dev
```

Open http://localhost:5173 in your browser.

### Build for Production

```bash
npm run build
```

The built files will be in the `dist/` directory.

## Architecture

```
frontend/
├── src/
│   ├── components/        # React components
│   │   ├── ConnectionStatus.tsx
│   │   ├── DownloadControls.tsx
│   │   ├── JobCard.tsx
│   │   ├── JobsList.tsx
│   │   ├── ScrapeControls.tsx
│   │   └── StatsPanel.tsx
│   ├── hooks/            # Custom React hooks
│   │   └── useWebSocket.ts
│   ├── services/         # API client
│   │   └── api.ts
│   ├── types/            # TypeScript types
│   │   └── index.ts
│   ├── App.tsx           # Main app component
│   ├── main.tsx          # Entry point
│   └── index.css         # Global styles
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── tailwind.config.js
```

## Components

### ScrapeControls
Date/month/year pickers with scrape buttons. Displays success/error messages and job IDs.

### DownloadControls
Buttons for downloading all CSVs, removing duplicates, and matching downloads.

### StatsPanel
Real-time statistics dashboard showing:
- Total CSVs and tracks scraped
- Download counts and storage usage
- NAS storage stats
- Date range coverage

### JobsList
Live-updating list of all jobs with:
- Status indicators (pending/running/completed/failed)
- Progress bars for running jobs
- Job results and error messages
- Timestamps and job IDs

### ConnectionStatus
WebSocket connection indicator with auto-reconnect.

## WebSocket Integration

The frontend connects to `ws://localhost:3002/api/v1/ws` (local dev) or `ws://your-domain/api/v1/ws` (production) for real-time updates:

```typescript
const { isConnected, jobs } = useWebSocket();
```

Jobs are automatically updated as they progress via WebSocket messages.

## API Integration

All API calls go through the `apiService`:

```typescript
import apiService from './services/api';

// Scraping
await apiService.scrapeDate('2026-01-28');
await apiService.scrapeMonth('2026-01');
await apiService.scrapeYear('2026');

// Downloads
await apiService.downloadAll();

// Utilities
await apiService.removeDuplicates();
await apiService.matchDownloads();
await apiService.getStats();
```

## Styling

Uses Tailwind CSS with a dark slate theme. Custom colors:
- Background: `bg-slate-900`
- Cards: `bg-slate-800`
- Borders: `border-slate-700`
- Text: `text-white`, `text-slate-300`, `text-slate-400`

## Proxy Configuration

Vite proxy redirects `/api` requests to the backend:

```typescript
proxy: {
  '/api': {
    target: 'http://localhost:3000',
    changeOrigin: true,
  }
}
```
