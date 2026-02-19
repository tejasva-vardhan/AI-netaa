import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useComplaintState } from '../state/ComplaintContext';
import { getRandomPhrase } from '../utils/botPhrases';
import './CameraScreen.css';

function CameraScreen() {
  const navigate = useNavigate();
  const { complaintData, updateComplaintData, addMessage, goToStepRef } = useComplaintState();
  const [stream, setStream] = useState(null);
  const [photo, setPhoto] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);
  const [permissionRequested, setPermissionRequested] = useState(false);
  const videoRef = useRef(null);
  const canvasRef = useRef(null);

  useEffect(() => {
    // STRICT VALIDATION: Only allow if prerequisites met
    if (!complaintData.summary || !complaintData.description) {
      navigate('/chat-legacy');
      return;
    }
    if (!complaintData.location) {
      navigate('/location');
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
      const photoData = { blob, url };
      setPhoto(photoData);
      stopCamera();
      
      // Update state
      const completedSteps = complaintData.completedSteps || [];
      updateComplaintData({ 
        photo: photoData,
        completedSteps: [...completedSteps, 'photo']
      });
      
      addMessage({ type: 'bot', text: `${getRandomPhrase('okay')}`, timestamp: new Date() });
      // EVENT-DRIVEN: Direct step transition on async completion
      setTimeout(() => {
        navigate('/chat-legacy');
        if (goToStepRef?.current) {
          // Check if phone verification needed
          const phoneVerified = localStorage.getItem('phone_verified') === 'true';
          if (!phoneVerified) {
            goToStepRef.current('phone-verify-prompt');
          } else {
            goToStepRef.current('confirmation');
          }
        }
      }, 1000);
    }, 'image/jpeg', 0.8);
  };

  const retakePhoto = () => {
    setPhoto(null);
    startCamera();
  };


  const handleSkip = () => {
    const completedSteps = complaintData.completedSteps || [];
    // Update state
    updateComplaintData({ 
      completedSteps: [...completedSteps, 'photo']
    });
    addMessage({ type: 'bot', text: 'Thik hai, bina photo ke bhi main aage bhej raha hoon.', timestamp: new Date() });
    // EVENT-DRIVEN: Direct step transition on skip
    navigate('/chat-legacy');
    if (goToStepRef?.current) {
      // Check if phone verification needed
      const phoneVerified = localStorage.getItem('phone_verified') === 'true';
      if (!phoneVerified) {
        goToStepRef.current('phone-verify-prompt');
      } else {
        goToStepRef.current('confirmation');
      }
    }
  };

  return (
    <div className="camera-screen">
      <div className="camera-header">
        <button type="button" className="back-button" onClick={() => navigate('/location')}>
          Back
        </button>
        <h2>Photo</h2>
      </div>

      <div className="camera-container">
        {loading && !error && (
          <div className="loading">
            <p>Camera khol raha hoon…</p>
            <p className="loading-subtext">Camera permission allow kar dijiye.</p>
          </div>
        )}

        {error && (
          <div className="error">
            <p>{error}</p>
            <p className="error-subtext" style={{ fontSize: '14px', marginTop: '8px' }}>
              Photo skip karke aage badh sakte hain, koi baat nahi.
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
                Photo Lein
              </button>
            </div>
          </>
        )}

        {!loading && !error && !photo && !stream && (
          <div className="camera-prompt">
            <p>Agar possible ho toh ek photo bhej dijiye, taaki main samajh sakoon.</p>
            <button type="button" className="btn btn-primary" onClick={startCamera}>
              Camera Khol Dein
            </button>
          </div>
        )}

        {photo && (
          <>
            <img src={photo.url} alt="Captured" className="photo-preview" />
            <div className="photo-controls">
              <button type="button" className="btn btn-secondary" onClick={retakePhoto}>
                Phir Se Lein
              </button>
            </div>
          </>
        )}
      </div>

      <div className="camera-footer">
        <button type="button" className="link-button" onClick={handleSkip}>
            Abhi Skip Kar Dein
        </button>
      </div>
    </div>
  );
}

export default CameraScreen;
