import { useState, useRef, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useComplaintState } from '../state/ComplaintContext';
import api, { ApiError } from '../services/api';
import { saveToQueue } from '../utils/offlineQueue';
import { ensureCategory } from '../utils/categoryInference';
import { getRandomPhrase } from '../utils/botPhrases';
import MessageList from '../components/MessageList';
import ChatInput from '../components/ChatInput';
import './ChatScreen.css';

// STRICT FLOW ORDER - steps must complete in this sequence
const FLOW_STEPS = [
  'summary',
  'description',
  'description_confirm',
  'location',
  'location_confirm',
  'photo',
  'voice', // optional
  'submit'
];

function ChatScreen() {
  const navigate = useNavigate();
  const { complaintData, updateComplaintData, addMessage, clearComplaintData, isTyping, goToStepRef } = useComplaintState();
  const [inputText, setInputText] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [stage, setStage] = useState('initial'); // initial | collecting_details | confirming | submitting | completed
  const [waitingForConfirmation, setWaitingForConfirmation] = useState(false);
  const [waitingForDescriptionConfirmation, setWaitingForDescriptionConfirmation] = useState(false);
  const [waitingForLocationConfirmation, setWaitingForLocationConfirmation] = useState(false);
  const [waitingForPhoneVerification, setWaitingForPhoneVerification] = useState(false);
  const [waitingForLocationChoice, setWaitingForLocationChoice] = useState(false);
  const [pendingDescription, setPendingDescription] = useState('');
  const [expectingCorrection, setExpectingCorrection] = useState(false);
  const messagesEndRef = useRef(null);
  
  // HARD STEP TRANSITION CONTROL
  const currentActiveStepRef = useRef(null); // Only ONE step active at a time
  const isAwaitingAsyncRef = useRef(false); // Lock during GPS/photo/voice
  const isStepProcessingRef = useRef(false); // Lock during step execution
  const stepTransitionLockRef = useRef(false); // Prevent concurrent transitions
  const goToStepRefLocal = useRef(null); // Local ref for goToStep function

  // Chat-first: chat only after phone/OTP. Unauthenticated users go to phone-verify.
  const phoneVerified = localStorage.getItem('phone_verified') === 'true';
  useEffect(() => {
    if (!phoneVerified) {
      navigate('/phone-verify', { replace: true });
    }
  }, [navigate, phoneVerified]);

  // Initialize conversation on mount (only once)
  useEffect(() => {
    if (complaintData.conversation.length === 0) {
      goToStep('summary', { force: true });
    }
  }, []); // Empty deps - only run once on mount

  // Expose goToStep to context for async screens (LocationScreen, CameraScreen)
  useEffect(() => {
    if (goToStepRef) {
      goToStepRef.current = (nextStep, options) => {
        if (goToStepRefLocal.current) {
          goToStepRefLocal.current(nextStep, options);
        }
      };
    }
    return () => {
      if (goToStepRef) {
        goToStepRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [complaintData.conversation]);

  // Helper: Check if step is completed
  const isStepCompleted = (stepName) => {
    const completedSteps = complaintData.completedSteps || [];
    return completedSteps.includes(stepName);
  };

  // Helper: Mark step as completed
  const markStepCompleted = (stepName) => {
    const completedSteps = complaintData.completedSteps || [];
    if (!completedSteps.includes(stepName)) {
      updateComplaintData({ 
        completedSteps: [...completedSteps, stepName]
      });
    }
  };

  // STRICT VALIDATION: Check prerequisites before step execution
  const validateStepPrerequisites = (stepName) => {
    switch (stepName) {
      case 'description':
        return isStepCompleted('summary');
      case 'description_confirm':
        return !!complaintData.description;
      case 'location':
        return isStepCompleted('description_confirm');
      case 'location_confirm':
        return !!complaintData.location;
      case 'photo':
        return isStepCompleted('location_confirm');
      case 'voice':
        return isStepCompleted('photo');
      case 'submit':
        return isStepCompleted('photo') && phoneVerified;
      default:
        return true;
    }
  };

  // HARD STEP TRANSITION FUNCTION - ONLY entry point for step changes
  const goToStep = (nextStep, options = {}) => {
    // No step changes while submitting or after completed
    if ((stage === 'submitting' || stage === 'completed') && !options.force) {
      return;
    }
    // Prevent concurrent transitions
    if (stepTransitionLockRef.current) {
      return;
    }

    // Lock async events
    if (isAwaitingAsyncRef.current && !options.force) {
      return;
    }

    // Lock step processing
    if (isStepProcessingRef.current && !options.force) {
      return;
    }

    // Validate prerequisites
    if (!validateStepPrerequisites(nextStep) && !options.skipValidation) {
      return;
    }

    // Check if step is already active
    if (currentActiveStepRef.current === nextStep && !options.force) {
      return;
    }

    stepTransitionLockRef.current = true;
    isStepProcessingRef.current = true;

    try {
      // Update active step
      currentActiveStepRef.current = nextStep;
      
      // Update state
      updateComplaintData({ step: nextStep });

      // Stage: collecting_details for any step before confirmation; initial only on force restart
      if (options.force && nextStep === 'summary') {
        setStage('initial');
      } else if (['summary', 'description', 'description_confirm', 'location', 'location_confirm', 'location-confirmation', 'photo', 'voice'].includes(nextStep)) {
        setStage('collecting_details');
      }

      // Execute step logic
      executeStep(nextStep, options);

    } finally {
      stepTransitionLockRef.current = false;
      // isStepProcessingRef will be cleared after step completes
    }
  };
  
  // Store goToStep in ref for stable reference
  goToStepRefLocal.current = goToStep;

  // Execute step logic - ONLY called from goToStep
  // Each step directly calls goToStep for next step (event-driven)
  const executeStep = (stepName, options = {}) => {
    switch (stepName) {
      case 'summary':
        if (complaintData.conversation.length === 0) {
          addMessage({ type: 'bot', text: 'Namaste.', timestamp: new Date() });
          addMessage({ 
            type: 'bot', 
            text: 'Batayiye, kya samasya aa rahi hai. (Jaise: sadak par gaddha, paani ki samasya.)', 
            timestamp: new Date() 
          });
        } else if (complaintData.summary && !isStepCompleted('summary')) {
          markStepCompleted('summary');
          const shortRef = complaintData.summary.length > 40 
            ? complaintData.summary.substring(0, 40) + '...' 
            : complaintData.summary;
          addMessage({
            type: 'bot',
            text: `${getRandomPhrase('understood')}. ${shortRef}.\nAb zara detail mein bataiye: kab se hai, kahan hai.`,
            timestamp: new Date()
          });
          // DIRECT TRANSITION: summary → description
          setTimeout(() => {
            isStepProcessingRef.current = false;
            goToStep('description');
          }, 100);
        }
        break;

      case 'description':
        // Description step just waits for user input
        // User input will trigger description_confirm via handleSend
        isStepProcessingRef.current = false;
        break;

      case 'description_confirm':
        if (complaintData.description && !waitingForDescriptionConfirmation) {
          addMessage({
            type: 'bot',
            text: `Aap keh rahe hain ki: ${complaintData.description}\n\nKya yeh sahi hai?`,
            timestamp: new Date()
          });
          setWaitingForDescriptionConfirmation(true);
        }
        isStepProcessingRef.current = false;
        break;

      case 'location':
        if (complaintData.summary && complaintData.description && isStepCompleted('description_confirm')) {
          navigate('/location');
        }
        isStepProcessingRef.current = false;
        break;

      case 'location_confirm':
        if (complaintData.location && !complaintData.photo) {
          // Check if manual location exists - ask for GPS choice
          if (complaintData.manualLocation && (!complaintData.location.latitude || complaintData.location.manual)) {
            if (!waitingForLocationChoice) {
              addMessage({
                type: 'bot',
                text: `GPS use karun ya ${complaintData.manualLocation} se aage badhun?`,
                timestamp: new Date()
              });
              setWaitingForLocationChoice(true);
            }
          } else if (!waitingForLocationConfirmation) {
            showLocationConfirmationMessage();
          }
        }
        isStepProcessingRef.current = false;
        break;

      case 'camera':
      case 'photo':
        if (complaintData.location && isStepCompleted('location_confirm')) {
          navigate('/camera');
        }
        isStepProcessingRef.current = false;
        break;

      case 'voice':
        isStepProcessingRef.current = false;
        break;

      case 'phone-verify-prompt':
        if (complaintData.photo && complaintData.location && !phoneVerified && !waitingForPhoneVerification) {
          addMessage({
            type: 'bot',
            text: 'Aage badhne ke liye ek baar phone verify kar lein.',
            timestamp: new Date()
          });
          setWaitingForPhoneVerification(true);
        }
        isStepProcessingRef.current = false;
        break;

      case 'confirmation':
      case 'review':
        if (complaintData.summary && 
            complaintData.description && 
            complaintData.location && 
            complaintData.photo &&
            phoneVerified &&
            isStepCompleted('location_confirm') &&
            !waitingForLocationConfirmation &&
            !waitingForConfirmation) {
          showConfirmationMessage();
        }
        isStepProcessingRef.current = false;
        break;
    }
  };

  const handleSend = async () => {
    if (!inputText.trim() || isProcessing || isSubmitting) return;
    if (isStepProcessingRef.current) return;
    // No input handling after submit started or flow completed
    if (stage === 'submitting' || stage === 'completed') return;

    const raw = inputText.trim();
    const normalized = raw.toLowerCase();
    const currentStep = complaintData.step || 'summary';

    // PHOTO STEP: accept image (handled elsewhere) or text skip/yes/no → move to voice
    if (currentStep === 'photo') {
      addMessage({ type: 'user', text: raw, timestamp: new Date() });
      setInputText('');
      const wantSkip = /^(skip|no|nahi|na|yes|haan|han|sahi|theek|ok)$/.test(normalized) || normalized === 'n' || normalized === 'h';
      if (wantSkip) {
        markStepCompleted('photo');
        updateComplaintData({ photo: { skipped: true } });
        addMessage({
          type: 'bot',
          text: `${getRandomPhrase('acknowledge')}, aage badhte hain.`,
          timestamp: new Date()
        });
        goToStep('voice');
      } else {
        addMessage({
          type: 'bot',
          text: "Photo bhej sakte hain, ya 'skip' likh dijiye.",
          timestamp: new Date()
        });
      }
      return;
    }
    // VOICE STEP: accept audio (handled elsewhere) or text skip/yes/no → move to confirmation
    if (currentStep === 'voice') {
      addMessage({ type: 'user', text: raw, timestamp: new Date() });
      setInputText('');
      const wantSkip = /^(skip|no|nahi|na|yes|haan|han|sahi|theek|ok)$/.test(normalized) || normalized === 'n' || normalized === 'h';
      if (wantSkip) {
        addMessage({
          type: 'bot',
          text: `${getRandomPhrase('acknowledge')}, ab aapki complaint confirm karte hain.`,
          timestamp: new Date()
        });
        goToStep('confirmation');
      } else {
        addMessage({
          type: 'bot',
          text: "Voice message bhejna chahte hain ya 'skip' likh dijiye.",
          timestamp: new Date()
        });
      }
      return;
    }

    // CONFIRMATION ONLY: yes/no as confirm/reject only when stage or step is confirmation (no global yes/no)
    const isConfirming = stage === 'confirming' || currentStep === 'confirmation';
    const isYes = /^(haan|han|yes|sahi|theek|ok|correct|bilkul|sahi hai)$/.test(normalized);
    const isNo = /^(nahi|na|no|galat|n)$/.test(normalized) || normalized.includes('galat') || normalized.includes('change') || normalized.includes('badal') || normalized.includes('edit');
    if (isConfirming) {
      const userMessage = { type: 'user', text: raw, timestamp: new Date() };
      addMessage(userMessage);
      setInputText('');
      if (isYes) {
        setWaitingForConfirmation(false);
        setStage('submitting');
        await submitComplaint();
        return;
      }
      if (isNo) {
        setWaitingForConfirmation(false);
        setStage('collecting_details');
        setExpectingCorrection(true);
        addMessage({
          type: 'bot',
          text: `${getRandomPhrase('acknowledge')}, bata dijiye kya sahi hai.`,
          timestamp: new Date()
        });
        return;
      }
      addMessage({
        type: 'bot',
        text: 'Bas haan bol dijiye, main aage badhata hoon.',
        timestamp: new Date()
      });
      return;
    }

    // Hidden reset: user says "restart"
    if (normalized === 'restart') {
      setInputText('');
      setIsProcessing(true);
      setStage('initial');
      setWaitingForConfirmation(false);
      setWaitingForDescriptionConfirmation(false);
      setWaitingForLocationConfirmation(false);
      setWaitingForPhoneVerification(false);
      setWaitingForLocationChoice(false);
      setExpectingCorrection(false);
      setPendingDescription('');
      currentActiveStepRef.current = null;
      isAwaitingAsyncRef.current = false;
      isStepProcessingRef.current = false;
      try {
        await api.resetChatDraft();
      } catch (_) {}
      clearComplaintData();
      goToStep('summary', { force: true });
      setIsProcessing(false);
      return;
    }

    // Handle location choice (GPS vs manual)
    if (waitingForLocationChoice) {
      const userMessage = { type: 'user', text: raw, timestamp: new Date() };
      addMessage(userMessage);
      setInputText('');
      setWaitingForLocationChoice(false);

      if (normalized.includes('gps') || normalized.includes('location') || normalized === 'haan' || normalized === 'yes') {
        updateComplaintData({ 
          manualLocation: null,
          step: 'location'
        });
        addMessage({
          type: 'bot',
          text: `${getRandomPhrase('acknowledge')}, GPS se location le raha hoon.`,
          timestamp: new Date()
        });
        goToStep('location');
      } else {
        updateComplaintData({ 
          location: { manual: true, name: complaintData.manualLocation },
          step: 'location-confirmation'
        });
        markStepCompleted('location');
        addMessage({
          type: 'bot',
          text: `${getRandomPhrase('acknowledge')}, ${complaintData.manualLocation} se aage badh raha hoon.`,
          timestamp: new Date()
        });
        // DIRECT TRANSITION: location choice → location_confirm
        goToStep('location_confirm');
      }
      return;
    }

    // Handle phone verification prompt response
    if (waitingForPhoneVerification) {
      const userMessage = { type: 'user', text: raw, timestamp: new Date() };
      addMessage(userMessage);
      setInputText('');
      
      if (normalized === 'haan' || normalized === 'han' || normalized === 'yes' || normalized === 'h' || normalized === 'sahi' || normalized === 'theek' || normalized === 'ok' || normalized === 'bilkul' || normalized === 'kar lein' || normalized === 'kar lete hain') {
        setWaitingForPhoneVerification(false);
        navigate('/phone-verify');
      } else if (normalized === 'nahi' || normalized === 'na' || normalized === 'no' || normalized === 'n' || normalized === 'baad mein') {
        setWaitingForPhoneVerification(false);
        addMessage({
          type: 'bot',
          text: `${getRandomPhrase('acknowledge')}, baad mein verify kar sakte hain. Abhi complaint register kar lein?`,
          timestamp: new Date()
        });
        if (complaintData.summary && complaintData.description && complaintData.location && complaintData.photo) {
          setTimeout(() => {
            goToStep('confirmation');
          }, 1000);
        }
      } else {
        addMessage({
          type: 'bot',
          text: 'Phone verify karna hai ya baad mein? Bas haan ya nahi bata dijiye.',
          timestamp: new Date()
        });
      }
      return;
    }

    // Handle location confirmation response
    if (waitingForLocationConfirmation) {
      const userMessage = { type: 'user', text: raw, timestamp: new Date() };
      addMessage(userMessage);
      setInputText('');
      
      if (normalized === 'haan' || normalized === 'han' || normalized === 'yes' || normalized === 'h' || normalized === 'sahi' || normalized === 'theek' || normalized === 'ok' || normalized === 'bilkul') {
        markStepCompleted('location_confirm');
        setWaitingForLocationConfirmation(false);
        addMessage({
          type: 'bot',
          text: 'Agar possible ho toh ek photo bhej dijiye, isse madad milegi.',
          timestamp: new Date()
        });
        // DIRECT TRANSITION: location_confirm → photo
        goToStep('photo');
      } else if (normalized === 'nahi' || normalized === 'na' || normalized === 'no' || normalized === 'n' || normalized === 'galat' || normalized.includes('galat') || normalized.includes('change') || normalized.includes('badal') || normalized.includes('sahi nahi')) {
        setWaitingForLocationConfirmation(false);
        updateComplaintData({ 
          location: null,
          manualLocation: null,
          step: 'location',
          completedSteps: complaintData.completedSteps.filter(s => s !== 'location' && s !== 'location_confirm')
        });
        currentActiveStepRef.current = null;
        addMessage({
          type: 'bot',
          text: `${getRandomPhrase('acknowledge')}, phir se location share kar dijiye.`,
          timestamp: new Date()
        });
        // DIRECT TRANSITION: reject → location
        goToStep('location');
      } else {
        addMessage({
          type: 'bot',
          text: 'Location sahi hai na?',
          timestamp: new Date()
        });
      }
      return;
    }

    // Handle description confirmation response
    if (waitingForDescriptionConfirmation) {
      const userMessage = { type: 'user', text: raw, timestamp: new Date() };
      addMessage(userMessage);
      setInputText('');
      
      if (normalized === 'haan' || normalized === 'han' || normalized === 'yes' || normalized === 'h' || normalized === 'sahi' || normalized === 'theek' || normalized === 'ok') {
        const completedSteps = complaintData.completedSteps || [];
        const categoryResult = inferCategory(pendingDescription);
        const finalCategory = complaintData.category || categoryResult.category;
        
        markStepCompleted('description_confirm');
        updateComplaintData({ 
          description: pendingDescription,
          category: finalCategory,
          completedSteps: [...completedSteps, 'description', 'description_confirm']
        });
        
        setWaitingForDescriptionConfirmation(false);
        setPendingDescription('');
        
        addMessage({
          type: 'bot',
          text: 'Chinta mat kariye, main dekh raha hoon 🙏',
          timestamp: new Date()
        });
        
        // DIRECT TRANSITION: description_confirm → location
        goToStep('location');
      } else if (normalized === 'nahi' || normalized === 'na' || normalized === 'no' || normalized === 'n' || normalized === 'galat') {
        setWaitingForDescriptionConfirmation(false);
        setPendingDescription('');
        addMessage({
          type: 'bot',
          text: `${getRandomPhrase('acknowledge')}, phir se bata dijiye.`,
          timestamp: new Date()
        });
        // Stay on description step
        goToStep('description');
      } else {
        addMessage({
          type: 'bot',
          text: 'Yeh description sahi hai na?',
          timestamp: new Date()
        });
      }
      return;
    }

    const userMessage = { type: 'user', text: raw, timestamp: new Date() };
    addMessage(userMessage);
    setInputText('');
    setIsProcessing(true);

    // Bot response (local flow; persistence is async via context debounce)
    const capturedInput = raw;
    setTimeout(() => {
      // Handle correction if expecting one
      if (expectingCorrection) {
        setExpectingCorrection(false);
        
        const lowerText = capturedInput.toLowerCase();
        const locationKeywords = ['location', 'jagah', 'kahan', 'karera', 'kolaras', 'pohri', 'shivpuri', 'mein hoon', 'yahan', 'idhar'];
        const isLocationCorrection = locationKeywords.some(keyword => lowerText.includes(keyword)) || 
                                     (lowerText.includes('galat') && (lowerText.includes('location') || lowerText.includes('jagah')));
        
        if (isLocationCorrection) {
          updateComplaintData({ 
            location: null,
            manualLocation: null,
            step: 'location',
            completedSteps: complaintData.completedSteps.filter(s => s !== 'location' && s !== 'location_confirm')
          });
          currentActiveStepRef.current = null;
          addMessage({
            type: 'bot',
            text: `${getRandomPhrase('acknowledge')}, location update kar raha hoon.`,
            timestamp: new Date()
          });
          goToStep('location');
          setIsProcessing(false);
          return;
        }
        
        if (capturedInput.length > 30 || lowerText.includes('kab se') || lowerText.includes('detail') || lowerText.includes('hota hai')) {
          const categoryResult = inferCategory(capturedInput);
          const finalCategory = complaintData.category || categoryResult.category;
          updateComplaintData({ 
            description: capturedInput,
            category: finalCategory,
            completedSteps: complaintData.completedSteps.filter(s => s !== 'description' && s !== 'description_confirm')
          });
          currentActiveStepRef.current = null;
          addMessage({
            type: 'bot',
            text: `${getRandomPhrase('acknowledge')}, description update kar diya.`,
            timestamp: new Date()
          });
          if (complaintData.summary && complaintData.location && complaintData.photo) {
            setTimeout(() => {
              goToStep('confirmation');
            }, 1000);
          } else {
            goToStep('description_confirm');
          }
          setIsProcessing(false);
          return;
        }
        
        if (capturedInput.length <= 50) {
          const categoryResult = inferCategory(capturedInput);
          updateComplaintData({ 
            summary: capturedInput,
            category: categoryResult.category,
            step: 'description',
            completedSteps: complaintData.completedSteps.filter(s => s !== 'summary')
          });
          currentActiveStepRef.current = null;
          addMessage({
            type: 'bot',
            text: `${getRandomPhrase('acknowledge')}, summary update kar diya.`,
            timestamp: new Date()
          });
          if (complaintData.description) {
            setTimeout(() => {
              goToStep('location');
            }, 1000);
          } else {
            goToStep('description');
          }
          setIsProcessing(false);
          return;
        }
        
        const categoryResult = inferCategory(capturedInput);
        const finalCategory = complaintData.category || categoryResult.category;
        updateComplaintData({ 
          description: capturedInput,
          category: finalCategory
        });
        addMessage({
          type: 'bot',
          text: `${getRandomPhrase('acknowledge')}, update kar diya.`,
          timestamp: new Date()
        });
        if (complaintData.summary && complaintData.location && complaintData.photo) {
          setTimeout(() => {
            goToStep('confirmation');
          }, 1000);
        } else {
          goToStep('description_confirm');
        }
        setIsProcessing(false);
        return;
      }
      
      const botResponse = getBotResponse(capturedInput);
      
      if (botResponse.needsDescriptionConfirmation) {
        setPendingDescription(botResponse.pendingDescriptionText);
        setWaitingForDescriptionConfirmation(true);
        // DIRECT TRANSITION: user input → description_confirm
        goToStep('description_confirm');
      }
      
      if (botResponse.needsLocationConfirmation && botResponse.updateData?.manualLocation) {
        updateComplaintData({ 
          manualLocation: botResponse.updateData.manualLocation,
          step: 'location-confirmation'
        });
        setWaitingForLocationConfirmation(true);
        // DIRECT TRANSITION: manual location → location_confirm
        goToStep('location_confirm');
      }
      
      if (botResponse.text && botResponse.text.trim()) {
        addMessage({
          type: 'bot',
          text: botResponse.text,
          timestamp: new Date()
        });
      }

      if (!botResponse.needsDescriptionConfirmation && !botResponse.needsLocationConfirmation && botResponse.updateData && Object.keys(botResponse.updateData).length > 0) {
        updateComplaintData(botResponse.updateData);
        if (botResponse.updateData.step) {
          // DIRECT TRANSITION: bot response → next step
          goToStep(botResponse.updateData.step);
        }
      }

      setIsProcessing(false);
    }, 500);
  };

  const getBotResponse = (userText) => {
    const currentStep = complaintData.step || 'summary';
    const completedSteps = new Set(complaintData.completedSteps || []);

    // Never treat confirmation-step input as complaint text (yes/haan handled in handleSend)
    if (currentStep === 'confirmation') {
      return { text: 'Bas haan bol dijiye, main aage badhata hoon.', updateData: {} };
    }
    // photo/voice: prompt only (skip/yes/no handled in handleSend)
    if (currentStep === 'photo') {
      return { text: "Photo bhej sakte hain, ya 'skip' likh dijiye.", updateData: {} };
    }
    if (currentStep === 'voice') {
      return { text: "Voice message bhejna chahte hain ya 'skip' likh dijiye.", updateData: {} };
    }
    
    // Step-based flow logic
    if (currentStep === 'summary' && !completedSteps.has('summary')) {
      const clarificationAsked = complaintData.clarificationAsked || false;
      
      if (clarificationAsked && !complaintData.category) {
        const categoryResult = inferCategory(userText);
        const finalCategory = categoryResult.category;
        
        return {
          text: 'Aap bataiye: samasya kya hai, kab se hai, kahan hai.',
          updateData: { 
            category: finalCategory,
            clarificationAsked: false,
            step: 'description',
            completedSteps: [...Array.from(completedSteps), 'summary']
          }
        };
      }
      
      const categoryResult = inferCategory(userText);
      
      if (categoryResult.isAmbiguous && !clarificationAsked && !complaintData.category) {
        const categoryOptions = categoryResult.matchedCategories.slice(0, 3).join(', ');
        return {
          text: `Ye issue in categories mein aa sakta hai: ${categoryOptions}. Aap bataiye kaunsi sahi lagti hai.`,
          updateData: { 
            summary: userText,
            clarificationAsked: true,
            step: 'summary'
          }
        };
      }
      
      const finalCategory = complaintData.category || categoryResult.category;
      const shortSummary = userText.length > 40 ? userText.substring(0, 40) + '...' : userText;
      return {
        text: `${getRandomPhrase('understood')}. ${shortSummary}.\nAb zara detail mein bataiye: kab se hai, kahan hai.`,
        updateData: { 
          summary: userText,
          category: finalCategory,
          clarificationAsked: false,
          step: 'description',
          completedSteps: [...Array.from(completedSteps), 'summary']
        }
      };
    } else if (currentStep === 'description' && !completedSteps.has('description')) {
      return {
        text: `Aap keh rahe hain ki: ${userText}\n\nKya yeh sahi hai?`,
        updateData: {},
        needsDescriptionConfirmation: true,
        pendingDescriptionText: userText
      };
    }

    if (!complaintData.summary) {
      return {
        text: 'Aap bataiye kya dikkat hai.',
        updateData: { step: 'summary' }
      };
    }
    if (!complaintData.description) {
      return {
        text: 'Zara aur detail mein bataiye, kya ho raha hai.',
        updateData: { step: 'description' }
      };
    }
    
    // Check if user is providing location manually
    const lowerText = userText.toLowerCase();
    const locationKeywords = ['nagar', 'colony', 'road', 'street', 'karera', 'kolaras', 'pohri', 'shivpuri', 'mein hoon', 'yahan', 'idhar', 'area'];
    const isManualLocation = locationKeywords.some(keyword => lowerText.includes(keyword)) && 
                             (userText.length < 100);
    
    if (isManualLocation && !complaintData.location) {
      const locationName = userText.trim();
      return {
        text: `${getRandomPhrase('acknowledge')}, ${locationName} ki problem hai na?`,
        updateData: { 
          step: 'location-confirmation',
          manualLocation: locationName
        },
        needsLocationConfirmation: true
      };
    }
    
    return {
      text: '',
      updateData: { step: 'location' },
      nextStep: '/location'
    };
  };

  // Rule-based category inference
  const inferCategory = (text) => {
    const lowerText = text.toLowerCase();
    const matchedCategories = [];
    
    if (lowerText.match(/\b(road|sadak|gaddha|pothole|bridge|nala|drain|street|path|way)\b/)) {
      matchedCategories.push('infrastructure');
    }
    if (lowerText.match(/\b(water|paani|tap|pipeline|supply|leak|shortage|nalka)\b/)) {
      matchedCategories.push('water');
    }
    if (lowerText.match(/\b(electricity|bijli|power|light|wire|pole|transformer|connection)\b/)) {
      matchedCategories.push('electricity');
    }
    if (lowerText.match(/\b(toilet|swachh|garbage|waste|kooda|clean|dirty|sanitation)\b/)) {
      matchedCategories.push('sanitation');
    }
    if (lowerText.match(/\b(health|hospital|doctor|medicine|treatment|disease|illness)\b/)) {
      matchedCategories.push('health');
    }
    if (lowerText.match(/\b(school|education|padhai|teacher|student|book|exam)\b/)) {
      matchedCategories.push('education');
    }
    
    if (matchedCategories.length === 0) {
      return { category: 'general', isAmbiguous: false, matchedCategories: [] };
    } else if (matchedCategories.length === 1) {
      return { category: matchedCategories[0], isAmbiguous: false, matchedCategories };
    } else {
      return { category: matchedCategories[0], isAmbiguous: true, matchedCategories };
    }
  };

  const reverseGeocode = async (latitude, longitude) => {
    try {
      const response = await fetch(
        `https://nominatim.openstreetmap.org/reverse?format=json&lat=${latitude}&lon=${longitude}`,
        {
          headers: {
            'User-Agent': 'AI Neta Complaint App'
          }
        }
      );
      
      if (!response.ok) {
        return null;
      }
      
      const data = await response.json();
      const locationName = 
        data.address?.city || 
        data.address?.town || 
        data.address?.village || 
        data.address?.suburb ||
        data.address?.county ||
        null;
      
      return locationName;
    } catch (error) {
      console.warn('Reverse geocoding failed:', error);
      return null;
    }
  };

  const showLocationConfirmationMessage = async () => {
    let confirmationText = 'Aap yahin ke area mein hain na?';
    
    if (complaintData.manualLocation) {
      confirmationText = `${getRandomPhrase('acknowledge')}, ${complaintData.manualLocation} ki problem hai na?`;
    } else if (complaintData.location && complaintData.location.latitude && complaintData.location.longitude) {
      const locationName = await reverseGeocode(
        complaintData.location.latitude,
        complaintData.location.longitude
      );
      if (locationName) {
        confirmationText = `Aap abhi ${locationName} mein hain, sahi?`;
      }
    }
    
    addMessage({
      type: 'bot',
      text: confirmationText,
      timestamp: new Date()
    });
    
    setWaitingForLocationConfirmation(true);
  };

  const showConfirmationMessage = () => {
    const summary = complaintData.summary || '';
    const description = complaintData.description || '';
    const complaintText = [summary, description].filter(Boolean).join('\n\n');
    
    const confirmationText = `Maine yeh samjha hai:

${complaintText}

Kya yeh sahi hai?`;

    addMessage({
      type: 'bot',
      text: confirmationText,
      timestamp: new Date()
    });
    
    setWaitingForConfirmation(true);
    setStage('confirming');
    updateComplaintData({ step: 'confirmation' });
  };

  const submitComplaint = async () => {
    if (!complaintData.summary || !complaintData.description) {
      addMessage({
        type: 'bot',
        text: 'Aap bataiye kya dikkat hai aur kahan hai.',
        timestamp: new Date()
      });
      return;
    }

    if (!complaintData.location) {
      addMessage({
        type: 'bot',
        text: 'Location chahiye. Wapas jaake location share karein.',
        timestamp: new Date()
      });
      return;
    }

    if (!complaintData.photo || (!complaintData.photo.blob && !complaintData.photo.url)) {
      addMessage({
        type: 'bot',
        text: 'Agar possible ho toh ek photo bhej dijiye, isse madad milegi.',
        timestamp: new Date()
      });
      addMessage({
        type: 'bot',
        text: 'Nahi ho paye toh skip karke aage badh sakte hain.',
        timestamp: new Date()
      });
      return;
    }

    const userID = localStorage.getItem('user_id');
    if (!userID) {
      addMessage({
        type: 'bot',
        text: 'Phone verify zaroori hai.',
        timestamp: new Date()
      });
      navigate('/phone-verify');
      return;
    }

    setIsSubmitting(true);
    
    addMessage({
      type: 'bot',
      text: 'Main ise sahi jagah tak pahucha raha hoon.',
      timestamp: new Date()
    });
    
    addMessage({
      type: 'bot',
      text: 'Main ise register kar raha hoon. Thoda wait karein…',
      timestamp: new Date()
    });

    try {
      const finalCategory = ensureCategory(
        complaintData.category,
        complaintData.summary || complaintData.description || ''
      );
      
      const complaintDataWithCategory = {
        ...complaintData,
        category: finalCategory
      };
      
      if (import.meta.env.DEV) {
        console.log('Submitting complaint with photo:', {
          hasBlob: !!complaintData.photo.blob,
          hasUrl: !!complaintData.photo.url,
          category: finalCategory,
          photo: complaintData.photo
        });
      }

      const response = await api.createComplaint(complaintDataWithCategory);
      
      const complaintNumber = response.complaint_number || response.complaint_id;
      const idDisplay = complaintNumber || response.complaint_id || '—';
      
      addMessage({
        type: 'bot',
        text: `✅ Aapki complaint register ho gayi hai (ID: ${idDisplay})\nRelevant department ko bhej di gayi hai.`,
        timestamp: new Date()
      });
      
      setStage('completed');
      clearComplaintData({ keepConversation: true });
      setIsSubmitting(false);
      
    } catch (err) {
      let errorMessage = 'Thoda issue aa gaya. Main dobara try kar raha hoon.';
      
      if (err instanceof ApiError) {
        if (err.status === 400) {
          errorMessage = err.message || 'Kuch data sahi nahi lag raha. Ek baar check karein.';
        } else if (err.status === 401) {
          errorMessage = err.message || 'Phone verify karein.';
          localStorage.removeItem('auth_token');
          localStorage.removeItem('phone_verified');
          navigate('/phone-verify');
        } else if (err.status === 0) {
          if (err.code === 'CORS_ERROR') {
            errorMessage = 'Backend CORS error. Connection check karein.';
          } else if (err.code === 'TIMEOUT') {
            errorMessage = 'Request timeout. Phir se try karein.';
          } else {
            const isOffline = !navigator.onLine;
            if (isOffline) {
              errorMessage = 'Aap offline hain. Complaint local save ho gayi, online aate hi submit ho jayegi.';
              saveToQueue(complaintData);
            } else {
              errorMessage = 'Request fail. Connection check karein.';
            }
          }
        } else {
          errorMessage = err.message || `Error: ${err.status}. Phir se try karein.`;
        }
      } else {
        errorMessage = err.message || errorMessage;
      }

      addMessage({
        type: 'bot',
        text: errorMessage,
        timestamp: new Date()
      });
      setIsSubmitting(false);
      setWaitingForConfirmation(true);
      setStage('confirming');
    }
  };

  const processingLabel = () => {
    const step = complaintData.step || 'summary';
    if (step === 'location') return 'Location dekh raha hoon…';
    if (step === 'description' || step === 'summary') return 'Samajh raha hoon…';
    if (isSubmitting) return 'Register ho raha hai…';
    return 'Dekh raha hoon…';
  };

  if (!phoneVerified) {
    return null;
  }

  return (
    <div className="chat-screen">
      <div className="chat-shell">
        <div className="chat-avatar-placeholder" aria-hidden="true" />
        <div className="chat-main">
          <div className="chat-header">
            <button type="button" className="chat-back-button" onClick={() => navigate('/')}>
              Back
            </button>
            <h2 className="chat-title">AI Neta</h2>
          </div>

          <MessageList messages={complaintData.conversation} isTyping={isTyping} />

          {isProcessing && (
            <div className="chat-processing-indicator" role="status">
              {processingLabel()}
            </div>
          )}

          <ChatInput
            value={inputText}
            onChange={setInputText}
            onSend={handleSend}
            disabled={isProcessing || isSubmitting}
            placeholder="Type your message..."
          />
        </div>
      </div>
    </div>
  );
}

export default ChatScreen;
