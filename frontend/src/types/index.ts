export interface Job {
  id: string;
  type: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress: number;
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
