const API_BASE = '/api';

export async function apiRequest(endpoint, options = {}) {
  const url = `${API_BASE}${endpoint}`;
  
  const config = {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    credentials: 'include',
  };
  
  const response = await fetch(url, config);
  
  if (!response.ok) {
    const text = await response.text();
    console.error(`API Error (${response.status}):`, text);
    throw new Error(text || `HTTP ${response.status}`);
  }
  
  // Handle empty responses
  const contentType = response.headers.get('content-type');
  if (contentType && contentType.includes('application/json')) {
    return response.json();
  }
  
  return response.text();
}

// Auth API
export async function signup(email, password) {
  return apiRequest('/auth/signup', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export async function login(email, password) {
  return apiRequest('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export async function logout() {
  return apiRequest('/auth/logout', { method: 'POST' });
}

export async function getCurrentUser() {
  return apiRequest('/auth/me');
}

// Rooms API
export async function createRoom(name) {
  return apiRequest('/rooms', {
    method: 'POST',
    body: JSON.stringify({ name }),
  });
}

export async function getRooms() {
  return apiRequest('/rooms');
}

export async function getRoom(roomId) {
  return apiRequest(`/rooms/${roomId}`);
}

export async function joinRoom(token) {
  return apiRequest('/rooms/join', {
    method: 'POST',
    body: JSON.stringify({ token }),
  });
}

export async function leaveRoom(roomId) {
  return apiRequest('/rooms/leave', {
    method: 'POST',
    body: JSON.stringify({ room_id: roomId }),
  });
}

export async function getRoomMembers(roomId) {
  return apiRequest(`/rooms/members?room_id=${roomId}`);
}

// Messages API
export async function getMessages(roomId, limit = 50, offset = 0) {
  return apiRequest(`/messages?room_id=${roomId}&limit=${limit}&offset=${offset}`);
}
