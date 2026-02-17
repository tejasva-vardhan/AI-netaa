// Error handling utilities
import { ApiError } from '../services/api';

/**
 * Get user-friendly error message from API error
 */
export function getErrorMessage(error) {
  if (!error) {
    return 'An unexpected error occurred.';
  }

  if (error instanceof ApiError) {
    switch (error.status) {
      case 0:
        return 'Network error. Please check your connection.';
      case 400:
        return error.message || 'Invalid data. Please check your input.';
      case 401:
        return 'Please verify your phone number first.';
      case 403:
        return 'You do not have permission to perform this action.';
      case 404:
        return 'Resource not found.';
      case 500:
      case 502:
      case 503:
      case 504:
        return 'Server error. Please try again later.';
      default:
        return error.message || `Error ${error.status}. Please try again.`;
    }
  }

  return error.message || 'An unexpected error occurred.';
}

/**
 * Check if error is retryable
 */
export function isRetryable(error) {
  if (!error) return false;
  
  if (error instanceof ApiError) {
    // Network errors and server errors are retryable
    return error.status === 0 || 
           error.status >= 500 || 
           error.code === 'NETWORK_ERROR' ||
           error.code === 'TIMEOUT';
  }
  
  return false;
}

/**
 * Check if error is due to authentication
 */
export function isAuthError(error) {
  if (!error) return false;
  
  if (error instanceof ApiError) {
    return error.status === 401 || error.status === 403;
  }
  
  return false;
}
