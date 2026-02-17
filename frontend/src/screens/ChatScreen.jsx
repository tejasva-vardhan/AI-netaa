import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useComplaintState } from '../state/ComplaintContext';
import api from '../services/api';
import MessageList from '../components/MessageList';
import ChatInput from '../components/ChatInput';
import './ChatScreen.css';

function ChatScreen() {
  const navigate = useNavigate();
  const { complaintData, updateComplaintData, addMessage, clearComplaintData } = useComplaintState();
  const [inputText, setInputText] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const messagesEndRef = useRef(null);

  // Chat-first: chat only after phone/OTP. Unauthenticated users go to phone-verify.
  const phoneVerified = localStorage.getItem('phone_verified') === 'true';
  useEffect(() => {
    if (!phoneVerified) {
      navigate('/phone-verify', { replace: true });
    }
  }, [navigate, phoneVerified]);

  useEffect(() => {
    // Guard: do not run step logic if not verified (we're redirecting)
    if (localStorage.getItem('phone_verified') !== 'true') return;

    // ISSUE 4: Prevent partial drafts from jumping ahead
    // Only navigate if current step matches AND previous steps are complete
    if (complaintData.step === 'location') {
      // Can only go to location if summary and description are complete
      if (complaintData.summary && complaintData.description) {
        navigate('/location');
        return;
      }
    } else if (complaintData.step === 'camera') {
      // Can only go to camera if location is complete
      if (complaintData.location) {
        navigate('/camera');
        return;
      }
    } else if (complaintData.step === 'phone-verify') {
      // Can only go to phone-verify if photo is complete
      if (complaintData.photo) {
        navigate('/phone-verify');
        return;
      }
    } else if (complaintData.step === 'review') {
      // Can only go to review if ALL steps are complete
      if (complaintData.summary && complaintData.description && complaintData.location && complaintData.photo) {
        navigate('/review');
        return;
      }
    }

    // Initialize conversation if empty
    if (complaintData.conversation.length === 0) {
      addMessage({
        type: 'bot',
        text: 'Namaste.\n\nBatayiye, kya samasya aa rahi hai. (Jaise: sadak par gaddha, paani ki samasya.)',
        timestamp: new Date()
      });
      updateComplaintData({ step: 'summary', completedSteps: [] });
    } else {
      // Resume conversation - check what step we're on and prompt accordingly
      const completedSteps = complaintData.completedSteps || [];
      
      // If summary exists but not marked complete, mark it complete
      if (complaintData.summary && !completedSteps.includes('summary')) {
        updateComplaintData({ 
          completedSteps: [...completedSteps, 'summary'],
          step: 'description'
        });
        // Only prompt if description is not yet provided
        if (!complaintData.description) {
          const hasDescriptionPrompt = complaintData.conversation.some(
            m => m.type === 'bot' && (m.text.includes('विस्तृत विवरण') || m.text.includes('samasya kya hai'))
          );
          if (!hasDescriptionPrompt) {
            addMessage({
              type: 'bot',
              text: 'Batayiye: samasya kya hai, kab se hai, kahan hai.',
              timestamp: new Date()
            });
          }
        }
      }
      
      // If description exists but not marked complete, mark it complete and navigate
      if (complaintData.description && !completedSteps.includes('description')) {
        updateComplaintData({ 
          completedSteps: [...completedSteps, 'description'],
          step: 'location'
        });
        // Auto-navigate to location
        setTimeout(() => {
          navigate('/location');
        }, 500);
        return;
      }
      
      // If we're on description step but haven't completed it, prompt
      if (complaintData.step === 'description' && !completedSteps.includes('description') && !complaintData.description) {
        const hasDescriptionPrompt = complaintData.conversation.some(
          m => m.type === 'bot' && (m.text.includes('विस्तृत विवरण') || m.text.includes('samasya kya hai'))
        );
        if (!hasDescriptionPrompt) {
          addMessage({
            type: 'bot',
            text: 'Batayiye: samasya kya hai, kab se hai, kahan hai.',
            timestamp: new Date()
          });
        }
      }
      
      // If we're on summary step but haven't completed it, prompt
      if (complaintData.step === 'summary' && !completedSteps.includes('summary') && !complaintData.summary) {
        const hasSummaryPrompt = complaintData.conversation.some(
          m => m.type === 'bot' && (m.text.includes('संक्षिप्त विवरण') || m.text.includes('kya samasya aa rahi hai'))
        );
        if (!hasSummaryPrompt) {
          addMessage({
            type: 'bot',
            text: 'Apni samasya ka sankshipt vivaran bataiye.',
            timestamp: new Date()
          });
        }
      }
    }
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [complaintData.conversation]);

  const handleSend = async () => {
    if (!inputText.trim() || isProcessing) return;

    const raw = inputText.trim();
    const normalized = raw.toLowerCase();

    // Hidden reset: user says "restart" -> clear chat state and draft, auth unchanged
    if (normalized === 'restart') {
      setInputText('');
      setIsProcessing(true);
      try {
        await api.resetChatDraft();
      } catch (_) {
        // Proceed with local clear even if backend fails (e.g. offline)
      }
      clearComplaintData();
      addMessage({
        type: 'bot',
        text: 'Namaste.\n\nBatayiye, kya samasya aa rahi hai. (Jaise: sadak par gaddha, paani ki samasya.)',
        timestamp: new Date()
      });
      updateComplaintData({ step: 'summary', completedSteps: [] });
      setIsProcessing(false);
      return;
    }

    const userMessage = {
      type: 'user',
      text: raw,
      timestamp: new Date()
    };

    addMessage(userMessage);
    setInputText('');
    setIsProcessing(true);

    // Bot response (local flow; persistence is async via context debounce)
    setTimeout(() => {
      const botResponse = getBotResponse(inputText);
      
      // Only add message if there's actual text
      if (botResponse.text && botResponse.text.trim()) {
        addMessage({
          type: 'bot',
          text: botResponse.text,
          timestamp: new Date()
        });
      }

      // Update complaint data based on conversation
      if (botResponse.updateData && Object.keys(botResponse.updateData).length > 0) {
        updateComplaintData(botResponse.updateData);
      }

      // Navigate to next step if needed
      if (botResponse.nextStep) {
        setTimeout(() => {
          navigate(botResponse.nextStep);
        }, 1500);
      } else if (!botResponse.text || botResponse.text.trim() === '') {
        // Empty response means we should navigate but already have the message
        // Check if we should auto-navigate
        const updatedStep = botResponse.updateData?.step;
        if (updatedStep === 'location') {
          setTimeout(() => {
            navigate('/location');
          }, 500);
        }
      }

      setIsProcessing(false);
    }, 500);
  };

  const getBotResponse = (userText) => {
    const currentStep = complaintData.step || 'summary';
    const completedSteps = complaintData.completedSteps || [];
    
    // Prevent duplicate processing - if step is already completed, don't process again
    if (currentStep === 'summary' && completedSteps.includes('summary')) {
      // Summary already completed but user sent another message - ignore or move forward
      if (!complaintData.description) {
        // Already asked for description, don't repeat
        return {
          text: '',
          updateData: {}
        };
      }
    }
    
    if (currentStep === 'description' && completedSteps.includes('description')) {
      // Description already completed - should have navigated already
      return {
        text: '',
        updateData: {},
        nextStep: '/location'
      };
    }
    
    // Step-based flow logic
    if (currentStep === 'summary' && !completedSteps.includes('summary')) {
      const clarificationAsked = complaintData.clarificationAsked || false;
      
      // SAFETY CHECK 1: If clarification was asked, user is responding to it - lock category
      if (clarificationAsked && !complaintData.category) {
        // User responded to clarification - try to match their response to a category
        const categoryResult = inferCategory(userText);
        const finalCategory = categoryResult.category; // Use inferred category from response
        
        return {
          text: 'Batayiye: samasya kya hai, kab se hai, kahan hai.',
          updateData: { 
            category: finalCategory,
            clarificationAsked: false,
            step: 'description',
            completedSteps: [...completedSteps, 'summary']
          }
        };
      }
      
      // Infer category from summary text
      const categoryResult = inferCategory(userText);
      
      // SAFETY CHECK 1: If ambiguous and clarification not yet asked, ask ONE clarification question
      if (categoryResult.isAmbiguous && !clarificationAsked && !complaintData.category) {
        const categoryOptions = categoryResult.matchedCategories.slice(0, 3).join(', '); // Limit to 3 options
        return {
          text: `Ye issue in categories mein aa sakta hai: ${categoryOptions}. Sabse upyukt category bataiye.`,
          updateData: { 
            summary: userText,
            clarificationAsked: true,
            step: 'summary'
          }
        };
      }
      
      // Category is determined (either unambiguous or after clarification) - proceed
      const finalCategory = complaintData.category || categoryResult.category;
      
      // User provided summary - mark as completed
      return {
        text: 'Batayiye: samasya kya hai, kab se hai, kahan hai.',
        updateData: { 
          summary: userText,
          category: finalCategory,
          clarificationAsked: false,
          step: 'description',
          completedSteps: [...completedSteps, 'summary']
        }
      };
    } else if (currentStep === 'description' && !completedSteps.includes('description')) {
      const categoryResult = inferCategory(userText);
      const finalCategory = complaintData.category || categoryResult.category;
      
      return {
        text: 'Location chahiye. Ab location screen par ja rahe hain.',
        updateData: { 
          description: userText,
          category: finalCategory,
          step: 'location',
          completedSteps: [...completedSteps, 'description']
        },
        nextStep: '/location'
      };
    }

    if (!complaintData.summary) {
      return {
        text: 'Apni samasya ka sankshipt vivaran bataiye.',
        updateData: { step: 'summary' }
      };
    }
    if (!complaintData.description) {
      return {
        text: 'Zara aur detail mein bataiye.',
        updateData: { step: 'description' }
      };
    }
    
    // Both summary and description exist - move to location
    return {
      text: '',
      updateData: { step: 'location' },
      nextStep: '/location'
    };
  };

  // Rule-based category inference (no AI guessing)
  // Returns: { category: string, isAmbiguous: boolean, matchedCategories: string[] }
  const inferCategory = (text) => {
    const lowerText = text.toLowerCase();
    const matchedCategories = [];
    
    // Road/Infrastructure keywords
    if (lowerText.match(/\b(road|sadak|gaddha|pothole|bridge|nala|drain|street|path|way)\b/)) {
      matchedCategories.push('infrastructure');
    }
    
    // Water keywords
    if (lowerText.match(/\b(water|paani|tap|pipeline|supply|leak|shortage|nalka)\b/)) {
      matchedCategories.push('water');
    }
    
    // Electricity keywords
    if (lowerText.match(/\b(electricity|bijli|power|light|wire|pole|transformer|connection)\b/)) {
      matchedCategories.push('electricity');
    }
    
    // Sanitation keywords
    if (lowerText.match(/\b(toilet|swachh|garbage|waste|kooda|clean|dirty|sanitation)\b/)) {
      matchedCategories.push('sanitation');
    }
    
    // Health keywords
    if (lowerText.match(/\b(health|hospital|doctor|medicine|treatment|disease|illness)\b/)) {
      matchedCategories.push('health');
    }
    
    // Education keywords
    if (lowerText.match(/\b(school|education|padhai|teacher|student|book|exam)\b/)) {
      matchedCategories.push('education');
    }
    
    // Determine if ambiguous (multiple matches) or default
    if (matchedCategories.length === 0) {
      return { category: 'general', isAmbiguous: false, matchedCategories: [] };
    } else if (matchedCategories.length === 1) {
      return { category: matchedCategories[0], isAmbiguous: false, matchedCategories };
    } else {
      // Multiple matches = ambiguous, default to first match but mark as ambiguous
      return { category: matchedCategories[0], isAmbiguous: true, matchedCategories };
    }
  };

  const handleVoiceClick = () => {
    // Placeholder for voice input
    alert('Voice input will be available soon. Please use text for now.');
  };

  const processingLabel = () => {
    const step = complaintData.step || 'summary';
    if (step === 'location') return 'Location confirm ho rahi hai…';
    if (step === 'description' || step === 'summary') return 'Details verify ho rahi hain…';
    return 'Reviewing…';
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

          <MessageList messages={complaintData.conversation} />

          {isProcessing && (
            <div className="chat-processing-indicator" role="status">
              {processingLabel()}
            </div>
          )}

          <ChatInput
            value={inputText}
            onChange={setInputText}
            onSend={handleSend}
            onVoiceClick={handleVoiceClick}
            disabled={isProcessing}
          />
        </div>
      </div>
    </div>
  );
}

export default ChatScreen;
