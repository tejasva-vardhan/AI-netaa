import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { useComplaintState } from '../state/ComplaintContext';
import './LandingScreen.css';

const AVATAR_ENTER_MS = 600;
const AVATAR_PAUSE_MS = 500;
const AVATAR_TO_LEFT_MS = 500;
const SETTLED_DELAY_MS = AVATAR_ENTER_MS + AVATAR_PAUSE_MS + AVATAR_TO_LEFT_MS;

function LandingScreen() {
  const navigate = useNavigate();
  const { complaintData, clearComplaintData } = useComplaintState();
  const [hasDraft, setHasDraft] = useState(false);
  const [avatarPhase, setAvatarPhase] = useState('top'); // 'top' | 'center' | 'left'
  const [contentVisible, setContentVisible] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const mq = window.matchMedia('(max-width: 639px)');
    setIsMobile(mq.matches);
    const handler = () => setIsMobile(mq.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  useEffect(() => {
    const draftExists = !!(complaintData.summary || complaintData.description || complaintData.location || complaintData.photo);
    setHasDraft(draftExists);
  }, [complaintData.summary, complaintData.description, complaintData.location, complaintData.photo]);

  useEffect(() => {
    const t1 = setTimeout(() => setAvatarPhase('center'), 50);
    const t2 = setTimeout(() => setAvatarPhase('left'), 50 + AVATAR_ENTER_MS + AVATAR_PAUSE_MS);
    const t3 = setTimeout(() => setContentVisible(true), SETTLED_DELAY_MS + 100);
    return () => {
      clearTimeout(t1);
      clearTimeout(t2);
      clearTimeout(t3);
    };
  }, []);

  const handleStart = () => {
    const phoneVerified = localStorage.getItem('phone_verified') === 'true';
    if (!phoneVerified) {
      navigate('/phone-verify');
      return;
    }
    clearComplaintData();
    navigate('/chat');
  };

  const handleContinueDraft = () => {
    const step = complaintData.step || 'summary';
    if (step === 'summary' || step === 'description') navigate('/chat');
    else if (step === 'location') navigate('/location');
    else if (step === 'location-confirmation' || step === 'chat' || step === 'phone-verify-prompt') navigate('/chat-legacy');
    else if (step === 'camera') navigate('/camera');
    else if (step === 'phone-verify') navigate('/phone-verify');
    else if (step === 'confirmation') navigate('/chat-legacy');
    else navigate('/chat');
  };

  return (
    <div className="landing-screen">
      <div className="landing-home-container">
        <motion.div
          className="landing-avatar-placeholder"
          initial={{
            top: '0%',
            left: '50%',
            x: '-50%',
            y: '-50%',
          }}
          animate={
            avatarPhase === 'top'
              ? { top: '0%', left: '50%', x: '-50%', y: '-50%' }
              : avatarPhase === 'center'
              ? { top: '50%', left: '50%', x: '-50%', y: '-50%' }
              : isMobile
              ? { top: '18%', left: '50%', x: '-50%', y: '-50%' }
              : { top: '50%', left: '18%', x: '-50%', y: '-50%' }
          }
          transition={{
            top: { duration: avatarPhase === 'center' ? AVATAR_ENTER_MS / 1000 : avatarPhase === 'left' ? AVATAR_TO_LEFT_MS / 1000 : 0 },
            left: { duration: avatarPhase === 'left' ? AVATAR_TO_LEFT_MS / 1000 : 0 },
            delay: 0,
            ease: 'easeInOut',
          }}
        />

        <AnimatePresence>
          {contentVisible && (
            <motion.div
              className="landing-right-content"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.4 }}
            >
              <p className="landing-greeting">Namaste.</p>
              <p className="landing-intro">Main AI Neta hoon.</p>
              <p className="landing-line">Aap apni samasya mujhe bata sakte hain.</p>
              <p className="landing-line">Main ise sahi authority tak pahunchaoonga.</p>

              <div className="landing-cta">
                <button type="button" className="landing-cta-button" onClick={handleStart}>
                  Talk with me
                </button>
                {hasDraft && (
                  <button
                    type="button"
                    className="landing-cta-secondary"
                    onClick={handleContinueDraft}
                  >
                    Continue Previous Complaint
                  </button>
                )}
              </div>

              <div className="landing-footer">
                <button
                  type="button"
                  className="landing-footer-link"
                  onClick={() => navigate('/complaints')}
                >
                  View My Complaints
                </button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}

export default LandingScreen;
