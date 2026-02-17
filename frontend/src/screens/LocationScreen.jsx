import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useComplaintState } from '../state/ComplaintContext';
import './LocationScreen.css';

function LocationScreen() {
  const navigate = useNavigate();
  const { complaintData, updateComplaintData, addMessage } = useComplaintState();
  const [location, setLocation] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);
  const [permissionRequested, setPermissionRequested] = useState(false);
  const [showExplanation, setShowExplanation] = useState(true); // ISSUE 2: Show explanation first

  useEffect(() => {
    // ISSUE 5: Defensive validation - ensure we should be on this screen
    if (!complaintData.summary || !complaintData.description) {
      navigate('/chat');
      return;
    }
    
    if (complaintData.step === 'camera' || complaintData.step === 'phone-verify' || complaintData.step === 'review') {
      navigate(`/${complaintData.step}`);
      return;
    }
    
    // ISSUE 2: Don't auto-request location - wait for user to click button after reading explanation
    // If location already exists, show success state
    if (complaintData.location) {
      setLocation(complaintData.location);
    }
  }, []);

  const requestLocation = async () => {
    // ISSUE 2: Hide explanation and show loading when user clicks
    setShowExplanation(false);
    setLoading(true);
    setError(null);
    setPermissionRequested(true);

    if (!navigator.geolocation) {
      setError('Aapke browser mein geolocation support nahi hai.');
      setLoading(false);
      return;
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
        updateComplaintData({ 
          location: loc,
          step: 'camera',
          completedSteps: [...completedSteps, 'location']
        });
        addMessage({ type: 'system', text: 'Location confirm ho gayi.', timestamp: new Date() });
        setLoading(false);
        
        // Automatically transition to camera after 2 seconds (give user time to see success)
        setTimeout(() => {
          navigate('/camera');
        }, 2000);
      },
      (err) => {
        let errorMsg = 'Location access deny ho gaya.';
        if (err.code === 1) {
          errorMsg = 'Location permission deny. Browser settings mein allow karein.';
        } else if (err.code === 2) {
          errorMsg = 'Location unavailable. GPS check karein.';
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

  const handleContinue = () => {
    if (location) {
      const completedSteps = complaintData.completedSteps || [];
      updateComplaintData({ 
        step: 'camera',
        completedSteps: [...completedSteps, 'location']
      });
      addMessage({ type: 'system', text: 'Location confirm ho gayi.', timestamp: new Date() });
      navigate('/camera');
    } else {
      setError('Location access allow karein ya manually enter karein.');
    }
  };

  const handleSkip = () => {
    const completedSteps = complaintData.completedSteps || [];
    updateComplaintData({ 
      step: 'camera',
      completedSteps: [...completedSteps, 'location']
    });
    navigate('/camera');
  };

  return (
    <div className="location-screen">
      <div className="location-header">
        <button type="button" className="back-button" onClick={() => navigate('/chat')}>
          Back
        </button>
        <h2>Location</h2>
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
              Aapke phone ki location chahiye
            </p>
            <p style={{ fontSize: '14px', color: 'var(--text-secondary)', lineHeight: '1.6' }}>
              Taaki complaint sahi jagah tak pahunche.
            </p>
          </div>
        )}

        {loading && !error && (
          <div className="loading">
            <p>Location confirm ho rahi hai…</p>
            <p className="loading-subtext">Location permission allow karein.</p>
          </div>
        )}

        {error && (
          <div className="error">
            <p>{error}</p>
            <p className="error-subtext" style={{ fontSize: '14px', marginTop: '8px' }}>
              Manually location enter karein ya skip karein.
            </p>
          </div>
        )}

        {location && !loading && (
          <div className="success">
            <p>Location capture ho gaya.</p>
            <p className="location-coords" style={{ fontSize: '12px', color: 'var(--text-secondary)', marginTop: '8px' }}>
              Lat: {location.latitude.toFixed(6)}, Lng: {location.longitude.toFixed(6)}
            </p>
            <p className="success-subtext" style={{ fontSize: '14px', marginTop: '12px' }}>
              Ab photo capture par ja rahe hain.
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
                {showExplanation ? 'Allow Location' : 'Location confirm ho rahi hai…'}
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
                  Try Again
                </button>
              )}
            </>
          )}

          {location && (
            <button className="btn btn-primary" onClick={handleContinue}>
              Continue
            </button>
          )}
        </div>

        <div className="location-footer">
          <button className="link-button" onClick={handleSkip}>
            Skip Location
          </button>
        </div>
      </div>
    </div>
  );
}

export default LocationScreen;
