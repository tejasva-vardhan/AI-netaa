import { useState } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { authorityApi } from '../../services/authorityApi';

// Allowed transitions (Authority only): submitted→under_review, under_review→in_progress, in_progress→resolved, escalated→under_review|in_progress
const NEXT_STATUS = {
  submitted: ['under_review'],
  under_review: ['in_progress'],
  in_progress: ['resolved'],
  escalated: ['under_review', 'in_progress'],
};

export default function AuthorityComplaintDetailScreen() {
  const { id } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const complaint = location.state?.complaint;

  const [reason, setReason] = useState('');
  const [selectedStatus, setSelectedStatus] = useState('');
  const [message, setMessage] = useState(null); // { type: 'success'|'error', text }
  const [loading, setLoading] = useState(false);

  if (!complaint || String(complaint.complaint_id) !== String(id)) {
    return (
      <div style={{ padding: 20 }}>
        <p>Complaint not found. Go back to list.</p>
        <button onClick={() => navigate('/authority/dashboard')}>Back to dashboard</button>
      </div>
    );
  }

  const currentStatus = complaint.current_status || '';
  const allowedNext = NEXT_STATUS[currentStatus] || [];

  const handleUpdateStatus = async (e) => {
    e.preventDefault();
    if (!selectedStatus || !reason.trim()) {
      setMessage({ type: 'error', text: 'Select a status and enter a reason.' });
      return;
    }
    setMessage(null);
    setLoading(true);
    try {
      await authorityApi.updateStatus(complaint.complaint_id, selectedStatus, reason.trim());
      setMessage({ type: 'success', text: 'Status updated successfully.' });
      setReason('');
      setSelectedStatus('');
      // Update local state so UI reflects new status
      complaint.current_status = selectedStatus;
    } catch (err) {
      setMessage({ type: 'error', text: err.message || 'Update failed.' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ padding: 20, maxWidth: 600 }}>
      <button onClick={() => navigate('/authority/dashboard')} style={{ marginBottom: 16 }}>← Back to list</button>
      <h2>Complaint: {complaint.complaint_number}</h2>
      <div style={{ marginBottom: 16, padding: 12, background: '#f5f5f5' }}>
        <p><strong>Title:</strong> {complaint.title}</p>
        <p><strong>Status:</strong> {complaint.current_status}</p>
        <p><strong>Priority:</strong> {complaint.priority}</p>
        <p><strong>Created:</strong> {complaint.created_at}</p>
        {complaint.supporter_count != null && <p><strong>Supporters:</strong> {complaint.supporter_count}</p>}
      </div>

      <h3>Update status</h3>
      {allowedNext.length === 0 ? (
        <p>No further status changes allowed from &quot;{currentStatus}&quot; (e.g. resolved or closed).</p>
      ) : (
        <form onSubmit={handleUpdateStatus}>
          <div style={{ marginBottom: 8 }}>
            <label>New status </label>
            <select
              value={selectedStatus}
              onChange={(e) => setSelectedStatus(e.target.value)}
              style={{ marginLeft: 8, padding: 4 }}
            >
              <option value="">-- Select --</option>
              {allowedNext.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          </div>
          <div style={{ marginBottom: 8 }}>
            <label>Reason (required) </label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              required
              rows={3}
              style={{ display: 'block', width: '100%', marginTop: 4, padding: 8 }}
            />
          </div>
          {message && (
            <p style={{ color: message.type === 'error' ? 'red' : 'green', marginBottom: 8 }}>{message.text}</p>
          )}
          <button type="submit" disabled={loading} style={{ padding: '8px 16px' }}>
            {loading ? 'Updating...' : 'Update status'}
          </button>
        </form>
      )}
    </div>
  );
}
