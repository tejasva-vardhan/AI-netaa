import { create } from 'zustand';
import api from '../services/api';
import toast from 'react-hot-toast';

// Auth uses existing backend: phone OTP only (no email/password). Token from /users/otp/verify.
function getStoredAuth() {
  const token = localStorage.getItem('auth_token');
  const phoneVerified = localStorage.getItem('phone_verified') === 'true';
  const userId = localStorage.getItem('user_id');
  return {
    token,
    isAuthenticated: !!(token && phoneVerified),
    userId: userId || null,
  };
}

export const useAuthStore = create((set, get) => ({
  ...getStoredAuth(),

  sendOTP: async (phoneNumber) => {
    try {
      await api.sendOTP(phoneNumber);
      toast.success('OTP bheja gaya. Apna number check karein.');
      return true;
    } catch (error) {
      toast.error(error.message || 'OTP bhejne mein error');
      return false;
    }
  },

  verifyOTP: async (phoneNumber, otp) => {
    try {
      const response = await api.verifyOTP(phoneNumber, otp);
      localStorage.setItem('user_phone', phoneNumber);
      set({
        token: response.token,
        isAuthenticated: true,
        userId: String(response.user_id),
      });
      toast.success('Phone verify ho gaya!');
      return true;
    } catch (error) {
      toast.error(error.message || 'Verify fail');
      return false;
    }
  },

  logout: () => {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('user_id');
    localStorage.removeItem('phone_verified');
    localStorage.removeItem('phone_verified_at');
    localStorage.removeItem('user_id_source');
    set({ token: null, isAuthenticated: false, userId: null });
    toast.success('Logged out');
  },

  hydrate: () => set(getStoredAuth()),
}));
