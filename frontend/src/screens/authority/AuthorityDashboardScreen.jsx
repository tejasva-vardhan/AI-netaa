import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { authorityApi, getAuthorityToken } from '../../services/authorityApi';

export default function AuthorityDashboardScreen() {
  const navigate = useNavigate();
  const [complaints, setComplaints] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!getAuthorityToken()) {
      navigate('/authority/login', { replace: true });
      return;
    }
    let cancelled = false;
    setLoading(true);
    authorityApi.complaints({ page, page_size: pageSize, status: statusFilter || undefined })
      .then((res) => {
        if (!cancelled) {
          setComplaints(res.complaints || []);
          setTotal(res.total ?? 0);
        }
      })
      .catch((err) => { if (!cancelled) setError(err.message); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [navigate, page, pageSize, statusFilter]);

  const handleLogout = () => {
    localStorage.removeItem('authority_token');
    navigate('/authority/login', { replace: true });
  };

  return (
    <div style={{ padding: 20 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h1>Assigned Complaints</h1>
        <button onClick={handleLogout} style={{ padding: '6px 12px' }}>Logout</button>
      </div>
      <div style={{ marginBottom: 12 }}>
        <label>Status filter </label>
        <select
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
          style={{ marginLeft: 8, padding: 4 }}
        >
          <option value="">All</option>
          <option value="submitted">submitted</option>
          <option value="under_review">under_review</option>
          <option value="in_progress">in_progress</option>
          <option value="resolved">resolved</option>
          <option value="escalated">escalated</option>
        </select>
      </div>
      {error && <p style={{ color: 'red' }}>{error}</p>}
      {loading ? <p>Loading...</p> : (
        <>
          <p>Total: {total}</p>
          <ul style={{ listStyle: 'none', padding: 0 }}>
            {complaints.map((c) => (
              <li key={c.complaint_id} style={{ border: '1px solid #ccc', marginBottom: 8, padding: 12 }}>
                <strong>{c.complaint_number}</strong> — {c.title}
                <br />
                Status: {c.current_status} | Priority: {c.priority} | Created: {c.created_at}
                <br />
                <button
                  type="button"
                  onClick={() => navigate(`/authority/complaints/${c.complaint_id}`, { state: { complaint: c } })}
                  style={{ marginTop: 8, padding: '4px 8px' }}
                >
                  View &amp; update status
                </button>
              </li>
            ))}
          </ul>
          {total > pageSize && (
            <div style={{ marginTop: 12 }}>
              <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>Prev</button>
              <span style={{ margin: '0 12px' }}>Page {page}</span>
              <button disabled={page * pageSize >= total} onClick={() => setPage((p) => p + 1)}>Next</button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
