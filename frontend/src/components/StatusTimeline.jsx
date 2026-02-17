import './StatusTimeline.css';

function getActorLabel(changedByType) {
  if (!changedByType) return '';
  const t = String(changedByType).toLowerCase();
  if (t === 'system') return 'System';
  if (t === 'authority' || t === 'officer') return 'Authority';
  if (t === 'user') return 'Aap';
  if (t === 'admin') return 'Admin';
  return changedByType;
}

function StatusTimeline({ timeline }) {
  if (!timeline || timeline.length === 0) {
    return (
      <div className="timeline-empty">
        <p>Abhi tak koi status update nahi.</p>
        <p className="timeline-empty-sub">Jab action hoga, yahan dikhega.</p>
      </div>
    );
  }

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleString('hi-IN', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  const sorted = [...timeline].sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
  const actorClass = (type) => {
    const t = type ? String(type).toLowerCase() : '';
    if (t === 'authority' || t === 'officer') return 'timeline-marker--authority';
    if (t === 'user') return 'timeline-marker--user';
    return 'timeline-marker--system';
  };

  return (
    <div className="status-timeline">
      {sorted.map((item) => (
        <div key={item.history_id} className="timeline-item">
          <div className={`timeline-marker ${actorClass(item.changed_by_type)}`} aria-hidden="true" />
          <div className="timeline-content">
            <div className="timeline-status">
              {item.old_status && (
                <span className="status-old">{String(item.old_status).replace(/_/g, ' ')}</span>
              )}
              <span className="status-arrow">→</span>
              <span className="status-new">{String(item.new_status).replace(/_/g, ' ')}</span>
            </div>
            <div className="timeline-meta">
              <span className="timeline-actor">{getActorLabel(item.changed_by_type)}</span>
              <span className="timeline-date">{formatDate(item.created_at)}</span>
            </div>
            {(item.reason || (item.notes && item.notes.trim())) && (
              <div className="timeline-notes">
                {item.reason && <span>{item.reason}</span>}
                {item.reason && item.notes && item.notes.trim() && ' · '}
                {item.notes && item.notes.trim() && <span>{item.notes}</span>}
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

export default StatusTimeline;
