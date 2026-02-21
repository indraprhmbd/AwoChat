import { apiRequest } from '../api';

export default function DeleteConfirmModal({ roomId, roomName, onConfirm, onCancel }) {
  const handleDelete = async () => {
    try {
      await apiRequest(`/rooms/delete?room_id=${roomId}`, {
        method: 'DELETE',
      });
      onConfirm();
      window.location.href = '/'; // Redirect to dashboard
    } catch (err) {
      alert('Failed to delete room: ' + err.message);
    }
  };

  return (
    <div className="modal-overlay" onClick={onCancel}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>Delete Room</h2>
        <p className="modal-description">
          Are you sure you want to delete "<strong>{roomName}</strong>"? This action cannot be undone.
        </p>
        <div className="modal-actions">
          <button onClick={onCancel}>Cancel</button>
          <button onClick={handleDelete} className="btn btn-danger">
            Delete Room
          </button>
        </div>
      </div>
    </div>
  );
}
