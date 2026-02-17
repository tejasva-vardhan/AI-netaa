import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useComplaintState } from '../state/ComplaintContext';
import './CameraScreen.css';

function CameraScreen() {
  const navigate = useNavigate();
  const { complaintData, updateComplaintData, addMessage } = useComplaintState();
  const [stream, setStream] = useState(null);
  const [photo, setPhoto] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [permissionRequested, setPermissionRequested] = useState(false);
  const videoRef = useRef(null);
  const canvasRef = useRef(null);

  useEffect(() => {
    // ISSUE 5: Defensive validation - ensure we should be on this screen
    if (!complaintData.summary || !complaintData.description) {
      navigate('/chat');
      return;
    }
    if (!complaintData.location) {
      navigate('/location');
      return;
    }
    
    if (complaintData.step === 'phone-verify' || complaintData.step === 'review') {
      navigate(`/${complaintData.step}`);
      return;
    }
    
    // Only start camera if photo doesn't exist yet
    if (!complaintData.photo) {
      startCamera();
    }
    
    return () => {
      stopCamera();
    };
  }, []);

  const startCamera = async () => {
    setLoading(true);
    setPermissionRequested(true);
    setError(null);
    
    try {
      const mediaStream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'environment' },
        audio: false
      });
      setStream(mediaStream);
      if (videoRef.current) {
        videoRef.current.srcObject = mediaStream;
      }
      setLoading(false);
    } catch (err) {
      let errorMsg = 'Camera access deny ho gaya.';
      if (err.name === 'NotAllowedError') {
        errorMsg = 'Camera permission deny. Browser settings mein allow karein.';
      } else if (err.name === 'NotFoundError') {
        errorMsg = 'Camera nahi mila. Device check karein.';
      } else if (err.name === 'NotReadableError') {
        errorMsg = 'Camera kisi aur app mein use ho raha hai.';
      }
      setError(errorMsg);
      setLoading(false);
    }
  };

  const stopCamera = () => {
    if (stream) {
      stream.getTracks().forEach(track => track.stop());
      setStream(null);
    }
  };

  const capturePhoto = () => {
    if (!videoRef.current || !canvasRef.current) return;

    const video = videoRef.current;
    const canvas = canvasRef.current;
    const context = canvas.getContext('2d');

    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    context.drawImage(video, 0, 0);

    canvas.toBlob((blob) => {
      const url = URL.createObjectURL(blob);
      setPhoto({ blob, url });
      stopCamera();
    }, 'image/jpeg', 0.8);
  };

  const retakePhoto = () => {
    setPhoto(null);
    startCamera();
  };

  const handleContinue = () => {
    if (photo) {
      const completedSteps = complaintData.completedSteps || [];
      updateComplaintData({ 
        photo,
        step: 'phone-verify',
        completedSteps: [...completedSteps, 'camera']
      });
      addMessage({ type: 'system', text: 'Photo capture ho gaya.', timestamp: new Date() });
      navigate('/phone-verify');
    }
  };

  const handleSkip = () => {
    const completedSteps = complaintData.completedSteps || [];
    updateComplaintData({ 
      step: 'phone-verify',
      completedSteps: [...completedSteps, 'camera']
    });
    navigate('/phone-verify');
  };

  return (
    <div className="camera-screen">
      <div className="camera-header">
        <button type="button" className="back-button" onClick={() => navigate('/location')}>
          Back
        </button>
        <h2>Live Photo</h2>
      </div>

      <div className="camera-container">
        {loading && !error && (
          <div className="loading">
            <p>Camera start ho raha hai…</p>
            <p className="loading-subtext">Camera permission allow karein.</p>
          </div>
        )}

        {error && (
          <div className="error">
            <p>{error}</p>
            <p className="error-subtext" style={{ fontSize: '14px', marginTop: '8px' }}>
              Photo skip karke aage badh sakte hain.
            </p>
          </div>
        )}

        {!loading && !error && !photo && stream && (
          <>
            <video
              ref={videoRef}
              autoPlay
              playsInline
              className="camera-preview"
            />
            <canvas ref={canvasRef} className="hidden" />
            <div className="camera-controls">
              <button type="button" className="btn btn-primary" onClick={capturePhoto}>
                Capture Photo
              </button>
            </div>
          </>
        )}

        {!loading && !error && !photo && !stream && (
          <div className="camera-prompt">
            <p>Live photo ke liye camera permission chahiye.</p>
            <button type="button" className="btn btn-primary" onClick={startCamera}>
              Allow Camera
            </button>
          </div>
        )}

        {photo && (
          <>
            <img src={photo.url} alt="Captured" className="photo-preview" />
            <div className="photo-controls">
              <button type="button" className="btn btn-secondary" onClick={retakePhoto}>
                Retake
              </button>
              <button type="button" className="btn btn-primary" onClick={handleContinue}>
                Continue
              </button>
            </div>
          </>
        )}
      </div>

      <div className="camera-footer">
        <button type="button" className="link-button" onClick={handleSkip}>
          Skip Photo
        </button>
      </div>
    </div>
  );
}

export default CameraScreen;
