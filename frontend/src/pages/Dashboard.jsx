import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { getRooms, createRoom, joinRoom } from '../api';
import { useAuth } from '../contexts/AuthContext';
import AwoChatLogo from '../../assets/AwoChat.webp';
import ChatRoom from './ChatRoom';
import { House, Plus, SignOut, List, X } from '@phosphor-icons/react';

export default function Dashboard() {
  const { roomId } = useParams();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const [rooms, setRooms] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newRoomName, setNewRoomName] = useState('');
  const [inviteToken, setInviteToken] = useState('');
  const [showMobileMenu, setShowMobileMenu] = useState(false);

  useEffect(() => {
    loadRooms();
  }, []);

  async function loadRooms() {
    try {
      const data = await getRooms();
      setRooms(data);
      setError('');
    } catch (err) {
      console.error('Failed to load rooms:', err);
      setError('Failed to load rooms: ' + (err.message || 'Unknown error'));
    } finally {
      setLoading(false);
    }
  }

  async function handleCreateRoom(e) {
    e.preventDefault();
    if (!newRoomName.trim()) return;

    try {
      const room = await createRoom(newRoomName.trim());
      setRooms((prev) => [...prev, room]);
      setNewRoomName('');
      setShowCreateModal(false);
      navigate(`/room/${room.id}`);
    } catch (err) {
      console.error('Failed to create room:', err);
      alert('Failed to create room: ' + (err.message || 'Unknown error'));
    }
  }

  async function handleJoinRoom(e) {
    e.preventDefault();
    if (!inviteToken.trim()) return;

    try {
      const room = await joinRoom(inviteToken.trim());
      await loadRooms();
      setInviteToken('');
      navigate(`/room/${room.id}`);
    } catch (err) {
      alert('Failed to join room: ' + err.message);
    }
  }

  async function handleLogout() {
    await logout();
    navigate('/login');
  }

  if (loading) {
    return <div className="dashboard loading">Loading...</div>;
  }

  return (
    <div className="dashboard">
      {/* Mobile Menu Toggle */}
      <button 
        className="mobile-menu-toggle"
        onClick={() => setShowMobileMenu(!showMobileMenu)}
      >
        {showMobileMenu ? <X size={24} /> : <List size={24} />}
      </button>

      {/* Sidebar - Always Visible (WhatsApp Web Style) */}
      <aside className={`sidebar ${showMobileMenu && roomId ? 'sidebar-hidden-mobile' : ''}`}>
        <div className="sidebar-header">
          <div className="logo-container-full">
            <img src={AwoChatLogo} alt="AwoChat" className="logo-img-full" />
          </div>
          <div className="user-menu">
            <span className="user-email">{user?.email}</span>
            <button onClick={handleLogout} className="btn btn-small btn-logout">
              <SignOut size={16} />
            </button>
          </div>
        </div>

        <div className="sidebar-actions">
          <button 
            className="btn btn-primary w-full" 
            onClick={() => setShowCreateModal(true)}
          >
            <Plus size={20} weight="bold" /> New Room
          </button>
        </div>

        <div className="join-room-form">
          <form onSubmit={handleJoinRoom}>
            <input
              type="text"
              placeholder="Invite token"
              value={inviteToken}
              onChange={(e) => setInviteToken(e.target.value)}
              className="w-full"
            />
            <button type="submit" className="btn btn-secondary w-full mt-sm">
              Join Room
            </button>
          </form>
        </div>

        <nav className="room-list">
          <h3>Your Rooms</h3>
          {rooms.length === 0 ? (
            <p className="empty-state">No rooms yet. Create one or join with an invite.</p>
          ) : (
            <ul>
              {rooms.map((room) => (
                <li key={room.id}>
                  <Link to={`/room/${room.id}`}>{room.name}</Link>
                </li>
              ))}
            </ul>
          )}
        </nav>
      </aside>

      {/* Main Content - Changes Based on Selection */}
      <main className="main-content">
        {roomId ? (
          <ChatRoom />
        ) : (
          <div className="welcome-panel">
            <h2>Welcome to AwoChat</h2>
            <p>Select a room from the sidebar or create a new one to start chatting.</p>
            <div className="features">
              <div className="feature">
                <h4>🔒 Private Rooms</h4>
                <p>Join rooms via invite links only</p>
              </div>
              <div className="feature">
                <h4>⚡ Real-time</h4>
                <p>Instant messaging via WebSocket</p>
              </div>
              <div className="feature">
                <h4>💾 Persistent</h4>
                <p>Message history saved</p>
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Create Room Modal */}
      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>Create New Room</h2>
            <form onSubmit={handleCreateRoom}>
              <div className="form-group">
                <label htmlFor="roomName">Room Name</label>
                <input
                  type="text"
                  id="roomName"
                  value={newRoomName}
                  onChange={(e) => setNewRoomName(e.target.value)}
                  placeholder="My Room"
                  autoFocus
                  required
                />
              </div>
              <div className="modal-actions">
                <button type="button" onClick={() => setShowCreateModal(false)}>
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary">
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
