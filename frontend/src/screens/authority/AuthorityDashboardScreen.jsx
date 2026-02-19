import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { authorityApi, getAuthorityToken } from '../../services/authorityApi';

export default function AuthorityDashboardScreen() {
  const navigate = useNavigate();
  const [complaints, setComplaints] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!getAuthorityToken()) {
      navigate('/authority/login', { replace: true });
      return;
    }
    let cancelled = false;
    setLoading(true);
    authorityApi.complaints({ page, page_size: pageSize, status: statusFilter === 'all' ? undefined : statusFilter })
      .then((res) => {
        if (!cancelled) {
          setComplaints(res.complaints || []);
          setTotal(res.total ?? 0);
        }
      })
      .catch((err) => { if (!cancelled) setError(err.message); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [navigate, page, pageSize, statusFilter]);

  const stats = {
    total: total,
    pending: complaints.filter(c => c.current_status === 'submitted').length,
    inProgress: complaints.filter(c => c.current_status === 'in_progress').length,
    resolved: complaints.filter(c => c.current_status === 'resolved').length,
  };

  const handleLogout = () => {
    localStorage.removeItem('authority_token');
    navigate('/authority/login', { replace: true });
  };

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold text-gray-900">Authority Dashboard</h1>
          <Button variant="secondary" onClick={handleLogout}>
            Logout
          </Button>
        </div>
        
        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
          {Object.entries(stats).map(([key, value]) => (
            <motion.div
              key={key}
              whileHover={{ y: -2 }}
              className="bg-white rounded-xl shadow-sm p-6 border border-gray-100"
            >
              <p className="text-sm text-gray-500 capitalize">{key}</p>
              <p className="text-2xl font-bold text-gray-900">{value}</p>
            </motion.div>
          ))}
        </div>

        {/* Filters */}
        <div className="flex space-x-2 mb-4">
          {['all', 'submitted', 'under_review', 'in_progress', 'resolved'].map((status) => (
            <Button
              key={status}
              variant={statusFilter === status ? 'primary' : 'secondary'}
              onClick={() => { setStatusFilter(status); setPage(1); }}
              className="text-sm"
            >
              {status.replace('_', ' ')}
            </Button>
          ))}
        </div>

        {/* Error */}
        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-4">
            {error}
          </div>
        )}

        {/* Loading */}
        {loading ? (
          <div className="text-center py-12">
            <p className="text-gray-500">Loading...</p>
          </div>
        ) : (
          <>
            {/* Complaint List */}
            <div className="space-y-4 mb-6">
              {complaints
                .filter(c => statusFilter === 'all' || c.current_status === statusFilter)
                .map(complaint => (
                  <ComplaintCard
                    key={complaint.complaint_id}
                    complaint={{
                      complaint_number: complaint.complaint_number,
                      current_status: complaint.current_status,
                      created_at: complaint.created_at,
                      title: complaint.title
                    }}
                    onClick={() => navigate(`/authority/complaints/${complaint.complaint_id}`, { state: { complaint } })}
                  />
                ))}
            </div>

            {/* Pagination */}
            {total > pageSize && (
              <div className="flex justify-center items-center space-x-4 mt-6">
                <Button
                  variant="secondary"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                >
                  Prev
                </Button>
                <span className="text-gray-600">Page {page}</span>
                <Button
                  variant="secondary"
                  disabled={page * pageSize >= total}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
