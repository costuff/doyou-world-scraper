import { useEffect, useRef, useState, useCallback } from 'react';
import type { Job, WSMessage } from '../types';
import { apiService } from '../services/api';

// Dynamically determine WebSocket URL based on environment
const getWebSocketURL = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.host;
  
  // Use current host for production, localhost:3002 for local development
  if (host.includes('localhost') || host.includes('127.0.0.1')) {
    return 'ws://localhost:3002/api/v1/ws'; // Port 3002 for local dev (avoids VS Code conflict)
  }
  
  return `${protocol}//${host}/api/v1/ws`;
};

const WS_URL = getWebSocketURL();

export const useWebSocket = () => {
  const [isConnected, setIsConnected] = useState(false);
  const [jobs, setJobs] = useState<Record<string, Job>>({});
  const ws = useRef<WebSocket | null>(null);
  const reconnectTimeout = useRef<number>();

  // Fetch existing jobs on connect
  const fetchJobs = useCallback(async () => {
    try {
      const response = await apiService.getAllJobs();
      const jobsMap: Record<string, Job> = {};
      // API returns { jobs: Job[], total: number }
      if (response.data && Array.isArray(response.data.jobs)) {
        response.data.jobs.forEach((job: Job) => {
          jobsMap[job.id] = job;
        });
        setJobs(jobsMap);
      }
    } catch (error) {
      console.error('Failed to fetch jobs:', error);
    }
  }, []);

  const connect = useCallback(() => {
    if (ws.current?.readyState === WebSocket.OPEN) return;

    ws.current = new WebSocket(WS_URL);

    ws.current.onopen = () => {
      console.log('WebSocket connected');
      setIsConnected(true);
      // Fetch existing jobs when WebSocket connects
      fetchJobs();
    };

    ws.current.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data);
        
        if (message.type === 'job_update' && message.data && message.data.id) {
          setJobs(prev => {
            const existingJob = prev[message.data.id];
            // Quick check: only update if status, progress, message, or error changed
            if (existingJob && 
                existingJob.status === message.data.status && 
                existingJob.progress === message.data.progress &&
                existingJob.message === message.data.message &&
                existingJob.error === message.data.error) {
              return prev;
            }
            return {
              ...prev,
              [message.data.id]: message.data
            };
          });
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error, event.data);
      }
    };

    ws.current.onclose = () => {
      console.log('WebSocket disconnected');
      setIsConnected(false);
      
      // Attempt to reconnect after 3 seconds
      reconnectTimeout.current = setTimeout(() => {
        console.log('Attempting to reconnect...');
        connect();
      }, 3000);
    };

    ws.current.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }, []);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      ws.current?.close();
    };
  }, [connect]);

  return { isConnected, jobs };
};
