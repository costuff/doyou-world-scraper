import axios from 'axios';
import type { Job, Stats, TrackListResult, TrackScanStatus } from '../types';

const API_BASE = '/api/v1';

const api = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const apiService = {
  // Health check
  health: () => api.get('/health'),

  // Scraping
  scrapeDate: (date: string) => 
    api.post<{ job_id: string }>('/scrape/date', { date }),
  
  scrapeMonth: (month: string) => 
    api.post<{ job_id: string }>('/scrape/month', { month }),
  
  scrapeYear: (year: string) => 
    api.post<{ job_id: string }>('/scrape/year', { year }),

  // Downloading
  downloadFromCSV: (csvPath: string) =>
    api.post<{ job_id: string }>('/download/csv', { csv_path: csvPath }),
  
  downloadAll: () =>
    api.post<{ job_id: string }>('/download/all'),

  // Utilities
  removeDuplicates: () =>
    api.post<{ job_id: string }>('/util/deduplicate'),
  
  matchDownloads: () =>
    api.post<{ job_id: string }>('/util/match'),
  
  getStats: () =>
    api.get<Stats>('/util/stats'),

  // Jobs
  getJob: (jobId: string) =>
    api.get<Job>(`/jobs/${jobId}`),
  
  getAllJobs: () =>
    api.get<Job[]>('/jobs'),

  // Tracks
  getTracks: (params: {
    page?: number;
    per_page?: number;
    search?: string;
    sort_by?: string;
    sort_order?: string;
    downloads_dir?: string;
    async?: boolean;
  }) =>
    api.get<TrackListResult>('/tracks', { params }),

  getTrackScanStatus: (params: { downloads_dir?: string }) =>
    api.get<TrackScanStatus>('/tracks/status', { params }),
  
  getStreamUrl: (filePath: string) =>
    `${API_BASE}/tracks/stream?file=${encodeURIComponent(filePath)}`,
};

export default apiService;
