import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { useComplaintState } from '../state/ComplaintContext';
import api, { ApiError } from '../services/api';
import { saveToQueue } from '../utils/offlineQueue';
import { ensureCategory } from '../utils/categoryInference';
import './ReviewScreen.css';

function ReviewScreen() {
  const navigate = useNavigate();
  const { complaintData, clearComplaintData, updateComplaintData, addMessage } = useComplaintState();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState(null);
  const [retryCount, setRetryCount] = useState(0);
  const [isRetrying, setIsRetrying] = useState(false);
  const retryTimeoutRef = useRef(null);

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
      setError('Aap bataiye kya dikkat hai aur kahan hai.');
      return;
    }

    if (!complaintData.location) {
      setError('Location chahiye. Wapas jaake location share karein.');
      return;
    }

    if (!complaintData.photo || (!complaintData.photo.blob && !complaintData.photo.url)) {
      setError('Agar possible ho toh ek photo bhej dijiye, isse madad milegi. Ya phir skip karke aage badh sakte hain.');
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
      // Ensure category is ALWAYS set before submission
      const finalCategory = ensureCategory(
        complaintData.category,
        complaintData.summary || complaintData.description || ''
      );
      
      // Update complaintData with ensured category
      const complaintDataWithCategory = {
        ...complaintData,
        category: finalCategory
      };
      
      // Log photo data before submission for debugging (only in dev mode)
      if (import.meta.env.DEV) {
        console.log('Submitting complaint with photo:', {
          hasBlob: !!complaintData.photo.blob,
          hasUrl: !!complaintData.photo.url,
          category: finalCategory,
          photo: complaintData.photo
        });
      }

      const response = await api.createComplaint(complaintDataWithCategory);
      
      // Show success message in chat instead of navigating to success screen
      const complaintNumber = response.complaint_number || response.complaint_id;
      const successMessage = complaintNumber 
        ? `Ho gaya.\n\nMaine aapki problem sahi adhikari tak pahucha di hai.\n\nAapko updates milte rahenge.\n\nComplaint ID: ${complaintNumber}`
        : `Ho gaya.\n\nMaine aapki problem sahi adhikari tak pahucha di hai.\n\nAapko updates milte rahenge.`;
      
      addMessage({
        type: 'bot',
        text: successMessage,
        timestamp: new Date()
      });
      
      clearComplaintData({ keepConversation: true });
      setSubmitting(false);
      setIsRetrying(false);
      setError(null);
      
      // Navigate back to chat instead of success screen
      navigate('/chat-legacy');
    } catch (err) {
      // Check if this is a retry attempt
      const isRetryAttempt = isRetrying;
      
      // For non-retryable errors (401, validation errors), show error immediately
      if (err instanceof ApiError) {
        if (err.status === 400) {
          // Validation error - don't retry
          setError(err.message || 'Kuch data sahi nahi lag raha. Ek baar check karein.');
          setSubmitting(false);
          setIsRetrying(false);
          return;
        } else if (err.status === 401) {
          // Auth error - don't retry
          setError(err.message || 'Phone verify karein.');
          // Clear invalid token
          localStorage.removeItem('auth_token');
          localStorage.removeItem('phone_verified');
          setSubmitting(false);
          setIsRetrying(false);
          // Navigate to phone verification screen
          navigate('/phone-verify');
          return;
        } else if (err.code === 'CORS_ERROR') {
          // CORS error - don't retry
          setError('Backend CORS error. Connection check karein.');
          setSubmitting(false);
          setIsRetrying(false);
          return;
        }
      }
      
      // For retryable errors (network, server errors)
      if (!isRetryAttempt) {
        // First failure: Show initial message and auto-retry once
        setError('Thoda issue aa gaya.');
        setSubmitting(false);
        setIsRetrying(true);
        
        // Clear any existing retry timeout
        if (retryTimeoutRef.current) {
          clearTimeout(retryTimeoutRef.current);
        }
        
        // Automatically retry once in background after a short delay
        retryTimeoutRef.current = setTimeout(async () => {
          try {
            // Ensure category is set before retry
            const finalCategory = ensureCategory(
              complaintData.category,
              complaintData.summary || complaintData.description || ''
            );
            
            const complaintDataWithCategory = {
              ...complaintData,
              category: finalCategory
            };
            
            const response = await api.createComplaint(complaintDataWithCategory);
            
            // Retry succeeded - continue normal success flow
            const complaintNumber = response.complaint_number || response.complaint_id;
            const successMessage = complaintNumber 
              ? `Ho gaya.\n\nMaine aapki problem sahi adhikari tak pahucha di hai.\n\nAapko updates milte rahenge.\n\nComplaint ID: ${complaintNumber}`
              : `Ho gaya.\n\nMaine aapki problem sahi adhikari tak pahucha di hai.\n\nAapko updates milte rahenge.`;
            
            addMessage({
              type: 'bot',
              text: successMessage,
              timestamp: new Date()
            });
            
            clearComplaintData({ keepConversation: true });
            setIsRetrying(false);
            setError(null);
            retryTimeoutRef.current = null;
            
            // Navigate back to chat
            navigate('/chat-legacy');
          } catch (retryErr) {
            // Retry also failed - show final error message
            setError('Abhi thoda network issue lag raha hai.\n\nThodi der baad phir try kar sakte hain.');
            setIsRetrying(false);
            retryTimeoutRef.current = null;
            
            // Save to offline queue as fallback
            if (!navigator.onLine || (retryErr instanceof ApiError && retryErr.status === 0)) {
              saveToQueue(complaintData);
            }
          }
        }, 1000); // Wait 1 second before retry
      } else {
        // This is the retry attempt that failed - should not reach here as it's handled above
        setError('Abhi thoda network issue lag raha hai.\n\nThodi der baad phir try kar sakte hain.');
        setIsRetrying(false);
        retryTimeoutRef.current = null;
        
        // Save to offline queue as fallback
        if (!navigator.onLine || (err instanceof ApiError && err.status === 0)) {
          saveToQueue(complaintData);
        }
      }
    }
  };

  const handleRetry = () => {
    // Clear any pending automatic retry
    if (retryTimeoutRef.current) {
      clearTimeout(retryTimeoutRef.current);
      retryTimeoutRef.current = null;
    }
    setIsRetrying(false);
    setError(null);
    setRetryCount(prev => prev + 1);
    handleSubmit();
  };

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current);
      }
    };
  }, []);

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
        <h2>Ek Baar Dekh Lein</h2>
      </div>

      <div className="review-content">
        {submitting && (
          <div className="loading">
            <p>Main ise register kar raha hoon.</p>
            <p className="loading-subtext">Thoda wait karein.</p>
          </div>
        )}

        {error && (
          <div className="error" style={{ whiteSpace: 'pre-line' }}>
            {error}
          </div>
        )}

        {/* ISSUE 1: Only Summary and Description are editable and shown */}
        <div className="review-section">
          <div className="review-section-header">
            <h3>Aap ne kya kaha</h3>
            <button type="button" className="edit-button" onClick={() => handleEdit('summary')}>
              Badal Dein
            </button>
          </div>
          <p className="review-text">{complaintData.summary || 'Nahi batai'}</p>
        </div>

        <div className="review-section">
          <div className="review-section-header">
            <h3>Detail</h3>
            <button type="button" className="edit-button" onClick={() => handleEdit('description')}>
              Badal Dein
            </button>
          </div>
          <p className="review-text">{complaintData.description || 'Nahi batai'}</p>
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
              Agar possible ho toh ek photo bhej dijiye, isse madad milegi.
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
                Phir Se Try Karein
              </button>
            </div>
          )}
          <button
            className="btn btn-primary"
            onClick={handleSubmit}
            disabled={submitting || !complaintData.summary || !complaintData.description || !complaintData.photo}
          >
            {submitting ? 'Register ho raha hai…' : 'Theek Hai, Register Kar Dein'}
          </button>
          {submitting && (
            <p className="submitting-hint">
              Main ise register kar raha hoon.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

export default ReviewScreen;
