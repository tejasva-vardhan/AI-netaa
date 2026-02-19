import { memo } from 'react';
import './MessageList.css';

const MessageItem = memo(function MessageItem({ message, index }) {
  const isSystem = message.type === 'system';
  const isBot = message.type === 'bot';
  return (
    <div
      className={`message message-${message.type}`}
      data-index={index}
    >
      {isBot && !isSystem && <div className="message-avatar" aria-hidden="true" />}
      <div className="message-content">
        <div className="message-text">{message.text}</div>
        {message.timestamp && !isSystem && (
          <div className="message-time">
            {new Date(message.timestamp).toLocaleTimeString('hi-IN', {
              hour: '2-digit',
              minute: '2-digit'
            })}
          </div>
        )}
      </div>
    </div>
  );
});

function TypingIndicator() {
  return (
    <div className="message message-bot">
      <div className="message-avatar" aria-hidden="true" />
      <div className="message-content">
        <div className="message-text typing-indicator">
          <span>.</span>
          <span>.</span>
          <span>.</span>
        </div>
      </div>
    </div>
  );
}

function MessageList({ messages, isTyping }) {
  return (
    <div className="message-list">
      {messages.map((message, index) => (
        <MessageItem key={index} message={message} index={index} />
      ))}
      {isTyping && <TypingIndicator />}
    </div>
  );
}

export default memo(MessageList);
