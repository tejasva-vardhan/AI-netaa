import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { FaCheckCircle, FaClock, FaExclamationTriangle, FaBuilding, FaUser } from 'react-icons/fa';
import api from '../services/api';
import toast from 'react-hot-toast';

const ComplaintStatus = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const [complaint, setComplaint] = useState(null);
  const [timeline, setTimeline] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadComplaint();
  }, [id]);

  const loadComplaint = async () => {
    try {
      const data = await api.getComplaint(id);
      setComplaint(data);
      try {
        const tlData = await api.getComplaintTimeline(id);
        setTimeline(tlData.timeline || []);
      } catch {
        setTimeline([]);
      }
    } catch (error) {
      toast.error('Complaint not found');
      navigate('/dashboard');
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status) => {
    const s = (status || '').toLowerCase();
    if (s === 'submitted' || s === 'pending') return <FaClock className="text-yellow-500" />;
    if (s === 'verified') return <FaCheckCircle className="text-blue-500" />;
    if (s === 'assigned') return <FaBuilding className="text-purple-500" />;
    if (s === 'in_progress' || s === 'in-progress') return <FaUser className="text-orange-500" />;
    if (s === 'resolved') return <FaCheckCircle className="text-green-500" />;
    if (s === 'escalated') return <FaExclamationTriangle className="text-red-500" />;
    return <FaClock className="text-gray-500" />;
  };

  const getStatusColor = (status) => {
    const s = (status || '').toLowerCase();
    if (s === 'submitted' || s === 'pending') return 'bg-yellow-100 text-yellow-800';
    if (s === 'verified') return 'bg-blue-100 text-blue-800';
    if (s === 'assigned') return 'bg-purple-100 text-purple-800';
    if (s === 'in_progress' || s === 'in-progress') return 'bg-orange-100 text-orange-800';
    if (s === 'resolved') return 'bg-green-100 text-green-800';
    if (s === 'escalated') return 'bg-red-100 text-red-800';
    return 'bg-gray-100 text-gray-800';
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center pt-20">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-netaji-saffron" />
      </div>
    );
  }

  const status = complaint.current_status || complaint.status || 'submitted';
  const complaintNumber = complaint.complaint_number || complaint.complaint_id || id;
  const timestamp = complaint.created_at || complaint.timestamp;
  const problem = complaint.title || complaint.problem || complaint.description || '—';
  const address = complaint.address || (complaint.latitude && complaint.longitude ? 'Location captured' : '—');
  const location = (complaint.latitude != null && complaint.longitude != null)
    ? { lat: complaint.latitude, lng: complaint.longitude }
    : complaint.location;
  const photo = (complaint.attachments && complaint.attachments[0]) || complaint.attachment_urls?.[0] || complaint.photo;
  const timelineMapped = timeline.map((item) => ({
    status: item.new_status || item.status,
    description: item.notes || `${(item.old_status || '').replace(/_/g, ' ')} → ${(item.new_status || '').replace(/_/g, ' ')}`,
    timestamp: item.created_at
  }));

  return (
    <div className="min-h-screen bg-gray-50 pt-20 pb-12">
      <div className="container mx-auto px-4 max-w-4xl">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="bg-white rounded-3xl shadow-xl p-8"
        >
          <div className="flex justify-between items-start mb-8">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Complaint #{complaintNumber}</h1>
              <p className="text-gray-600 mt-2">
                Filed on {timestamp ? new Date(timestamp).toLocaleDateString() : '—'}
              </p>
            </div>
            <span className={`px-4 py-2 rounded-full text-sm font-semibold ${getStatusColor(status)}`}>
              {(status || 'SUBMITTED').toString().replace(/_/g, ' ').toUpperCase()}
            </span>
          </div>

          <div className="mb-8">
            <h2 className="text-xl font-semibold mb-4">Status Timeline</h2>
            <div className="space-y-4">
              {timelineMapped.length === 0 ? (
                <p className="text-gray-500">Abhi tak koi status update nahi.</p>
              ) : (
                timelineMapped.map((event, index) => (
                  <div key={index} className="flex items-start gap-4">
                    <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gray-100 flex items-center justify-center">
                      {getStatusIcon(event.status)}
                    </div>
                    <div className="flex-1">
                      <p className="font-medium">{event.description}</p>
                      <p className="text-sm text-gray-500">
                        {event.timestamp ? new Date(event.timestamp).toLocaleString() : ''}
                      </p>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          <div className="grid md:grid-cols-2 gap-6 mb-8">
            <div className="bg-gray-50 p-4 rounded-xl">
              <h3 className="font-semibold text-gray-700 mb-2">Problem</h3>
              <p className="text-gray-600">{problem}</p>
            </div>
            <div className="bg-gray-50 p-4 rounded-xl">
              <h3 className="font-semibold text-gray-700 mb-2">Location</h3>
              <p className="text-gray-600">{address}</p>
              {location && (
                <a
                  href={`https://maps.google.com/?q=${location.lat},${location.lng}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-netaji-saffron hover:underline text-sm mt-2 inline-block"
                >
                  View on Map →
                </a>
              )}
            </div>
            {photo && (
              <div className="bg-gray-50 p-4 rounded-xl">
                <h3 className="font-semibold text-gray-700 mb-2">Photo Evidence</h3>
                <img
                  src={typeof photo === 'string' ? photo : photo.url}
                  alt="Complaint"
                  className="rounded-lg max-h-48 object-cover"
                />
              </div>
            )}
            <div className="bg-gray-50 p-4 rounded-xl">
              <h3 className="font-semibold text-gray-700 mb-2">Assigned To</h3>
              <p className="text-gray-600">{complaint.assigned_to || complaint.assignedTo || 'Not assigned yet'}</p>
              {complaint.department && (
                <p className="text-sm text-gray-500 mt-1">Department: {complaint.department}</p>
              )}
            </div>
          </div>

          {complaint.escalation_level > 0 && (
            <div className="bg-red-50 border border-red-200 rounded-xl p-4 mb-8">
              <h3 className="font-semibold text-red-800 flex items-center gap-2">
                <FaExclamationTriangle />
                Escalated to Level {complaint.escalation_level}
              </h3>
              <p className="text-red-600 text-sm mt-1">
                {complaint.escalation_reason || complaint.escalationReason || 'No response within SLA'}
              </p>
            </div>
          )}

          <div className="flex gap-4">
            <button
              type="button"
              onClick={() => navigate('/dashboard')}
              className="px-6 py-3 border-2 border-gray-300 rounded-full font-semibold hover:bg-gray-50 transition flex-1"
            >
              Back to Dashboard
            </button>
            <button
              type="button"
              onClick={loadComplaint}
              className="px-6 py-3 bg-netaji-saffron text-white rounded-full font-semibold hover:bg-netaji-green transition flex-1"
            >
              Refresh Status
            </button>
          </div>
        </motion.div>
      </div>
    </div>
  );
};

export default ComplaintStatus;
