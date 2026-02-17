import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../stores/authStore';

// Backend has no email/password signup - only phone OTP. Redirect to Login (phone verify).
function Signup() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuthStore();

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard');
    } else {
      navigate('/login', { replace: true });
    }
  }, [isAuthenticated, navigate]);

  return null;
}

export default Signup;
