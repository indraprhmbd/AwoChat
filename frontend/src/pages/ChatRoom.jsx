import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { getMessages, getRoom, leaveRoom } from '../api';
import { useAuth } from '../contexts/AuthContext';
import { useWebSocket } from '../hooks/useWebSocket';
import { ArrowLeft, DotsThreeVertical, Users, Link as LinkIcon, Pencil, Trash, Door, Plus, PaperPlaneRight } from '@phosphor-icons/react';
import MembersModal from '../components/MembersModal';
import EditRoomModal from '../components/EditRoomModal';
import InviteLinkModal from '../components/InviteLinkModal';
import DeleteConfirmModal from '../components/DeleteConfirmModal';

export default function ChatRoom() {
  const { roomId } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();

  const [room, setRoom] = useState(null);
  const [messages, setMessages] = useState([]);
  const [newMessage, setNewMessage] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState('');
  const messagesEndRef = useRef(null);
  const messagesContainerRef = useRef(null);
  const typingTimeoutRef = useRef(null);
  const loadedCountRef = useRef(50);

  const [showMenu, setShowMenu] = useState(false);
  const menuRef = useRef(null);

  const [showMembers, setShowMembers] = useState(false);
  const [showEditRoom, setShowEditRoom] = useState(false);
  const [showInviteLink, setShowInviteLink] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const { sendMessage, typingUsers, isConnected } = useWebSocket(roomId, {
    onMessage: (msg) => {
      setMessages((prev) => [...prev, msg]);
    },
    onTyping: () => {},
  });

  useEffect(() => {
    function handleClickOutside(event) {
      if (menuRef.current && !menuRef.current.contains(event.target)) {
        setShowMenu(false);
      }
    }
    if (showMenu) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [showMenu]);

  useEffect(() => {
    async function loadRoom() {
      try {
        const [roomData, messagesData] = await Promise.all([
          getRoom(roomId),
          getMessages(roomId, 50, 0),
        ]);
        setRoom(roomData);
        setMessages(messagesData || []);
        loadedCountRef.current = (messagesData || []).length;
        setError('');
      } catch (err) {
        console.error('Failed to load room:', err);
        setError('Failed to load room: ' + (err.message || 'Unknown error'));
      } finally {
        setLoading(false);
      }
    }
    loadRoom();
  }, [roomId]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleScroll = useCallback(async () => {
    const container = messagesContainerRef.current;
    if (!container || loadingMore || loadedCountRef.current < 50) return;

    if (container.scrollTop < 50) {
      setLoadingMore(true);
      try {
        const moreMessages = await getMessages(roomId, 50, loadedCountRef.current);
        if (moreMessages.length > 0) {
          const previousHeight = container.scrollHeight;
          setMessages((prev) => [...moreMessages, ...prev]);
          loadedCountRef.current += moreMessages.length;
          setTimeout(() => {
            container.scrollTop = container.scrollHeight - previousHeight;
          }, 0);
        }
      } catch (err) {
        console.error('Failed to load more messages:', err);
      } finally {
        setLoadingMore(false);
      }
    }
  }, [roomId, loadingMore]);

  const handleSendMessage = useCallback(() => {
    if (!newMessage.trim()) return;
    sendMessage(newMessage.trim());
    setNewMessage('');
  }, [newMessage, sendMessage]);

  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  const handleTyping = useCallback(() => {
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }
    typingTimeoutRef.current = setTimeout(() => {}, 2000);
  }, []);

  const handleLeaveRoom = async () => {
    if (!confirm('Are you sure you want to leave this room?')) return;
    try {
      await leaveRoom(roomId);
      navigate('/');
    } catch (err) {
      alert('Failed to leave room: ' + err.message);
    }
  };

  if (loading) {
    return <div className="chat-room loading">Loading...</div>;
  }

  if (error) {
    return (
      <div className="chat-room error">
        <p>{error}</p>
        <button onClick={() => navigate('/')}>Back to Rooms</button>
      </div>
    );
  }

  return (
    <div className="chat-room-container">
      <div className="chat-header">
        <div className="chat-header-info">
          <button className="btn-back" onClick={() => navigate('/')} aria-label="Back to rooms">
            <ArrowLeft size={24} />
          </button>
          <h2>{room?.name}</h2>
          <span className={`connection-status ${isConnected ? 'connected' : 'disconnected'}`} title={isConnected ? 'Connected' : 'Connecting...'}>
            ●
          </span>
        </div>
        <div className="chat-header-actions">
          <div className="chat-header-members">
            {room?.members?.length || 0}
          </div>
          <div className="menu-container" ref={menuRef}>
            <button className="btn-menu" onClick={() => setShowMenu(!showMenu)} aria-label="Menu">
              <DotsThreeVertical size={24} />
            </button>
            {showMenu && (
              <div className="menu-dropdown">
                <button onClick={() => { setShowInviteLink(true); setShowMenu(false); }}>
                  <LinkIcon size={18} weight="bold" /> Create Invite Link
                </button>
                <button onClick={() => { setShowMembers(true); setShowMenu(false); }}>
                  <Users size={18} weight="bold" /> View Members
                </button>
                {room?.owner_id === user?.id && (
                  <>
                    <button onClick={() => { setShowEditRoom(true); setShowMenu(false); }}>
                      <Pencil size={18} weight="bold" /> Edit Room Name
                    </button>
                    <button onClick={() => { setShowDeleteConfirm(true); setShowMenu(false); }} className="danger">
                      <Trash size={18} weight="bold" /> Delete Room
                    </button>
                  </>
                )}
                <button onClick={handleLeaveRoom} className="danger">
                  <Door size={18} weight="bold" /> Leave Room
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="messages-container" ref={messagesContainerRef} onScroll={handleScroll}>
        {loadingMore && <div className="loading-more">Loading older messages...</div>}
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`message ${msg.user_id === user?.id ? 'message-own' : 'message-other'}`}
          >
            <div className="message-header">
              <span className="message-author">{msg.user_email}</span>
              <span className="message-time">
                {new Date(msg.created_at).toLocaleTimeString()}
              </span>
            </div>
            <div className="message-content">{msg.content}</div>
          </div>
        ))}
        <div ref={messagesEndRef} />
        {typingUsers.length > 0 && (
          <div className="typing-indicator">
            {typingUsers.length} {typingUsers.length === 1 ? 'person' : 'people'} typing...
          </div>
        )}
      </div>

      <div className="message-input-container">
        <textarea
          value={newMessage}
          onChange={(e) => {
            setNewMessage(e.target.value);
            handleTyping();
          }}
          onKeyPress={handleKeyPress}
          placeholder="Type a message..."
          rows={1}
        />
        <button className="btn btn-primary btn-send" onClick={handleSendMessage} disabled={!newMessage.trim()}>
          <PaperPlaneRight size={20} weight="fill" />
        </button>
      </div>

      {showMembers && room && (
        <MembersModal members={room.members} onClose={() => setShowMembers(false)} />
      )}
      {showEditRoom && room && (
        <EditRoomModal room={room} onClose={() => setShowEditRoom(false)} />
      )}
      {showInviteLink && room && (
        <InviteLinkModal roomId={room.id} onClose={() => setShowInviteLink(false)} />
      )}
      {showDeleteConfirm && room && (
        <DeleteConfirmModal
          roomId={room.id}
          roomName={room.name}
          onConfirm={() => setShowDeleteConfirm(false)}
          onCancel={() => setShowDeleteConfirm(false)}
        />
      )}
    </div>
  );
}
