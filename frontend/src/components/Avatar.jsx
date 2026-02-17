import React, { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { FaMicrophone } from 'react-icons/fa';

const Avatar = ({ isSpeaking, onTalkClick }) => {
  const [eyeDirection, setEyeDirection] = useState({ x: 0, y: 0 });

  useEffect(() => {
    const handleMouseMove = (e) => {
      const x = (e.clientX / window.innerWidth - 0.5) * 20;
      const y = (e.clientY / window.innerHeight - 0.5) * 20;
      setEyeDirection({ x, y });
    };

    window.addEventListener('mousemove', handleMouseMove);
    return () => window.removeEventListener('mousemove', handleMouseMove);
  }, []);

  return (
    <div className="relative flex flex-col items-center justify-center min-h-[60vh]">
      <motion.div
        className="avatar-container w-64 h-64 md:w-80 md:h-80 cursor-pointer relative"
        animate={{
          scale: isSpeaking ? [1, 1.02, 1] : 1,
        }}
        transition={{
          duration: 2,
          repeat: isSpeaking ? Infinity : 0,
          ease: "easeInOut"
        }}
        onClick={onTalkClick}
      >
        {/* Animated Netaji-style Avatar */}
        <svg viewBox="0 0 200 200" className="w-full h-full">
          {/* Background glow */}
          <circle cx="100" cy="100" r="90" fill="url(#grad1)" />
          
          {/* Face */}
          <circle cx="100" cy="90" r="50" fill="#FFE0BD" />
          
          {/* Turban with Netaji style */}
          <path d="M60 60 Q100 30 140 60 L140 80 Q100 110 60 80" fill="#C3B091" />
          <circle cx="120" cy="55" r="5" fill="#FFD700" />
          
          {/* Animated Eyes */}
          <motion.g
            animate={{
              x: eyeDirection.x,
              y: eyeDirection.y
            }}
          >
            <circle cx="80" cy="85" r="8" fill="white" />
            <circle cx="120" cy="85" r="8" fill="white" />
            <circle cx="83" cy="87" r="4" fill="black" />
            <circle cx="123" cy="87" r="4" fill="black" />
          </motion.g>
          
          {/* Mouth - animates when speaking */}
          <motion.path
            d={isSpeaking ? "M80 110 Q100 125 120 110" : "M80 110 Q100 115 120 110"}
            stroke="#8B4513"
            strokeWidth="3"
            fill="none"
            animate={{
              d: isSpeaking ? [
                "M80 110 Q100 125 120 110",
                "M80 115 Q100 130 120 115",
                "M80 110 Q100 125 120 110"
              ] : "M80 110 Q100 115 120 110"
            }}
            transition={{
              duration: 0.5,
              repeat: isSpeaking ? Infinity : 0,
              ease: "easeInOut"
            }}
          />
          
          {/* Mustache */}
          <path d="M85 95 Q100 100 115 95" stroke="#4A3B2A" strokeWidth="3" fill="none" />
          
          <defs>
            <linearGradient id="grad1" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" style={{ stopColor: '#FF9933', stopOpacity: 0.2 }} />
              <stop offset="100%" style={{ stopColor: '#138808', stopOpacity: 0.2 }} />
            </linearGradient>
          </defs>
        </svg>

        {/* Speaking animation rings */}
        <AnimatePresence>
          {isSpeaking && (
            <>
              <motion.div
                className="absolute inset-0 rounded-full border-4 border-netaji-saffron"
                initial={{ scale: 1, opacity: 1 }}
                animate={{ scale: 1.5, opacity: 0 }}
                transition={{ duration: 1.5, repeat: Infinity }}
              />
              <motion.div
                className="absolute inset-0 rounded-full border-4 border-netaji-green"
                initial={{ scale: 1, opacity: 1 }}
                animate={{ scale: 2, opacity: 0 }}
                transition={{ duration: 1.5, delay: 0.5, repeat: Infinity }}
              />
            </>
          )}
        </AnimatePresence>
      </motion.div>

      <motion.button
        type="button"
        className="mt-8 px-8 py-4 bg-gradient-to-r from-netaji-navy to-netaji-green text-white rounded-full font-semibold text-xl shadow-xl flex items-center gap-3 hover:shadow-2xl transition-all"
        whileHover={{ scale: 1.05 }}
        whileTap={{ scale: 0.95 }}
        onClick={onTalkClick}
      >
        <FaMicrophone className="text-netaji-gold" />
        Talk with me
      </motion.button>
      
      <p className="mt-4 text-gray-600 text-lg">Your AI Janpratinidhi is here to help</p>
    </div>
  );
};

export default Avatar;
