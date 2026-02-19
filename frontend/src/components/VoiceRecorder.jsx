import React, { useState, useRef } from 'react';
import { motion } from 'framer-motion';
import { FaMicrophone, FaStop, FaPlay, FaPause } from 'react-icons/fa';
import RecordRTC from 'recordrtc';
import './VoiceRecorder.css';

const VoiceRecorder = ({ onClose, onRecord }) => {
  const [recording, setRecording] = useState(false);
  const [audioURL, setAudioURL] = useState(null);
  const [playing, setPlaying] = useState(false);
  const recorder = useRef(null);
  const stream = useRef(null);
  const audioPlaybackRef = useRef(null);

  const startRecording = async () => {
    try {
      stream.current = await navigator.mediaDevices.getUserMedia({ audio: true });
      recorder.current = new RecordRTC(stream.current, {
        type: 'audio',
        mimeType: 'audio/webm',
        recorderType: RecordRTC.StereoAudioRecorder,
        numberOfAudioChannels: 1,
        desiredSampRate: 16000
      });
      recorder.current.startRecording();
      setRecording(true);
    } catch (err) {
      console.error('Error accessing microphone:', err);
    }
  };

  const stopRecording = () => {
    if (!recorder.current) return;
    recorder.current.stopRecording(() => {
      const blob = recorder.current.getBlob();
      setAudioURL(URL.createObjectURL(blob));
      setRecording(false);
      if (stream.current) stream.current.getTracks().forEach(t => t.stop());
    });
  };

  const togglePlayback = () => {
    const audio = audioPlaybackRef.current;
    if (!audio) return;
    if (playing) {
      audio.pause();
    } else {
      audio.currentTime = 0;
      audio.play();
    }
    setPlaying(!playing);
  };

  const handleSubmit = () => {
    if (recorder.current) {
      onRecord(recorder.current.getBlob());
      onClose();
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="voice-recorder-overlay"
      onClick={onClose}
    >
      <motion.div
        initial={{ scale: 0.9 }}
        animate={{ scale: 1 }}
        exit={{ scale: 0.9 }}
        className="voice-recorder-modal"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="voice-recorder-title">Voice Me Bataiye</h3>
        <div className="voice-recorder-body">
          <button
            type="button"
            onClick={recording ? stopRecording : startRecording}
            className={`voice-recorder-btn ${recording ? 'voice-recorder-btn--recording' : ''}`}
          >
            {recording ? <FaStop className="voice-recorder-btn-icon" /> : <FaMicrophone className="voice-recorder-btn-icon" />}
          </button>
          {recording && (
            <div className="voice-waveform" aria-hidden>
              {[1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map((i) => (
                <motion.span
                  key={i}
                  className="voice-waveform-bar"
                  animate={{ height: ['40%', '90%', '40%'] }}
                  transition={{ duration: 0.5, repeat: Infinity, delay: i * 0.05 }}
                />
              ))}
            </div>
          )}
          {audioURL && !recording && (
            <div className="voice-playback">
              <audio ref={audioPlaybackRef} src={audioURL} onEnded={() => setPlaying(false)} />
              <button type="button" onClick={togglePlayback} className="voice-playback-btn">
                {playing ? <FaPause /> : <FaPlay />}
              </button>
              <span className="voice-playback-label">{playing ? 'Rok Dein' : 'Sun Lein'}</span>
            </div>
          )}
          <div className="voice-recorder-actions">
            <button type="button" onClick={onClose} className="voice-recorder-cancel">Cancel</button>
            {audioURL && (
              <button type="button" onClick={handleSubmit} className="voice-recorder-submit">Theek Hai</button>
            )}
          </div>
        </div>
      </motion.div>
    </motion.div>
  );
};

export default VoiceRecorder;
