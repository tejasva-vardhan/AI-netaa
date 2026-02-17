import { create } from 'zustand';
import api from '../services/api';
import DepartmentRouter from '../services/departmentRouter';

const initialMessage = {
  id: '1',
  text: 'Namaste! Main aapka AI NETA hoon. Kya problem hai? Aap mujhe bata sakte hain.',
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
        aiResponse = 'Dhanyavaad! Kahan ki problem hai? Kripya location share karein.';
        set({ currentStep: 'location' });
        break;
      case 'location':
        aiResponse = 'Kripya complaint ke liye department chuniye. Neeche list se select karein.';
        set({ currentStep: 'department' });
        break;
      case 'department':
        aiResponse = 'Dhanyavaad! Ab kya aap problem ki photo bhej sakte hain?';
        set({ currentStep: 'photo' });
        break;
      case 'photo':
        aiResponse = 'Bahut accha. Kya aap voice me batana chahenge? Aap record kar sakte hain.';
        set({ currentStep: 'voice' });
        break;
      case 'voice':
        aiResponse = 'Dhanyavaad! Main aapki complaint process kar raha hoon. Ek minute.';
        set({ currentStep: 'processing' });
        setTimeout(() => {
          get().submitComplaint();
        }, 2000);
        break;
      default:
        aiResponse = 'Main aapki complaint track kar raha hoon.';
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
      text: `📍 Location shared: ${address || 'Location captured'}`,
      sender: 'user',
      timestamp: new Date(),
      type: 'location'
    };

    set((state) => ({
      messages: [...state.messages, locationMessage],
      currentStep: 'department'
    }));

    setTimeout(() => {
      const aiMessage = {
        id: (Date.now() + 1).toString(),
        text: 'Location mil gayi. Ab neeche se department chuniye (विभाग चुनें).',
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
    set((state) => ({
      messages: [...state.messages, deptMessage],
      currentStep: 'photo'
    }));
    setTimeout(() => {
      const aiMessage = {
        id: (Date.now() + 1).toString(),
        text: 'Department select ho gaya. Ab photo bhejiye.',
        sender: 'ai',
        timestamp: new Date()
      };
      set((state) => ({
        messages: [...state.messages, aiMessage]
      }));
    }, 500);
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
        text: '📸 Photo captured',
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
          text: 'Photo mil gayi. Ab voice record karein ya text mein bata dein.',
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
        text: '🎤 Voice note recorded',
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

      const seriousKeywords = /बहुत गंभीर|जान का खतरा|urgent|critical|गंभीर|खतरा/i;
      const isSerious = complaintData.problem && seriousKeywords.test(complaintData.problem);
      const complaintForRouter = {
        ...complaintData,
        severity: complaintData.severity || (isSerious ? 'high' : 'normal'),
        escalationLevel: complaintData.escalationLevel || (isSerious ? 2 : 0)
      };
      const recipients = DepartmentRouter.getRecipients(complaintForRouter);
      if (import.meta.env.DEV) {
        console.log('📧 [Department Routing] Recipients:', recipients);
        console.log('📍 Area:', complaintData.location?.area, '| Dept:', complaintData.selectedDepartment, '| Severity:', complaintForRouter.severity);
      }

      const payload = {
        summary: complaintData.problem,
        description: complaintData.address || complaintData.problem,
        category: complaintData.category || '',
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

      const successMessage = {
        id: Date.now().toString(),
        text: `✅ Complaint #${complaintNumber} registered! Sent to ${recipients.length} department(s). Main ab ise track karunga.`,
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
                text: 'Voice note safely attached.',
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
        text: 'Complaint submit nahi ho payi. Kripya phir se koshish karein.',
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
