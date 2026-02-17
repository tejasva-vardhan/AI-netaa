import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';

// Read-only public case page by complaint_number (shareable). No auth; complaint_id never exposed.
const API_BASE = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

function PublicCaseScreen() {
  const { complaintNumber } = useParams();
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (!complaintNumber) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    fetch(`${API_BASE}/public/complaints/by-number/${encodeURIComponent(complaintNumber)}`, { method: 'GET' })
      .then((res) => {
        if (!res.ok) {
          if (res.status === 404) throw new Error('Complaint not found');
          throw new Error(res.statusText || 'Failed to load');
        }
        return res.json();
      })
      .then((json) => {
        if (!cancelled) setData(json);
      })
      .catch((err) => {
        if (!cancelled) setError(err.message || 'Failed to load case');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, [complaintNumber]);

  if (loading) return <div style={{ padding: 16 }}>Loading...</div>;
  if (error) return <div style={{ padding: 16 }}>{error}</div>;
  if (!data) return null;

  return (
    <div style={{ padding: 16, maxWidth: 600 }}>
      <h1>Case: {data.complaint_number}</h1>
      <dl style={{ marginTop: 8 }}>
        <dt>Status</dt>
        <dd>{data.current_status}</dd>
        <dt>Department</dt>
        <dd>{data.department_id || '—'}</dd>
        <dt>Location</dt>
        <dd>{data.location_id}</dd>
        <dt>Created</dt>
        <dd>{data.created_at}</dd>
      </dl>
      <h2 style={{ marginTop: 24 }}>Timeline</h2>
      <ul style={{ listStyle: 'none', padding: 0 }}>
        {(data.timeline || []).map((entry, i) => (
          <li key={i} style={{ marginBottom: 8, padding: 8, border: '1px solid #eee' }}>
            <strong>{entry.new_status}</strong>
            {entry.old_status && <> (from {entry.old_status})</>}
            {' — '}{entry.created_at}
            {entry.actor_type && <> — {entry.actor_type}</>}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default PublicCaseScreen;
