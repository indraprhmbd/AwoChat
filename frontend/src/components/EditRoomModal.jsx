import { useState } from 'react';
import { apiRequest } from '../api';

export default function EditRoomModal({ room, onClose }) {
  const [name, setName] = useState(room.name);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!name.trim()) return;

    setLoading(true);
    try {
      await apiRequest(`/rooms/update?room_id=${room.id}`, {
        method: 'PUT',
        body: JSON.stringify({ name: name.trim() }),
      });
      onClose();
      window.location.reload(); // Reload to show updated name
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>Edit Room Name</h2>
        <form onSubmit={handleSubmit}>
          {error && <div className="modal-error">{error}</div>}
          <div className="form-group">
            <label htmlFor="roomName">Room Name</label>
            <input
              type="text"
              id="roomName"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My Room"
              autoFocus
              required
            />
          </div>
          <div className="modal-actions">
            <button type="button" onClick={onClose}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={loading}>
              {loading ? 'Saving...' : 'Save'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
