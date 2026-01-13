import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import api from '../services/api';
import { useAuth } from '../context/AuthContext';
import { Plus, Database, ChevronRight, LogOut, Terminal, Activity, X, Trash2, Loader2 } from 'lucide-react';
import { motion } from 'framer-motion';

const Dashboard = () => {
  const [workspaces, setWorkspaces] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newWorkspaceName, setNewWorkspaceName] = useState('');
  const [isCreating, setIsCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const { user, logout } = useAuth();

  useEffect(() => {
    fetchWorkspaces();
  }, []);

  const fetchWorkspaces = async () => {
    try {
      const response = await api.get('/workspaces');
      if (response.data.success) {
        setWorkspaces(response.data.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch workspaces', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateWorkspace = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newWorkspaceName.trim()) return;
    
    setIsCreating(true);
    try {
      const res = await api.post('/workspaces', { name: newWorkspaceName });
      if (res.data.success) {
        setWorkspaces([...workspaces, res.data.data]);
        setShowCreateModal(false);
        setNewWorkspaceName('');
      }
    } catch (error) {
      console.error('Failed to create workspace', error);
    } finally {
        setIsCreating(false);
    }
  };

  const handleDeleteWorkspace = async (e: React.MouseEvent, id: string) => {
    e.preventDefault(); // Prevent navigation
    e.stopPropagation();
    
    if (!window.confirm('Are you sure you want to delete this workspace?')) return;

    setDeletingId(id);
    try {
      await api.delete(`/workspaces/${id}`);
      setWorkspaces(workspaces.filter(w => w.id !== id));
    } catch (error) {
       console.error('Failed to delete workspace', error);
    } finally {
        setDeletingId(null);
    }
  };

  return (
    <div className="min-h-screen bg-background p-6">
      {/* Header */}
      <header className="flex justify-between items-center mb-10 max-w-6xl mx-auto">
        <div className="flex items-center gap-3">
          <div className="bg-primary/20 p-2 rounded-lg">
            <Terminal className="w-6 h-6 text-primary" />
          </div>
          <h1 className="text-xl font-bold">Text-to-SQL Platform</h1>
        </div>
        <div className="flex items-center gap-4">
          <span className="text-gray-400 text-sm hidden md:block">Signed in as <span className="text-white">{user?.email}</span></span>
          <button 
            onClick={logout}
            className="btn-secondary flex items-center gap-2 text-sm"
          >
            <LogOut className="w-4 h-4" />
            Logout
          </button>
        </div>
      </header>

      <main className="max-w-6xl mx-auto">
        <div className="flex justify-between items-end mb-6">
          <div>
            <h2 className="text-3xl font-bold mb-2">Workspaces</h2>
            <p className="text-gray-400">Manage your data environments and query contexts</p>
          </div>
          <button 
            onClick={() => setShowCreateModal(true)} 
            className="btn-primary flex items-center gap-2"
          >
            <Plus className="w-4 h-4" />
            New Workspace
          </button>
        </div>

        {loading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {[1, 2, 3].map((i) => (
              <div key={i} className="glass-card h-48 animate-pulse bg-white/5"></div>
            ))}
          </div>
        ) : workspaces.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 bg-white/5 rounded-2xl border border-white/10 backdrop-blur-sm dashed-border">
            <div className="w-20 h-20 bg-primary/10 rounded-full flex items-center justify-center mb-6">
              <Database className="w-10 h-10 text-primary" />
            </div>
            <h3 className="text-xl font-bold mb-2">No Workspaces Yet</h3>
            <p className="text-gray-400 max-w-md text-center mb-8">
              Create your first workspace to start connecting to databases and asking questions.
            </p>
            <button 
              onClick={() => setShowCreateModal(true)} 
              className="btn-primary flex items-center gap-2"
            >
              <Plus className="w-5 h-5" />
              Create Workspace
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {workspaces.map((ws, i) => (
              <motion.div
                key={ws.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: i * 0.1 }}
              >
                <Link to={`/workspace/${ws.id}`} className="block group relative">
                  <div className="glass-card p-6 h-full hover:bg-white/5 transition-colors border-white/5 hover:border-primary/30 relative overflow-hidden">
                    <div className="absolute top-0 right-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity">
                      <Database className="w-24 h-24" />
                    </div>
                    
                    <button 
                        onClick={(e) => handleDeleteWorkspace(e, ws.id)}
                        disabled={deletingId === ws.id}
                        className="absolute top-4 right-4 p-2 bg-red-500/10 hover:bg-red-500 text-red-500 hover:text-white rounded-lg opacity-0 group-hover:opacity-100 transition-all z-10"
                    >
                        {deletingId === ws.id ? <Loader2 className="w-4 h-4 animate-spin"/> : <Trash2 className="w-4 h-4" />}
                    </button>

                    <div className="flex justify-between items-start mb-4">
                      <div className="bg-gradient-to-br from-gray-800 to-black p-3 rounded-xl border border-white/5">
                        <Database className="w-6 h-6 text-primary" />
                      </div>
                      <span className="text-xs font-mono text-gray-500 bg-black/20 px-2 py-1 rounded">
                        ID: {ws.id.slice(0, 8)}
                      </span>
                    </div>

                    <h3 className="text-xl font-semibold mb-2 group-hover:text-primary transition-colors truncate pr-8">
                      {ws.name}
                    </h3>
                    
                    <div className="flex items-center gap-4 text-sm text-gray-400 mt-4">
                      <span className="flex items-center gap-1">
                        <Activity className="w-4 h-4" />
                        Active
                      </span>
                      <span>•</span>
                      <span>{new Date(ws.created_at).toLocaleDateString()}</span>
                    </div>

                    <div className="mt-6 flex items-center text-sm font-medium text-primary opacity-0 group-hover:opacity-100 transform translate-y-2 group-hover:translate-y-0 transition-all">
                      Open Workspace <ChevronRight className="w-4 h-4 ml-1" />
                    </div>
                  </div>
                </Link>
              </motion.div>
            ))}

            {/* Empty State / Create New Card */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: workspaces.length * 0.1 }}
              className="group cursor-pointer"
              onClick={() => setShowCreateModal(true)}
            >
              <div className="glass-card p-6 h-full border-dashed border-white/20 hover:border-primary/50 hover:bg-primary/5 transition-all flex flex-col items-center justify-center text-center">
                <div className="bg-white/5 p-4 rounded-full mb-4 group-hover:scale-110 transition-transform">
                  <Plus className="w-8 h-8 text-gray-400 group-hover:text-primary" />
                </div>
                <h3 className="text-lg font-medium text-gray-300 group-hover:text-white">Create New Workspace</h3>
                <p className="text-sm text-gray-500 mt-2">Connect a new database and start querying</p>
              </div>
            </motion.div>
          </div>
        )}
      </main>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <motion.div 
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="glass-card w-full max-w-md p-6"
          >
            <div className="flex justify-between items-center mb-6">
                <h3 className="text-xl font-bold">Create New Workspace</h3>
                <button onClick={() => setShowCreateModal(false)} className="p-2 hover:bg-white/10 rounded-lg">
                    <X className="w-5 h-5 text-gray-400" />
                </button>
            </div>
            
            <form onSubmit={handleCreateWorkspace}>
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">Workspace Name</label>
                        <input 
                            type="text" 
                            value={newWorkspaceName}
                            onChange={(e) => setNewWorkspaceName(e.target.value)}
                            className="glass-input w-full"
                            placeholder="e.g. Sales Analytics"
                            autoFocus
                        />
                    </div>
                    
                    <button 
                        type="submit" 
                        disabled={!newWorkspaceName.trim() || isCreating}
                        className="btn-primary w-full flex items-center justify-center gap-2 mt-6"
                    >
                        {isCreating ? <Loader2 className="w-4 h-4 animate-spin"/> : <Plus className="w-4 h-4" />}
                        Create Workspace
                    </button>
                </div>
            </form>
          </motion.div>
        </div>
      )}
    </div>
  );
};

export default Dashboard;
