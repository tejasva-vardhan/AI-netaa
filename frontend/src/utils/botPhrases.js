/**
 * Helper function to get random variations of common bot phrases
 * Keeps responses natural and avoids repetition
 * 
 * @param {string} type - Type of phrase ('acknowledge', 'understood', 'okay')
 * @returns {string} Random variation of the phrase
 */
export function getRandomPhrase(type) {
  const phrases = {
    // Acknowledgment phrases (like "Theek hai")
    acknowledge: [
      'Theek hai',
      'Achha',
      'Theek hai 👍'
    ],
    
    // Understanding phrases (like "Samajh gaya")
    understood: [
      'Samajh gaya',
      'Theek hai, samajh gaya',
      'Achha, samajh gaya'
    ],
    
    // Simple okay (with emoji)
    okay: [
      'Theek hai 👍',
      'Achha 👍',
      'Samajh gaya 👍'
    ],
    
    // Location-related acknowledgment
    locationAck: [
      'Theek hai',
      'Achha',
      'Theek hai'
    ]
  };

  const options = phrases[type] || ['Theek hai'];
  
  // Return random option
  return options[Math.floor(Math.random() * options.length)];
}
