import { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import Navbar from './components/Navbar';
import LandingScreen from './screens/LandingScreen';
import ChatScreen from './screens/ChatScreen';
import CameraScreen from './screens/CameraScreen';
import LocationScreen from './screens/LocationScreen';
import PhoneVerificationScreen from './screens/PhoneVerificationScreen';
import ReviewScreen from './screens/ReviewScreen';
import ComplaintsListScreen from './screens/ComplaintsListScreen';
import ComplaintDetailScreen from './screens/ComplaintDetailScreen';
import AuthorityLayout, { AuthorityIndexRedirect } from './screens/authority/AuthorityLayout';
import AuthorityLoginScreen from './screens/authority/AuthorityLoginScreen';
import AuthorityDashboardScreen from './screens/authority/AuthorityDashboardScreen';
import AuthorityComplaintDetailScreen from './screens/authority/AuthorityComplaintDetailScreen';
import PublicCaseScreen from './screens/PublicCaseScreen';
import { ComplaintProvider } from './state/ComplaintContext';
import { startAutoRetry, stopAutoRetry } from './utils/offlineQueue';
import { useAuthStore } from './stores/authStore';
import Login from './pages/Login';
import Signup from './pages/Signup';
import Dashboard from './pages/Dashboard';
import Chat from './pages/Chat';
import ComplaintTracker from './pages/ComplaintTracker';
import ComplaintStatus from './pages/ComplaintStatus';
import './App.css';

function App() {
  const [isOnline, setIsOnline] = useState(navigator.onLine);
  const { isAuthenticated } = useAuthStore();

  useEffect(() => {
    const handleOnline = () => {
      setIsOnline(true);
      startAutoRetry();
    };
    const handleOffline = () => {
      setIsOnline(false);
      stopAutoRetry();
    };
    setIsOnline(navigator.onLine);
    if (navigator.onLine) startAutoRetry();
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);
    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
      stopAutoRetry();
    };
  }, []);

  return (
    <ComplaintProvider>
      <Router>
        <div className="app">
          <Navbar />
          <Toaster position="top-right" />
          {!isOnline && (
            <div className="offline-banner">
              No internet connection. Your data will be saved and synced when online.
            </div>
          )}
          <Routes>
            <Route path="/" element={<LandingScreen />} />
            <Route path="/login" element={!isAuthenticated ? <Login /> : <Navigate to="/dashboard" replace />} />
            <Route path="/signup" element={!isAuthenticated ? <Signup /> : <Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={isAuthenticated ? <Dashboard /> : <Navigate to="/login" replace />} />
            <Route path="/chat" element={isAuthenticated ? <Chat /> : <Navigate to="/login" replace />} />
            <Route path="/chat-legacy" element={<ChatScreen />} />
            <Route path="/camera" element={<CameraScreen />} />
            <Route path="/location" element={<LocationScreen />} />
            <Route path="/phone-verify" element={<PhoneVerificationScreen />} />
            <Route path="/review" element={<ReviewScreen />} />
            <Route path="/complaints" element={isAuthenticated ? <ComplaintsListScreen /> : <Navigate to="/login" replace />} />
            <Route path="/complaint/:id" element={isAuthenticated ? <ComplaintStatus /> : <Navigate to="/login" replace />} />
            <Route path="/complaints/:id" element={isAuthenticated ? <ComplaintTracker /> : <Navigate to="/login" replace />} />
            <Route path="/case/:complaintNumber" element={<PublicCaseScreen />} />
            <Route path="/authority" element={<AuthorityLayout />}>
              <Route index element={<AuthorityIndexRedirect />} />
              <Route path="login" element={<AuthorityLoginScreen />} />
              <Route path="dashboard" element={<AuthorityDashboardScreen />} />
              <Route path="complaints/:id" element={<AuthorityComplaintDetailScreen />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </div>
      </Router>
    </ComplaintProvider>
  );
}

export default App;
