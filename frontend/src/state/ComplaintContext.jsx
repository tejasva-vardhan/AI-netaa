import { createContext, useContext, useState, useEffect, useRef } from 'react';

const ComplaintContext = createContext();

// Debounce delay for async persistence (avoid blocking UI)
const PERSIST_DEBOUNCE_MS = 300;

export function useComplaintState() {
  const context = useContext(ComplaintContext);
  if (!context) {
    throw new Error('useComplaintState must be used within ComplaintProvider');
  }
  return context;
}

export function ComplaintProvider({ children }) {
  const [isTyping, setIsTyping] = useState(false);
  const messageQueueRef = useRef([]);
  const isProcessingMessageRef = useRef(false);
  
  const [complaintData, setComplaintData] = useState(() => {
    // ISSUE 2 & 4: Load from localStorage if exists, but DON'T auto-resume
    // App always starts from LandingScreen - user must explicitly choose to continue
    const saved = localStorage.getItem('complaint_draft');
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        // Determine current step based on completed data (for resume functionality)
        const step = determineCurrentStep(parsed);
        return {
          ...parsed,
          step: step
        };
      } catch (error) {
        // If JSON is invalid, clear it and use defaults
        console.warn('Invalid complaint_draft data in localStorage, clearing:', error);
        localStorage.removeItem('complaint_draft');
      }
    }
    return {
      summary: '',
      description: '',
      category: '',
      urgency: 'medium',
      location: null,
      photo: null,
      conversation: [],
      step: 'summary', // 'summary' | 'description' | 'location' | 'location-confirmation' | 'camera' | 'chat' | 'phone-verify-prompt' | 'phone-verify' | 'confirmation'
      completedSteps: [], // Track which steps are completed
      clarificationAsked: false, // Track if clarification question was asked (max = 1)
      userPhone: null,
      userID: null
    };
  });

  // Helper function to determine current step based on completed data
  function determineCurrentStep(data) {
    if (!data.summary) {
      return 'summary';
    }
    if (!data.description) {
      return 'description';
    }
    if (!data.location) {
      return 'location';
    }
    if (!data.photo) {
      return 'camera';
    }
    // Check if phone is verified
    const phoneVerified = localStorage.getItem('phone_verified') === 'true';
    const userID = localStorage.getItem('user_id');
    if (!phoneVerified || !userID) {
      return 'phone-verify';
    }
    return 'confirmation';
  }

  const [userPhone, setUserPhone] = useState(() => {
    return localStorage.getItem('user_phone') || null;
  });

  // Async persistence: debounce localStorage write so it doesn't block UI after each message
  const persistTimeoutRef = useRef(null);
  useEffect(() => {
    if (persistTimeoutRef.current) clearTimeout(persistTimeoutRef.current);
    if (!complaintData.summary && !complaintData.description && complaintData.conversation?.length === 0) {
      return;
    }
    persistTimeoutRef.current = setTimeout(() => {
      try {
        localStorage.setItem('complaint_draft', JSON.stringify(complaintData));
      } finally {
        persistTimeoutRef.current = null;
      }
    }, PERSIST_DEBOUNCE_MS);
    return () => {
      if (persistTimeoutRef.current) clearTimeout(persistTimeoutRef.current);
    };
  }, [complaintData]);

  const updateComplaintData = (updates) => {
    setComplaintData(prev => ({ ...prev, ...updates }));
  };

  // Calculate delay based on message length
  const calculateTypingDelay = (text) => {
    if (!text) return 400;
    const length = text.length;
    // Short messages (< 50 chars): ~400ms
    // Medium messages (50-150 chars): ~600-900ms
    // Long messages (> 150 chars): ~900-1200ms
    if (length < 50) {
      return 400 + Math.random() * 200; // 400-600ms
    } else if (length < 150) {
      return 600 + Math.random() * 300; // 600-900ms
    } else {
      return 900 + Math.random() * 300; // 900-1200ms
    }
  };

  // Process message queue (one at a time)
  const processMessageQueue = () => {
    if (isProcessingMessageRef.current || messageQueueRef.current.length === 0) {
      return;
    }

    isProcessingMessageRef.current = true;
    const message = messageQueueRef.current.shift();

    if (message.type === 'bot') {
      const delay = calculateTypingDelay(message.text);
      
      // Show typing indicator
      setIsTyping(true);
      
      // Add message after delay
      setTimeout(() => {
        setIsTyping(false);
        setComplaintData(prev => ({
          ...prev,
          conversation: [...prev.conversation, message]
        }));
        isProcessingMessageRef.current = false;
        // Process next message in queue
        processMessageQueue();
      }, delay);
    } else {
      // User messages appear instantly
      setComplaintData(prev => ({
        ...prev,
        conversation: [...prev.conversation, message]
      }));
      isProcessingMessageRef.current = false;
      // Process next message in queue
      processMessageQueue();
    }
  };

  const addMessage = (message) => {
    // Add to queue
    messageQueueRef.current.push(message);
    // Process queue
    processMessageQueue();
  };

  const clearComplaintData = (options = {}) => {
    const keepConversation = options.keepConversation === true;
    setComplaintData(prev => {
      const next = {
        summary: '',
        description: '',
        category: '',
        urgency: 'medium',
        location: null,
        photo: null,
        conversation: keepConversation ? prev.conversation : [],
        step: 'summary',
        completedSteps: [],
        clarificationAsked: false,
        userPhone: prev.userPhone ?? null,
        userID: prev.userID ?? null
      };
      if (!keepConversation) {
        localStorage.removeItem('complaint_draft');
      } else {
        setTimeout(() => {
          try {
            localStorage.setItem('complaint_draft', JSON.stringify(next));
          } catch (_) {}
        }, 0);
      }
      return next;
    });
  };

  const setPhone = (phone) => {
    setUserPhone(phone);
    localStorage.setItem('user_phone', phone);
  };

  // Expose goToStep function ref (will be set by ChatScreen)
  const goToStepRef = useRef(null);

  return (
    <ComplaintContext.Provider
      value={{
        complaintData,
        updateComplaintData,
        addMessage,
        clearComplaintData,
        userPhone,
        setPhone,
        isTyping,
        goToStepRef
      }}
    >
      {children}
    </ComplaintContext.Provider>
  );
}
