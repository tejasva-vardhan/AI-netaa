import './ChatInput.css';

function ChatInput({ value, onChange, onSend, onVoiceClick, disabled }) {
  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      onSend();
    }
  };

  return (
    <div className="chat-input-container">
      <div className="chat-input-wrapper">
        <button
          className="voice-button"
          onClick={onVoiceClick}
          disabled={disabled}
          aria-label="Voice input"
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
            <path
              d="M12 1C10.34 1 9 2.34 9 4V12C9 13.66 10.34 15 12 15C13.66 15 15 13.66 15 12V4C15 2.34 13.66 1 12 1Z"
              fill="currentColor"
            />
            <path
              d="M19 10V12C19 15.87 15.87 19 12 19C8.13 19 5 15.87 5 12V10H3V12C3 16.97 7.03 21 12 21C16.97 21 21 16.97 21 12V10H19Z"
              fill="currentColor"
            />
            <path
              d="M11 22H13V24H11V22Z"
              fill="currentColor"
            />
          </svg>
        </button>

        <textarea
          className="chat-input"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder="Type your message…"
          rows="1"
          disabled={disabled}
        />

        <button
          className="send-button"
          onClick={onSend}
          disabled={!value.trim() || disabled}
          aria-label="Send message"
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
            <path
              d="M2.01 21L23 12L2.01 3L2 10L17 12L2 14L2.01 21Z"
              fill="currentColor"
            />
          </svg>
        </button>
      </div>
    </div>
  );
}

export default ChatInput;
