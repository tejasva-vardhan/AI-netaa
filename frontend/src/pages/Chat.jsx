import React, { useState, useRef, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { FaMicrophone, FaCamera, FaMapMarkerAlt, FaPaperPlane, FaStop } from 'react-icons/fa';
import toast from 'react-hot-toast';
import Avatar from '../components/Avatar';
import { useChatStore } from '../stores/chatStore';
import VoiceRecorder from '../components/VoiceRecorder';
import CameraCapture from '../components/CameraCapture';
import DepartmentRouter from '../services/departmentRouter';
import { getAreaFromAddress } from '../utils/locationArea';
import './Chat.css';

const Chat = () => {
  const [message, setMessage] = useState('');
  const [isRecording, setIsRecording] = useState(false);
  const [showCamera, setShowCamera] = useState(false);
  const messagesEndRef = useRef(null);
  const {
    messages,
    sendMessage,
    currentStep,
    complaintData,
    setLocation,
    uploadPhoto,
    uploadVoiceNote,
    resetChat,
    isProcessing
  } = useChatStore();

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  useEffect(() => {
    if (import.meta.env.DEV && typeof window !== 'undefined') {
      window.testRecipients = function () {
        const state = useChatStore.getState();
        const complaint = {
          ...state.complaintData,
          severity: state.complaintData.severity || 'normal',
          escalationLevel: state.complaintData.escalationLevel || 0
        };
        const recipients = DepartmentRouter.getRecipients(complaint);
        const primary = 'aineta502@gmail.com';
        const sdmEmails = [
          'sdoshiv@gmail.com',
          'sdmkolaras2013@gmail.com',
          'ropohari24@gmail.com',
          'sdmkarera13@gmail.com',
          'sdmpichhore@gmail.com'
        ];
        const out = {
          recipients,
          area: state.complaintData.location?.area,
          department: state.complaintData.selectedDepartment,
          problem: state.complaintData.problem,
          includesSDM: recipients.some((r) => sdmEmails.includes(r)),
          includesCollector: recipients.includes('dmshivpuri@nic.in'),
          includesRuralEngg: recipients.includes('eeresshivpuri-mp@nic.in')
        };
        console.log('📧 testRecipients() result:', JSON.stringify(out, null, 2));
        return out;
      };

      window.runAllRoutingTests = function () {
        const PRIMARY = 'aineta502@gmail.com';
        const tests = [
          {
            name: 'TEST 1: Shivpuri City (No SDM)',
            complaint: { problem: 'सड़क टूटी हुई है', location: { area: 'shivpuri' }, selectedDepartment: 'pwd' },
            expect: { area: 'shivpuri', dept: 'pwd', includesSDM: false, includesCollector: false, mustInclude: [PRIMARY, 'eepwdshivpuri@mp.nic.in'] }
          },
          {
            name: 'TEST 2: Kolaras (SDM + Rural Engg)',
            complaint: { problem: 'बिजली नहीं है', location: { area: 'kolaras' }, selectedDepartment: 'electricity' },
            expect: { area: 'kolaras', includesSDM: true, includesRuralEngg: true, mustInclude: [PRIMARY, 'seshivpuri.cz@mp.gov.in', 'sdmkolaras2013@gmail.com', 'eeresshivpuri-mp@nic.in'] }
          },
          {
            name: 'TEST 3: Pohri (SDM Pohri + Water)',
            complaint: { problem: 'पानी की समस्या', location: { area: 'pohri' }, selectedDepartment: 'water' },
            expect: { mustInclude: [PRIMARY, 'ropohari24@gmail.com', 'eeresshivpuri-mp@nic.in', 'eewrdshivpuri@nic.in'] }
          },
          {
            name: 'TEST 4: Serious keywords (Collector)',
            complaint: { problem: 'बहुत गंभीर समस्या - सड़क टूटी', location: { area: 'shivpuri' }, selectedDepartment: 'pwd', severity: 'high', escalationLevel: 2 },
            expect: { includesCollector: true, mustInclude: [PRIMARY, 'dmshivpuri@nic.in'] }
          },
          {
            name: 'TEST 6: Multiple keywords (Kolaras)',
            complaint: { problem: 'बिजली और सड़क दोनों खराब', location: { area: 'kolaras' }, selectedDepartment: 'pwd' },
            expect: { mustInclude: [PRIMARY, 'seshivpuri.cz@mp.gov.in', 'eepwdshivpuri@mp.nic.in', 'sdmkolaras2013@gmail.com', 'eeresshivpuri-mp@nic.in'] }
          },
          {
            name: 'TEST 7: No department selected',
            complaint: { problem: 'सड़क टूटी', location: { area: 'kolaras' }, selectedDepartment: null },
            expect: { mustInclude: [PRIMARY], includesSDM: true }
          }
        ];
        let passed = 0;
        let failed = 0;
        tests.forEach((t) => {
          const complaint = { ...t.complaint, severity: t.complaint.severity || 'normal', escalationLevel: t.complaint.escalationLevel ?? 0 };
          const recipients = DepartmentRouter.getRecipients(complaint);
          const sdmEmails = ['sdoshiv@gmail.com', 'sdmkolaras2013@gmail.com', 'ropohari24@gmail.com', 'sdmkarera13@gmail.com', 'sdmpichhore@gmail.com'];
          const includesSDM = recipients.some((r) => sdmEmails.includes(r));
          const includesCollector = recipients.includes('dmshivpuri@nic.in');
          const includesRuralEngg = recipients.includes('eeresshivpuri-mp@nic.in');
          const mustInclude = (t.expect.mustInclude || []).every((email) => recipients.includes(email));
          const sdmOk = t.expect.includesSDM === undefined || t.expect.includesSDM === includesSDM;
          const collectorOk = t.expect.includesCollector === undefined || t.expect.includesCollector === includesCollector;
          const ruralOk = t.expect.includesRuralEngg === undefined || t.expect.includesRuralEngg === includesRuralEngg;
          const ok = mustInclude && sdmOk && collectorOk && ruralOk;
          if (ok) {
            passed++;
            console.log(`✅ ${t.name}`);
          } else {
            failed++;
            console.log(`❌ ${t.name}`);
            console.log('   Expected:', t.expect);
            console.log('   Got recipients:', recipients);
            console.log('   includesSDM:', includesSDM, 'includesCollector:', includesCollector, 'includesRuralEngg:', includesRuralEngg);
          }
        });
        console.log(`\n📊 Result: ${passed} passed, ${failed} failed (total ${tests.length})`);
        return { passed, failed, total: tests.length };
      };
    }
  }, []);

  const handleSendMessage = async () => {
    if (!message.trim()) return;
    await sendMessage(message);
    setMessage('');
  };

  const handleLocationShare = () => {
    if (!navigator.geolocation) {
      toast.error('Geolocation not supported');
      return;
    }
    toast.loading('Getting your location...');
    navigator.geolocation.getCurrentPosition(
      async (position) => {
        try {
          const { latitude, longitude } = position.coords;
          const res = await fetch(
            `https://nominatim.openstreetmap.org/reverse?format=json&lat=${latitude}&lon=${longitude}`
          );
          const data = await res.json();
          const address = data.display_name || 'Yahin se';
          const area = getAreaFromAddress(address);
          useChatStore.getState().setLocation({ lat: latitude, lng: longitude, area }, address);
          toast.dismiss();
          toast.success('Location mil gaya!');
        } catch (err) {
          toast.dismiss();
          useChatStore.getState().setLocation(
            { lat: position.coords.latitude, lng: position.coords.longitude, area: undefined },
            'Yahin se'
          );
          toast.success('Location mil gaya!');
        }
      },
      () => {
        toast.dismiss();
        toast.error('Could not get location');
      }
    );
  };

  const handleNewComplaint = () => {
    resetChat();
    setMessage('');
  };

  const handleVoiceRecord = async (audioBlob) => {
    await uploadVoiceNote(audioBlob);
  };

  const handlePhotoCapture = async (photoBlob) => {
    await uploadPhoto(photoBlob);
    setShowCamera(false);
  };

  const showInput = currentStep !== 'completed' && currentStep !== 'processing';
  const hasLocation = complaintData?.location && complaintData?.address;
  const showActionAddress = (currentStep === 'processing' || currentStep === 'completed') && complaintData?.address;

  return (
    <div className="min-h-screen bg-gradient-to-b from-gray-50 to-white pt-20">
      <div className="container mx-auto px-4 max-w-4xl">
        <Avatar
          isSpeaking={currentStep === 'processing'}
          onTalkClick={handleNewComplaint}
        />

        <div className="chat-messages-box">
          <AnimatePresence>
            {messages.map((msg) => (
              <motion.div
                key={msg.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0 }}
                className={`chat-msg ${msg.sender === 'user' ? 'chat-msg--user' : 'chat-msg--ai'}`}
              >
                <div className={msg.sender === 'user' ? 'user-message' : 'ai-message'}>
                  {msg.type === 'photo' && msg.image && (
                    <img src={msg.image} alt="Complaint" className="rounded-lg max-h-32 mb-2 block" />
                  )}
                  {msg.text}
                </div>
              </motion.div>
            ))}
          </AnimatePresence>

          {/* Location verified block */}
          {hasLocation && currentStep !== 'problem' && currentStep !== 'location' && (
            <div className="chat-location-verified">
              <span className="chat-location-pin">📍 {complaintData.address}</span>
              {complaintData.location?.lat != null && (
                <a
                  href={`https://www.openstreetmap.org/?mlat=${complaintData.location.lat}&mlon=${complaintData.location.lng}&zoom=16`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="chat-map-link"
                >
                  View on map
                </a>
              )}
            </div>
          )}

          {/* Action will be taken at */}
          {showActionAddress && (
            <div className="chat-action-address">
              <strong>Yahin action liya jayega:</strong> {complaintData.address}
            </div>
          )}

          {isProcessing && (
            <div className="chat-msg chat-msg--ai">
              <div className="ai-message">Dekh raha hoon…</div>
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>

        {currentStep === 'completed' && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="mt-4 text-center"
          >
            <button
              type="button"
              onClick={handleNewComplaint}
              className="px-6 py-3 bg-netaji-green text-white rounded-full font-semibold hover:bg-green-700 transition"
            >
              Naya Complaint
            </button>
          </motion.div>
        )}

        {showInput && (
          <div className="chat-input-bar">
            <button
              type="button"
              onClick={() => setIsRecording(!isRecording)}
              className={`chat-input-btn ${isRecording ? 'chat-input-btn--recording' : ''}`}
              title="Voice"
            >
              {isRecording ? <FaStop /> : <FaMicrophone />}
            </button>
            <button
              type="button"
              onClick={() => setShowCamera(true)}
              className="chat-input-btn"
              title="Photo"
            >
              <FaCamera />
            </button>
            <button
              type="button"
              onClick={handleLocationShare}
              className="chat-input-btn"
              title="Share location"
            >
              <FaMapMarkerAlt />
            </button>
            <input
              type="text"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && handleSendMessage()}
              placeholder="Aap bataiye..."
              className="chat-input-field"
            />
            <button
              type="button"
              onClick={handleSendMessage}
              className="chat-input-btn chat-input-btn--send"
            >
              <FaPaperPlane />
            </button>
          </div>
        )}

        <AnimatePresence>
          {isRecording && (
            <VoiceRecorder onClose={() => setIsRecording(false)} onRecord={handleVoiceRecord} />
          )}
        </AnimatePresence>
        <AnimatePresence>
          {showCamera && (
            <CameraCapture onClose={() => setShowCamera(false)} onCapture={handlePhotoCapture} />
          )}
        </AnimatePresence>
      </div>
    </div>
  );
};

export default Chat;
