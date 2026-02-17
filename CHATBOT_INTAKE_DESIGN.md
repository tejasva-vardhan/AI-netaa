# AI Chatbot - Conversational Intake Layer Design

## Overview

A conversational intake system that guides users through filing complaints via natural dialogue. The system collects structured complaint data through step-by-step questions in a neutral, respectful, and legally-safe manner.

## Design Principles

1. **Neutral Tone**: Netaji-style respectful, formal but approachable
2. **No Assumptions**: Never assume truth or validity of claims
3. **Legal Safety**: Avoid leading questions, maintain neutrality
4. **Bilingual Support**: Hindi + English, code-mixed allowed
5. **Progressive Disclosure**: Ask one question at a time
6. **User Control**: Allow users to skip, go back, or clarify

## Conversation Flow (Dialog Tree)

### Phase 1: Greeting & Initial Setup

```
START
  ↓
[Greeting] "Namaste / Hello. I'm here to help you file a complaint..."
  ↓
[Language Preference] "किस भाषा में बात करें? / Which language would you prefer?"
  ↓
[Confirmation] "Thank you. Let's begin..."
  ↓
PHASE 2
```

### Phase 2: Complaint Summary

```
[Open Question] "Please briefly describe your issue..."
  ↓
[Clarification if needed] "Could you provide more details about..."
  ↓
[Summary Confirmation] "Let me confirm: Your issue is about..."
  ↓
PHASE 3
```

### Phase 3: Detailed Description

```
[Detailed Description] "Now, please provide more details..."
  ↓
[Follow-up Questions]
  - "When did this happen?"
  - "Where did this happen?"
  - "Who is involved?"
  ↓
[Description Confirmation] "Based on what you've shared..."
  ↓
PHASE 4
```

### Phase 4: Category Classification

```
[Category Options] "What category best describes your complaint?"
  ↓
[Category List] (Present options)
  ↓
[Category Selection] "You selected: [category]"
  ↓
PHASE 5
```

### Phase 5: Urgency Assessment

```
[Urgency Question] "How urgent is this issue?"
  ↓
[Urgency Options]
  - Low: Can wait a few days
  - Medium: Should be addressed soon
  - High: Needs immediate attention
  - Urgent: Critical situation
  ↓
[Urgency Confirmation] "You've marked this as [urgency]"
  ↓
PHASE 6
```

### Phase 6: Location Collection

```
[Location Question] "Where did this issue occur?"
  ↓
[Location Options]
  - "I can share my current location"
  - "I can provide an address"
  - "I can describe the location"
  ↓
[Location Collection] (GPS, address, or description)
  ↓
[Location Confirmation] "Location recorded: [location]"
  ↓
PHASE 7
```

### Phase 7: Photo/Evidence Request

```
[Photo Question] "Do you have a photo or video of this issue?"
  ↓
[Photo Options]
  - "Yes, I can share now"
  - "Yes, but later"
  - "No, I don't have"
  ↓
[Photo Collection] (If yes, request upload)
  ↓
PHASE 8
```

### Phase 8: Review & Confirmation

```
[Summary Review] "Let me summarize what you've shared..."
  ↓
[Review Display]
  - Summary: [summary]
  - Description: [description]
  - Category: [category]
  - Urgency: [urgency]
  - Location: [location]
  - Photo: [yes/no]
  ↓
[Final Confirmation] "Is this information correct?"
  ↓
[User Confirmation] Yes/No/Edit
  ↓
[If Edit] → Return to relevant phase
[If Yes] → Generate JSON
[If No] → Start over
```

## Detailed Dialog States

### State 1: GREETING

**Bot**: 
```
Namaste! मैं आपकी मदद करने के लिए यहाँ हूँ। / Hello! I'm here to help you file a complaint 
about a public issue. This process will take a few minutes. 

क्या हम शुरू कर सकते हैं? / Shall we begin?
```

**User Options**:
- "Yes / हाँ"
- "No / नहीं" (exit)
- "What is this? / यह क्या है?"

**Bot Response (if "What is this?"**):
```
This is a public accountability platform. You can file complaints about 
public issues like infrastructure problems, service delivery issues, 
or other concerns. All complaints are tracked and addressed by 
responsible departments.

क्या आप शिकायत दर्ज करना चाहते हैं? / Would you like to file a complaint?
```

### State 2: LANGUAGE_SELECTION

**Bot**:
```
किस भाषा में बात करें? / Which language would you prefer?

1. हिंदी / Hindi
2. English
3. Both / दोनों (code-mixed)
```

**User**: Selects option

**Bot**:
```
धन्यवाद। चलिए शुरू करते हैं। / Thank you. Let's begin.
```

### State 3: SUMMARY_COLLECTION

**Bot**:
```
कृपया अपनी समस्या का संक्षिप्त विवरण दें। / Please briefly describe 
your issue in a few sentences. What is the main problem you want to report?
```

**User**: Provides summary

**Bot** (if summary is too short):
```
क्या आप थोड़ा और विस्तार से बता सकते हैं? / Could you provide a bit 
more detail? What exactly is the issue?
```

**Bot** (if summary is adequate):
```
मैं समझ गया। आपकी समस्या [summary] के बारे में है। / I understand. 
Your issue is about [summary]. Is this correct?
```

### State 4: DETAILED_DESCRIPTION

**Bot**:
```
अब कृपया अधिक विवरण दें। / Now, please provide more details:

1. यह कब हुआ? / When did this happen?
2. यह कहाँ हुआ? / Where did this happen?
3. कौन शामिल है? / Who is involved?

आप इन सवालों के जवाब दे सकते हैं या अपने शब्दों में वर्णन कर सकते हैं।
```

**User**: Provides details

**Bot** (follow-up if needed):
```
क्या आप और कुछ जानकारी साझा कर सकते हैं? / Is there anything else 
you'd like to add?
```

**Bot** (when sufficient):
```
धन्यवाद। मैंने आपकी जानकारी रिकॉर्ड की है। / Thank you. I've recorded 
your information.
```

### State 5: CATEGORY_SELECTION

**Bot**:
```
आपकी शिकायत किस श्रेणी में आती है? / What category best describes 
your complaint?

1. Infrastructure / बुनियादी ढांचा
   - Roads, bridges, water supply, electricity
2. Sanitation / स्वच्छता
   - Garbage, drainage, public toilets
3. Public Safety / सार्वजनिक सुरक्षा
   - Street lights, security, accidents
4. Service Delivery / सेवा वितरण
   - Government services, delays, corruption
5. Environment / पर्यावरण
   - Pollution, trees, parks
6. Other / अन्य
```

**User**: Selects category

**Bot**:
```
आपने [category] चुना है। / You've selected [category]. Is this correct?
```

### State 6: URGENCY_ASSESSMENT

**Bot**:
```
यह मुद्दा कितना जरूरी है? / How urgent is this issue?

1. Low / कम - Can wait a few days / कुछ दिन इंतज़ार कर सकता है
2. Medium / मध्यम - Should be addressed soon / जल्दी संबोधित होना चाहिए
3. High / उच्च - Needs immediate attention / तत्काल ध्यान चाहिए
4. Urgent / अत्यावश्यक - Critical situation / गंभीर स्थिति
```

**User**: Selects urgency

**Bot**:
```
आपने इसे [urgency] के रूप में चिह्नित किया है। / You've marked this 
as [urgency].
```

### State 7: LOCATION_COLLECTION

**Bot**:
```
यह समस्या कहाँ हुई? / Where did this issue occur?

1. मैं अपना वर्तमान स्थान साझा कर सकता हूँ / I can share my current location
2. मैं एक पता दे सकता हूँ / I can provide an address
3. मैं स्थान का वर्णन कर सकता हूँ / I can describe the location
```

**User**: Selects option

**Bot** (if GPS):
```
कृपया अपना स्थान साझा करें। / Please share your location.
[Location picker/GPS prompt]
```

**Bot** (if address):
```
कृपया पूरा पता दें। / Please provide the complete address.
```

**Bot** (if description):
```
कृपया स्थान का वर्णन करें। / Please describe the location.
```

**Bot** (after collection):
```
स्थान रिकॉर्ड किया गया: [location] / Location recorded: [location]
```

### State 8: PHOTO_REQUEST

**Bot**:
```
क्या आपके पास इस मुद्दे की कोई तस्वीर या वीडियो है? / Do you have a 
photo or video of this issue?

1. हाँ, अभी साझा कर सकता हूँ / Yes, I can share now
2. हाँ, लेकिन बाद में / Yes, but later
3. नहीं, मेरे पास नहीं है / No, I don't have
```

**User**: Selects option

**Bot** (if "Yes, now"):
```
कृपया तस्वीर या वीडियो अपलोड करें। / Please upload the photo or video.
[File upload prompt]
```

**Bot** (if "Yes, later"):
```
ठीक है। आप बाद में तस्वीर जोड़ सकते हैं। / Okay. You can add the photo later.
```

**Bot** (if "No"):
```
कोई बात नहीं। आप बिना तस्वीर के भी शिकायत दर्ज कर सकते हैं। / No problem. 
You can file the complaint without a photo.
```

### State 9: REVIEW_CONFIRMATION

**Bot**:
```
मैंने जो जानकारी एकत्र की है, उसकी समीक्षा करें: / Let me summarize 
what you've shared:

**Summary / सारांश**: [summary]
**Description / विवरण**: [description]
**Category / श्रेणी**: [category]
**Urgency / जरूरत**: [urgency]
**Location / स्थान**: [location]
**Photo / तस्वीर**: [yes/no]

क्या यह जानकारी सही है? / Is this information correct?

1. Yes / हाँ - Submit / जमा करें
2. No / नहीं - Start over / फिर से शुरू करें
3. Edit / संपादित करें
```

**User**: Selects option

**Bot** (if "Yes"):
```
धन्यवाद। आपकी शिकायत दर्ज की गई है। / Thank you. Your complaint has 
been filed. You will receive a complaint number shortly.
```

**Bot** (if "No"):
```
ठीक है। चलिए फिर से शुरू करते हैं। / Okay. Let's start over.
[Return to State 1]
```

**Bot** (if "Edit"):
```
कौन सा भाग संपादित करना चाहते हैं? / Which part would you like to edit?
[Show edit options]
```

## Sample Conversations

### Sample 1: Text Input (English)

```
Bot: Hello! I'm here to help you file a complaint about a public issue. 
     This process will take a few minutes. Shall we begin?

User: Yes

Bot: Which language would you prefer?
     1. Hindi
     2. English
     3. Both

User: 2

Bot: Thank you. Let's begin. Please briefly describe your issue in a 
     few sentences. What is the main problem you want to report?

User: There's a huge pothole on Main Street near the market. It's 
      causing traffic jams and vehicles are getting damaged.

Bot: I understand. Your issue is about a pothole on Main Street near 
     the market causing traffic and vehicle damage. Is this correct?

User: Yes

Bot: Now, please provide more details:
     1. When did this happen?
     2. Where did this happen?
     3. Who is involved?
     You can answer these questions or describe in your own words.

User: This pothole has been there for about 2 weeks. It's getting 
      bigger every day. Many people have complained but nothing 
      happened. Yesterday, a motorcycle rider fell and got injured.

Bot: Thank you. I've recorded your information. What category best 
     describes your complaint?
     1. Infrastructure
     2. Sanitation
     3. Public Safety
     4. Service Delivery
     5. Environment
     6. Other

User: 1

Bot: You've selected Infrastructure. Is this correct?

User: Yes

Bot: How urgent is this issue?
     1. Low - Can wait a few days
     2. Medium - Should be addressed soon
     3. High - Needs immediate attention
     4. Urgent - Critical situation

User: 3

Bot: You've marked this as High. Where did this issue occur?
     1. I can share my current location
     2. I can provide an address
     3. I can describe the location

User: 1

Bot: Please share your location.
[User shares GPS location]

Bot: Location recorded: Main Street, Near Market, GPS: 28.6139, 77.2090

Bot: Do you have a photo or video of this issue?
     1. Yes, I can share now
     2. Yes, but later
     3. No, I don't have

User: 1

Bot: Please upload the photo or video.
[User uploads photo]

Bot: Let me summarize what you've shared:
     **Summary**: Pothole on Main Street near market causing traffic 
                  and vehicle damage
     **Description**: Pothole present for 2 weeks, getting bigger, 
                      motorcycle rider injured yesterday
     **Category**: Infrastructure
     **Urgency**: High
     **Location**: Main Street, Near Market, GPS: 28.6139, 77.2090
     **Photo**: Yes

     Is this information correct?
     1. Yes - Submit
     2. No - Start over
     3. Edit

User: 1

Bot: Thank you. Your complaint has been filed. You will receive a 
     complaint number shortly.
```

### Sample 2: Voice Input (Hindi-English Code-Mixed)

```
Bot: Namaste! मैं आपकी मदद करने के लिए यहाँ हूँ। This process will 
     take a few minutes. क्या हम शुरू कर सकते हैं?

User: [Voice] हाँ, चलिए start करते हैं

Bot: किस भाषा में बात करें?
     1. हिंदी
     2. English
     3. Both

User: [Voice] तीन, दोनों languages में

Bot: धन्यवाद। चलिए शुरू करते हैं। कृपया अपनी समस्या का संक्षिप्त 
     विवरण दें।

User: [Voice] मेरे area में garbage collection नहीं हो रही है। 
      बहुत दिनों से। Smell बहुत आ रही है और mosquitoes भी हैं।

Bot: मैं समझ गया। आपकी समस्या garbage collection न होने के बारे में 
     है। Is this correct?

User: [Voice] हाँ, बिल्कुल

Bot: अब कृपया अधिक विवरण दें। यह कब से हो रहा है?

User: [Voice] लगभग एक महीने से। पहले weekly collection होती थी, 
      अब बिल्कुल नहीं हो रही।

Bot: यह कहाँ हो रहा है?

User: [Voice] Sector 15, Block C, near the park

Bot: क्या आप और कुछ जानकारी साझा कर सकते हैं?

User: [Voice] हाँ, local residents ने complaint की थी लेकिन कोई 
      action नहीं हुआ। Health issue हो रहा है।

Bot: धन्यवाद। What category best describes your complaint?
     1. Infrastructure
     2. Sanitation / स्वच्छता
     3. Public Safety
     4. Service Delivery
     5. Environment
     6. Other

User: [Voice] दो, sanitation

Bot: आपने Sanitation चुना है। How urgent is this?
     1. Low
     2. Medium
     3. High
     4. Urgent

User: [Voice] तीन, high

Bot: Where did this issue occur?
     1. I can share my current location
     2. I can provide an address
     3. I can describe the location

User: [Voice] मैं location share कर सकता हूँ
[User shares GPS]

Bot: Location recorded: Sector 15, Block C, GPS: 28.6543, 77.1234

Bot: Do you have a photo?
     1. Yes, I can share now
     2. Yes, but later
     3. No

User: [Voice] हाँ, अभी share करता हूँ
[User uploads photo]

Bot: Let me summarize:
     **Summary**: Garbage collection not happening in area
     **Description**: No collection for 1 month, smell and mosquitoes, 
                     health issues, previous complaints ignored
     **Category**: Sanitation
     **Urgency**: High
     **Location**: Sector 15, Block C, GPS: 28.6543, 77.1234
     **Photo**: Yes

     Is this correct?

User: [Voice] हाँ, submit कर दो

Bot: धन्यवाद। Your complaint has been filed.
```

### Sample 3: Minimal Input (User provides all info upfront)

```
Bot: Hello! I'm here to help you file a complaint. Shall we begin?

User: Yes

Bot: Which language would you prefer?

User: English

Bot: Please briefly describe your issue.

User: Street light not working on Park Road for 3 days. It's dark 
      and unsafe at night. Location: Park Road, near Metro Station. 
      Category: Public Safety. Urgency: High. I have photos.

Bot: I understand. Your issue is about a non-working street light 
     on Park Road causing safety concerns. Is this correct?

User: Yes

Bot: Now, please provide more details. When did this happen?

User: The light stopped working 3 days ago. It was working fine before.

Bot: Thank you. What category best describes your complaint?

User: Public Safety

Bot: How urgent is this issue?

User: High

Bot: Where did this issue occur?

User: Park Road, near Metro Station. I can share location.

Bot: Please share your location.
[User shares GPS]

Bot: Do you have a photo?

User: Yes, here
[User uploads photo]

Bot: Let me summarize:
     **Summary**: Street light not working on Park Road
     **Description**: Light stopped 3 days ago, dark and unsafe at night
     **Category**: Public Safety
     **Urgency**: High
     **Location**: Park Road, near Metro Station, GPS: [coordinates]
     **Photo**: Yes

     Is this correct?

User: Yes

Bot: Thank you. Your complaint has been filed.
```

## JSON Output Format

### Example 1: Complete Complaint

```json
{
  "summary": "Large pothole on Main Street near market causing traffic jams and vehicle damage",
  "description": "Pothole has been present for approximately 2 weeks and is getting bigger every day. Many people have complained but no action taken. Yesterday, a motorcycle rider fell and got injured. Location: Main Street, Near Market. The pothole is approximately 2 feet wide and 6 inches deep.",
  "category": "infrastructure",
  "urgency": "high",
  "language": "en",
  "needs_live_photo": true,
  "needs_location": true
}
```

### Example 2: Hindi-English Code-Mixed

```json
{
  "summary": "Garbage collection not happening in area, causing smell and mosquitoes",
  "description": "Garbage collection has stopped for approximately 1 month. Previously, weekly collection was happening but now completely stopped. Local residents have filed complaints but no action taken. Health issues are arising due to smell and mosquitoes. Location: Sector 15, Block C, near the park.",
  "category": "sanitation",
  "urgency": "high",
  "language": "hi-en",
  "needs_live_photo": true,
  "needs_location": true
}
```

### Example 3: Minimal Information

```json
{
  "summary": "Street light not working on Park Road",
  "description": "Street light stopped working 3 days ago. It was working fine before. Now it's dark and unsafe at night. Location: Park Road, near Metro Station.",
  "category": "public_safety",
  "urgency": "high",
  "language": "en",
  "needs_live_photo": true,
  "needs_location": true
}
```

### Example 4: No Photo Available

```json
{
  "summary": "Water leakage from main pipeline",
  "description": "Water is leaking from the main pipeline on Station Road. This has been happening for 5 days. Water is being wasted and road is getting damaged. Location: Station Road, near Bus Stand.",
  "category": "infrastructure",
  "urgency": "medium",
  "language": "en",
  "needs_live_photo": false,
  "needs_location": true
}
```

## Field Mapping Rules

### Summary
- **Source**: User's initial brief description
- **Length**: 50-200 characters
- **Format**: Single sentence or short paragraph
- **Language**: Preserve original language

### Description
- **Source**: Detailed description phase + follow-up questions
- **Length**: 200-1000 characters
- **Format**: Paragraph with complete details
- **Includes**: When, where, who, impact, previous actions

### Category
- **Mapping**:
  - Infrastructure → "infrastructure"
  - Sanitation → "sanitation"
  - Public Safety → "public_safety"
  - Service Delivery → "service_delivery"
  - Environment → "environment"
  - Other → "other"

### Urgency
- **Mapping**:
  - Low → "low"
  - Medium → "medium"
  - High → "high"
  - Urgent → "urgent"

### Language
- **Mapping**:
  - Hindi only → "hi"
  - English only → "en"
  - Code-mixed → "hi-en"

### needs_live_photo
- **true**: User has photo/video or will provide later
- **false**: User confirmed no photo available

### needs_location
- **true**: Location collected (GPS, address, or description)
- **false**: User skipped location (should not happen in normal flow)

## Error Handling & Edge Cases

### User Provides Incomplete Information

**Bot Response**:
```
मुझे कुछ और जानकारी चाहिए। / I need a bit more information to proceed. 
कृपया [missing field] के बारे में बताएं। / Please tell me about [missing field].
```

### User Wants to Skip a Question

**Bot Response**:
```
क्या आप इस सवाल को छोड़ना चाहते हैं? / Would you like to skip this question?
ध्यान दें: [field] जानकारी complaint processing के लिए helpful हो सकती है।
/ Note: [field] information can be helpful for complaint processing.
```

### User Provides Contradictory Information

**Bot Response**:
```
मैंने देखा कि आपने पहले [info1] कहा था, लेकिन अब [info2] कह रहे हैं।
/ I noticed you said [info1] earlier, but now you're saying [info2].
कौन सा सही है? / Which one is correct?
```

### User Wants to Start Over

**Bot Response**:
```
ठीक है। चलिए फिर से शुरू करते हैं। / Okay. Let's start over.
[Reset to State 1]
```

### User Provides Offensive Content

**Bot Response**:
```
मैं समझता हूँ कि आप परेशान हैं, लेकिन कृपया respectful language का 
उपयोग करें। / I understand you're upset, but please use respectful language.
क्या आप अपनी concern को differently express कर सकते हैं?
/ Can you express your concern differently?
```

## Voice Input Handling

### Speech-to-Text Conversion
- Support Hindi and English recognition
- Handle code-mixed speech
- Provide transcription feedback: "मैंने सुना: [transcription]"

### Voice Command Recognition
- "Skip / छोड़ दो" → Skip current question
- "Go back / पीछे जाओ" → Previous question
- "Repeat / दोहराओ" → Repeat question
- "Start over / फिर से शुरू करो" → Reset conversation

## Integration Points

### After JSON Generation

1. **Pass to Complaint Creation API**
   - Use `summary` as `title`
   - Use `description` as `description`
   - Map `category` to complaint category
   - Map `urgency` to complaint priority
   - Use `language` for localization

2. **Handle Photo Upload**
   - If `needs_live_photo: true`, request photo upload
   - Link photo to complaint after creation

3. **Handle Location**
   - If `needs_location: true`, use collected location
   - Map to `location_id` or coordinates

## Testing Scenarios

### Scenario 1: Complete Flow
- User provides all information step-by-step
- All fields populated correctly
- JSON generated successfully

### Scenario 2: Partial Information
- User skips some questions
- System handles gracefully
- JSON generated with available information

### Scenario 3: Code-Mixed Input
- User mixes Hindi and English
- System preserves language in JSON
- `language` field set to "hi-en"

### Scenario 4: Voice Input
- User speaks instead of typing
- Speech-to-text conversion works
- Conversation flow continues normally

### Scenario 5: Edit Flow
- User reviews and wants to edit
- System returns to relevant state
- Changes reflected in final JSON

## Notes for Implementation

1. **State Management**: Maintain conversation state (current phase, collected data)
2. **Natural Language Understanding**: Parse user responses to extract information
3. **Validation**: Validate collected data before moving to next phase
4. **Persistence**: Save conversation state for resumption
5. **Timeout Handling**: Handle inactive users gracefully
6. **Multi-turn Clarification**: Support follow-up questions within same phase
