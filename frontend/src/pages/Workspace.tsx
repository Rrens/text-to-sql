import { useState, useEffect, useRef } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import api from '../services/api';
import { userService } from '../services/user';
import { useAuth } from '../context/AuthContext';
import { 
    Send, Database, ArrowLeft, Code, Table, Clock, Bot, Sparkles, Loader2, StopCircle, 
    Plus, X, Settings, MessageSquare, Trash2, Edit2, Check, AlertCircle, Save, User, Menu, Key
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import clsx from 'clsx';
import Sidebar from '../components/Sidebar';
import { ChartVisualizer } from '../components/ChartVisualizer';
import ErrorBoundary from '../components/ErrorBoundary';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  sql?: string;
  result?: any;
  error?: string;
  metadata?: any;
  timestamp: Date;
}

interface Connection {
    id: string;
    name: string;
    database_type: string;
    host: string;
    port: number;
    database: string;
    username: string;
    ssl_mode: string;
    max_rows: number;
    created_at: string;
}

interface Session {
    id: string;
    title: string;
    created_at: string;
}

const Workspace = () => {
  const { workspaceId } = useParams();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<'chat' | 'connections' | 'settings'>('chat');
  
  // Workspace State
  const [workspaceName, setWorkspaceName] = useState('Workspace');
  const [isLoadingWorkspace, setIsLoadingWorkspace] = useState(true);
  const [workspaceError, setWorkspaceError] = useState<string | null>(null);

  // Session State
  const [sessions, setSessions] = useState<Session[]>([]);
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
  const [isLoadingSessions, setIsLoadingSessions] = useState(false);
  const [showSidebar, setShowSidebar] = useState(true);

  // Suggestions State
  const [suggestions, setSuggestions] = useState<string[]>([]);


  // Chat State
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Connection State
  const [connections, setConnections] = useState<Connection[]>([]);
  const [selectedConnection, setSelectedConnection] = useState<string>('');
  const [isLoadingConnections, setIsLoadingConnections] = useState(false);
  
  // Connection CRUD State
  const [showConnectionModal, setShowConnectionModal] = useState(false);
  const [editingConnection, setEditingConnection] = useState<Connection | null>(null);
  const [connectionForm, setConnectionForm] = useState({
    name: '',
    type: 'postgres',
    host: 'localhost',
    port: '5432',
    user: '',
    password: '',
    dbname: '',
    sslMode: 'disable',
    maxRows: 1000,
    timeout: 30
  });
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [isSubmittingConnection, setIsSubmittingConnection] = useState(false);
  const [deletingConnectionId, setDeletingConnectionId] = useState<string | null>(null);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [testConnectionResult, setTestConnectionResult] = useState<{ success: boolean; message: string } | null>(null);
  const [sqliteInputMode, setSqliteInputMode] = useState<'upload' | 'url'>('upload');
  const [isUploadingSqlite, setIsUploadingSqlite] = useState(false);
  const [uploadedFileName, setUploadedFileName] = useState<string>('');

  // LLM State
  const [providers, setProviders] = useState<any[]>([]);
  const [selectedProvider, setSelectedProvider] = useState<string>('ollama');
  const [selectedModel, setSelectedModel] = useState<string>('qwen2.5-coder:7b');

  // LLM Config State
  const { user, login } = useAuth(); // Need login to update user in context
  const [llmConfigForm, setLlmConfigForm] = useState({
      ollama_host: '',
      openai_key: '',
      anthropic_key: '',
      deepseek_key: '',
      gemini_key: ''
  });
  const [isSavingLLM, setIsSavingLLM] = useState(false);
  const [isLLMSaved, setIsLLMSaved] = useState(false);

  useEffect(() => {
    if (isLLMSaved) {
        const timer = setTimeout(() => setIsLLMSaved(false), 3000);
        return () => clearTimeout(timer);
    }
  }, [isLLMSaved]);

  useEffect(() => {
      if (user?.llm_config) {
          setLlmConfigForm({
              ollama_host: (user.llm_config.ollama as any)?.host || '',
              openai_key: (user.llm_config.openai as any)?.api_key || '',
              anthropic_key: (user.llm_config.anthropic as any)?.api_key || '',
              deepseek_key: (user.llm_config.deepseek as any)?.api_key || '',
              gemini_key: (user.llm_config.gemini as any)?.api_key || ''
          });
      }
  }, [user]);

  const handleUpdateLLMConfig = async (e: React.FormEvent) => {
      e.preventDefault();
      setIsSavingLLM(true);
      try {
          const config = {
              ollama: { host: llmConfigForm.ollama_host },
              openai: { api_key: llmConfigForm.openai_key },
              anthropic: { api_key: llmConfigForm.anthropic_key },
              deepseek: { api_key: llmConfigForm.deepseek_key },
              gemini: { api_key: llmConfigForm.gemini_key }
          };
          const updatedUser = await userService.updateLLMConfig(config);
          // Update user in context (we need token, assume it's same)
          if (token && updatedUser && updatedUser.data) {
              login(token, updatedUser.data);
          }
          setIsLLMSaved(true);
      } catch (error) {
          console.error("Failed to update LLM config", error);
      } finally {
          setIsSavingLLM(false);
      }
  };

  useEffect(() => {
    fetchWorkspaceDetails();
    fetchConnections();
    fetchProviders();
    fetchProviders();
    fetchSessions(); 
    fetchSuggestions();
  }, [workspaceId]);

  useEffect(() => {
    // Reset model when provider changes
    const provider = providers.find(p => p.name === selectedProvider);
    if (provider && provider.models.length > 0) {
        if (!provider.models.includes(selectedModel)) {
            setSelectedModel(provider.models[0]);
        }
    }
  }, [selectedProvider, providers]);

  useEffect(() => {
    scrollToBottom();
  }, [messages, activeTab]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const fetchWorkspaceDetails = async () => {
    setIsLoadingWorkspace(true);
    setWorkspaceError(null);
    try {
      const res = await api.get(`/workspaces/${workspaceId}`);
      if (res.data.success) {
        setWorkspaceName(res.data.data.name);
      }
    } catch (err: any) {
      console.error(err);
      const status = err.response?.status;
      if (status === 403) {
        setWorkspaceError('You do not have access to this workspace.');
      } else if (status === 404) {
        setWorkspaceError('Workspace not found.');
      } else {
        setWorkspaceError('Failed to load workspace. Please try again.');
      }
    } finally {
      setIsLoadingWorkspace(false);
    }
  };

  const fetchConnections = async () => {
    setIsLoadingConnections(true);
    try {
      const res = await api.get(`/workspaces/${workspaceId}/connections`);
      if (res.data.success) {
        setConnections(res.data.data);
        if (res.data.data.length > 0 && !selectedConnection) {
          setSelectedConnection(res.data.data[0].id);
        }
      }
    } catch (err) {
      console.error(err);
    } finally {
        setIsLoadingConnections(false);
    }
  };

  const fetchProviders = async () => {
    try {
      const res = await api.get('/llm-providers');
      if (res.data.success) {
        setProviders(res.data.data.providers);
        if (res.data.data.default_provider) {
             setSelectedProvider(res.data.data.default_provider);
        }
      }
    } catch (err) {
      console.error(err);
    }
  };

  const fetchSessions = async () => {
    setIsLoadingSessions(true);
    try {
        const res = await api.get(`/workspaces/${workspaceId}/sessions`);
        if (res.data.success) {
            setSessions(res.data.data || []);
        }
    } catch (err) {
        console.error("Failed to fetch sessions", err);
    } finally {
        setIsLoadingSessions(false);
    }
  };

  const fetchSuggestions = async () => {

    try {
        const res = await api.get(`/workspaces/${workspaceId}/suggestions`);
        if (res.data.success && res.data.data && res.data.data.length > 0) {
            setSuggestions(res.data.data);
        } else {
             // Fallback default suggestions if none found
             setSuggestions(['Show total revenue by month', 'Top 5 customers in 2024', 'Count active users', 'List orders > $500']);
        }
    } catch (err) {
        console.error("Failed to fetch suggestions", err);
        // Fallback on error
        setSuggestions(['Show total revenue by month', 'Top 5 customers in 2024', 'Count active users', 'List orders > $500']);
    } finally {

    }
  };

  const fetchSessionHistory = async (sessionId: string) => {
    setLoading(true);
    try {
      const res = await api.get(`/workspaces/${workspaceId}/sessions/${sessionId}`);
      if (res.data.success) {
        // Map backend messages to frontend format
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const history = res.data.data.map((msg: any) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          sql: msg.sql,
          result: msg.result,
          error: msg.error,
          metadata: msg.metadata,
          timestamp: new Date(msg.created_at)
        }));
        setMessages(history);
      }
    } catch (err) {
      console.error("Failed to fetch chat history", err);
    } finally {
        setLoading(false);
    }
  };

  const handleSelectSession = (sessionId: string) => {
      if (currentSessionId === sessionId) return;
      setCurrentSessionId(sessionId);
      fetchSessionHistory(sessionId);
      setActiveTab('chat');
  };

  const handleNewChat = () => {
      setCurrentSessionId(null);
      setMessages([]);
      setActiveTab('chat');
  };
  
  const handleDeleteSession = async (sessionId: string) => {
    if (!window.confirm("Delete this chat session?")) return;
    try {
        await api.delete(`/workspaces/${workspaceId}/sessions/${sessionId}`);
        setSessions(sessions.filter(s => s.id !== sessionId));
        if (currentSessionId === sessionId) {
            handleNewChat();
        }
    } catch (err) {
        console.error("Failed to delete session", err);
    }
  };

  // --- Connection Management ---

  const openCreateModal = () => {
      setEditingConnection(null);
      setConnectionForm({
        name: '',
        type: 'postgres',
        host: 'localhost',
        port: '5432',
        user: '',
        password: '',
        dbname: '',
        sslMode: 'disable',
        maxRows: 1000,
        timeout: 30
      });
      setShowAdvanced(false);
      setShowConnectionModal(true);
  };

  const openEditModal = (conn: Connection) => {
      setEditingConnection(conn);
      setConnectionForm({
          name: conn.name,
          type: conn.database_type,
          host: conn.host,
          port: conn.port.toString(),
          user: conn.username,
          password: '', // Don't fill password for security
          dbname: conn.database,
          sslMode: conn.ssl_mode || 'disable',
          maxRows: conn.max_rows || 1000,
          timeout: 30
      });
      setShowAdvanced(false);
      setShowConnectionModal(true);
  };

  const handleConnectionSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmittingConnection(true);

    try {
        const isSQLite = connectionForm.type === 'sqlite';
        const payload = {
            name: connectionForm.name,
            database_type: connectionForm.type,
            host: isSQLite ? 'localhost' : connectionForm.host,
            port: isSQLite ? 1 : parseInt(connectionForm.port),
            database: connectionForm.dbname,
            username: isSQLite ? 'sqlite' : connectionForm.user,
            password: isSQLite ? 'sqlite' : connectionForm.password,
            ssl_mode: isSQLite ? 'disable' : connectionForm.sslMode,
            max_rows: parseInt(connectionForm.maxRows.toString()),
            timeout_seconds: parseInt(connectionForm.timeout.toString())
        };

        let res;
        if (editingConnection) {
            res = await api.patch(`/workspaces/${workspaceId}/connections/${editingConnection.id}`, payload);
        } else {
            if (!isSQLite && !connectionForm.password) {
                 alert("Password is required for new connections");
                 setIsSubmittingConnection(false);
                 return;
            }
            res = await api.post(`/workspaces/${workspaceId}/connections`, payload);
        }

        if (res.data.success) {
            if (editingConnection) {
                setConnections(connections.map(c => c.id === editingConnection.id ? res.data.data : c));
            } else {
                setConnections([...connections, res.data.data]);
                if (!selectedConnection) setSelectedConnection(res.data.data.id);
            }
            setShowConnectionModal(false);
        }
    } catch (error) {
        console.error("Failed to save connection", error);
        alert("Failed to save connection");
    } finally {
        setIsSubmittingConnection(false);
    }
  };

  const handleDeleteConnection = async (id: string) => {
      if (!window.confirm("Are you sure you want to delete this connection?")) return;
      setDeletingConnectionId(id);
      try {
          await api.delete(`/workspaces/${workspaceId}/connections/${id}`);
          setConnections(connections.filter(c => c.id !== id));
          if (selectedConnection === id) {
              setSelectedConnection(connections.find(c => c.id !== id)?.id || '');
          }
      } catch (error) {
          console.error("Failed to delete connection", error);
      } finally {
          setDeletingConnectionId(null);
      }
  };

  const handleTestConnection = async () => {
      setIsTestingConnection(true);
      setTestConnectionResult(null);
      try {
          const isSQLiteTest = connectionForm.type === 'sqlite';
          const payload = {
              name: connectionForm.name || 'test',
              database_type: connectionForm.type,
              host: isSQLiteTest ? 'localhost' : connectionForm.host,
              port: isSQLiteTest ? 1 : parseInt(connectionForm.port),
              database: connectionForm.dbname,
              username: isSQLiteTest ? 'sqlite' : connectionForm.user,
              password: isSQLiteTest ? 'sqlite' : connectionForm.password,
              ssl_mode: isSQLiteTest ? 'disable' : connectionForm.sslMode,
              max_rows: parseInt(connectionForm.maxRows.toString()),
              timeout_seconds: parseInt(connectionForm.timeout.toString())
          };
          // Use a dummy connection id for test endpoint
          const res = await api.post(`/workspaces/${workspaceId}/connections/00000000-0000-0000-0000-000000000000/test`, payload);
          if (res.data.data?.connected) {
              setTestConnectionResult({ success: true, message: 'Connection successful!' });
          } else {
              setTestConnectionResult({ success: false, message: res.data.data?.error || 'Connection failed' });
          }
      } catch (error: unknown) {
          const err = error as { response?: { data?: { error?: string | { error?: string } } } };
          let errorMessage = 'Connection test failed';
          const backendError = err.response?.data?.error;
          
          if (typeof backendError === 'string') {
              errorMessage = backendError;
          } else if (typeof backendError === 'object' && backendError?.error) {
              errorMessage = backendError.error;
          }
          
          setTestConnectionResult({ success: false, message: errorMessage });
      } finally {
          setIsTestingConnection(false);
      }
  };

  // No Connection Modal Logic
  const [showNoConnectionModal, setShowNoConnectionModal] = useState(false);

  const handleCreateConnectionFromModal = () => {
      setShowNoConnectionModal(false);
      setActiveTab('connections');
      openCreateModal();
  };

  const handleChatSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (connections.length === 0) {
        setShowNoConnectionModal(true);
        return;
    }

    if (!input.trim() || !selectedConnection) return;

    const userMsg: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: input,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMsg]);
    setInput('');
    setLoading(true);

    try {
      const res = await api.post(`/workspaces/${workspaceId}/query`, {
        session_id: currentSessionId, 
        connection_id: selectedConnection,
        question: userMsg.content,
        execute: true,
        llm_provider: selectedProvider,
        llm_model: selectedModel
      });

      if (res.data.success) {
        const data = res.data.data;
        const assistantMsg: Message = {
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: data.explanation || 'Here is the result:',
          sql: data.sql,
          result: data.result,
          metadata: data.metadata,
          timestamp: new Date(),
        };
        setMessages(prev => [...prev, assistantMsg]);
        
        // Update session ID if it was a new chat
        if (!currentSessionId && data.session_id) {
            setCurrentSessionId(data.session_id);
            // Refresh sessions list to show new session
            fetchSessions();
        }
      }
    } catch (err: any) {
      const errorMsg: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: 'I encountered an error processing your request.',
        error: err.response?.data?.error || err.message,
        timestamp: new Date(),
      };
      setMessages(prev => [...prev, errorMsg]);
    } finally {
      setLoading(false);
    }
  };

  // --- Workspace Management (Settings) ---
  const handleUpdateWorkspace = async (e: React.FormEvent) => {
      e.preventDefault();
      try {
           /* await api.patch(`/workspaces/${workspaceId}`, { name: workspaceName }); */
           alert("Workspace updated (mock)");
      } catch (error) {
          console.error(error);
      }
  };
  
  const handleDeleteWorkspace = async () => {
      if (!window.confirm("Delete this workspace permanently?")) return;
      try {
          await api.delete(`/workspaces/${workspaceId}`);
          navigate('/');
      } catch (error) {
          console.error(error);
      }
  };


  if (isLoadingWorkspace) {
    return (
      <div className="h-screen bg-background flex items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="w-10 h-10 animate-spin text-primary" />
          <p className="text-gray-400">Loading workspace...</p>
        </div>
      </div>
    );
  }

  if (workspaceError) {
    return (
      <div className="h-screen bg-background flex items-center justify-center">
        <div className="flex flex-col items-center gap-4 text-center max-w-md">
          <AlertCircle className="w-12 h-12 text-red-400" />
          <h2 className="text-xl font-semibold text-white">Unable to Load Workspace</h2>
          <p className="text-gray-400">{workspaceError}</p>
          <div className="flex gap-3">
            <button onClick={() => navigate('/')} className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg transition-colors">
              Go Home
            </button>
            <button onClick={() => { setWorkspaceError(null); fetchWorkspaceDetails(); }} className="px-4 py-2 bg-primary hover:bg-primary/80 text-white rounded-lg transition-colors">
              Retry
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="h-screen bg-background flex overflow-hidden">
        {/* Sidebar (Desktop) */}
        <div className={clsx(
            "fixed inset-y-0 left-0 z-30 transform transition-transform duration-300 ease-in-out md:relative md:translate-x-0",
            showSidebar ? "translate-x-0" : "-translate-x-full",
            "md:w-64 shrink-0"
        )}>
            <Sidebar 
                sessions={sessions}
                currentSessionId={currentSessionId}
                onSelectSession={(id) => {
                    handleSelectSession(id);
                    if (window.innerWidth < 768) setShowSidebar(false);
                }}
                onNewChat={() => {
                   handleNewChat();
                   if (window.innerWidth < 768) setShowSidebar(false);
                }}
                onDeleteSession={handleDeleteSession}
                isLoading={isLoadingSessions}
            />
        </div>

        {/* Sidebar Overlay (Mobile) */}
        {showSidebar && (
            <div 
                className="fixed inset-0 bg-black/50 z-20 md:hidden"
                onClick={() => setShowSidebar(false)}
            />
        )}
      
      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 h-full overflow-hidden">
        {/* Header */}
        <header className="h-16 border-b border-white/10 flex items-center justify-between px-6 bg-surface/50 backdrop-blur shrink-0 z-10">
            <div className="flex items-center gap-4">
                <button 
                    onClick={() => setShowSidebar(!showSidebar)}
                    className="md:hidden p-2 hover:bg-white/5 rounded-lg text-gray-400"
                >
                    <Menu className="w-5 h-5" />
                </button>

            <Link to="/" className="p-2 hover:bg-white/5 rounded-full transition-colors">
                <ArrowLeft className="w-5 h-5 text-gray-400" />
            </Link>
            <div>
                <h1 className="font-semibold text-lg">{workspaceName}</h1>
                <div className="flex items-center gap-2 text-xs text-gray-400">
                <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></span>
                Online
                </div>
            </div>
            </div>

            {/* Tab Navigation */}
            <div className="flex bg-black/20 p-1 rounded-lg border border-white/5">
                <button 
                    onClick={() => setActiveTab('chat')}
                    className={clsx(
                        "px-4 py-1.5 rounded-md text-sm font-medium transition-all flex items-center gap-2",
                        activeTab === 'chat' ? "bg-primary text-white shadow-lg" : "text-gray-400 hover:text-white hover:bg-white/5"
                    )}
                >
                    <MessageSquare className="w-4 h-4" />
                    Chat
                </button>
                <button 
                    onClick={() => setActiveTab('connections')}
                    className={clsx(
                        "px-4 py-1.5 rounded-md text-sm font-medium transition-all flex items-center gap-2",
                        activeTab === 'connections' ? "bg-primary text-white shadow-lg" : "text-gray-400 hover:text-white hover:bg-white/5"
                    )}
                >
                    <Database className="w-4 h-4" />
                    Connections
                </button>
                <button 
                    onClick={() => setActiveTab('settings')}
                    className={clsx(
                        "px-4 py-1.5 rounded-md text-sm font-medium transition-all flex items-center gap-2",
                        activeTab === 'settings' ? "bg-primary text-white shadow-lg" : "text-gray-400 hover:text-white hover:bg-white/5"
                    )}
                >
                    <Settings className="w-4 h-4" />
                    Settings
                </button>
            </div>

            <div className="flex items-center gap-4">
                {activeTab === 'chat' && (
                    <>
                    <div className="flex items-center gap-2 bg-black/20 p-1 pr-3 rounded-lg border border-white/5">
                        <div className="bg-primary/20 p-1.5 rounded text-primary">
                        <Database className="w-4 h-4" />
                        </div>
                        <select 
                        className="bg-transparent text-sm border-none focus:ring-0 text-gray-300 outline-none w-32 cursor-pointer"
                        value={selectedConnection}
                        onChange={(e) => setSelectedConnection(e.target.value)}
                        >
                        {connections.map(c => (
                            <option key={c.id} value={c.id} className="bg-surface">{c.name}</option>
                        ))}
                        {connections.length === 0 && <option value="">No connections</option>}
                        </select>
                    </div>

                    <div className="flex items-center gap-2 bg-black/20 p-1 pr-3 rounded-lg border border-white/5">
                        <div className="bg-accent/20 p-1.5 rounded text-accent">
                        <Bot className="w-4 h-4" />
                        </div>
                        <select 
                        className="bg-transparent text-sm border-none focus:ring-0 text-gray-300 outline-none w-24 cursor-pointer"
                        value={selectedProvider}
                        onChange={(e) => setSelectedProvider(e.target.value)}
                        >
                        {providers.map(p => (
                            <option key={p.name} value={p.name} className="bg-surface">{p.name}</option>
                        ))}
                            {providers.length === 0 && <option value="ollama">ollama</option>}
                        </select>
                    </div>
                    
                    <div className="flex items-center gap-2 bg-black/20 p-1 pr-3 rounded-lg border border-white/5">
                        <div className="bg-purple-500/20 p-1.5 rounded text-purple-400">
                        <Sparkles className="w-4 h-4" />
                        </div>
                        <select 
                        className="bg-transparent text-sm border-none focus:ring-0 text-gray-300 outline-none max-w-[120px] cursor-pointer"
                        value={selectedModel}
                        onChange={(e) => setSelectedModel(e.target.value)}
                        >
                            {providers.find(p => p.name === selectedProvider)?.models.map((m: string) => (
                                <option key={m} value={m} className="bg-surface">{m}</option>
                            ))}
                            {(!providers.find(p => p.name === selectedProvider)?.models) && <option value="qwen2.5-coder:7b">Default</option>}
                        </select>
                    </div>
                    </>
                )}
            </div>
        </header>

        {/* Main Content */}
        <main className="flex-1 overflow-y-auto bg-background/50 scroll-smooth relative">
            <AnimatePresence mode="wait">
                
                {/* CHAT TAB */}
                {activeTab === 'chat' && (
                    <motion.div 
                        key="chat"
                        initial={{ opacity: 0, x: -20 }}
                        animate={{ opacity: 1, x: 0 }}
                        exit={{ opacity: 0, x: 20 }}
                        className="h-full flex flex-col"
                    >
                        <div className="flex-1 overflow-y-auto p-4 pb-0">
                            <div className="max-w-4xl mx-auto space-y-6 pb-24">
                                {messages.length === 0 ? (
                                    <div className="flex flex-col items-center justify-center h-[60vh] text-center opacity-50">
                                    <div className="w-20 h-20 bg-gradient-to-tr from-primary to-accent rounded-3xl flex items-center justify-center mb-6 shadow-2xl shadow-primary/20 animate-float">
                                        <Sparkles className="w-10 h-10 text-white" />
                                    </div>
                                    <h2 className="text-2xl font-bold mb-2">How can I help you today?</h2>
                                    <p className="text-gray-400 max-w-md">
                                        Ask questions about your data in plain English. I'll translate them to SQL and show you the results.
                                    </p>
                                    
                                    <div className="grid grid-cols-2 gap-4 mt-8 w-full max-w-lg">
                                        {suggestions.map((suggestion, i) => (
                                        <button 
                                            key={i}
                                            onClick={() => setInput(suggestion)}
                                            className="p-3 text-sm bg-white/5 hover:bg-white/10 border border-white/5 rounded-xl transition-all text-left"
                                        >
                                            {suggestion}
                                        </button>
                                        ))}
                                    </div>
                                    </div>
                                ) : (
                                    messages.map((msg) => (
                                        <motion.div 
                                            key={msg.id}
                                            initial={{ opacity: 0, y: 10 }}
                                            animate={{ opacity: 1, y: 0 }}
                                            className={clsx(
                                            "flex gap-4",
                                            msg.role === 'user' ? "flex-row-reverse" : "flex-row"
                                            )}
                                        >
                                            <div className={clsx(
                                            "w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 mt-1",
                                            msg.role === 'user' ? "bg-white/10" : "bg-primary"
                                            )}>
                                            {msg.role === 'user' ? <User className="w-5 h-5 text-white" /> : <Bot className="w-5 h-5 text-white" />}
                                            </div>

                                            <div className={clsx(
                                            "max-w-[85%] space-y-4",
                                            msg.role === 'user' ? "bg-primary text-white p-4 rounded-2xl rounded-tr-sm" : "w-full"
                                            )}>
                                            {msg.role === 'user' ? (
                                                <p>{msg.content}</p>
                                            ) : (
                                                <div className="space-y-4">
                                                {msg.content && !msg.sql && (
                                                    <div className="bg-white/5 p-4 rounded-xl text-gray-200 prose prose-invert max-w-none overflow-x-auto">
                                                        <ReactMarkdown 
                                                            remarkPlugins={[remarkGfm]}
                                                            components={{
                                                                // Custom styling for code blocks to match the theme
                                                                code({node, inline, className, children, ...props}: any) {
                                                                    const match = /language-(\w+)/.exec(className || '')
                                                                    return !inline && match ? (
                                                                        <div className="rounded-md overflow-hidden my-2 border border-white/10 bg-[#0d1117]">
                                                                            <div className="px-3 py-1 bg-white/5 border-b border-white/5 text-xs text-gray-400 font-mono">
                                                                                {match[1]}
                                                                            </div>
                                                                            <code className={clsx("block p-3 overflow-x-auto text-sm font-mono", className)} {...props}>
                                                                                {children}
                                                                            </code>
                                                                        </div>
                                                                    ) : (
                                                                        <code className={clsx("bg-white/10 px-1.5 py-0.5 rounded text-sm font-mono text-blue-300", className)} {...props}>
                                                                            {children}
                                                                        </code>
                                                                    )
                                                                },
                                                                // Style tables
                                                                table({children}: any) {
                                                                    return <div className="overflow-x-auto my-4 border border-white/10 rounded-lg"><table className="min-w-full divide-y divide-white/10 text-sm">{children}</table></div>
                                                                },
                                                                thead({children}: any) {
                                                                    return <thead className="bg-white/5">{children}</thead>
                                                                },
                                                                th({children}: any) {
                                                                    return <th className="px-4 py-2 text-left text-xs font-medium text-gray-400 uppercase tracking-wider">{children}</th>
                                                                },
                                                                td({children}: any) {
                                                                    return <td className="px-4 py-2 whitespace-nowrap text-gray-300 border-t border-white/5">{children}</td>
                                                                },
                                                                // Style links
                                                                a({children, href}: any) {
                                                                    return <a href={href} target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:text-blue-300 underline">{children}</a>
                                                                }
                                                            }}
                                                        >
                                                            {msg.content}
                                                        </ReactMarkdown>
                                                    </div>
                                                )}
                                                {msg.sql && (
                                                    <div className="bg-[#0d1117] rounded-xl border border-white/10 overflow-hidden">
                                                    <div className="px-4 py-2 bg-white/5 border-b border-white/5 flex items-center justify-between">
                                                        <div className="flex items-center gap-2 text-xs text-gray-400">
                                                        <Code className="w-4 h-4" />
                                                        Generated SQL
                                                        </div>
                                                        <div className="flex items-center gap-3 text-xs text-gray-500">
                                                            {msg.metadata && <>
                                                                <span className="flex items-center gap-1"><Clock className="w-3 h-3"/> {msg.metadata.execution_time_ms}ms</span>
                                                            </>}
                                                        </div>
                                                    </div>
                                                    <pre className="p-4 overflow-x-auto text-sm text-blue-200 font-mono">
                                                        {msg.sql}
                                                    </pre>
                                                    </div>
                                                )}

                                                {msg.result && (
                                                    <div className="bg-surface rounded-xl border border-white/10 overflow-hidden">
                                                      <div className="px-4 py-2 bg-white/5 border-b border-white/5 flex items-center justify-between text-xs text-gray-400">
                                                        <div className="flex items-center gap-2">
                                                          <Table className="w-4 h-4" />
                                                          Result ({msg.result.row_count} rows)
                                                        </div>
                                                      </div>
                                                      
                                                      <div className="p-4 border-b border-white/5">
                                                        <ErrorBoundary fallback={<div className="p-4 text-center text-gray-500 text-sm">Chart could not be rendered</div>}>
                                                          <ChartVisualizer result={msg.result} />
                                                        </ErrorBoundary>
                                                      </div>

                                                      <div className="overflow-x-auto max-h-96">
                                                          <table className="w-full text-sm text-left">
                                                          <thead className="bg-[#11141d] text-gray-400 sticky top-0 z-10 shadow-sm">
                                                              <tr>
                                                              {msg.result?.columns?.map((col: string, i: number) => (
                                                                  <th key={i} className="px-4 py-3 font-medium whitespace-nowrap">{col}</th>
                                                              ))}
                                                              </tr>
                                                          </thead>
                                                          <tbody className="divide-y divide-white/5">
                                                              {msg.result?.rows?.map((row: any[], i: number) => (
                                                              <tr key={i} className="hover:bg-white/5 transition-colors">
                                                                  {row?.map((cell: any, j: number) => (
                                                                  <td key={j} className="px-4 py-3 text-gray-300 whitespace-nowrap">
                                                                      {cell !== null && cell !== undefined ? cell.toString() : <span className="text-gray-600 italic">null</span>}
                                                                  </td>
                                                                  ))}
                                                              </tr>
                                                              ))}
                                                          </tbody>
                                                          </table>
                                                      </div>
                                                    </div>
                                                )}

                                                {msg.error && (
                                                    <div className="bg-red-500/10 border border-red-500/20 text-red-200 p-4 rounded-xl flex items-start gap-3">
                                                        <div className="bg-red-500/20 p-1.5 rounded text-red-400 mt-0.5">
                                                            <StopCircle className="w-4 h-4" />
                                                        </div>
                                                        <div className="flex-1">
                                                            <h4 className="font-semibold text-sm mb-1">Execution Failed</h4>
                                                            <p className="text-sm opacity-90">{msg.error}</p>
                                                        </div>
                                                    </div>
                                                )}
                                                </div>
                                            )}
                                            </div>
                                        </motion.div>
                                    ))
                                )}
                                {loading && (
                                    <motion.div 
                                        initial={{ opacity: 0 }}
                                        animate={{ opacity: 1 }}
                                        className="flex gap-4"
                                    >
                                        <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center flex-shrink-0 mt-1">
                                            <Bot className="w-5 h-5 text-white" />
                                        </div>
                                        <div className="flex items-center gap-2 text-gray-400 bg-white/5 px-4 py-3 rounded-2xl rounded-tl-sm">
                                            <Loader2 className="w-4 h-4 animate-spin" />
                                            <span className="text-sm">Generating SQL & querying database...</span>
                                        </div>
                                    </motion.div>
                                )}
                                <div ref={messagesEndRef} />
                            </div>
                        </div>
                    
                        {/* Input Area */}
                        <div className="p-4 bg-background border-t border-white/10 shrink-0">
                            <form onSubmit={handleChatSubmit} className="max-w-4xl mx-auto relative">
                            <input
                                type="text"
                                value={input}
                                onChange={(e) => setInput(e.target.value)}
                                placeholder="Ask a question about your data..."
                                className="w-full bg-surface border border-white/10 rounded-xl px-4 py-4 pr-14 focus:outline-none focus:ring-2 focus:ring-primary/50 text-white placeholder-gray-500 shadow-xl"
                                disabled={loading}
                            />
                            <button 
                                type="submit" 
                                disabled={!input.trim() || loading}
                                className="absolute right-2 top-2 p-2 bg-primary hover:bg-blue-600 disabled:opacity-50 disabled:hover:bg-primary text-white rounded-lg transition-colors"
                            >
                                <Send className="w-5 h-5" />
                            </button>
                            </form>
                            <p className="text-center text-xs text-gray-600 mt-2">
                                Connected to {connections.length > 0 ? (connections.find(c => c.id === selectedConnection)?.name || 'Database') : 'No Database'} | AI Model: {selectedModel}
                                {' | '}
                                {selectedProvider === 'ollama' ? (
                                    user?.llm_config?.ollama && (user.llm_config.ollama as { host?: string })?.host 
                                        ? <span className="text-green-500">🔑 Your Ollama</span>
                                        : <span className="text-gray-500">⚙️ System Ollama</span>
                                ) : (
                                    (() => {
                                        const config = user?.llm_config?.[selectedProvider as keyof typeof user.llm_config] as { api_key?: string } | undefined;
                                        return config?.api_key 
                                            ? <span className="text-green-500">🔑 Your API Key</span>
                                            : <span className="text-gray-500">⚙️ System API Key</span>;
                                    })()
                                )}
                            </p>
                        </div>
                    </motion.div>
                )}

                {/* CONNECTIONS TAB */}
                {activeTab === 'connections' && (
                    <motion.div
                        key="connections"
                        initial={{ opacity: 0, x: 20 }}
                        animate={{ opacity: 1, x: 0 }}
                        exit={{ opacity: 0, x: -20 }}
                        className="p-8 max-w-5xl mx-auto"
                    >
                        <div className="flex justify-between items-center mb-8">
                            <div>
                                <h2 className="text-2xl font-bold mb-1">Data Connections</h2>
                                <p className="text-gray-400 text-sm">Manage your database connections for this workspace</p>
                            </div>
                            <button 
                                onClick={openCreateModal}
                                className="btn-primary flex items-center gap-2"
                            >
                                <Plus className="w-4 h-4" />
                                Add Connection
                            </button>
                        </div>

                        {isLoadingConnections ? (
                            <div className="flex justify-center p-12">
                                <Loader2 className="w-8 h-8 animate-spin text-primary" />
                            </div>
                        ) : (
                            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                                {connections.map(conn => (
                                    <div key={conn.id} className="glass-card p-6 relative group hover:border-primary/30 transition-all">
                                        <div className="absolute top-4 right-4 flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                                            <button 
                                                onClick={() => openEditModal(conn)}
                                                className="p-2 bg-white/5 hover:bg-white/10 rounded-lg text-gray-300 hover:text-white"
                                                title="Edit Connection"
                                            >
                                                <Edit2 className="w-4 h-4" />
                                            </button>
                                            <button 
                                                onClick={() => handleDeleteConnection(conn.id)}
                                                className="p-2 bg-red-500/10 hover:bg-red-500 rounded-lg text-red-400 hover:text-white"
                                                title="Delete Connection"
                                            >
                                                {deletingConnectionId === conn.id ? <Loader2 className="w-4 h-4 animate-spin"/> :<Trash2 className="w-4 h-4" />}
                                            </button>
                                        </div>

                                        <div className="flex items-start gap-4 mb-4">
                                            <div className="bg-primary/20 p-3 rounded-xl">
                                                <Database className="w-6 h-6 text-primary" />
                                            </div>
                                            <div>
                                                <h3 className="font-semibold text-lg">{conn.name}</h3>
                                                <span className="text-xs text-uppercase bg-white/10 px-2 py-0.5 rounded text-gray-400 uppercase">{conn.database_type}</span>
                                            </div>
                                        </div>

                                        <div className="space-y-2 text-sm text-gray-400">
                                            {conn.database_type === 'sqlite' ? (
                                                <div className="flex justify-between">
                                                    <span>File:</span>
                                                    <span className="text-gray-300 truncate ml-2 max-w-[200px]" title={conn.database}>{conn.database}</span>
                                                </div>
                                            ) : (
                                            <>
                                            <div className="flex justify-between">
                                                <span>Host:</span>
                                                <span className="text-gray-300">{conn.host}</span>
                                            </div>
                                            <div className="flex justify-between">
                                                <span>Port:</span>
                                                <span className="text-gray-300">{conn.port}</span>
                                            </div>
                                            <div className="flex justify-between">
                                                <span>Database:</span>
                                                <span className="text-gray-300">{conn.database}</span>
                                            </div>
                                            <div className="flex justify-between">
                                                <span>User:</span>
                                                <span className="text-gray-300">{conn.username}</span>
                                            </div>
                                            </>
                                            )}
                                        </div>
                                        
                                        {selectedConnection === conn.id && (
                                            <div className="mt-4 flex items-center gap-2 text-green-400 text-sm bg-green-500/10 p-2 rounded justify-center">
                                                <Check className="w-4 h-4" />
                                                Active in Chat
                                            </div>
                                        )}
                                    </div>
                                ))}

                                {connections.length === 0 && (
                                    <div className="col-span-full py-12 text-center border border-dashed border-white/10 rounded-2xl bg-white/5">
                                        <Database className="w-12 h-12 text-gray-500 mx-auto mb-4" />
                                        <h3 className="text-lg font-medium text-gray-300">No connections yet</h3>
                                        <p className="text-gray-500 mb-6">Add a database to start querying</p>
                                        <button onClick={openCreateModal} className="btn-secondary">
                                            Connect Database
                                        </button>
                                    </div>
                                )}
                            </div>
                        )}
                    </motion.div>
                )}

                {/* SETTINGS TAB */}
                {activeTab === 'settings' && (
                    <motion.div
                        key="settings"
                        initial={{ opacity: 0, x: 20 }}
                        animate={{ opacity: 1, x: 0 }}
                        exit={{ opacity: 0, x: -20 }}
                        className="p-8 max-w-2xl mx-auto"
                    >
                        <h2 className="text-2xl font-bold mb-6">Workspace Settings</h2>
                        
                        <div className="glass-card p-6 space-y-6">
                            <section>
                                <h3 className="text-lg font-medium mb-4 flex items-center gap-2">
                                    <Settings className="w-5 h-5 text-primary" />
                                    General
                                </h3>
                                <form onSubmit={handleUpdateWorkspace} className="space-y-4">
                                    <div>
                                        <label className="block text-sm font-medium text-gray-300 mb-1">Workspace Name</label>
                                        <div className="flex gap-2">
                                            <input 
                                                type="text" 
                                                value={workspaceName}
                                                onChange={(e) => setWorkspaceName(e.target.value)}
                                                className="glass-input flex-1"
                                            />
                                            <button type="submit" className="btn-primary">
                                                <Save className="w-4 h-4" />
                                            </button>
                                        </div>
                                    </div>
                                </form>
                            </section>

                            <div className="border-t border-white/10 my-6"></div>

                            <section>
                                <h3 className="text-lg font-medium mb-4 flex items-center gap-2">
                                    <Key className="w-5 h-5 text-primary" />
                                    LLM Credentials
                                </h3>
                                <div className="bg-white/5 border border-white/10 p-4 rounded-xl mb-4">
                                    <p className="text-sm text-gray-400 mb-2">
                                        Configure your own API keys for LLM providers. These will override the system defaults.
                                        Leave empty to use system defaults.
                                    </p>
                                </div>
                                <form onSubmit={handleUpdateLLMConfig} className="space-y-4">
                                    <div>
                                        <label className="block text-sm font-medium text-gray-300 mb-1">Ollama Host URL</label>
                                        <input 
                                            type="text" 
                                            value={llmConfigForm.ollama_host}
                                            onChange={(e) => setLlmConfigForm({...llmConfigForm, ollama_host: e.target.value})}
                                            className="glass-input w-full"
                                            placeholder="http://localhost:11434"
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-300 mb-1">OpenAI API Key</label>
                                        <input 
                                            type="password" 
                                            value={llmConfigForm.openai_key}
                                            onChange={(e) => setLlmConfigForm({...llmConfigForm, openai_key: e.target.value})}
                                            className="glass-input w-full"
                                            placeholder="sk-..."
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-300 mb-1">Anthropic API Key</label>
                                        <input 
                                            type="password" 
                                            value={llmConfigForm.anthropic_key}
                                            onChange={(e) => setLlmConfigForm({...llmConfigForm, anthropic_key: e.target.value})}
                                            className="glass-input w-full"
                                            placeholder="sk-ant-..."
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-300 mb-1">DeepSeek API Key</label>
                                        <input 
                                            type="password" 
                                            value={llmConfigForm.deepseek_key}
                                            onChange={(e) => setLlmConfigForm({...llmConfigForm, deepseek_key: e.target.value})}
                                            className="glass-input w-full"
                                            placeholder="sk-..."
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-sm font-medium text-gray-300 mb-1">Gemini API Key</label>
                                        <input 
                                            type="password" 
                                            value={llmConfigForm.gemini_key}
                                            onChange={(e) => setLlmConfigForm({...llmConfigForm, gemini_key: e.target.value})}
                                            className="glass-input w-full"
                                            placeholder="AIza..."
                                        />
                                    </div>
                                    <div className="flex justify-end items-center gap-4">
                                        {isSavingLLM ? null : (isLLMSaved ? <span className="text-green-400 text-sm animate-fade-in flex items-center gap-1"><Check className="w-4 h-4" /> Saved successfully</span> : null)}
                                        <button type="submit" className={clsx("btn-primary transition-all duration-300", isLLMSaved && "bg-green-600 hover:bg-green-700")} disabled={isSavingLLM}>
                                            {isSavingLLM ? <Loader2 className="w-4 h-4 animate-spin" /> : (isLLMSaved ? <Check className="w-4 h-4" /> : <Save className="w-4 h-4" />)}
                                            {isLLMSaved ? "Saved!" : "Save Credentials"}
                                        </button>
                                    </div>
                                </form>
                            </section>

                            <div className="border-t border-white/10 my-6"></div>

                            <section>
                                <h3 className="text-lg font-medium mb-4 flex items-center gap-2 text-red-400">
                                    <AlertCircle className="w-5 h-5" />
                                    Danger Zone
                                </h3>
                                <div className="bg-red-500/5 border border-red-500/20 p-4 rounded-xl">
                                    <h4 className="font-semibold text-red-200 mb-1">Delete Workspace</h4>
                                    <p className="text-sm text-red-200/60 mb-4">
                                        Permanently delete this workspace and all its data. This action cannot be undone.
                                    </p>
                                    <button onClick={handleDeleteWorkspace} className="px-4 py-2 bg-red-500 hover:bg-red-600 text-white rounded-lg text-sm font-medium transition-colors">
                                        Delete Workspace
                                    </button>
                                </div>
                            </section>
                        </div>
                    </motion.div>
                )}

            </AnimatePresence>
      </main>
      </div>

      {/* Connection Modal */}
      {showConnectionModal && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
             <motion.div 
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                className="glass-card w-full max-w-md p-6 max-h-[90vh] overflow-y-auto"
            >
                <div className="flex justify-between items-center mb-6">
                    <h3 className="text-xl font-bold">{editingConnection ? 'Edit Connection' : 'Add New Connection'}</h3>
                    <button onClick={() => setShowConnectionModal(false)} className="p-2 hover:bg-white/10 rounded-lg">
                        <X className="w-5 h-5 text-gray-400" />
                    </button>
                </div>

                <form onSubmit={handleConnectionSubmit} className="space-y-5">
                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">Connection Name</label>
                        <input 
                            type="text" 
                            value={connectionForm.name}
                            onChange={(e) => setConnectionForm({...connectionForm, name: e.target.value})}
                            className="glass-input w-full"
                            placeholder="My Postgres DB"
                            required
                        />
                    </div>
                     <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">Type</label>
                        <select
                            value={connectionForm.type}
                            onChange={(e) => {
                                const type = e.target.value;
                                let port = connectionForm.port;
                                // Set default port based on database type
                                if (type === 'postgres') port = '5432';
                                else if (type === 'mysql') port = '3306';
                                else if (type === 'clickhouse') port = '8123';
                                else if (type === 'mongodb') port = '27017';
                                else if (type === 'sqlite') port = '0';
                                setConnectionForm({...connectionForm, type, port});
                            }}
                            className="glass-input w-full bg-[#11141d] cursor-pointer" // Force dark bg for select
                        >
                            <option value="postgres">PostgreSQL</option>
                            <option value="mysql">MySQL</option>
                            <option value="clickhouse">ClickHouse</option>
                            <option value="mongodb">MongoDB</option>
                            <option value="sqlite">SQLite</option>
                        </select>
                    </div>

                    {connectionForm.type === 'sqlite' ? (
                    <div>
                        <div className="flex gap-1 mb-3 bg-black/20 p-1 rounded-lg">
                            <button
                                type="button"
                                onClick={() => { setSqliteInputMode('upload'); setConnectionForm({...connectionForm, dbname: ''}); setUploadedFileName(''); }}
                                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
                                    sqliteInputMode === 'upload' ? 'bg-primary text-white shadow' : 'text-gray-400 hover:text-white'
                                }`}
                            >
                                📁 Upload File
                            </button>
                            <button
                                type="button"
                                onClick={() => { setSqliteInputMode('url'); setConnectionForm({...connectionForm, dbname: ''}); setUploadedFileName(''); }}
                                className={`flex-1 px-3 py-1.5 rounded-md text-sm font-medium transition-all ${
                                    sqliteInputMode === 'url' ? 'bg-primary text-white shadow' : 'text-gray-400 hover:text-white'
                                }`}
                            >
                                🔗 URL / Path
                            </button>
                        </div>

                        {sqliteInputMode === 'upload' ? (
                            <div>
                                <label className="block text-sm font-medium text-gray-300 mb-1">SQLite Database File</label>
                                <div className={`border-2 border-dashed rounded-xl p-6 text-center transition-all ${
                                    connectionForm.dbname && uploadedFileName
                                        ? 'border-green-500/50 bg-green-500/5' 
                                        : 'border-white/10 hover:border-primary/50 bg-white/5'
                                }`}>
                                    {connectionForm.dbname && uploadedFileName ? (
                                        <div className="space-y-2">
                                            <div className="text-green-400 text-2xl">✅</div>
                                            <p className="text-sm text-green-300 font-medium">{uploadedFileName}</p>
                                            <p className="text-xs text-gray-500">File uploaded successfully</p>
                                            <button
                                                type="button"
                                                onClick={() => { setConnectionForm({...connectionForm, dbname: ''}); setUploadedFileName(''); }}
                                                className="text-xs text-red-400 hover:text-red-300 underline"
                                            >
                                                Remove
                                            </button>
                                        </div>
                                    ) : (
                                        <div className="space-y-2">
                                            {isUploadingSqlite ? (
                                                <div className="flex flex-col items-center gap-2">
                                                    <Loader2 className="w-8 h-8 animate-spin text-primary" />
                                                    <p className="text-sm text-gray-400">Uploading...</p>
                                                </div>
                                            ) : (
                                                <>
                                                    <div className="text-gray-500 text-2xl">📂</div>
                                                    <p className="text-sm text-gray-400">Click to select a SQLite file</p>
                                                    <label className="inline-block px-4 py-2 bg-primary/20 text-primary rounded-lg text-sm cursor-pointer hover:bg-primary/30 transition-colors">
                                                        Browse Files
                                                        <input
                                                            type="file"
                                                            accept=".db,.sqlite,.sqlite3,.db3"
                                                            className="hidden"
                                                            onChange={async (e) => {
                                                                const file = e.target.files?.[0];
                                                                if (!file) return;
                                                                setIsUploadingSqlite(true);
                                                                try {
                                                                    const formData = new FormData();
                                                                    formData.append('file', file);
                                                                    const res = await api.post(`/workspaces/${workspaceId}/upload-sqlite`, formData, {
                                                                        headers: { 'Content-Type': 'multipart/form-data' }
                                                                    });
                                                                    if (res.data?.data?.file_path) {
                                                                        setConnectionForm(prev => ({...prev, dbname: res.data.data.file_path}));
                                                                        setUploadedFileName(file.name);
                                                                    }
                                                                } catch (err) {
                                                                    console.error('Upload failed', err);
                                                                    alert('Failed to upload SQLite file');
                                                                } finally {
                                                                    setIsUploadingSqlite(false);
                                                                }
                                                            }}
                                                        />
                                                    </label>
                                                    <p className="text-xs text-gray-600">Supports: .db, .sqlite, .sqlite3, .db3</p>
                                                </>
                                            )}
                                        </div>
                                    )}
                                </div>
                            </div>
                        ) : (
                            <div>
                                <label className="block text-sm font-medium text-gray-300 mb-1">Database URL / File Path</label>
                                <input 
                                    type="text" 
                                    value={connectionForm.dbname}
                                    onChange={(e) => setConnectionForm({...connectionForm, dbname: e.target.value})}
                                    className="glass-input w-full"
                                    placeholder="/path/to/database.db or https://example.com/data.db"
                                    required
                                />
                                <p className="text-xs text-gray-600 mt-1">Enter a local file path or URL to a SQLite database</p>
                            </div>
                        )}
                    </div>
                    ) : (
                    <>
                    <div className="grid grid-cols-3 gap-4">
                        <div className="col-span-2">
                             <label className="block text-sm font-medium text-gray-300 mb-1">Host</label>
                             <input 
                                type="text" 
                                value={connectionForm.host}
                                onChange={(e) => setConnectionForm({...connectionForm, host: e.target.value})}
                                className="glass-input w-full"
                                placeholder="localhost"
                                required
                            />
                        </div>
                        <div>
                             <label className="block text-sm font-medium text-gray-300 mb-1">Port</label>
                             <input 
                                type="text" 
                                value={connectionForm.port}
                                onChange={(e) => setConnectionForm({...connectionForm, port: e.target.value})}
                                className="glass-input w-full"
                                placeholder="5432"
                                required
                            />
                        </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div>
                             <label className="block text-sm font-medium text-gray-300 mb-1">Database Name</label>
                             <input 
                                type="text" 
                                value={connectionForm.dbname}
                                onChange={(e) => setConnectionForm({...connectionForm, dbname: e.target.value})}
                                className="glass-input w-full"
                                placeholder="mydb"
                                required
                            />
                        </div>
                         <div>
                             <label className="block text-sm font-medium text-gray-300 mb-1">Username</label>
                             <input 
                                type="text" 
                                value={connectionForm.user}
                                onChange={(e) => setConnectionForm({...connectionForm, user: e.target.value})}
                                className="glass-input w-full"
                                placeholder="user"
                            />
                        </div>
                    </div>
                    </>
                    )}

                    {connectionForm.type !== 'sqlite' && (
                    <div>
                         <label className="block text-sm font-medium text-gray-300 mb-1">Password</label>
                         <input 
                            type="password" 
                            value={connectionForm.password}
                            onChange={(e) => setConnectionForm({...connectionForm, password: e.target.value})}
                            className="glass-input w-full"
                            placeholder={editingConnection ? "(Unchanged)" : "password"}
                        />
                    </div>
                    )}
                    
                    <div className="pt-2">
                        <button 
                            type="button" 
                            onClick={() => setShowAdvanced(!showAdvanced)}
                            className="flex items-center gap-2 text-sm text-primary hover:text-primary/80 transition-colors"
                        >
                            <span>Advanced Options</span>
                            <div className={clsx("transition-transform duration-200", showAdvanced ? "rotate-180" : "")}>
                                <ArrowLeft className="w-3 h-3 -rotate-90" />
                            </div>
                        </button>

                        <AnimatePresence>
                            {showAdvanced && (
                                <motion.div
                                    initial={{ height: 0, opacity: 0 }}
                                    animate={{ height: 'auto', opacity: 1 }}
                                    exit={{ height: 0, opacity: 0 }}
                                    className="overflow-hidden"
                                >
                                    <div className="pt-4 space-y-4 border-t border-white/5 mt-2">
                                        <div>
                                            <label className="block text-sm font-medium text-gray-300 mb-1">SSL Mode</label>
                                            <select
                                                value={connectionForm.sslMode}
                                                onChange={(e) => setConnectionForm({...connectionForm, sslMode: e.target.value})}
                                                className="glass-input w-full bg-[#11141d] cursor-pointer"
                                            >
                                                <option value="disable">Disable (No SSL)</option>
                                                <option value="require">Require (True)</option>
                                                <option value="verify-full">Verify CA (Strict)</option>
                                            </select>
                                        </div>

                                        <div className="grid grid-cols-2 gap-4">
                                            <div>
                                                 <label className="block text-sm font-medium text-gray-300 mb-1 leading-none py-1">Max Rows</label>
                                                 <input 
                                                    type="number" 
                                                    value={connectionForm.maxRows}
                                                    onChange={(e) => setConnectionForm({...connectionForm, maxRows: parseInt(e.target.value) || 1000})}
                                                    className="glass-input w-full"
                                                />
                                            </div>
                                             <div>
                                                 <label className="block text-sm font-medium text-gray-300 mb-1 leading-none py-1">Timeout (sec)</label>
                                                 <input 
                                                    type="number" 
                                                    value={connectionForm.timeout}
                                                    onChange={(e) => setConnectionForm({...connectionForm, timeout: parseInt(e.target.value) || 30})}
                                                    className="glass-input w-full"
                                                />
                                            </div>
                                        </div>
                                    </div>
                                </motion.div>
                            )}
                        </AnimatePresence>
                    </div>

                    {/* Test Connection Result */}
                    {testConnectionResult && (
                        <div className={clsx(
                            'p-3 rounded-lg text-sm mb-4',
                            testConnectionResult.success 
                                ? 'bg-green-500/20 text-green-300 border border-green-500/30' 
                                : 'bg-red-500/20 text-red-300 border border-red-500/30'
                        )}>
                            {testConnectionResult.success ? '✅' : '❌'} {testConnectionResult.message}
                        </div>
                    )}
                    
                    <div className="flex justify-end gap-3 mt-6">
                        <button 
                            type="button"
                            onClick={() => setShowConnectionModal(false)}
                            className="btn-secondary px-4 py-2"
                        >
                            Cancel
                        </button>
                        <button 
                            type="button"
                            onClick={handleTestConnection}
                            disabled={isTestingConnection}
                            className="btn-secondary px-4 py-2 flex items-center gap-2"
                        >
                            {isTestingConnection ? <Loader2 className="w-4 h-4 animate-spin" /> : <Database className="w-4 h-4" />}
                            <span>Test</span>
                        </button>
                        <button 
                            type="submit" 
                            disabled={isSubmittingConnection}
                            className="btn-primary px-4 py-2 flex items-center gap-2"
                        >
                            {isSubmittingConnection ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                            <span>Save</span>
                        </button>
                    </div>
                </form>
            </motion.div>
        </div>
      )}
      {/* No Connection Modal */}
      {showNoConnectionModal && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
            <motion.div 
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                className="glass-card w-full max-w-sm p-6 text-center border-orange-500/30"
            >
                <div className="w-16 h-16 bg-orange-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                    <Database className="w-8 h-8 text-orange-400" />
                </div>
                <h3 className="text-xl font-bold mb-2">No Database Connected</h3>
                <p className="text-gray-400 mb-6 text-sm">
                    You need to connect to a database to start chatting and querying your data.
                </p>
                <div className="flex gap-3">
                    <button 
                        onClick={() => setShowNoConnectionModal(false)}
                        className="flex-1 px-4 py-2 hover:bg-white/5 rounded-lg text-gray-300 transition-colors"
                    >
                        Cancel
                    </button>
                    <button 
                        onClick={handleCreateConnectionFromModal}
                        className="flex-1 btn-primary bg-orange-500 hover:bg-orange-600 border-none"
                    >
                        Connect Database
                    </button>
                </div>
            </motion.div>
        </div>
      )}
    </div>
  );
};

export default Workspace;
