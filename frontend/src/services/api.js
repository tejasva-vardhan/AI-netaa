// API service - integrated with backend
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

class ApiError extends Error {
  constructor(message, status, code) {
    super(message);
    this.status = status;
    this.code = code;
    this.name = 'ApiError';
  }
}

// Retry configuration
const RETRY_CONFIG = {
  maxRetries: 3,
  retryDelay: 1000, // Start with 1 second
  retryableStatuses: [0, 500, 502, 503, 504], // Network errors and server errors
};

// Retry logic with exponential backoff
async function requestWithRetry(endpoint, options = {}, retryCount = 0) {
  const url = `${API_BASE_URL}${endpoint}`;
  const config = {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers
    },
    ...options
  };

  // ISSUE 2: Add Authorization header with JWT token (REQUIRED for authenticated requests)
  // NOTE: OTP endpoints don't need auth, so suppress warning for those
  const token = localStorage.getItem('auth_token');
  const isOTPEndpoint = endpoint.includes('/otp/');
  
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`;
  } else if (!isOTPEndpoint) {
    // Log warning if token missing for non-OTP endpoints (helps debug 401 errors)
    if (import.meta.env.DEV) {
      console.warn('[WARNING] Authorization token not found - user may need to verify phone');
    }
  }

  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 30000); // 30 second timeout

    const response = await fetch(url, {
      ...config,
      signal: controller.signal
    });

    clearTimeout(timeoutId);

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      const error = new ApiError(
        errorData.message || errorData.error || `HTTP error! status: ${response.status}`,
        response.status,
        errorData.code
      );

      // ISSUE 3: Distinguish 401/4xx from network errors - do NOT retry 401
      // 401 = authentication error - user needs to verify phone
      if (response.status === 401) {
        throw error; // Don't retry auth errors
      }

      // Retry on server errors (5xx) only
      if (RETRY_CONFIG.retryableStatuses.includes(response.status) && retryCount < RETRY_CONFIG.maxRetries) {
        const delay = RETRY_CONFIG.retryDelay * Math.pow(2, retryCount);
        await new Promise(resolve => setTimeout(resolve, delay));
        return requestWithRetry(endpoint, options, retryCount + 1);
      }

      throw error;
    }

    // Success (200/201): parse body safely so empty or invalid JSON never causes false-negative
    const body = await response.json().catch(() => ({}));
    return body;
  } catch (error) {
    // Handle abort (timeout)
    if (error.name === 'AbortError') {
      throw new ApiError('Request timeout. Please check your connection.', 0, 'TIMEOUT');
    }

    // Handle network errors with retry
    if (error instanceof ApiError) {
      throw error;
    }

    // ISSUE 3: Check if navigator.onLine to determine real network failure
    const isActuallyOffline = !navigator.onLine;

    // Check for CORS errors (TypeError with CORS-related message)
    if (error instanceof TypeError) {
      const errorMessage = error.message || '';
      if (errorMessage.includes('CORS') || errorMessage.includes('Failed to fetch') || errorMessage.includes('NetworkError')) {
        // CORS or network error
        if (isActuallyOffline) {
          // Real offline - can retry when online
          if (retryCount < RETRY_CONFIG.maxRetries) {
            const delay = RETRY_CONFIG.retryDelay * Math.pow(2, retryCount);
            await new Promise(resolve => setTimeout(resolve, delay));
            return requestWithRetry(endpoint, options, retryCount + 1);
          }
          throw new ApiError('Network error. You are offline. Your data will be saved locally.', 0, 'NETWORK_ERROR');
        }
        // CORS error - don't retry
        throw new ApiError('CORS error. Please check backend CORS configuration.', 0, 'CORS_ERROR');
      }
    }

    // Network error - only retry if actually offline
    if (isActuallyOffline && retryCount < RETRY_CONFIG.maxRetries) {
      const delay = RETRY_CONFIG.retryDelay * Math.pow(2, retryCount);
      await new Promise(resolve => setTimeout(resolve, delay));
      return requestWithRetry(endpoint, options, retryCount + 1);
    }

    // Final network error - only show if actually offline
    if (isActuallyOffline) {
      throw new ApiError('Network error. You are offline. Your data will be saved locally.', 0, 'NETWORK_ERROR');
    }

    // If online but fetch failed, it's likely a different error (CORS, server down, etc.)
    throw new ApiError('Request failed. Please check your connection and try again.', 0, 'NETWORK_ERROR');
  }
}

async function request(endpoint, options = {}) {
  return requestWithRetry(endpoint, options, 0);
}

export const api = {
  // Create complaint
  // POST /api/v1/complaints
  // Request: CreateComplaintRequest
  // Response: CreateComplaintResponse { complaint_id, complaint_number, status, message }
  createComplaint: async (complaintData) => {
    // ISSUE 1 & 2: Ensure auth token exists (required for authenticated requests)
    const token = localStorage.getItem('auth_token');
    const phoneVerified = localStorage.getItem('phone_verified') === 'true';
    
    if (!token || !phoneVerified) {
      throw new ApiError('Please verify your phone number to submit complaint', 401, 'PHONE_NOT_VERIFIED');
    }
    
    // Log token presence for debugging (don't log actual token)
    if (import.meta.env.DEV) {
      console.log('[DEBUG] Submitting complaint with auth token present');
    }
    
    // ISSUE 1 & 4: Handle photo upload - CRITICAL: photo MUST be uploaded before submission
    let attachmentUrls = [];
    if (complaintData.photo?.blob) {
      try {
        console.log('Uploading photo blob...', { blobSize: complaintData.photo.blob.size, blobType: complaintData.photo.blob.type });
        const uploadResult = await api.uploadPhoto(complaintData.photo.blob);
        const uploadedUrl = uploadResult.url || uploadResult.file_path || uploadResult.fileUrl;
        if (uploadedUrl) {
          attachmentUrls = [uploadedUrl];
          console.log('Photo uploaded successfully:', uploadedUrl);
        } else {
          throw new Error('Upload succeeded but no URL returned');
        }
      } catch (err) {
        console.error('Photo upload failed:', err);
        // ISSUE 1: Don't continue without photo - throw error to block submission
        throw new ApiError(
          'Failed to upload photo. Please try again or capture a new photo.',
          400,
          'PHOTO_UPLOAD_FAILED'
        );
      }
    } else if (complaintData.photo?.url) {
      // If URL already exists (e.g., from previous upload), use it
      attachmentUrls = [complaintData.photo.url];
      console.log('Using existing photo URL:', complaintData.photo.url);
    } else {
      // ISSUE 1: No photo at all - this should be caught by ReviewScreen validation, but double-check
      console.error('No photo found in complaintData:', complaintData.photo);
      throw new ApiError(
        'Photo is required for live proof. Please capture a photo.',
        400,
        'PHOTO_MISSING'
      );
    }

    // Map location: payload may have location.latitude/longitude or location.lat/lng
    const lat = complaintData.location?.latitude ?? complaintData.location?.lat ?? null;
    const lng = complaintData.location?.longitude ?? complaintData.location?.lng ?? null;

    // ISSUE 4: Map frontend complaint data to backend API format
    const requestBody = {
      title: complaintData.summary,
      description: complaintData.description,
      category: complaintData.category || null,
      location_id: complaintData.location?.location_id || 1,
      latitude: lat,
      longitude: lng,
      priority: complaintData.urgency || 'medium',
      public_consent_given: true,
      attachment_urls: attachmentUrls
    };
    if (complaintData.notifyEmails && Array.isArray(complaintData.notifyEmails) && complaintData.notifyEmails.length > 0) {
      requestBody.notify_emails = complaintData.notifyEmails;
    }

    // Log request payload for debugging (without sensitive data)
    console.log('Submitting complaint with payload:', {
      title: requestBody.title,
      hasDescription: !!requestBody.description,
      hasLocation: !!(requestBody.latitude && requestBody.longitude),
      attachmentUrlsCount: attachmentUrls.length,
      attachmentUrls: attachmentUrls
    });

    const raw = await request('/complaints', {
      method: 'POST',
      body: JSON.stringify(requestBody)
    });
    // Normalize: backend may return { complaint_id, complaint_number } or wrap in .data or use .id
    const data = raw && typeof raw === 'object' && raw.data != null ? raw.data : raw;
    const out = { ...(typeof data === 'object' && data !== null ? data : {}) };
    out.complaint_id = out.complaint_id ?? out.id ?? null;
    out.complaint_number = out.complaint_number ?? (out.complaint_id != null ? String(out.complaint_id) : null) ?? (out.id != null ? String(out.id) : null);
    if (import.meta.env.DEV) {
      console.log('[createComplaint] response status OK, body:', raw, 'normalized:', out);
    }
    return out;
  },

  // Upload voice note for a complaint (after creation). Does not block; failure must not fail complaint flow.
  // POST /api/v1/complaints/{id}/voice — Content-Type: audio/webm or audio/wav, body = blob
  uploadVoice: async (complaintId, blob) => {
    const token = localStorage.getItem('auth_token');
    if (!token) {
      throw new ApiError('Authentication required', 401, 'UNAUTHORIZED');
    }
    const baseUrl = API_BASE_URL.replace(/\/$/, '');
    const url = `${baseUrl}/complaints/${complaintId}/voice`;
    const contentType = blob.type && blob.type.startsWith('audio/') ? blob.type : 'audio/webm';
    const res = await fetch(url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': contentType
      },
      body: blob
    });
    if (!res.ok) {
      const errBody = await res.json().catch(() => ({}));
      throw new ApiError(errBody.message || res.statusText, res.status, errBody.code || 'VOICE_UPLOAD_FAILED');
    }
    return res.json().catch(() => ({}));
  },

  // Get complaint by ID
  // GET /api/v1/complaints/{id}
  // Response: ComplaintDetailResponse
  getComplaint: async (complaintId) => {
    return request(`/complaints/${complaintId}`);
  },

  // Get complaint timeline
  // GET /api/v1/complaints/{id}/timeline
  // Response: StatusTimelineResponse { complaint_id, complaint_number, timeline[] }
  getComplaintTimeline: async (complaintId) => {
    return request(`/complaints/${complaintId}/timeline`);
  },

  // Get user's complaints
  // Note: Backend doesn't have dedicated endpoint, so we'll need to filter client-side
  // or backend needs to add GET /api/v1/complaints?user_id=X
  // For now, we'll use a placeholder that returns empty array
  // In production, backend should add: GET /api/v1/complaints (returns user's complaints when X-User-ID header present)
  getUserComplaints: async () => {
    const userID = localStorage.getItem('user_id') || localStorage.getItem('user_phone');
    if (!userID) {
      return [];
    }

    try {
      // Try to fetch complaints (backend should filter by user_id from header)
      // If backend doesn't support this yet, return empty array
      const response = await request('/complaints', {
        method: 'GET'
      });
      
      // Backend might return array directly or wrapped in object
      if (Array.isArray(response)) {
        return response;
      }
      return response.complaints || response.items || [];
    } catch (err) {
      // If endpoint doesn't exist or fails, return empty array
      console.warn('Failed to fetch complaints list:', err);
      return [];
    }
  },

  // Send OTP to phone number
  // POST /api/v1/users/otp/send
  sendOTP: async (phoneNumber) => {
    // ISSUE 1: Call real backend API for OTP sending
    try {
      const response = await request('/users/otp/send', {
        method: 'POST',
        body: JSON.stringify({ phone_number: phoneNumber })
      });
      
      // DEV MODE: Backend returns OTP in response for testing
      console.log('[DEBUG] sendOTP response:', response);
      if (import.meta.env.DEV && response.otp) {
        console.log('\n========================================');
        console.log('✅ [DEV MODE] OTP CODE:', response.otp);
        console.log('Phone:', phoneNumber);
        console.log('========================================\n');
      } else if (import.meta.env.DEV) {
        console.warn('[WARNING] OTP not found in response. Response keys:', Object.keys(response));
        console.log('[INFO] OTP sent. Check backend console/terminal for the OTP code.');
        console.log('[INFO] Backend will print: [PILOT MODE] OTP for <phone>: <code>');
      }
      
      return response;
    } catch (err) {
      // ISSUE 4: Remove fake fallbacks - backend MUST be available
      throw err;
    }
  },

  // Verify OTP
  // POST /api/v1/users/otp/verify
  verifyOTP: async (phoneNumber, otp) => {
    // ISSUE 1 & 2: Call real backend API - backend creates user and returns user_id
    try {
      const response = await request('/users/otp/verify', {
        method: 'POST',
        body: JSON.stringify({ phone_number: phoneNumber, otp: otp })
      });
      
      // Backend returns: { success: true, user_id: number, phone_verified: true, token: string }
      // Clear local OTP storage
      localStorage.removeItem(`otp_${phoneNumber}`);
      localStorage.removeItem(`otp_expires_${phoneNumber}`);
      
      // ISSUE 1: Store JWT token for authenticated requests
      if (!response.token) {
        throw new ApiError('Backend did not return authentication token. Please try again.', 500, 'NO_TOKEN');
      }
      
      localStorage.setItem('auth_token', response.token);
      localStorage.setItem('user_id', response.user_id.toString());
      localStorage.setItem('phone_verified', 'true');
      localStorage.setItem('phone_verified_at', new Date().toISOString());
      localStorage.setItem('user_id_source', 'backend');
      
      console.log('[DEBUG] Phone verified, token stored, user_id:', response.user_id);
      
      return { 
        user_id: response.user_id, 
        phone_verified: response.phone_verified || true,
        token: response.token
      };
    } catch (err) {
      // Log detailed error for debugging
      if (import.meta.env.DEV) {
        console.error('[ERROR] OTP verification failed:', {
          status: err.status,
          message: err.message,
          phoneNumber: phoneNumber,
          otpLength: otp?.length
        });
      }
      
      // ISSUE 4: Remove fake fallbacks - backend MUST be available
      throw err;
    }
  },

  // Upload photo
  // ISSUE 4: For pilot, convert blob to data URL since backend expects URLs
  // In production, backend should have upload endpoint that returns URL
  uploadPhoto: async (file) => {
    // For pilot: Convert blob to data URL (backend accepts any URL string)
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onloadend = () => {
        // Return data URL as the "uploaded" URL
        // Backend will accept this as attachment_urls[0]
        resolve({ 
          url: reader.result, 
          file_path: reader.result,
          fileUrl: reader.result 
        });
      };
      reader.onerror = () => {
        reject(new ApiError('Failed to read photo file', 400, 'PHOTO_READ_ERROR'));
      };
      reader.readAsDataURL(file);
    });

    // Production code (when backend has upload endpoint):
    /*
    const formData = new FormData();
    formData.append('file', file);

    const userID = localStorage.getItem('user_id') || localStorage.getItem('user_phone');
    const headers = {};
    if (userID) {
      headers['X-User-ID'] = userID;
    }

    const response = await fetch(`${API_BASE_URL}/upload`, {
      method: 'POST',
      headers,
      body: formData
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new ApiError(
        errorData.message || 'Failed to upload photo',
        response.status,
        errorData.code
      );
    }

    return await response.json();
    */
  },

  // Reset chat/draft state (hidden "restart" UX). Auth remains; clears server-side chat_state and draft.
  resetChatDraft: async () => {
    return request('/users/chat/reset', { method: 'POST' });
  }
};

export { ApiError };
export { request };
export default api;
