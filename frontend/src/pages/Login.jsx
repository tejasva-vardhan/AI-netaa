import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import toast from 'react-hot-toast';
import { useAuthStore } from '../stores/authStore';

function Login() {
  const navigate = useNavigate();
  const { isAuthenticated, sendOTP, verifyOTP, hydrate } = useAuthStore();
  const [phone, setPhone] = useState('');
  const [otp, setOtp] = useState('');
  const [step, setStep] = useState('phone');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard');
    }
  }, [isAuthenticated, navigate]);

  const handleSendOTP = async (e) => {
    e.preventDefault();
    setError('');
    const clean = phone.replace(/\D/g, '');
    if (clean.length !== 10 || !/^[6-9]/.test(clean)) {
      setError('Valid 10-digit phone number dalein.');
      return;
    }
    setLoading(true);
    const ok = await sendOTP(clean);
    setLoading(false);
    if (ok) {
      setStep('otp');
    }
  };

  const handleVerifyOTP = async (e) => {
    e.preventDefault();
    setError('');
    if (!otp || otp.length < 4) {
      setError('OTP dalein (4-6 digits).');
      return;
    }
    const cleanPhone = phone.replace(/\D/g, '');
    setLoading(true);
    const ok = await verifyOTP(cleanPhone, otp.trim());
    setLoading(false);
    if (ok) {
      navigate('/dashboard');
    }
  };

  if (isAuthenticated) return null;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-gray-100 pt-20 flex items-center justify-center px-4">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="bg-white rounded-3xl shadow-xl p-8 max-w-md w-full"
      >
        <h1 className="text-2xl font-bold text-center text-netaji-navy mb-2">Login</h1>
        <p className="text-gray-600 text-center mb-6">Phone number se login karein</p>

        {step === 'phone' ? (
          <form onSubmit={handleSendOTP} className="space-y-4">
            <input
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value.replace(/\D/g, '').slice(0, 10))}
              placeholder="10-digit mobile number"
              className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:border-netaji-navy outline-none"
              maxLength={10}
            />
            {error && <p className="text-red-500 text-sm">{error}</p>}
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-gradient-to-r from-netaji-navy to-netaji-green text-white rounded-xl font-semibold hover:shadow-lg transition disabled:opacity-50"
            >
              {loading ? 'Bhej rahe hain...' : 'OTP bhejein'}
            </button>
          </form>
        ) : (
          <form onSubmit={handleVerifyOTP} className="space-y-4">
            <p className="text-sm text-gray-600">OTP bheja gaya: {phone}</p>
            <input
              type="text"
              value={otp}
              onChange={(e) => setOtp(e.target.value.replace(/\D/g, '').slice(0, 6))}
              placeholder="OTP dalein"
              className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:border-netaji-navy outline-none"
              maxLength={6}
            />
            {error && <p className="text-red-500 text-sm">{error}</p>}
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-gradient-to-r from-netaji-green to-netaji-navy text-white rounded-xl font-semibold hover:shadow-lg transition disabled:opacity-50"
            >
              {loading ? 'Verify ho raha hai...' : 'Verify karein'}
            </button>
            <button
              type="button"
              onClick={() => { setStep('phone'); setOtp(''); setError(''); }}
              className="w-full py-2 text-gray-600 hover:text-netaji-navy"
            >
              Number change karein
            </button>
          </form>
        )}
      </motion.div>
    </div>
  );
}

export default Login;
