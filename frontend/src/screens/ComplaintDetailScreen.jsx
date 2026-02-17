import { useState, useEffect } from 'react';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import api, { ApiError } from '../services/api';
import StatusTimeline from '../components/StatusTimeline';
import './ComplaintDetailScreen.css';

function ComplaintDetailScreen() {
  const navigate = useNavigate();
  const location = useLocation();
  const { id } = useParams();
  const [complaint, setComplaint] = useState(null);
  const [timeline, setTimeline] = useState([]);
  const [loading, setLoading] = useState(true);
  const [timelineLoading, setTimelineLoading] = useState(false);
  const [error, setError] = useState(null);
  const [timelineError, setTimelineError] = useState(null);
  const [showTimeline, setShowTimeline] = useState(false);
  const [successMessage, setSuccessMessage] = useState(null);

  useEffect(() => {
    // Check if phone is verified
    const phoneVerified = localStorage.getItem('phone_verified') === 'true';
    if (!phoneVerified) {
      // Redirect to phone verification if not verified
      navigate('/phone-verify');
      return;
    }
    
    // Check for success message from navigation state
    if (location.state?.success) {
      setSuccessMessage(location.state.message);
      // Clear state after showing message
      window.history.replaceState({}, document.title);
    }
    loadComplaint();
  }, [id]);

  useEffect(() => {
    if (showTimeline && timeline.length === 0) {
      loadTimeline();
    }
  }, [showTimeline]);

  const loadComplaint = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.getComplaint(id);
      setComplaint(data);
    } catch (err) {
      let errorMessage = 'Complaint load nahi ho payi.';
      if (err instanceof ApiError) {
        if (err.status === 404) {
          errorMessage = 'Complaint nahi mili.';
        } else if (err.status === 401) {
          errorMessage = 'Is complaint ko dekhne ki permission nahi hai.';
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
    }
  };

  const loadTimeline = async () => {
    try {
      setTimelineLoading(true);
      setTimelineError(null);
      const data = await api.getComplaintTimeline(id);
      setTimeline(data.timeline || []);
    } catch (err) {
      let errorMessage = 'Timeline load nahi ho payi.';
      if (err instanceof ApiError) {
        if (err.status === 0) {
          errorMessage = 'Network error. Connection check karein.';
        } else {
          errorMessage = err.message || errorMessage;
        }
      }

      setTimelineError(errorMessage);
      console.error('Failed to load timeline:', err);
    } finally {
      setTimelineLoading(false);
    }
  };

  const handleRetry = () => {
    loadComplaint();
  };

  const getStatusColor = (status) => {
    const colors = {
      'submitted': '#1E3A8A',
      'verified': '#0EA5A4',
      'under_review': '#F59E0B',
      'in_progress': '#1E3A8A',
      'resolved': '#0EA5A4',
      'rejected': '#DC2626',
      'closed': '#64748B',
      'escalated': '#1E40AF'
    };
    return colors[status] || '#64748B';
  };

  if (loading) {
    return (
      <div className="complaint-detail-screen">
        <div className="complaint-header">
          <button type="button" className="back-button" onClick={() => navigate('/complaints')}>
            Back
          </button>
          <h2>Complaint Details</h2>
        </div>
        <div className="loading">Load ho raha hai…</div>
      </div>
    );
  }

  if (error || !complaint) {
    return (
      <div className="complaint-detail-screen">
        <div className="complaint-header">
          <button type="button" className="back-button" onClick={() => navigate('/complaints')}>
            Back
          </button>
          <h2>Complaint Details</h2>
        </div>
        <div className="error">{error || 'Complaint nahi mili.'}</div>
      </div>
    );
  }

  return (
    <div className="complaint-detail-screen">
      <div className="complaint-header">
        <button type="button" className="back-button" onClick={() => navigate('/complaints')}>
          Back
        </button>
        <h2>Complaint Details</h2>
      </div>

      <div className="complaint-content">
        {successMessage && (
          <div className="success">
            {successMessage}
          </div>
        )}

        {error && (
          <div className="error-container">
            <div className="error">{error}</div>
            {error.includes('Network error') && (
              <button className="btn btn-secondary mt-2" onClick={handleRetry}>
                Retry
              </button>
            )}
          </div>
        )}

        {!error && (
          <div className="complaint-detail-inner">
            <div
              className="complaint-section complaint-hero"
              style={{ borderLeftColor: getStatusColor(complaint.current_status) }}
            >
              <div className="complaint-number-large">{complaint.complaint_number}</div>
              <span
                className="status-badge-large"
                style={{ backgroundColor: getStatusColor(complaint.current_status) }}
              >
                {complaint.current_status.replace('_', ' ')}
              </span>
            </div>

        <div className="complaint-section">
          <h3>Summary</h3>
          <p>{complaint.title}</p>
        </div>

        <div className="complaint-section">
          <h3>Description</h3>
          <p>{complaint.description}</p>
        </div>

        {complaint.category && (
          <div className="complaint-section">
            <h3>Category</h3>
            <p>{complaint.category}</p>
          </div>
        )}

        <div className="complaint-section">
          <h3>Priority</h3>
          <p className="priority-text">{complaint.priority}</p>
        </div>

        <div className="complaint-section">
          <h3>Created</h3>
          <p>{new Date(complaint.created_at).toLocaleString('hi-IN')}</p>
        </div>

        {complaint.attachments && complaint.attachments.length > 0 && (
          <div className="complaint-section">
            <h3>Attachments</h3>
            <div className="attachments-grid">
              {complaint.attachments.map((att, idx) => (
                <img
                  key={idx}
                  src={att.file_path}
                  alt={att.file_name}
                  className="attachment-image"
                />
              ))}
            </div>
          </div>
        )}

            <div className="complaint-actions">
              <button
                type="button"
                className="btn btn-primary"
                onClick={() => setShowTimeline(!showTimeline)}
                disabled={timelineLoading}
              >
                {showTimeline ? 'Band karein' : 'Status history dekhein'}
              </button>
            </div>

            {showTimeline && (
              <div className="timeline-container">
                {timelineLoading && <div className="loading">Timeline load ho rahi hai…</div>}
                {timelineError && (
                  <div className="error">
                    {timelineError}
                    <button type="button" className="btn btn-secondary mt-2" onClick={loadTimeline}>
                      Retry
                    </button>
                  </div>
                )}
                {!timelineLoading && !timelineError && (
                  <StatusTimeline timeline={timeline} />
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export default ComplaintDetailScreen;
