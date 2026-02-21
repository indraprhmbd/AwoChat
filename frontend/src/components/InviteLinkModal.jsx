import { useState, useEffect } from 'react';
import { apiRequest } from '../api';

export default function InviteLinkModal({ roomId, onClose }) {
  const [inviteToken, setInviteToken] = useState('');
  const [loading, setLoading] = useState(true);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    async function fetchInviteToken() {
      try {
        const data = await apiRequest(`/rooms/invite?room_id=${roomId}`);
        setInviteToken(data.invite_token);
      } catch (err) {
        console.error('Failed to fetch invite token:', err);
      } finally {
        setLoading(false);
      }
    }
    fetchInviteToken();
  }, [roomId]);

  const inviteLink = `${window.location.origin}/join?token=${inviteToken}`;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(inviteLink);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  if (loading) {
    return (
      <div className="modal-overlay" onClick={onClose}>
        <div className="modal" onClick={(e) => e.stopPropagation()}>
          <h2>Invite Link</h2>
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>Invite Link</h2>
        <p className="modal-description">
          Share this link with others to let them join this room:
        </p>
        <div className="invite-link-box">
          <input type="text" value={inviteLink} readOnly className="invite-link-input" />
          <button onClick={handleCopy} className="btn btn-primary">
            {copied ? '✓ Copied!' : 'Copy'}
          </button>
        </div>
        <div className="modal-description" style={{ marginTop: '1rem', fontSize: '0.875rem' }}>
          Or share this token: <strong>{inviteToken}</strong>
        </div>
        <div className="modal-actions">
          <button onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}
