import { create } from 'zustand';
import api from '../services/api';
import DepartmentRouter from '../services/departmentRouter';
import { ensureCategory } from '../utils/categoryInference';
import { getRandomPhrase } from '../utils/botPhrases';

const initialMessage = {
  id: '1',
  text: 'Namaste! Main aapka AI NETA hoon. Aap bataiye, kya dikkat hai?',
  sender: 'ai',
  timestamp: new Date(),
  step: 'problem'
};

export const useChatStore = create((set, get) => ({
  messages: [initialMessage],
  currentStep: 'problem',
  complaintData: {
    problem: '',
    location: null,
    photo: null,
    voiceNote: null,
    voiceBlob: null,
    address: '',
    category: '',
    selectedDepartment: null
  },
  isRecording: false,
  isProcessing: false,

  sendMessage: async (text) => {
    const userMessage = {
      id: Date.now().toString(),
      text,
      sender: 'user',
      timestamp: new Date()
    };

    set((state) => ({
      messages: [...state.messages, userMessage],
      complaintData: { ...state.complaintData, problem: text }
    }));

    const { currentStep } = get();
    let aiResponse = '';

    switch (currentStep) {
      case 'problem':
        aiResponse = `${getRandomPhrase('understood')}. Ab bataiye kahan ki problem hai?`;
        set({ currentStep: 'location' });
        break;
      case 'location':
        aiResponse = 'Aap yahin ke rehne wale hain na? Location share kar dijiye.';
        set({ currentStep: 'photo' });
        break;
      case 'photo':
        aiResponse = 'Bahut accha photo mila. Kya aap voice me bhi batana chahenge?';
        set({ currentStep: 'voice' });
        break;
      case 'voice':
        aiResponse = 'Dhanyavaad! Main ab ise sahi adhikari tak pahucha raha hoon. Ek minute.';
        set({ currentStep: 'processing' });
        setTimeout(() => {
          get().submitComplaint();
        }, 2000);
        break;
      default:
        aiResponse = 'Main aapki complaint dekh raha hoon.';
    }

    const aiMessage = {
      id: (Date.now() + 1).toString(),
      text: aiResponse,
      sender: 'ai',
      timestamp: new Date()
    };

    set((state) => ({
      messages: [...state.messages, aiMessage]
    }));
  },

  setLocation: (location, address) => {
    set((state) => ({
      complaintData: { ...state.complaintData, location, address }
    }));

    const locationMessage = {
      id: Date.now().toString(),
      text: `📍 ${address || 'Yahin se'}`,
      sender: 'user',
      timestamp: new Date(),
      type: 'location'
    };

    set((state) => ({
      messages: [...state.messages, locationMessage],
      currentStep: 'photo'
    }));

    setTimeout(() => {
      const aiMessage = {
        id: (Date.now() + 1).toString(),
        text: 'Achha, location mil gayi. Ab agar possible ho toh ek photo bhej dijiye.',
        sender: 'ai',
        timestamp: new Date()
      };
      set((state) => ({
        messages: [...state.messages, aiMessage]
      }));
    }, 500);
  },

  setSelectedDepartment: (dept) => {
    if (!dept || !dept.id) return;
    set((state) => ({
      complaintData: { ...state.complaintData, selectedDepartment: dept.id }
    }));
    const deptMessage = {
      id: Date.now().toString(),
      text: `🏛️ Department: ${dept.name}`,
      sender: 'user',
      timestamp: new Date(),
      type: 'department'
    };
    // Department selection removed from UI - backend will auto-assign based on category
    // Keep this function for backward compatibility but don't show UI
    set((state) => ({
      currentStep: 'photo'
    }));
  },

  uploadPhoto: async (photoBlob) => {
    set({ isProcessing: true });

    try {
      const result = await api.uploadPhoto(photoBlob);
      const photoUrl = result.url || result.file_path || result.fileUrl;

      set((state) => ({
        complaintData: { ...state.complaintData, photo: photoUrl }
      }));

      const photoMessage = {
        id: Date.now().toString(),
        text: '📸 Photo bhej diya',
        sender: 'user',
        timestamp: new Date(),
        type: 'photo',
        image: photoUrl
      };

      set((state) => ({
        messages: [...state.messages, photoMessage],
        currentStep: 'voice'
      }));

      setTimeout(() => {
        const aiMessage = {
          id: (Date.now() + 1).toString(),
          text: 'Photo mil gayi, dhanyavaad! Ab agar chahe toh voice me bhi bata sakte hain.',
          sender: 'ai',
          timestamp: new Date()
        };
        set((state) => ({
          messages: [...state.messages, aiMessage]
        }));
      }, 500);
    } catch (error) {
      console.error('Photo upload failed:', error);
    } finally {
      set({ isProcessing: false });
    }
  },

  uploadVoiceNote: async (audioBlob) => {
    set({ isProcessing: true });

    try {
      set((state) => ({
        complaintData: { ...state.complaintData, voiceNote: true, voiceBlob: audioBlob }
      }));

      const voiceMessage = {
        id: Date.now().toString(),
        text: '🎤 Voice message bhej diya',
        sender: 'user',
        timestamp: new Date(),
        type: 'voice'
      };

      set((state) => ({
        messages: [...state.messages, voiceMessage],
        currentStep: 'processing'
      }));

      setTimeout(() => {
        get().submitComplaint();
      }, 1000);
    } catch (error) {
      console.error('Voice note failed:', error);
    } finally {
      set({ isProcessing: false });
    }
  },

  submitComplaint: async () => {
    if (get().isProcessing) return;
    set({ isProcessing: true });

    try {
      const { complaintData } = get();

      // Ensure category is ALWAYS set (infer from problem text if missing)
      const finalCategory = ensureCategory(complaintData.category, complaintData.problem);
      
      // Update complaintData with inferred category if it was missing
      if (!complaintData.category || complaintData.category.trim() === '') {
        set((state) => ({
          complaintData: { ...state.complaintData, category: finalCategory }
        }));
      }

      const seriousKeywords = /बहुत गंभीर|जान का खतरा|urgent|critical|गंभीर|खतरा/i;
      const isSerious = complaintData.problem && seriousKeywords.test(complaintData.problem);
      const complaintForRouter = {
        ...complaintData,
        category: finalCategory, // Use ensured category
        severity: complaintData.severity || (isSerious ? 'high' : 'normal'),
        escalationLevel: complaintData.escalationLevel || (isSerious ? 2 : 0)
      };
      const recipients = DepartmentRouter.getRecipients(complaintForRouter);
      if (import.meta.env.DEV) {
        console.log('📧 [Department Routing] Recipients:', recipients);
        console.log('📍 Area:', complaintData.location?.area, '| Dept:', complaintData.selectedDepartment, '| Category:', finalCategory, '| Severity:', complaintForRouter.severity);
      }

      // Show message if category is "general"
      if (finalCategory === 'general') {
        const generalMessage = {
          id: (Date.now() - 1).toString(),
          text: 'Main ise uchit adhikari tak pahucha raha hoon.',
          sender: 'ai',
          timestamp: new Date()
        };
        set((state) => ({
          messages: [...state.messages, generalMessage]
        }));
      }

      const payload = {
        summary: complaintData.problem,
        description: complaintData.address || complaintData.problem,
        category: finalCategory, // ALWAYS set (never null/empty)
        location: complaintData.location
          ? { latitude: complaintData.location.lat, longitude: complaintData.location.lng }
          : null,
        photo: complaintData.photo ? { url: complaintData.photo } : null,
        urgency: 'medium',
        notifyEmails: recipients
      };

      const response = await api.createComplaint(payload);
      if (import.meta.env.DEV) {
        console.log('[submitComplaint] API success, response:', response);
      }
      const data = response?.data != null ? response.data : response;
      const complaintNumber = data?.complaint_number ?? data?.complaint_id ?? data?.id ?? '—';
      const complaintId = data?.complaint_id ?? data?.id ?? null;
      const voiceBlobToUpload = get().complaintData.voiceBlob || null;

      const successMessageText = complaintNumber && complaintNumber !== '—'
        ? `Ho gaya.\n\nMaine aapki problem sahi adhikari tak pahucha di hai.\n\nAapko updates milte rahenge.\n\nComplaint ID: ${complaintNumber}`
        : `Ho gaya.\n\nMaine aapki problem sahi adhikari tak pahucha di hai.\n\nAapko updates milte rahenge.`;
      
      const successMessage = {
        id: Date.now().toString(),
        text: successMessageText,
        sender: 'ai',
        timestamp: new Date()
      };

      set((state) => ({
        messages: [...state.messages, successMessage],
        currentStep: 'completed',
        complaintData: { ...state.complaintData, complaintNumber, voiceBlob: null }
      }));

      if (complaintNumber !== '—') {
        localStorage.setItem('lastComplaint', String(complaintNumber));
      }

      if (voiceBlobToUpload && complaintId) {
        api.uploadVoice(complaintId, voiceBlobToUpload).then(() => {
          set((state) => ({
            messages: [
              ...state.messages,
              {
                id: Date.now().toString(),
                text: 'Voice message bhi attach ho gaya.',
                sender: 'ai',
                timestamp: new Date()
              }
            ]
          }));
        }).catch(() => {});
      }
    } catch (error) {
      if (import.meta.env.DEV) {
        console.error('[submitComplaint] API failed (show error only on real failure):', error?.message ?? error, error);
      }

      const errorMessage = {
        id: Date.now().toString(),
        text: 'Maaf kijiye, complaint submit nahi ho payi. Thoda wait karke phir se try karein.',
        sender: 'ai',
        timestamp: new Date()
      };

      set((state) => ({
        messages: [...state.messages, errorMessage]
      }));
    } finally {
      set({ isProcessing: false });
    }
  },

  resetChat: () => {
    set({
      messages: [{ ...initialMessage, timestamp: new Date() }],
      currentStep: 'problem',
      complaintData: {
        problem: '',
        location: null,
        photo: null,
        voiceNote: null,
        voiceBlob: null,
        address: '',
        category: '',
        selectedDepartment: null
      }
    });
  }
}));
