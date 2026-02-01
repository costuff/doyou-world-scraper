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
    api.post<{ message: string; job: Job }>('/scrape/date', { date }),
  
  scrapeMonth: (month: string) => 
    api.post<{ message: string; job: Job }>('/scrape/month', { month }),
  
  scrapeYear: (year: string) => 
    api.post<{ message: string; job: Job }>('/scrape/year', { year }),

  // Downloading
  downloadFromCSV: (csvPath: string) =>
    api.post<{ message: string; job: Job }>('/download/csv', { csv_path: csvPath }),
  
  downloadAll: () =>
    api.post<{ message: string; job: Job }>('/download/all'),

  // Utilities
  removeDuplicates: () =>
    api.post<{ message: string; job: Job }>('/util/deduplicate', {}),
  
  moveToNAS: () =>
    api.post<{ message: string; job: Job }>('/util/move-to-nas', {}),
  
  matchDownloads: () =>
    api.post<{ message: string; job: Job }>('/util/match', {}),
  
  clearCSVs: () =>
    api.post<{ message: string; job: Job }>('/util/clear-csvs', {}),
  
  getStats: () =>
    api.get<Stats>('/util/stats'),

  // Jobs
  getJob: (jobId: string) =>
    api.get<Job>(`/jobs/${jobId}`),
  
  getAllJobs: () =>
    api.get<{ jobs: Job[]; total: number }>('/jobs'),

  cancelJob: (jobId: string) =>
    api.post<{ message: string; job_id: string }>(`/jobs/${jobId}/cancel`, {}),

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
