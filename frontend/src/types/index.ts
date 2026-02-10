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
  analysis?: AudioAnalysis; // Optional audio analysis data
}

export interface AudioAnalysis {
  bpm: number;
  key: string;
  duration_seconds: number;
  energy: number; // 0-1
  danceability: number; // 0-1
  valence: number; // 0-1 (mood/happiness)
  acousticness: number; // 0-1
  instrumentalness: number; // 0-1
  speechiness: number; // 0-1
  liveness: number; // 0-1
  melodicness: number; // 0-1
  spectral_centroid_mean?: number;
  rms_energy_mean?: number;
  spectrogram_path?: string;
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

export interface VersionInfo {
  backend: string;
  frontend: string;
  commit?: string;
  build_time?: string;
}

export interface AnalyzedTrack {
  file_path: string;
  file_name: string;
  analysis: AudioAnalysis;
}

export interface AnalyzedTracksResult {
  tracks: AnalyzedTrack[];
  count: number;
}
