export default function MembersModal({ members, onClose }) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>Room Members</h2>
        <div className="members-list">
          {members.map((member) => (
            <div key={member.id} className="member-item">
              <span className="member-email">{member.email}</span>
              <span className={`member-role ${member.role}`}>{member.role}</span>
            </div>
          ))}
        </div>
        <div className="modal-actions">
          <button onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}
