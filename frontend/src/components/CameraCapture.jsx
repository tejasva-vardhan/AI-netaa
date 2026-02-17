import React, { useRef, useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { FaCamera, FaSync, FaImages } from 'react-icons/fa';

const CameraCapture = ({ onClose, onCapture }) => {
  const videoRef = useRef(null);
  const canvasRef = useRef(null);
  const fileInputRef = useRef(null);
  const [stream, setStream] = useState(null);
  const [photo, setPhoto] = useState(null);
  const [error, setError] = useState(null);
  const [cameraAvailable, setCameraAvailable] = useState(true);
  const photoBlobRef = useRef(null);

  useEffect(() => {
    startCamera();
    return () => {
      if (stream) {
        stream.getTracks().forEach(track => track.stop());
      }
    };
  }, []);

  const startCamera = async () => {
    try {
      const mediaStream = await navigator.mediaDevices.getUserMedia({ 
        video: { facingMode: 'environment' },
        audio: false 
      });
      setStream(mediaStream);
      if (videoRef.current) {
        videoRef.current.srcObject = mediaStream;
      }
    } catch (err) {
      setCameraAvailable(false);
      setError(null);
    }
  };

  const handleFileSelect = (e) => {
    const file = e.target.files?.[0];
    if (!file || !file.type.startsWith('image/')) return;
    photoBlobRef.current = file;
    setPhoto(URL.createObjectURL(file));
    if (stream) {
      stream.getTracks().forEach(track => track.stop());
      setStream(null);
    }
    e.target.value = '';
  };

  const capturePhoto = () => {
    if (videoRef.current && canvasRef.current && stream) {
      const context = canvasRef.current.getContext('2d');
      canvasRef.current.width = videoRef.current.videoWidth;
      canvasRef.current.height = videoRef.current.videoHeight;
      context.drawImage(videoRef.current, 0, 0);
      
      canvasRef.current.toBlob((blob) => {
        if (blob) {
          photoBlobRef.current = blob;
          setPhoto(URL.createObjectURL(blob));
          stream.getTracks().forEach(track => track.stop());
        }
      }, 'image/jpeg', 0.95);
    }
  };

  const retake = () => {
    setPhoto(null);
    photoBlobRef.current = null;
    startCamera();
  };

  const handleSubmit = () => {
    if (photoBlobRef.current) {
      onCapture(photoBlobRef.current);
      onClose();
    } else if (photo && canvasRef.current) {
      canvasRef.current.toBlob((blob) => {
        if (blob) onCapture(blob);
        onClose();
      }, 'image/jpeg', 0.95);
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <motion.div
        initial={{ scale: 0.9 }}
        animate={{ scale: 1 }}
        exit={{ scale: 0.9 }}
        className="bg-white rounded-3xl p-4 max-w-lg w-full mx-4"
        onClick={e => e.stopPropagation()}
      >
        <h3 className="text-2xl font-bold text-center mb-4">Add Photo</h3>
        
        {error ? (
          <div className="text-red-500 text-center p-4">{error}</div>
        ) : (
          <div className="space-y-4">
            <div className="relative rounded-2xl overflow-hidden bg-black aspect-video">
              {!photo ? (
                <>
                  {cameraAvailable && stream ? (
                    <video
                      ref={videoRef}
                      autoPlay
                      playsInline
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center text-white/80 text-center p-4">
                      {cameraAvailable ? 'Starting camera…' : 'Camera not available. Choose from gallery below.'}
                    </div>
                  )}
                </>
              ) : (
                <img src={photo} alt="Selected" className="w-full h-full object-contain bg-black" />
              )}
            </div>

            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handleFileSelect}
            />

            <canvas ref={canvasRef} className="hidden" />

            <div className="flex flex-wrap justify-center gap-3">
              {!photo ? (
                <>
                  {cameraAvailable && (
                    <button
                      type="button"
                      onClick={capturePhoto}
                      className="flex items-center gap-2 px-5 py-3 rounded-full bg-netaji-saffron text-white font-semibold hover:bg-netaji-gold transition"
                    >
                      <FaCamera /> Take live photo
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    className="flex items-center gap-2 px-5 py-3 rounded-full border-2 border-netaji-green text-netaji-green font-semibold hover:bg-netaji-green hover:text-white transition"
                  >
                    <FaImages /> Choose from gallery
                  </button>
                </>
              ) : (
                <>
                  <button
                    type="button"
                    onClick={retake}
                    className="flex items-center gap-2 px-5 py-3 border-2 border-gray-300 rounded-full font-semibold hover:bg-gray-50 transition"
                  >
                    <FaSync /> Retake / Change
                  </button>
                  <button
                    type="button"
                    onClick={handleSubmit}
                    className="px-5 py-3 bg-netaji-green text-white rounded-full font-semibold hover:bg-green-700 transition"
                  >
                    Use Photo
                  </button>
                </>
              )}
            </div>
          </div>
        )}

        <button
          type="button"
          onClick={onClose}
          className="mt-4 w-full px-6 py-3 bg-gray-200 rounded-full font-semibold hover:bg-gray-300 transition"
        >
          Cancel
        </button>
      </motion.div>
    </motion.div>
  );
};

export default CameraCapture;
