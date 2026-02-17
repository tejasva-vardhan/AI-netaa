# Chat UX Stabilization – Logic Notes & Backend Reset

## Frontend logic notes

### 1. Optimistic UI
- **User message**: Appended immediately in `ChatScreen.handleSend` via `addMessage(userMessage)` and `setInputText('')` before any async work.
- **Persistence**: Backend does not persist chat messages; state lives in `ComplaintContext` and is persisted to `localStorage` only. Persistence is **debounced** (300 ms) in `ComplaintContext` so each message/update does not block the UI or cause synchronous writes.

### 2. Preventing lag (no full re-render per message)
- **MessageList** is wrapped in `React.memo` so the list component re-renders only when the `messages` prop reference changes.
- **MessageItem** is a separate component wrapped in `React.memo`; each item re-renders only when its own `message`/`index` change. Appending a new message therefore re-renders only the new item (and the list container), not the entire history.

### 3. Hidden reset ("restart")
- **Trigger**: When the user sends exactly the message **"restart"** (trimmed, case-insensitive), the app treats it as a reset.
- **Flow**:
  1. Call `api.resetChatDraft()` (POST `/api/v1/users/chat/reset`). If the request fails (e.g. offline), the app still performs the local reset.
  2. `clearComplaintData()` – clears conversation, draft fields, step, completedSteps; clears `complaint_draft` from `localStorage`.
  3. Add a single bot welcome message and set `step: 'summary'`, `completedSteps: []`.
- **Auth**: No logout; `auth_token`, `phone_verified`, `user_id`, `user_phone` remain in `localStorage`.

### 4. System confirmations
- **Location verified**: When the user successfully captures location (GPS success or Continue after capture), `addMessage({ type: 'system', text: '📍 Location verified' })` is called in `LocationScreen` (in the success callback and in `handleContinue`).
- **Photo captured**: When the user clicks Continue after capturing a photo, `addMessage({ type: 'system', text: '📸 Photo captured' })` is called in `CameraScreen.handleContinue`.
- **Complaint submitted**: After a successful `api.createComplaint()`, `addMessage({ type: 'system', text: '📄 Complaint submitted' })` is called, then `clearComplaintData({ keepConversation: true })` so the conversation (including this line) is kept, then navigate to the complaint detail/success page.

### 5. Clear draft with optional conversation keep
- `clearComplaintData(options)` supports `options.keepConversation: true`. When set, draft fields are reset but `conversation` is kept and the resulting state is written to `localStorage` so the user can see the last run (including "📄 Complaint submitted") when returning to chat.

### 6. No UI changes
- No new buttons, CTA changes, or onboarding screens were added. Reset is text-only ("restart"); confirmations are system messages in the existing chat.

---

## Backend reset handler

### Route
- **POST** `/api/v1/users/chat/reset`
- **Auth**: Required (same JWT auth as other user endpoints). Auth remains intact; this endpoint only clears server-side chat/draft state.

### Handler
- **File**: `handler/chat_handler.go`
- **Method**: `ChatHandler.ResetChatDraft`
- **Behavior**: Returns `200 OK` with `{"message":"ok"}`. User identity is already enforced by auth middleware.
- **Future**: If you add server-side `chat_state` or temp complaint draft storage (e.g. per `user_id`), clear that data in this handler so "restart" clears both client and server state.

### Wiring
- In `routes/routes.go`, the route is registered on the `users` subrouter with `authMiddleware.RequireAuth`, so unauthenticated requests receive 401.
