import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useComplaintState } from '../state/ComplaintContext';
import api from '../services/api';
import './PhoneVerificationScreen.css';

function PhoneVerificationScreen() {
  const navigate = useNavigate();
  const { complaintData, updateComplaintData } = useComplaintState();
  const [phone, setPhone] = useState('');
  const [otp, setOtp] = useState('');
  const [step, setStep] = useState('phone'); // 'phone' | 'otp' | 'verifying'
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);
  const [otpSent, setOtpSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [resendAttempts, setResendAttempts] = useState(0);
  const [blockedUntil, setBlockedUntil] = useState(null);

  useEffect(() => {
    // Check if user is already verified (e.g. refreshed or came from "Talk with me" when already logged in)
    const verifiedUserID = localStorage.getItem('user_id');
    const verifiedPhone = localStorage.getItem('user_phone');
    const phoneVerified = localStorage.getItem('phone_verified') === 'true';

    if (phoneVerified && verifiedUserID && verifiedPhone) {
      // Already verified: go to chat (new flow) or review if resuming a full draft
      const hasFullDraft = complaintData.summary && complaintData.description && complaintData.location && complaintData.photo;
      if (hasFullDraft) {
        updateComplaintData({ step: 'review' });
        navigate('/review');
      } else {
        navigate('/chat');
      }
      return;
    }

    // Pre-fill phone if available
    if (verifiedPhone && !phone) {
      setPhone(verifiedPhone);
    }
  }, []);

  useEffect(() => {
    // Countdown timer for OTP resend
    if (countdown > 0) {
      const timer = setTimeout(() => setCountdown(countdown - 1), 1000);
      return () => clearTimeout(timer);
    }
  }, [countdown]);

  useEffect(() => {
    // Check for blocked state on mount
    const cleanPhone = phone.replace(/\D/g, '');
    if (cleanPhone) {
      const blockedUntilStr = localStorage.getItem(`otp_blocked_until_${cleanPhone}`);
      if (blockedUntilStr) {
        const blockedUntilTime = parseInt(blockedUntilStr);
        if (Date.now() < blockedUntilTime) {
          setBlockedUntil(blockedUntilTime);
        } else {
          // Block expired, clear it
          localStorage.removeItem(`otp_blocked_until_${cleanPhone}`);
          localStorage.removeItem(`otp_resend_attempts_${cleanPhone}`);
          setResendAttempts(0);
          setBlockedUntil(null);
        }
      }
      
      // Load resend attempts
      const attemptsStr = localStorage.getItem(`otp_resend_attempts_${cleanPhone}`);
      if (attemptsStr) {
        const attemptsData = JSON.parse(attemptsStr);
        setResendAttempts(attemptsData.count || 0);
        if (attemptsData.resetAt && Date.now() < attemptsData.resetAt) {
          // Still within the hour window
        } else {
          // Reset window expired, clear attempts
          localStorage.removeItem(`otp_resend_attempts_${cleanPhone}`);
          setResendAttempts(0);
        }
      }
    }
  }, [phone]);

  const handlePhoneSubmit = async (e) => {
    e.preventDefault();
    if (!phone.trim()) {
      setError('Phone number dalein.');
      return;
    }

    const phoneRegex = /^[6-9]\d{9}$/;
    const cleanPhone = phone.replace(/\D/g, '');
    if (cleanPhone.length !== 10 || !phoneRegex.test(cleanPhone)) {
      setError('Valid 10-digit number dalein.');
      return;
    }

    // SAFETY CHECK 2: Check if blocked due to excess resend attempts
    const blockedUntilStr = localStorage.getItem(`otp_blocked_until_${cleanPhone}`);
    if (blockedUntilStr) {
      const blockedUntilTime = parseInt(blockedUntilStr);
      if (Date.now() < blockedUntilTime) {
        const minutesLeft = Math.ceil((blockedUntilTime - Date.now()) / 60000);
        setError(`Too many OTP requests. Please try again after ${minutesLeft} minute(s). / बहुत सारे OTP requests। कृपया ${minutesLeft} मिनट बाद कोशिश करें।`);
        return;
      } else {
        // Block expired, clear it
        localStorage.removeItem(`otp_blocked_until_${cleanPhone}`);
        localStorage.removeItem(`otp_resend_attempts_${cleanPhone}`);
        setResendAttempts(0);
        setBlockedUntil(null);
      }
    }

    setLoading(true);
    setError(null);

    try {
      // Send OTP - backend returns OTP in dev mode
      const response = await api.sendOTP(cleanPhone);
      
      console.log('[DEBUG] PhoneVerificationScreen received response:', response);
      
      // Store OTP in localStorage for display (dev mode only)
      if (import.meta.env.DEV && response.otp) {
        console.log('[DEBUG] Storing OTP in localStorage:', response.otp);
        localStorage.setItem(`dev_otp_${cleanPhone}`, response.otp);
      } else if (import.meta.env.DEV) {
        console.warn('[WARNING] OTP not found in response. Available keys:', Object.keys(response || {}));
      }
      
      setOtpSent(true);
      setStep('otp');
      setCountdown(60); // 60 second countdown
      setLoading(false);
    } catch (err) {
      setError(err.message || 'OTP भेजने में समस्या / Failed to send OTP. Please try again.');
      setLoading(false);
    }
  };

  const handleOTPSubmit = async (e) => {
    e.preventDefault();
    if (!otp.trim() || otp.length !== 6) {
      setError('6-digit OTP dalein.');
      return;
    }

    setLoading(true);
    setError(null);
    setStep('verifying');

    try {
      const cleanPhone = phone.replace(/\D/g, '');
      const result = await api.verifyOTP(cleanPhone, otp);
      
      // ISSUE 1: Store auth token and user session
      if (!result.user_id || !result.token) {
        throw new Error('Backend did not return user_id or token. Please try again.');
      }
      
      console.log('[DEBUG] Phone verified, token received, user_id:', result.user_id);
      
      // Store authentication token (REQUIRED for all authenticated requests)
      localStorage.setItem('auth_token', result.token);
      localStorage.setItem('user_phone', cleanPhone);
      localStorage.setItem('user_id', result.user_id.toString());
      localStorage.setItem('phone_verified', 'true');
      localStorage.setItem('phone_verified_at', new Date().toISOString());

      // Update complaint data; then chat (new flow) or review if resuming full draft
      updateComplaintData({ 
        userPhone: cleanPhone,
        userID: result.user_id
      });
      const hasFullDraft = complaintData.summary && complaintData.description && complaintData.location && complaintData.photo;
      if (hasFullDraft) {
        updateComplaintData({ step: 'review' });
        navigate('/review');
      } else {
        navigate('/chat');
      }
    } catch (err) {
      // Display backend error message or fallback
      let errorMsg = err.message || 'Galat OTP. Phir se try karein.';
      if (err.message && (err.message.includes('not found') || err.message.includes('OTP not found'))) {
        errorMsg = 'OTP nahi mila. Naya OTP maangein.';
      }
      
      console.error('[ERROR] OTP verification failed:', err);
      setError(errorMsg);
      setStep('otp');
      setLoading(false);
      setOtp('');
    }
  };

  const handleResendOTP = async () => {
    if (countdown > 0) return;
    
    const cleanPhone = phone.replace(/\D/g, '');
    
    // SAFETY CHECK 2: Rate limiting - Max 5 resends per hour
    const MAX_RESEND_ATTEMPTS = 5;
    const RESEND_WINDOW_MS = 60 * 60 * 1000; // 1 hour
    const BLOCK_DURATION_MS = 60 * 60 * 1000; // Block for 1 hour after max attempts
    
    // Check if blocked
    const blockedUntilStr = localStorage.getItem(`otp_blocked_until_${cleanPhone}`);
    if (blockedUntilStr) {
      const blockedUntilTime = parseInt(blockedUntilStr);
      if (Date.now() < blockedUntilTime) {
        const minutesLeft = Math.ceil((blockedUntilTime - Date.now()) / 60000);
        setError(`Too many OTP requests. Please try again after ${minutesLeft} minute(s). / बहुत सारे OTP requests। कृपया ${minutesLeft} मिनट बाद कोशिश करें।`);
        setBlockedUntil(blockedUntilTime);
        return;
      } else {
        // Block expired, clear it
        localStorage.removeItem(`otp_blocked_until_${cleanPhone}`);
        localStorage.removeItem(`otp_resend_attempts_${cleanPhone}`);
        setResendAttempts(0);
        setBlockedUntil(null);
      }
    }
    
    // Check resend attempts
    const attemptsStr = localStorage.getItem(`otp_resend_attempts_${cleanPhone}`);
    let attemptsData = attemptsStr ? JSON.parse(attemptsStr) : { count: 0, resetAt: Date.now() + RESEND_WINDOW_MS };
    
    // Reset if window expired
    if (Date.now() >= attemptsData.resetAt) {
      attemptsData = { count: 0, resetAt: Date.now() + RESEND_WINDOW_MS };
    }
    
    // Check if max attempts reached
    if (attemptsData.count >= MAX_RESEND_ATTEMPTS) {
      const blockUntil = Date.now() + BLOCK_DURATION_MS;
      localStorage.setItem(`otp_blocked_until_${cleanPhone}`, blockUntil.toString());
      setBlockedUntil(blockUntil);
      setError(`Maximum OTP resend attempts reached. Please try again after 1 hour. / अधिकतम OTP resend attempts पहुंच गए। कृपया 1 घंटे बाद कोशिश करें।`);
      return;
    }
    
    // Increment attempts
    attemptsData.count += 1;
    localStorage.setItem(`otp_resend_attempts_${cleanPhone}`, JSON.stringify(attemptsData));
    setResendAttempts(attemptsData.count);
    
    setLoading(true);
    setError(null);
    
    try {
      await api.sendOTP(cleanPhone);
      setCountdown(60);
      setOtp('');
      setError(null);
      setLoading(false);
    } catch (err) {
      setError(err.message || 'OTP bhejne mein problem. Phir se try karein.');
      setLoading(false);
    }
  };

  return (
    <div className="phone-verification-screen">
      <div className="phone-verification-header">
        <button type="button" className="back-button" onClick={() => navigate('/camera')}>
          Back
        </button>
        <h2>Phone Verification</h2>
      </div>

      <div className="phone-verification-content">
        {error && <div className="error">{error}</div>}

        {step === 'phone' && (
          <form onSubmit={handlePhoneSubmit} className="phone-form">
            <div className="form-group">
              <label htmlFor="phone">
                Phone Number
              </label>
              <input
                id="phone"
                type="tel"
                className="input"
                placeholder="9876543210"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                disabled={loading}
                maxLength={10}
                autoFocus
              />
              <p className="form-hint">
                Aapko OTP bheja jayega.
              </p>
            </div>

            <button
              type="submit"
              className="btn btn-primary"
              disabled={loading || !phone.trim()}
            >
              {loading ? 'OTP bhej rahe hain…' : 'Send OTP'}
            </button>
          </form>
        )}

        {step === 'otp' && (
          <form onSubmit={handleOTPSubmit} className="otp-form">
            <div className="form-group">
              <label htmlFor="otp">
                Enter OTP
              </label>
              <p className="otp-sent-message">
                OTP sent to {phone.replace(/(\d{2})(\d{4})(\d{4})/, '$1****$2')}
              </p>
              {/* Development mode: Show OTP if available */}
              {import.meta.env.DEV && (() => {
                // Try to get OTP from localStorage (stored by api.js after sendOTP)
                const cleanPhone = phone.replace(/\D/g, '');
                const storedOTP = localStorage.getItem(`dev_otp_${cleanPhone}`);
                return storedOTP ? (
                  <p className="dev-otp-hint" style={{ 
                    fontSize: '14px', 
                    color: 'var(--primary-color)', 
                    fontWeight: '600',
                    marginTop: '8px',
                    padding: '12px',
                    background: 'var(--background)',
                    borderRadius: '6px',
                    border: '1px solid var(--border)'
                  }}>
                    Dev: OTP = <strong>{storedOTP}</strong>
                    <br />
                    <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>
                      (Console for details)
                    </span>
                  </p>
                ) : (
                  <p className="dev-otp-hint" style={{ 
                    fontSize: '12px', 
                    color: 'var(--error-color)', 
                    marginTop: '8px',
                    padding: '8px',
                    background: '#FEF2F2',
                    borderRadius: '6px',
                    border: '1px solid #FECACA'
                  }}>
                    Dev: Check console for OTP. Backend prints [PILOT MODE] OTP for {phone.replace(/(\d{2})(\d{4})(\d{4})/, '$1****$2')}.
                  </p>
                );
              })()}
              <input
                id="otp"
                type="text"
                className="input otp-input"
                placeholder="123456"
                value={otp}
                onChange={(e) => {
                  const value = e.target.value.replace(/\D/g, '').slice(0, 6);
                  setOtp(value);
                  setError(null);
                }}
                disabled={loading}
                maxLength={6}
                autoFocus
              />
              <p className="form-hint">
                6-digit OTP dalein.
              </p>
            </div>

            <button
              type="submit"
              className="btn btn-primary"
              disabled={loading || otp.length !== 6}
            >
              {loading ? 'Verify ho raha hai…' : 'Verify OTP'}
            </button>

            <div className="resend-otp">
              {blockedUntil && Date.now() < blockedUntil ? (
                <p className="error" style={{ fontSize: '14px', marginTop: '8px' }}>
                  Blocked: Try again after {Math.ceil((blockedUntil - Date.now()) / 60000)} minute(s) / 
                  ब्लॉक: {Math.ceil((blockedUntil - Date.now()) / 60000)} मिनट बाद कोशिश करें
                </p>
              ) : countdown > 0 ? (
                <p className="countdown">
                  Resend OTP in {countdown}s / {countdown} सेकंड में OTP फिर से भेजें
                  {resendAttempts > 0 && ` (${resendAttempts}/${5} attempts)`}
                </p>
              ) : (
                <div>
                  <button
                    type="button"
                    className="link-button"
                    onClick={handleResendOTP}
                    disabled={loading || (blockedUntil && Date.now() < blockedUntil)}
                  >
                    Resend OTP / OTP फिर से भेजें
                  </button>
                  {resendAttempts > 0 && (
                    <p className="form-hint" style={{ fontSize: '12px', marginTop: '4px' }}>
                      Resend attempts: {resendAttempts}/5 per hour / Resend attempts: {resendAttempts}/5 प्रति घंटा
                    </p>
                  )}
                </div>
              )}
            </div>
          </form>
        )}

        {step === 'verifying' && (
          <div className="verifying">
            <p>OTP verify ho raha hai…</p>
          </div>
        )}
      </div>
    </div>
  );
}

export default PhoneVerificationScreen;
