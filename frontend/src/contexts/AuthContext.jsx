import { createContext, useContext, useState, useEffect } from 'react';
import { getCurrentUser, logout as apiLogout } from '../api.js';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  // Get session token from cookie (set during login)
  const [sessionToken, setSessionToken] = useState(null);

  useEffect(() => {
    // Check if user is logged in on mount
    getCurrentUser()
      .then((userData) => {
        setUser(userData);
        // Session token is in cookie, no need to store in localStorage
      })
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  const login = (userData) => {
    setUser(userData);
    // Session token is already in HttpOnly cookie
    // We don't store it in localStorage for security
  };

  const logout = async () => {
    try {
      await apiLogout();
    } catch (e) {
      // Ignore errors
    }
    setUser(null);
    setSessionToken(null);
  };

  const value = {
    user,
    loading,
    isAuthenticated: !!user,
    sessionToken,
    login,
    logout,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
