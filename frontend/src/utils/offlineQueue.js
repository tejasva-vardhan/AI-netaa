// Offline queue manager for complaint submissions
import api from '../services/api';

const QUEUE_KEY = 'pending_submissions';
const RETRY_INTERVAL = 30000; // 30 seconds
const MAX_RETRIES = 5;

let retryIntervalId = null;

/**
 * Save complaint to offline queue
 */
export function saveToQueue(complaintData) {
  const queue = getQueue();
  queue.push({
    ...complaintData,
    timestamp: new Date().toISOString(),
    retryCount: 0,
    lastRetryAt: null
  });
  localStorage.setItem(QUEUE_KEY, JSON.stringify(queue));
}

/**
 * Get all pending submissions from queue
 */
export function getQueue() {
  const stored = localStorage.getItem(QUEUE_KEY);
  return stored ? JSON.parse(stored) : [];
}

/**
 * Remove complaint from queue
 */
export function removeFromQueue(index) {
  const queue = getQueue();
  queue.splice(index, 1);
  localStorage.setItem(QUEUE_KEY, JSON.stringify(queue));
}

/**
 * Process queue - attempt to submit all pending complaints
 */
export async function processQueue() {
  const queue = getQueue();
  if (queue.length === 0) {
    return;
  }

  const userID = localStorage.getItem('user_id');
  if (!userID) {
    // User not verified, can't submit
    return;
  }

  const results = [];
  
  for (let i = queue.length - 1; i >= 0; i--) {
    const item = queue[i];
    
    // Skip if exceeded max retries
    if (item.retryCount >= MAX_RETRIES) {
      // Move to failed queue or remove
      removeFromQueue(i);
      continue;
    }

    try {
      // Attempt submission
      const response = await api.createComplaint(item);
      
      // Success - remove from queue
      removeFromQueue(i);
      results.push({ success: true, complaintId: response.complaint_id });
    } catch (error) {
      // Update retry count
      item.retryCount = (item.retryCount || 0) + 1;
      item.lastRetryAt = new Date().toISOString();
      
      // Update queue
      const updatedQueue = getQueue();
      updatedQueue[i] = item;
      localStorage.setItem(QUEUE_KEY, JSON.stringify(updatedQueue));
      
      // Only retry on network errors, not validation errors
      if (error.status === 0 || (error.status >= 500 && error.status < 600)) {
        // Network/server error - will retry next time
        results.push({ success: false, error: 'Network error, will retry' });
      } else {
        // Validation or other error - remove from queue
        removeFromQueue(i);
        results.push({ success: false, error: 'Validation error, removed from queue' });
      }
    }
  }

  return results;
}

/**
 * Start automatic retry processing
 */
export function startAutoRetry() {
  if (retryIntervalId) {
    return; // Already running
  }

  // Process immediately
  processQueue();

  // Then process every RETRY_INTERVAL
  retryIntervalId = setInterval(() => {
    processQueue();
  }, RETRY_INTERVAL);
}

/**
 * Stop automatic retry processing
 */
export function stopAutoRetry() {
  if (retryIntervalId) {
    clearInterval(retryIntervalId);
    retryIntervalId = null;
  }
}

/**
 * Get queue status
 */
export function getQueueStatus() {
  const queue = getQueue();
  return {
    pending: queue.length,
    items: queue.map((item, index) => ({
      index,
      retryCount: item.retryCount || 0,
      timestamp: item.timestamp,
      lastRetryAt: item.lastRetryAt
    }))
  };
}
