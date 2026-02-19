// Category inference utility
// Simple keyword-based category detection

/**
 * Infers category from text using keyword matching
 * @param {string} text - User input text
 * @returns {string} - Category name ('infrastructure', 'water', 'electricity', 'sanitation', 'health', 'education', or 'general')
 */
export function inferCategory(text) {
  if (!text || typeof text !== 'string') {
    return 'general';
  }

  const lowerText = text.toLowerCase();
  
  // Road/Infrastructure keywords
  if (lowerText.match(/\b(road|sadak|गड्ढा|gaddha|pothole|bridge|nala|drain|street|path|way|सड़क)\b/)) {
    return 'infrastructure';
  }
  
  // Water keywords
  if (lowerText.match(/\b(water|paani|पानी|tap|pipeline|supply|leak|shortage|nalka|नल|जल)\b/)) {
    return 'water';
  }
  
  // Electricity keywords
  if (lowerText.match(/\b(electricity|bijli|बिजली|power|light|wire|pole|transformer|connection|लाइट)\b/)) {
    return 'electricity';
  }
  
  // Sanitation keywords
  if (lowerText.match(/\b(toilet|swachh|garbage|waste|kooda|कूड़ा|clean|dirty|sanitation|सफाई)\b/)) {
    return 'sanitation';
  }
  
  // Health keywords
  if (lowerText.match(/\b(health|hospital|doctor|डॉक्टर|medicine|treatment|disease|illness|अस्पताल)\b/)) {
    return 'health';
  }
  
  // Education keywords
  if (lowerText.match(/\b(school|education|padhai|शिक्षा|teacher|student|book|exam|स्कूल)\b/)) {
    return 'education';
  }
  
  // Default: general
  return 'general';
}

/**
 * Ensures category is always set (never null/undefined/empty)
 * @param {string|null|undefined} category - Current category value
 * @param {string} fallbackText - Text to infer from if category is missing
 * @returns {string} - Always returns a valid category string
 */
export function ensureCategory(category, fallbackText = '') {
  // If category exists and is not empty, use it
  if (category && typeof category === 'string' && category.trim() !== '') {
    return category.trim();
  }
  
  // Otherwise, infer from fallback text
  if (fallbackText && typeof fallbackText === 'string') {
    return inferCategory(fallbackText);
  }
  
  // Final fallback: general
  return 'general';
}
