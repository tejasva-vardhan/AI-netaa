# Public Accountability Platform - Frontend

Minimal, production-safe PWA frontend for voice-first public accountability platform.

## Features

- Landing screen with "Talk with me" CTA
- Chat-based complaint intake (text first, voice placeholder)
- Live camera capture (gallery disabled)
- Location permission capture
- Complaint submission
- Complaint list + status timeline (read-only)

## Tech Stack

- React 18
- React Router 6
- Vite (build tool)
- PWA support (Vite PWA plugin)
- Vanilla CSS (no frameworks)

## Setup

```bash
npm install
npm run dev
```

## Build

```bash
npm run build
```

## Project Structure

```
src/
├── components/          # Reusable components
│   ├── MessageList.jsx
│   ├── ChatInput.jsx
│   └── StatusTimeline.jsx
├── screens/            # Screen components
│   ├── LandingScreen.jsx
│   ├── ChatScreen.jsx
│   ├── CameraScreen.jsx
│   ├── LocationScreen.jsx
│   ├── ReviewScreen.jsx
│   ├── ComplaintsListScreen.jsx
│   └── ComplaintDetailScreen.jsx
├── services/           # API services
│   └── api.js
├── state/             # State management
│   └── ComplaintContext.jsx
├── App.jsx            # Main app component
├── main.jsx           # Entry point
└── index.css          # Global styles
```

## API Integration

API endpoints are defined in `src/services/api.js`. Update `API_BASE_URL` in the file or set `VITE_API_BASE_URL` environment variable.

## State Management

Uses React Context API for simple state management. Complaint draft data is persisted to localStorage.

## Permissions

- Microphone: Requested when user clicks voice button (placeholder)
- Camera: Requested on camera screen
- Location: Requested on location screen

Permissions are requested progressively, not all at once.

## Offline Support

- Offline banner shown when network unavailable
- Complaint draft saved to localStorage
- Syncs when online

## Browser Support

- Modern browsers with ES6+ support
- Mobile browsers (iOS Safari, Chrome Android)
- PWA installable on mobile devices
