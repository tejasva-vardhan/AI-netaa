import React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuthStore } from '../stores/authStore';
import { FaList, FaSignInAlt, FaSignOutAlt } from 'react-icons/fa';

function Navbar() {
  const navigate = useNavigate();
  const { isAuthenticated, logout } = useAuthStore();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <nav className="fixed top-0 left-0 right-0 z-40 bg-white/90 backdrop-blur border-b border-gray-200 shadow-sm">
      <div className="container mx-auto px-4 flex items-center justify-between h-14">
        <Link to={isAuthenticated ? '/dashboard' : '/'} className="flex items-center gap-2 font-bold text-netaji-navy text-lg">
          AI NETA
        </Link>
        <div className="flex items-center gap-4">
          {isAuthenticated ? (
            <>
              <Link to="/dashboard" className="flex items-center gap-2 text-gray-700 hover:text-netaji-navy transition">
                <FaList /> Complaints
              </Link>
              <button
                type="button"
                onClick={handleLogout}
                className="flex items-center gap-2 text-gray-700 hover:text-red-600 transition"
              >
                <FaSignOutAlt /> Logout
              </button>
            </>
          ) : (
            <Link to="/login" className="flex items-center gap-2 text-netaji-navy font-medium hover:text-netaji-saffron transition">
              <FaSignInAlt /> Login
            </Link>
          )}
        </div>
      </div>
    </nav>
  );
}

export default Navbar;
