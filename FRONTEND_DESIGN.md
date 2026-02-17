# Frontend Design: Voice-First Public Accountability PWA

## Overview

A Progressive Web App (PWA) designed for voice-first interaction on low-end mobile devices with slow/intermittent connectivity. The interface prioritizes accessibility, offline resilience, and minimal data usage.

## Design Principles

1. **Voice-First**: Voice input is primary, text is secondary
2. **Mobile-First**: Optimized for small screens, touch interactions
3. **Offline-First**: Works without internet, syncs when available
4. **Low-End Friendly**: Minimal animations, lightweight assets
5. **Progressive Enhancement**: Basic functionality works everywhere
6. **Graceful Degradation**: Handles failures elegantly

## Screen Flow

### Flow 1: First-Time User Journey

```
[Splash Screen]
  ↓
[Onboarding - Welcome]
  ↓
[Permission Request - Microphone]
  ↓
[Permission Request - Location] (optional)
  ↓
[Permission Request - Camera] (optional, deferred)
  ↓
[Home Screen - Chat Interface]
```

### Flow 2: Filing a Complaint

```
[Home Screen]
  ↓
[Tap "File Complaint" button]
  ↓
[Chat Interface - Greeting]
  ↓
[Conversation Flow]
  - Voice input (primary)
  - Text input (fallback)
  - Avatar animations
  ↓
[Location Collection Screen]
  ↓
[Photo Upload Screen] (if needed)
  ↓
[Review Screen]
  ↓
[Submission Screen]
  ↓
[Success Screen with Complaint Number]
```

### Flow 3: Viewing Complaint Status

```
[Home Screen]
  ↓
[Tap "My Complaints"]
  ↓
[Complaints List]
  ↓
[Tap Complaint]
  ↓
[Complaint Detail Screen]
  ↓
[Status Timeline View]
  ↓
[Back to List]
```

### Flow 4: Offline Scenario

```
[User starts complaint]
  ↓
[Network disconnects]
  ↓
[Offline Banner appears]
  ↓
[Continue conversation (stored locally)]
  ↓
[Submit button disabled]
  ↓
[Network reconnects]
  ↓
[Auto-sync notification]
  ↓
[Submit enabled]
```

## Screen Designs

### Screen 1: Splash Screen

**Purpose**: Initial load, PWA installation prompt

**Elements**:
- App logo (lightweight SVG)
- Loading indicator
- "Install App" banner (if not installed)

**States**:
- Loading
- Ready
- Install prompt

**Offline Handling**: Show immediately, don't wait for network

---

### Screen 2: Onboarding

**Purpose**: Introduce app, request permissions progressively

**Elements**:
- Welcome message
- Feature highlights (voice, offline, tracking)
- "Get Started" button

**Progressive Permissions**:
1. Microphone (required)
2. Location (optional, can skip)
3. Camera (deferred until needed)

**Permission Flow**:
```
[Request Mic] → [User allows/denies]
  ↓
[If denied] → [Show text-only mode explanation]
  ↓
[Request Location] → [User allows/denies/skips]
  ↓
[Camera permission deferred until photo needed]
```

---

### Screen 3: Home Screen

**Purpose**: Main entry point, quick actions

**Layout** (Mobile):
```
┌─────────────────────────┐
│  [Avatar]  [Menu]      │ Header
├─────────────────────────┤
│                         │
│   [File Complaint]      │ Primary CTA
│   (Large button)        │
│                         │
│   [My Complaints]       │ Secondary CTA
│                         │
│   [Help] [Settings]     │ Tertiary
│                         │
└─────────────────────────┘
```

**States**:
- Online: All features available
- Offline: Show offline banner, disable sync features
- Syncing: Show sync indicator

**Components**:
- Header with avatar
- Primary CTA button
- Quick actions
- Offline banner (conditional)
- Sync status indicator

---

### Screen 4: Chat Interface (Complaint Filing)

**Purpose**: Conversational complaint intake

**Layout**:
```
┌─────────────────────────┐
│  [Back]  File Complaint │ Header
├─────────────────────────┤
│                         │
│  [Animated Avatar]      │ Bot avatar
│  "Namaste! I'm here..." │ Bot message
│                         │
│  [User message]         │ User message
│                         │
│  [Animated Avatar]      │ Bot avatar
│  "Please describe..."   │ Bot message
│                         │
├─────────────────────────┤
│ [Mic] [Text] [Send]     │ Input area
└─────────────────────────┘
```

**Voice Input States**:
- Idle: Mic button ready
- Listening: Mic button pulsing, waveform animation
- Processing: Mic button disabled, "Processing..." text
- Error: Error message, retry button

**Text Input States**:
- Hidden (default, voice-first)
- Visible (when voice unavailable or user taps text)
- Typing indicator

**Avatar Animation**:
- Idle: Subtle breathing animation (CSS keyframes)
- Speaking: Mouth movement (simple SVG morphing)
- Thinking: Subtle rotation/pulse
- Error: Shake animation

**Components**:
- ChatContainer
- MessageList
- MessageBubble (bot/user)
- Avatar (animated)
- VoiceInput
- TextInput
- InputControls

---

### Screen 5: Location Collection

**Purpose**: Collect location for complaint

**Layout**:
```
┌─────────────────────────┐
│  [Back]  Location       │ Header
├─────────────────────────┤
│                         │
│  [Map Preview]          │ Map (if available)
│  or                     │
│  [Location Icon]        │ Placeholder
│                         │
│  "Where did this        │ Instructions
│   happen?"              │
│                         │
│  [Use Current Location] │ Button
│  [Enter Address]        │ Button
│  [Describe Location]    │ Button
│                         │
└─────────────────────────┘
```

**Permission Handling**:
- Request location permission if not granted
- Show map if permission granted
- Fallback to address input if denied
- Allow manual description

**Offline Handling**:
- Cache last known location
- Allow manual entry
- Queue for sync when online

---

### Screen 6: Photo Upload

**Purpose**: Collect photo evidence

**Layout**:
```
┌─────────────────────────┐
│  [Back]  Add Photo      │ Header
├─────────────────────────┤
│                         │
│  [Camera Preview]       │ Camera view
│  or                     │
│  [Photo Placeholder]    │ If no camera
│                         │
│  [Capture Photo]        │ Button
│  [Choose from Gallery] │ Button
│  [Skip]                 │ Button
│                         │
└─────────────────────────┘
```

**Permission Handling**:
- Request camera permission when screen opens
- Request gallery permission if user chooses gallery
- Allow skip if permission denied

**Offline Handling**:
- Store photo locally
- Queue upload when online
- Show upload progress

---

### Screen 7: Review Screen

**Purpose**: Review complaint before submission

**Layout**:
```
┌─────────────────────────┐
│  [Back]  Review         │ Header
├─────────────────────────┤
│                         │
│  Summary:               │
│  [Summary text]         │
│                         │
│  Description:           │
│  [Description text]     │
│                         │
│  Category: [Category]   │
│  Urgency: [Urgency]     │
│  Location: [Location]   │
│  Photo: [Preview]       │
│                         │
│  [Edit] [Submit]        │ Actions
│                         │
└─────────────────────────┘
```

**States**:
- Online: Submit enabled
- Offline: Submit disabled, "Save Draft" enabled
- Syncing: Submit disabled, show sync status

---

### Screen 8: Complaint List

**Purpose**: Show user's complaints

**Layout**:
```
┌─────────────────────────┐
│  [Back]  My Complaints  │ Header
├─────────────────────────┤
│  [Filter] [Sort]        │ Controls
├─────────────────────────┤
│                         │
│  [Complaint Card]       │
│  COMP-20260212-abc123   │
│  Status: Verified       │
│  [Status indicator]     │
│                         │
│  [Complaint Card]       │
│  COMP-20260211-xyz789   │
│  Status: In Progress    │
│  [Status indicator]     │
│                         │
└─────────────────────────┘
```

**Components**:
- ComplaintCard
- StatusBadge
- FilterControls
- EmptyState (if no complaints)
- LoadingState
- ErrorState

---

### Screen 9: Complaint Detail

**Purpose**: Show complaint details and timeline

**Layout**:
```
┌─────────────────────────┐
│  [Back]  Complaint      │ Header
├─────────────────────────┤
│                         │
│  COMP-20260212-abc123   │ Complaint number
│  [Status Badge]         │ Current status
│                         │
│  Summary:               │
│  [Summary text]         │
│                         │
│  Description:           │
│  [Description text]     │
│                         │
│  [View Timeline]        │ Button
│                         │
│  [Attachments]          │ Photo gallery
│                         │
└─────────────────────────┘
```

---

### Screen 10: Status Timeline

**Purpose**: Show complaint status history

**Layout**:
```
┌─────────────────────────┐
│  [Back]  Timeline       │ Header
├─────────────────────────┤
│                         │
│  ┌─────────────────┐   │
│  │ Submitted       │   │ Timeline item
│  │ 2 days ago      │   │
│  └─────────────────┘   │
│         │               │
│         ▼               │
│  ┌─────────────────┐   │
│  │ Verified        │   │ Timeline item
│  │ 1 day ago       │   │
│  └─────────────────┘   │
│         │               │
│         ▼               │
│  ┌─────────────────┐   │
│  │ In Progress     │   │ Timeline item
│  │ Today           │   │
│  └─────────────────┘   │
│                         │
└─────────────────────────┘
```

**Components**:
- TimelineContainer
- TimelineItem
- StatusIcon
- Timestamp

**Animations**:
- Fade-in on scroll (lightweight)
- Progress indicator (CSS only)

---

## Component Structure

### Component Hierarchy

```
App
├── SplashScreen
├── OnboardingFlow
│   ├── WelcomeScreen
│   ├── PermissionRequest
│   └── PermissionHandler
├── HomeScreen
│   ├── Header
│   ├── PrimaryCTA
│   ├── QuickActions
│   └── OfflineBanner
├── ChatInterface
│   ├── ChatContainer
│   ├── MessageList
│   │   └── MessageBubble
│   ├── Avatar (animated)
│   ├── VoiceInput
│   ├── TextInput
│   └── InputControls
├── LocationScreen
│   ├── MapView (optional)
│   ├── LocationInput
│   └── LocationControls
├── PhotoScreen
│   ├── CameraView (optional)
│   ├── PhotoPreview
│   └── PhotoControls
├── ReviewScreen
│   ├── ReviewSummary
│   ├── ReviewDetails
│   └── ReviewActions
├── ComplaintsList
│   ├── ComplaintCard
│   ├── FilterControls
│   └── EmptyState
├── ComplaintDetail
│   ├── ComplaintHeader
│   ├── ComplaintContent
│   └── ComplaintActions
└── StatusTimeline
    ├── TimelineContainer
    └── TimelineItem
```

### Core Components

#### 1. Avatar Component

**Purpose**: Animated bot avatar

**Props**:
```typescript
interface AvatarProps {
  state: 'idle' | 'speaking' | 'thinking' | 'error';
  size?: 'small' | 'medium' | 'large';
}
```

**Animation Strategy**:
- CSS keyframes (no JavaScript)
- SVG path morphing for mouth
- Transform animations for movement
- Max 60fps, will-change optimization

**Implementation**:
```css
/* Idle: Subtle breathing */
@keyframes breathe {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.02); }
}

/* Speaking: Mouth animation */
@keyframes speak {
  0%, 100% { d: path("M 10 20 Q 15 15 20 20"); }
  50% { d: path("M 10 20 Q 15 25 20 20"); }
}

/* Thinking: Rotation */
@keyframes think {
  0% { transform: rotate(-5deg); }
  100% { transform: rotate(5deg); }
}
```

---

#### 2. VoiceInput Component

**Purpose**: Handle voice input with visual feedback

**Props**:
```typescript
interface VoiceInputProps {
  onTranscript: (text: string) => void;
  onError: (error: Error) => void;
  language?: 'en' | 'hi' | 'hi-en';
  disabled?: boolean;
}
```

**States**:
- `idle`: Ready to listen
- `listening`: Active recording
- `processing`: Converting speech to text
- `error`: Error occurred

**Visual Feedback**:
- Mic button with pulse animation (listening)
- Waveform visualization (lightweight canvas)
- Error message with retry button

**Offline Handling**:
- Store audio locally if offline
- Queue transcription when online
- Show "Processing when online" message

---

#### 3. MessageBubble Component

**Purpose**: Display chat messages

**Props**:
```typescript
interface MessageBubbleProps {
  type: 'bot' | 'user';
  text: string;
  timestamp?: Date;
  status?: 'sending' | 'sent' | 'failed';
  avatar?: boolean; // Show avatar for bot messages
}
```

**Styling**:
- Bot: Left-aligned, avatar visible
- User: Right-aligned, no avatar
- Different colors for each type
- Typing indicator for bot

---

#### 4. ComplaintCard Component

**Purpose**: Display complaint in list

**Props**:
```typescript
interface ComplaintCardProps {
  complaintNumber: string;
  summary: string;
  status: ComplaintStatus;
  createdAt: Date;
  onClick: () => void;
}
```

**Features**:
- Status badge with color coding
- Truncated summary
- Relative timestamp
- Touch-friendly tap target

---

#### 5. TimelineItem Component

**Purpose**: Display status change in timeline

**Props**:
```typescript
interface TimelineItemProps {
  status: ComplaintStatus;
  timestamp: Date;
  notes?: string;
  isLatest?: boolean;
}
```

**Visual Design**:
- Vertical timeline with connecting line
- Status icon (lightweight SVG)
- Status text
- Timestamp (relative)
- Notes (if available)

---

## State Management Strategy

### Architecture: Redux-like Pattern (Simplified)

**Store Structure**:
```typescript
interface AppState {
  // User state
  user: {
    id: string | null;
    permissions: {
      microphone: boolean;
      location: boolean;
      camera: boolean;
    };
  };
  
  // Complaint state
  complaint: {
    current: ComplaintDraft | null;
    conversation: Message[];
    step: ConversationStep;
    isSubmitting: boolean;
  };
  
  // Complaints list
  complaints: {
    items: Complaint[];
    loading: boolean;
    error: string | null;
    filters: FilterState;
  };
  
  // Network state
  network: {
    isOnline: boolean;
    isSyncing: boolean;
    pendingActions: PendingAction[];
  };
  
  // UI state
  ui: {
    currentScreen: Screen;
    modals: ModalState[];
    notifications: Notification[];
  };
}
```

### State Management Approach

#### 1. Local-First Strategy

- **Primary**: LocalStorage/IndexedDB for persistence
- **Secondary**: Sync to server when online
- **Conflict Resolution**: Server wins, but show diff

#### 2. Optimistic Updates

- Update UI immediately
- Queue API call
- Rollback on error

#### 3. Offline Queue

```typescript
interface PendingAction {
  id: string;
  type: 'CREATE_COMPLAINT' | 'UPDATE_COMPLAINT' | 'UPLOAD_PHOTO';
  payload: any;
  timestamp: Date;
  retries: number;
}
```

**Queue Management**:
- Store in IndexedDB
- Process when online
- Retry with exponential backoff
- Show sync status to user

#### 4. Conversation State

```typescript
interface ConversationState {
  messages: Message[];
  currentStep: ConversationStep;
  collectedData: {
    summary?: string;
    description?: string;
    category?: string;
    urgency?: string;
    location?: LocationData;
    photo?: File;
  };
  isProcessing: boolean;
}
```

**Persistence**:
- Save to IndexedDB after each message
- Restore on app restart
- Clear after successful submission

---

## Error & Crash Handling

### Error Categories

#### 1. Network Errors

**Handling**:
```typescript
// Network error handler
function handleNetworkError(error: NetworkError) {
  // Show offline banner
  dispatch(setNetworkStatus('offline'));
  
  // Queue action for retry
  dispatch(queueAction(action));
  
  // Show user-friendly message
  showNotification('No internet. Your action will be saved and synced when online.');
}
```

**User Experience**:
- Offline banner appears
- Actions continue to work
- Queue indicator shows pending actions
- Auto-retry when online

---

#### 2. Permission Errors

**Handling**:
```typescript
// Permission error handler
function handlePermissionError(permission: string) {
  // Show permission request modal
  showPermissionModal({
    permission,
    onGrant: () => requestPermission(permission),
    onDeny: () => showAlternativeFlow(permission),
  });
}
```

**Fallbacks**:
- Microphone denied → Text input only
- Location denied → Manual address entry
- Camera denied → Gallery upload or skip

---

#### 3. API Errors

**Handling**:
```typescript
// API error handler
function handleAPIError(error: APIError) {
  switch (error.code) {
    case 400:
      showError('Invalid data. Please check your input.');
      break;
    case 401:
      showError('Session expired. Please login again.');
      break;
    case 500:
      showError('Server error. Please try again later.');
      queueForRetry(action);
      break;
    default:
      showError('Something went wrong. Please try again.');
  }
}
```

**Retry Strategy**:
- Immediate retry for 5xx errors
- Exponential backoff
- Max 3 retries
- Show retry button to user

---

#### 4. Storage Errors

**Handling**:
```typescript
// Storage error handler
function handleStorageError(error: StorageError) {
  // Try alternative storage
  if (error.type === 'quota') {
    clearOldData();
    retryAction();
  } else {
    // Fallback to memory-only mode
    showWarning('Storage unavailable. Data will be lost on page refresh.');
  }
}
```

---

#### 5. Crash Recovery

**Strategy**:
1. **Error Boundaries**: Catch React errors, show fallback UI
2. **Service Worker**: Handle crashes, restore state
3. **State Persistence**: Save state frequently, restore on restart

**Implementation**:
```typescript
// Error boundary
class ErrorBoundary extends React.Component {
  componentDidCatch(error, errorInfo) {
    // Log error
    logError(error, errorInfo);
    
    // Save current state
    saveStateToStorage();
    
    // Show recovery UI
    this.setState({ hasError: true });
  }
  
  render() {
    if (this.state.hasError) {
      return <ErrorRecoveryScreen />;
    }
    return this.props.children;
  }
}
```

---

### User-Friendly Error Messages

**Principles**:
- Avoid technical jargon
- Explain what happened
- Suggest next steps
- Provide recovery options

**Examples**:
- ❌ "NetworkError: Failed to fetch"
- ✅ "No internet connection. Your complaint is saved and will be submitted when online."

- ❌ "Permission denied"
- ✅ "Microphone access is needed for voice input. You can use text input instead."

- ❌ "500 Internal Server Error"
- ✅ "Server is temporarily unavailable. We'll retry automatically."

---

## Performance Optimizations

### 1. Asset Optimization

- **Images**: WebP format, lazy loading
- **Icons**: SVG sprites, inline for critical
- **Fonts**: System fonts preferred, web fonts subset
- **Animations**: CSS-only, GPU-accelerated

### 2. Code Splitting

```typescript
// Lazy load screens
const ChatInterface = React.lazy(() => import('./ChatInterface'));
const ComplaintsList = React.lazy(() => import('./ComplaintsList'));
const StatusTimeline = React.lazy(() => import('./StatusTimeline'));
```

### 3. Caching Strategy

**Service Worker Cache**:
- App shell (HTML, CSS, JS)
- Static assets (icons, images)
- API responses (with TTL)

**IndexedDB Cache**:
- User data
- Complaints list
- Conversation state
- Pending actions

### 4. Network Optimization

- **Request Batching**: Combine multiple requests
- **Request Debouncing**: Debounce rapid requests
- **Request Prioritization**: Critical requests first
- **Compression**: Gzip/Brotli for API responses

### 5. Rendering Optimization

- **Virtual Scrolling**: For long lists
- **Memoization**: React.memo for expensive components
- **Debounced Updates**: Debounce frequent state updates
- **Request Animation Frame**: For smooth animations

---

## Accessibility

### 1. Screen Reader Support

- ARIA labels for all interactive elements
- Live regions for dynamic content
- Proper heading hierarchy
- Alt text for images

### 2. Keyboard Navigation

- Tab order logical
- Focus indicators visible
- Keyboard shortcuts for common actions
- Skip links for navigation

### 3. Visual Accessibility

- High contrast mode support
- Font size scaling
- Touch targets minimum 44x44px
- Color not sole indicator

### 4. Voice Accessibility

- Voice commands supported
- Clear audio feedback
- Visual indicators for voice state
- Fallback to text always available

---

## PWA Features

### 1. Service Worker

**Responsibilities**:
- Cache app shell
- Handle offline requests
- Background sync
- Push notifications (future)

**Cache Strategy**:
- App shell: Cache-first
- API: Network-first, cache fallback
- Images: Cache-first, network update

### 2. Web App Manifest

```json
{
  "name": "Public Accountability Platform",
  "short_name": "Accountability",
  "start_url": "/",
  "display": "standalone",
  "theme_color": "#1976d2",
  "background_color": "#ffffff",
  "icons": [
    {
      "src": "/icon-192.png",
      "sizes": "192x192",
      "type": "image/png"
    },
    {
      "src": "/icon-512.png",
      "sizes": "512x512",
      "type": "image/png"
    }
  ]
}
```

### 3. Install Prompt

- Show install banner after user engagement
- Custom install button
- Explain benefits (offline, faster)

---

## Testing Strategy

### 1. Device Testing

- Low-end Android devices
- Slow 3G networks
- Intermittent connectivity
- Various screen sizes

### 2. Scenario Testing

- Complete complaint flow offline
- Network interruption during submission
- Permission denial flows
- Storage quota exceeded
- App crash recovery

### 3. Performance Testing

- First Contentful Paint < 2s
- Time to Interactive < 5s
- Lighthouse score > 90
- Bundle size < 200KB (gzipped)

---

## Implementation Notes

### Technology Stack Recommendations

- **Framework**: React (with hooks) or Vue.js
- **State Management**: Redux Toolkit or Zustand
- **Routing**: React Router or Vue Router
- **PWA**: Workbox
- **Voice**: Web Speech API
- **Storage**: IndexedDB (via idb library)
- **Styling**: CSS Modules or Tailwind CSS

### File Structure

```
src/
├── components/
│   ├── common/
│   ├── chat/
│   ├── complaints/
│   └── timeline/
├── screens/
├── store/
│   ├── slices/
│   └── middleware/
├── services/
│   ├── api/
│   ├── storage/
│   └── voice/
├── utils/
│   ├── errors/
│   └── offline/
└── assets/
    ├── icons/
    └── animations/
```

---

## Summary

This design provides:

1. **Complete Screen Flow**: 10 screens covering all user journeys
2. **Component Structure**: Hierarchical component organization
3. **State Management**: Local-first with offline sync
4. **Error Handling**: Comprehensive error scenarios and recovery
5. **Performance**: Optimizations for low-end devices
6. **Accessibility**: Full accessibility support
7. **PWA Features**: Offline-first progressive web app

The design prioritizes user experience on low-end devices with poor connectivity while maintaining a modern, voice-first interface.
