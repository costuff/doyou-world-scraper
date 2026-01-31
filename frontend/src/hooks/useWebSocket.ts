import { useEffect, useRef, useState, useCallback } from 'react';
import type { Job, WSMessage } from '../types';

const WS_URL = 'ws://localhost:3000/api/v1/ws';

export const useWebSocket = () => {
  const [isConnected, setIsConnected] = useState(false);
  const [jobs, setJobs] = useState<Record<string, Job>>({});
  const ws = useRef<WebSocket | null>(null);
  const reconnectTimeout = useRef<number>();

  const connect = useCallback(() => {
    if (ws.current?.readyState === WebSocket.OPEN) return;

    ws.current = new WebSocket(WS_URL);

    ws.current.onopen = () => {
      console.log('WebSocket connected');
      setIsConnected(true);
    };

    ws.current.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data);
        
        if (message.type === 'job_update' && message.data && message.data.id) {
          setJobs(prev => {
            // Prevent infinite re-renders by checking if job actually changed
            const existingJob = prev[message.data.id];
            if (existingJob && JSON.stringify(existingJob) === JSON.stringify(message.data)) {
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
