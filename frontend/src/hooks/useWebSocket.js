import { useState, useEffect, useCallback, useRef } from 'react';
import { useAuth } from '../contexts/AuthContext';

const WS_BASE = window.location.protocol === 'https:' ? 'wss:' : 'ws:';

// Use same origin as the frontend (works with reverse proxy)
const WS_HOST = window.location.host;

export function useWebSocket(roomId, callbacks = {}) {
  const { user, sessionToken } = useAuth();
  const [isConnected, setIsConnected] = useState(false);
  const [typingUsers, setTypingUsers] = useState([]);
  const wsRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);
  const lastTypingSentRef = useRef(0);
  const reconnectCountRef = useRef(0);
  const MAX_RECONNECTS = 3;
  const isConnectingRef = useRef(false);
  const callbacksRef = useRef(callbacks); // Store callbacks in ref

  // Update callbacks ref when callbacks change
  useEffect(() => {
    callbacksRef.current = callbacks;
  }, [callbacks]);

  const connect = useCallback(() => {
    if (!roomId || !user?.id) return;
    if (isConnectingRef.current) return;
    if (wsRef.current?.readyState === WebSocket.OPEN) return;
    if (wsRef.current?.readyState === WebSocket.CONNECTING) return;

    isConnectingRef.current = true;

    // WebSocket will use session cookie for authentication
    let wsUrl = `${WS_BASE}//${WS_HOST}/ws?room_id=${roomId}`;
    
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
      reconnectCountRef.current = 0;
      isConnectingRef.current = false;
    };

    ws.onclose = (e) => {
      setIsConnected(false);
      isConnectingRef.current = false;
      
      if (reconnectCountRef.current < MAX_RECONNECTS) {
        reconnectCountRef.current += 1;
        reconnectTimeoutRef.current = setTimeout(() => {
          connect();
        }, 5000);
      }
    };

    ws.onerror = (error) => {
      isConnectingRef.current = false;
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        const cbs = callbacksRef.current; // Use callbacks from ref
        
        switch (data.type) {
          case 'message':
            cbs?.onMessage?.(data);
            break;
          case 'typing':
            setTypingUsers((prev) => {
              if (!prev.includes(data.user_id)) {
                return [...prev, data.user_id];
              }
              return prev;
            });
            
            // Remove typing indicator after 6 seconds
            setTimeout(() => {
              setTypingUsers((prev) => prev.filter((id) => id !== data.user_id));
            }, 6000);
            break;
          case 'error':
            console.error('Server error:', data.error);
            break;
        }
      } catch (err) {
        console.error('Failed to parse message:', err);
      }
    };

    wsRef.current = ws;
  }, [roomId, user?.id]); // Removed callbacks from dependencies

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
      reconnectCountRef.current = 0;
      isConnectingRef.current = false;
      wsRef.current = null;
    };
  }, [connect]);

  const sendMessage = useCallback((content) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'message',
        content,
      }));
    }
  }, []);

  const sendTyping = useCallback(() => {
    const now = Date.now();
    
    // Throttle: max 1 typing event per second
    if (now - lastTypingSentRef.current < 1000) {
      return;
    }
    
    lastTypingSentRef.current = now;
    
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'typing',
      }));
    }
  }, []);

  return {
    isConnected,
    typingUsers,
    sendMessage,
    sendTyping,
  };
}
