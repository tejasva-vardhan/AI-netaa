import React from 'react';
import { useParams } from 'react-router-dom';
import ComplaintDetailScreen from '../screens/ComplaintDetailScreen';

// ComplaintTracker = existing ComplaintDetailScreen (no backend change).
function ComplaintTracker() {
  const { id } = useParams();
  return <ComplaintDetailScreen />;
}

export default ComplaintTracker;
