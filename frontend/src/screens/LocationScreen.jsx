import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useComplaintState } from '../state/ComplaintContext';
import { getRandomPhrase } from '../utils/botPhrases';
import './LocationScreen.css';

function LocationScreen() {
  const navigate = useNavigate();
  const { complaintData, updateComplaintData, addMessage, goToStepRef } = useComplaintState();
  const [location, setLocation] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);
  const [permissionRequested, setPermissionRequested] = useState(false);
  const [showExplanation, setShowExplanation] = useState(true); // ISSUE 2: Show explanation first
  const [guidanceShown, setGuidanceShown] = useState(false); // Prevent duplicate guidance messages

  useEffect(() => {
    // STRICT VALIDATION: Only allow if prerequisites met
    if (!complaintData.summary || !complaintData.description) {
      navigate('/chat-legacy');
      return;
    }
    
    // If location already exists, show success state
    if (complaintData.location) {
      setLocation(complaintData.location);
    }
  }, []);

  const requestLocation = async () => {
    // Hide explanation and show loading when user clicks
    setShowExplanation(false);
    setLoading(true);
    setError(null);
    setPermissionRequested(true);

      if (!navigator.geolocation) {
      setError('Aapke browser mein geolocation support nahi hai.');
      setLoading(false);
      // Offer manual entry fallback
      addMessage({
        type: 'bot',
        text: `${getRandomPhrase('acknowledge')}, aap area ka naam bata dijiye.`,
        timestamp: new Date()
      });
      return;
    }

    // Show guidance message before requesting permission
    if (!guidanceShown) {
      addMessage({
        type: 'bot',
        text: 'Aapki location chahiye taaki main sahi jagah tak complaint bhej sakoon.',
        timestamp: new Date()
      });
      addMessage({
        type: 'bot',
        text: 'Please location ON kar dijiye.',
        timestamp: new Date()
      });
      setGuidanceShown(true);
    }

    navigator.geolocation.getCurrentPosition(
      (position) => {
        const loc = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          accuracy: position.coords.accuracy
        };
        setLocation(loc);
        const completedSteps = complaintData.completedSteps || [];
        // Update state
        updateComplaintData({ 
          location: loc,
          completedSteps: [...completedSteps, 'location']
        });
        setLoading(false);
        
        // EVENT-DRIVEN: Direct step transition on async completion
        setTimeout(() => {
          navigate('/chat-legacy');
          if (goToStepRef?.current) {
            goToStepRef.current('location_confirm');
          }
        }, 1000);
      },
      (err) => {
        let errorMsg = 'Location access deny ho gaya.';
        if (err.code === 1) {
          errorMsg = 'Location permission deny. Browser settings mein allow karein.';
          addMessage({
            type: 'bot',
            text: `${getRandomPhrase('acknowledge')}, aap area ka naam bata dijiye.`,
            timestamp: new Date()
          });
        } else if (err.code === 2) {
          errorMsg = 'Location unavailable. GPS check karein.';
          addMessage({
            type: 'bot',
            text: `${getRandomPhrase('acknowledge')}, aap area ka naam bata dijiye.`,
            timestamp: new Date()
          });
        } else if (err.code === 3) {
          errorMsg = 'Location request timeout. Phir se try karein.';
        }
        setError(errorMsg);
        setLoading(false);
        setShowExplanation(false); // Keep explanation hidden on error
      },
      {
        enableHighAccuracy: true,
        timeout: 15000,
        maximumAge: 0
      }
    );
  };

  const handleManualEntry = () => {
    alert('Manual address abhi available nahi. Location access allow karein.');
  };

  // Auto-proceed when location is manually set via handleContinue (if user clicks continue)
  // Note: GPS capture already auto-proceeds, this handles manual "continue" case
  // But we're removing the button, so this is just for edge cases

  const handleSkip = () => {
    const completedSteps = complaintData.completedSteps || [];
    // Update state
    updateComplaintData({ 
      completedSteps: [...completedSteps, 'location', 'location_confirm']
    });
    addMessage({ 
      type: 'bot', 
      text: 'Aapki di hui jaankari ke base par aage badh raha hoon.', 
      timestamp: new Date() 
    });
    // EVENT-DRIVEN: Direct step transition on skip
    navigate('/camera');
    if (goToStepRef?.current) {
      goToStepRef.current('photo');
    }
  };

  return (
    <div className="location-screen">
      <div className="location-header">
        <button type="button" className="back-button" onClick={() => navigate('/chat')}>
          Back
        </button>
        <h2>Aap Kahan Hain</h2>
      </div>

      <div className="location-content">
        {/* ISSUE 2: Show explanation FIRST before requesting GPS */}
        {showExplanation && !loading && !error && !location && (
          <div className="location-explanation" style={{ 
            padding: '24px', 
            textAlign: 'center',
            backgroundColor: 'var(--background)',
            borderRadius: '8px',
            marginBottom: '24px'
          }}>
            <p style={{ fontSize: '16px', fontWeight: '600', marginBottom: '12px' }}>
              Aap yahin ke rehne wale hain na?
            </p>
            <p style={{ fontSize: '14px', color: 'var(--text-secondary)', lineHeight: '1.6' }}>
              Location share kar dijiye, taaki main samajh sakoon kahan ki problem hai.
            </p>
          </div>
        )}

        {loading && !error && (
          <div className="loading">
            <p>Location dekh raha hoon…</p>
            <p className="loading-subtext">Location permission allow kar dijiye.</p>
          </div>
        )}

        {error && (
          <div className="error">
            <p>{error}</p>
            <p className="error-subtext" style={{ fontSize: '14px', marginTop: '8px' }}>
              Ya phir skip karke aage badh sakte hain.
            </p>
          </div>
        )}

        {location && !loading && (
          <div className="success">
            <p>Location mil gaya, dhanyavaad!</p>
            <p className="location-coords" style={{ fontSize: '12px', color: 'var(--text-secondary)', marginTop: '8px' }}>
              Lat: {location.latitude.toFixed(6)}, Lng: {location.longitude.toFixed(6)}
            </p>
            <p className="success-subtext" style={{ fontSize: '14px', marginTop: '12px' }}>
              Agar possible ho toh ek photo bhej dijiye, isse madad milegi.
            </p>
          </div>
        )}

        <div className="location-actions">
          {!location && !loading && (
            <>
              <button
                className="btn btn-primary"
                onClick={requestLocation}
                disabled={loading || showExplanation === false}
              >
                {showExplanation ? 'Location Share Kar Dein' : 'Location dekh raha hoon…'}
              </button>
              {error && (
                <button
                  className="btn btn-secondary"
                  onClick={() => {
                    setError(null);
                    setShowExplanation(true);
                    requestLocation();
                  }}
                  disabled={loading}
                  style={{ marginTop: '8px' }}
                >
                  Phir Se Try Karein
                </button>
              )}
            </>
          )}

        </div>

        <div className="location-footer">
          <button className="link-button" onClick={handleSkip}>
            Abhi Skip Kar Dein
          </button>
        </div>
      </div>
    </div>
  );
}

export default LocationScreen;
