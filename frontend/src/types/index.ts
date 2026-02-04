export interface Job {
  id: string;
  type: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  progress: number;
  message?: string; // Current operation message (e.g., "Downloading: Track Name - Artist")
  created_at: string;
  updated_at: string;
  result?: any;
  error?: string;
}

export interface Stats {
  total_csvs: number;
  total_downloads: number;
  downloads_size_gb: number;
  nas_downloads: number;
  nas_size_gb: number;
  earliest_date: string;
  latest_date: string;
  total_tracks_scraped: number;
}

export interface ScrapeRequest {
  date?: string;
  month?: string;
  year?: string;
}

export interface DownloadRequest {
  csv_path: string;
}

export interface WSMessage {
  type: string;
  data?: any;
}

export interface TrackInfo {
  filename: string;
  title: string;
  artist: string;
  album: string;
  year: number;
  genre: string;
  duration: number; // in seconds
  file_size: number; // in bytes
  file_path: string;
  mod_time: string;
}

export interface TrackListResult {
  tracks: TrackInfo[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
  in_progress?: boolean;
  scanned?: number;
}

export interface TrackScanStatus {
  in_progress: boolean;
  scanned: number;
  total: number;
  last_updated: string;
  error?: string;
}
