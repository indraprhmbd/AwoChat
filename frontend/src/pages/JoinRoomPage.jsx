import { useState, useEffect } from 'react';
import { useSearchParams, useNavigate, Link } from 'react-router-dom';
import { joinRoom } from '../api';
import { useAuth } from '../contexts/AuthContext';
import Logo from '../../assets/android-chrome-192x192.png';

export default function JoinRoomPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { isAuthenticated, loading } = useAuth();
  
  const token = searchParams.get('token');
  const [roomName, setRoomName] = useState('');
  const [status, setStatus] = useState('loading'); // loading, joining, success, error
  const [error, setError] = useState('');

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      // Redirect to login if not authenticated
      navigate('/login?redirect=/join?token=' + token);
      return;
    }

    if (token && isAuthenticated) {
      handleJoin();
    }
  }, [token, isAuthenticated, loading]);

  async function handleJoin() {
    if (!token) {
      setError('Invalid invite link - no token provided');
      setStatus('error');
      return;
    }

    setStatus('joining');
    try {
      console.log('Joining room with token:', token);
      const room = await joinRoom(token);
      console.log('Joined room:', room);
      setRoomName(room.name);
      setStatus('success');
      // Redirect to room after 2 seconds
      setTimeout(() => {
        navigate(`/room/${room.id}`);
      }, 2000);
    } catch (err) {
      console.error('Join room error:', err);
      setError(err.message || 'Failed to join room. Make sure you are logged in and the invite link is valid.');
      setStatus('error');
    }
  }

  if (loading) {
    return (
      <div className="join-page">
        <div className="join-container">
          <div className="join-logo">
            <img src={Logo} alt="AwoChat" className="join-logo-img" />
          </div>
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return null; // Will redirect
  }

  return (
    <div className="join-page">
      <div className="join-container">
        <div className="join-logo">
          <img src={Logo} alt="AwoChat" className="join-logo-img" />
        </div>
        
        {status === 'loading' && (
          <p>Checking invite...</p>
        )}

        {status === 'joining' && (
          <>
            <h2>Joining Room...</h2>
            <div className="spinner"></div>
          </>
        )}

        {status === 'success' && (
          <>
            <h2>✓ Joined Successfully!</h2>
            <p>Welcome to <strong>{roomName}</strong></p>
            <p>Redirecting...</p>
          </>
        )}

        {status === 'error' && (
          <>
            <h2>Failed to Join</h2>
            <p className="error-message">{error}</p>
            <div className="join-actions">
              <Link to="/" className="btn btn-primary">Go to Dashboard</Link>
              <button onClick={handleJoin} className="btn btn-secondary">Try Again</button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
