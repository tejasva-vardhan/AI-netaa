import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useComplaintState } from '../state/ComplaintContext';
import api, { ApiError } from '../services/api';
import { saveToQueue } from '../utils/offlineQueue';
import './ReviewScreen.css';

function ReviewScreen() {
  const navigate = useNavigate();
  const { complaintData, clearComplaintData, updateComplaintData, addMessage } = useComplaintState();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);
  const [retryCount, setRetryCount] = useState(0);

  useEffect(() => {
    // ISSUE 3: ReviewScreen visibility rules - only render when ALL required steps complete
    const requiredSteps = {
      summary: !!complaintData.summary,
      description: !!complaintData.description,
      location: !!complaintData.location,
      photo: !!complaintData.photo
    };

    // Check if phone is verified
    const phoneVerified = localStorage.getItem('phone_verified') === 'true';
    if (!phoneVerified) {
      navigate('/phone-verify');
      return;
    }

    // Redirect to missing step if any required step is incomplete
    if (!requiredSteps.summary || !requiredSteps.description) {
      navigate('/chat');
      return;
    }
    if (!requiredSteps.location) {
      navigate('/location');
      return;
    }
    if (!requiredSteps.photo) {
      navigate('/camera');
      return;
    }
  }, []);

  const handleSubmit = async () => {
    // ISSUE 1: Defensive validation - photo MUST exist before submit
    if (!complaintData.summary || !complaintData.description) {
      setError('Summary aur description zaroori hain.');
      return;
    }

    if (!complaintData.location) {
      setError('Location zaroori hai. Wapas jaake location add karein.');
      return;
    }

    if (!complaintData.photo || (!complaintData.photo.blob && !complaintData.photo.url)) {
      setError('Live photo zaroori hai. Wapas jaake photo capture karein.');
      console.error('Photo missing in complaintData:', complaintData.photo);
      return;
    }

    // Ensure user_id is set
    const userID = localStorage.getItem('user_id');
    if (!userID) {
      setError('Phone verify zaroori hai.');
      navigate('/phone-verify');
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      // Log photo data before submission for debugging (only in dev mode)
      if (import.meta.env.DEV) {
        console.log('Submitting complaint with photo:', {
          hasBlob: !!complaintData.photo.blob,
          hasUrl: !!complaintData.photo.url,
          photo: complaintData.photo
        });
      }

      const response = await api.createComplaint(complaintData);
      
      addMessage({ type: 'system', text: 'Complaint formally register ho gayi.', timestamp: new Date() });
      clearComplaintData({ keepConversation: true });
      
      navigate(`/complaints/${response.complaint_id}`, {
        state: { 
          success: true,
          message: `Complaint register ho gayi. Number: ${response.complaint_number}`,
          complaintNumber: response.complaint_number
        }
      });
    } catch (err) {
      let errorMessage = 'Submit fail. Phir se try karein.';
      
      if (err instanceof ApiError) {
        if (err.status === 400) {
          errorMessage = err.message || 'Data sahi nahi. Check karein.';
        } else if (err.status === 401) {
          errorMessage = err.message || 'Phone verify karein.';
          // Clear invalid token
          localStorage.removeItem('auth_token');
          localStorage.removeItem('phone_verified');
          // Navigate to phone verification screen
          navigate('/phone-verify');
        } else if (err.status === 0) {
          // Check error code to distinguish CORS from actual network errors
          if (err.code === 'CORS_ERROR') {
            errorMessage = 'Backend CORS error. Connection check karein.';
          } else if (err.code === 'TIMEOUT') {
            errorMessage = 'Request timeout. Phir se try karein.';
          } else {
            const isOffline = !navigator.onLine;
            if (isOffline) {
              errorMessage = 'Aap offline hain. Complaint local save ho gayi, online aate hi submit ho jayegi.';
              saveToQueue(complaintData);
            } else {
              errorMessage = 'Request fail. Connection check karein.';
            }
          }
        } else {
          errorMessage = err.message || `Error: ${err.status}. Phir se try karein.`;
        }
      } else {
        errorMessage = err.message || errorMessage;
      }

      setError(errorMessage);
      setSubmitting(false);
    }
  };

  const handleRetry = () => {
    setRetryCount(prev => prev + 1);
    handleSubmit();
  };

  const handleEdit = (section) => {
    // ISSUE 1: Edit buttons - only Summary and Description are editable
    // Navigate back to ChatScreen with appropriate step
    if (section === 'summary') {
      // Reset to summary step
      updateComplaintData({ 
        step: 'summary',
        completedSteps: complaintData.completedSteps.filter(s => s !== 'summary')
      });
      navigate('/chat');
    } else if (section === 'description') {
      // Reset to description step
      updateComplaintData({ 
        step: 'description',
        completedSteps: complaintData.completedSteps.filter(s => s !== 'description')
      });
      navigate('/chat');
    }
    // Location and Photo are NOT editable (privacy + live proof requirement)
  };

  return (
    <div className="review-screen">
      <div className="review-header">
        <button type="button" className="back-button" onClick={() => navigate('/phone-verify')}>
          Back
        </button>
        <h2>Review Complaint</h2>
      </div>

      <div className="review-content">
        {submitting && (
          <div className="loading">
            <p>Main ise formally register kar raha hoon.</p>
            <p className="loading-subtext">Prastha karein.</p>
          </div>
        )}

        {error && <div className="error">{error}</div>}

        {/* ISSUE 1: Only Summary and Description are editable and shown */}
        <div className="review-section">
          <div className="review-section-header">
            <h3>Summary</h3>
            <button type="button" className="edit-button" onClick={() => handleEdit('summary')}>
              Edit
            </button>
          </div>
          <p className="review-text">{complaintData.summary || 'Not provided'}</p>
        </div>

        <div className="review-section">
          <div className="review-section-header">
            <h3>Description</h3>
            <button type="button" className="edit-button" onClick={() => handleEdit('description')}>
              Edit
            </button>
          </div>
          <p className="review-text">{complaintData.description || 'Not provided'}</p>
        </div>

        {/* ISSUE 3: Photo preview - VIEW ONLY (no edit) */}
        <div className="review-section">
          <div className="review-section-header">
            <h3>Photo</h3>
            <span className="photo-badge" style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
              Zaroori
            </span>
          </div>
          {complaintData.photo ? (
            <div className="photo-preview-container">
              <img 
                src={complaintData.photo.url} 
                alt="Complaint photo" 
                className="review-photo-preview"
                style={{ 
                  maxWidth: '100%', 
                  maxHeight: '200px', 
                  borderRadius: '8px',
                  border: '1px solid var(--border-color)',
                  marginTop: '8px'
                }}
              />
              <p className="photo-hint" style={{ fontSize: '12px', color: 'var(--text-secondary)', marginTop: '8px' }}>
                Ye photo complaint ke sath bheji jayegi.
              </p>
            </div>
          ) : (
            <div className="error" style={{ marginTop: '8px' }}>
              Photo nahi hai. Wapas jaake capture karein.
            </div>
          )}
        </div>

        <div className="review-actions">
          {error && (
            <div className="error-actions">
              <button
                className="btn btn-secondary"
                onClick={handleRetry}
                disabled={submitting}
              >
                Retry
              </button>
            </div>
          )}
          <button
            className="btn btn-primary"
            onClick={handleSubmit}
            disabled={submitting || !complaintData.summary || !complaintData.description || !complaintData.photo}
          >
            {submitting ? 'Register ho raha hai…' : 'Submit Complaint'}
          </button>
          {submitting && (
            <p className="submitting-hint">
              Main ise formally register kar raha hoon.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

export default ReviewScreen;
