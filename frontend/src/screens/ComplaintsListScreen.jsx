import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api, { ApiError } from '../services/api';
import './ComplaintsListScreen.css';

function ComplaintsListScreen() {
  const navigate = useNavigate();
  const [complaints, setComplaints] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [retrying, setRetrying] = useState(false);

  useEffect(() => {
    // Check if phone is verified
    const phoneVerified = localStorage.getItem('phone_verified') === 'true';
    if (!phoneVerified) {
      // Redirect to phone verification if not verified
      navigate('/phone-verify');
      return;
    }
    
    loadComplaints();
  }, []);

  const loadComplaints = async (isRetry = false) => {
    try {
      if (!isRetry) {
        setLoading(true);
      } else {
        setRetrying(true);
      }
      setError(null);

      const response = await api.getUserComplaints();
      
      // Handle different response formats
      let complaintsList = [];
      if (Array.isArray(response)) {
        complaintsList = response;
      } else if (response.complaints) {
        complaintsList = response.complaints;
      } else if (response.items) {
        complaintsList = response.items;
      }

      setComplaints(complaintsList);
      setError(null);
    } catch (err) {
      let errorMessage = 'Complaints load nahi ho payi.';
      if (err instanceof ApiError) {
        if (err.status === 401) {
          errorMessage = 'Complaints dekhne ke liye phone verify karein.';
        } else if (err.status === 0) {
          errorMessage = 'Network error. Connection check karein.';
        } else {
          errorMessage = err.message || errorMessage;
        }
      } else {
        errorMessage = err.message || errorMessage;
      }

      setError(errorMessage);
    } finally {
      setLoading(false);
      setRetrying(false);
    }
  };

  const handleRetry = () => {
    loadComplaints(true);
  };

  const getStatusBadgeClass = (status) => {
    const s = (status || '').toLowerCase();
    if (s === 'submitted' || s === 'pending' || s === 'verified' || s === 'under_review') return 'status-badge--yellow';
    if (s === 'in_progress' || s === 'in-progress' || s === 'assigned') return 'status-badge--blue';
    if (s === 'escalated') return 'status-badge--red';
    if (s === 'resolved' || s === 'closed') return 'status-badge--green';
    return 'status-badge--gray';
  };

  const getStatusLabel = (status) => {
    const s = (status || '').replace(/_/g, ' ');
    return s ? s.charAt(0).toUpperCase() + s.slice(1).toLowerCase() : 'Unknown';
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('hi-IN', {
      day: 'numeric',
      month: 'short',
      year: 'numeric'
    });
  };

  if (loading) {
    return (
      <div className="complaints-list-screen">
        <div className="complaints-header">
          <button type="button" className="back-button" onClick={() => navigate('/')}>
            Back
          </button>
          <h2>My Complaints</h2>
        </div>
        <div className="loading">Complaints load ho rahi hain…</div>
      </div>
    );
  }

  return (
    <div className="complaints-list-screen">
      <div className="complaints-header">
        <button type="button" className="back-button" onClick={() => navigate('/')}>
          Back
        </button>
        <h2>My Complaints</h2>
      </div>

      {error && (
        <div className="error-container">
          <div className="error">{error}</div>
          {error.includes('Network error') && (
            <button type="button" className="btn btn-secondary mt-2" onClick={handleRetry} disabled={retrying}>
              {retrying ? 'Phir se try ho raha hai…' : 'Retry'}
            </button>
          )}
        </div>
      )}

      {!loading && !error && (
        <div className="complaints-content">
          <div className="complaints-actions">
            <button
              type="button"
              className="btn-register-new btn-primary"
              onClick={() => navigate('/chat')}
            >
              Register New Complaint
            </button>
          </div>
          {complaints.length === 0 ? (
            <div className="empty-state">
              <p className="empty-state-title">No complaints yet.</p>
              <p className="empty-state-sub">Tap &quot;Register New Complaint&quot; above or go to Home and use &quot;Talk with me&quot; to file one.</p>
              <button type="button" className="btn btn-primary" onClick={() => navigate('/chat')}>
                File First Complaint
              </button>
            </div>
          ) : (
            <div className="complaints-list complaints-list--cards">
              {complaints.map((complaint) => (
                <div
                  key={complaint.complaint_id}
                  className="complaint-card complaint-card--clickable"
                  onClick={() => navigate(`/complaints/${complaint.complaint_id}`)}
                  onKeyDown={(e) => e.key === 'Enter' && navigate(`/complaints/${complaint.complaint_id}`)}
                  role="button"
                  tabIndex={0}
                >
                  <div className="complaint-card__accent" data-status={complaint.current_status} />
                  <div className="complaint-card__top">
                    <span className="complaint-number complaint-number--large">
                      #{complaint.complaint_number}
                    </span>
                    <span className={`status-badge ${getStatusBadgeClass(complaint.current_status)}`}>
                      {getStatusLabel(complaint.current_status)}
                    </span>
                  </div>
                  <p className="complaint-title">{complaint.title || 'Complaint'}</p>
                  <div className="complaint-meta">
                    <span className="complaint-date">{formatDate(complaint.created_at)}</span>
                    {complaint.supporter_count > 0 && (
                      <span className="supporter-count">{complaint.supporter_count} supporters</span>
                    )}
                  </div>
                  <p className="complaint-card__hint">Tap to view details</p>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default ComplaintsListScreen;
