/**
 * Generates complaint email content for notification/CC.
 * Backend may use this structure when email integration is added.
 */
export const generateComplaintEmail = (complaint) => {
  const {
    complaintNumber,
    problem,
    location,
    address,
    photo,
    voiceNote,
    timestamp,
    phoneNumber,
    category
  } = complaint;

  const loc = location || {};
  const lat = loc.lat ?? loc.latitude;
  const lng = loc.lng ?? loc.longitude;
  const mapLink =
    lat != null && lng != null
      ? `https://maps.google.com/?q=${lat},${lng}`
      : '';

  return {
    subject: `AI NETA Complaint #${complaintNumber || '—'} - ${category || 'General'}`,
    html: `
      <h2>नई शिकायत / New Complaint</h2>
      <p><strong>Complaint #:</strong> ${complaintNumber || '—'}</p>
      <p><strong>Date:</strong> ${timestamp ? new Date(timestamp).toLocaleString() : '—'}</p>
      <p><strong>Phone:</strong> ${phoneNumber || '—'}</p>
      <hr/>
      <h3>समस्या / Problem:</h3>
      <p>${problem || '—'}</p>
      <h3>स्थान / Location:</h3>
      <p>📍 ${address || '—'}</p>
      ${lat != null && lng != null ? `<p>🌐 ${lat}, ${lng}</p>` : ''}
      ${mapLink ? `<p>🔗 <a href="${mapLink}">View on Map</a></p>` : ''}
      ${photo ? `<h3>फोटो / Photo:</h3><img src="${photo}" width="300" alt="Complaint"/>` : ''}
      ${voiceNote ? `<h3>वॉइस नोट / Voice Note:</h3><audio controls src="${voiceNote}"></audio>` : ''}
      <hr/>
      <p><strong>Track your complaint:</strong> https://aineta.com/complaint/${complaintNumber || ''}</p>
      <p><em>This is an AI NETA platform automated message</em></p>
    `
  };
};
