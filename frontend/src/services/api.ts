import axios from 'axios';
import type { Job, Stats, TrackListResult, TrackScanStatus, VersionInfo, AnalyticsStats } from '../types';

const API_BASE = '/api/v1';
const ANALYTICS_API_URL = import.meta.env.VITE_ANALYTICS_API_URL || 'http://localhost:8080';

const api = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const apiService = {
  // Health check
  health: () => api.get('/health'),

  // Version info
  getVersion: () => api.get<VersionInfo>('/version'),

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
  removeDuplicates: (nasDir?: string) =>
    api.post<{ message: string; job: Job }>('/util/deduplicate', { nas_dir: nasDir }),
  
  moveToNAS: (nasDir: string) =>
    api.post<{ message: string; job: Job }>('/util/move-to-nas', { nas_dir: nasDir }),
  
  matchDownloads: () =>
    api.post<{ message: string; job: Job }>('/util/match', {}),
  
  clearCSVs: () =>
    api.post<{ message: string; job: Job }>('/util/clear-csvs', {}),
  
  clearDownloads: () =>
    api.post<{ message: string; job: Job }>('/util/clear-downloads', {}),
  
  clearNASDownloads: (nasDir: string) =>
    api.post<{ message: string; job: Job }>('/util/clear-nas-downloads', { nas_dir: nasDir }),
  
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

  getAnalyzedTracks: async () => {
    try {
      // Analytics API has a max limit of 50, so we need to paginate
      const allTracks: any[] = [];
      let offset = 0;
      const limit = 50;
      let total = 0;
      
      // Fetch first page to get total count
      const firstResponse = await axios.get(`${ANALYTICS_API_URL}/api/v1/tracks`, {
        params: { status: 'completed', limit, offset }
      });
      
      if (firstResponse?.data) {
        total = firstResponse.data.total;
        allTracks.push(...firstResponse.data.tracks);
        offset += limit;
        
        // Fetch remaining pages
        while (offset < total) {
          const response = await axios.get(`${ANALYTICS_API_URL}/api/v1/tracks`, {
            params: { status: 'completed', limit, offset }
          });
          if (response?.data?.tracks) {
            allTracks.push(...response.data.tracks);
          }
          offset += limit;
        }
      }
      
      return { data: { tracks: allTracks, total } };
    } catch (error) {
      console.error('Analytics API not available:', error);
      throw error;
    }
  },

  // Audio Analysis
  analyzeTrack: (filePath: string) =>
    api.post<{ message: string; job: Job }>('/analyze/track', { file_path: filePath }),
  
  analyzeAll: (generateSpectrograms: boolean = false) =>
    api.post<{ message: string; job: Job }>('/analyze/all', { generate_spectrograms: generateSpectrograms }),
  
  // Analytics Service Integration
  getAnalyticsStats: async () => {
    try {
      const response = await axios.get<AnalyticsStats>(`${ANALYTICS_API_URL}/api/v1/stats`);
      return response;
    } catch (error) {
      console.error('Analytics API not available:', error);
      return { data: null };
    }
  },
  
  getAnalyticsDetailed: async () => {
    try {
      const response = await axios.get<AnalyticsStats>(`${ANALYTICS_API_URL}/api/v1/stats/detailed`);
      return response;
    } catch (error) {
      console.error('Analytics API not available:', error);
      return { data: null };
    }
  },
  
  // Check which filenames have been analyzed
  checkAnalyzedTracks: async (filenames: string[]) => {
    try {
      // Query analytics API for each filename
      const results = await Promise.all(
        filenames.map(async (filename) => {
          try {
            const response = await axios.get(`${ANALYTICS_API_URL}/api/v1/tracks/search?q=${encodeURIComponent(filename)}`);
            const tracks = response.data?.tracks || [];
            // Check if any track with this filename has been analyzed
            const analyzed = tracks.some((track: any) => 
              track.file_name === filename && track.analysis_status === 'completed'
            );
            return { filename, analyzed };
          } catch {
            return { filename, analyzed: false };
          }
        })
      );
      return results.reduce((acc, { filename, analyzed }) => {
        acc[filename] = analyzed;
        return acc;
      }, {} as Record<string, boolean>);
    } catch (error) {
      console.error('Failed to check analyzed tracks:', error);
      return {};
    }
  },
};

export default apiService;
