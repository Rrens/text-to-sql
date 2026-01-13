import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext.tsx';
import Login from './pages/Login.tsx';
import Register from './pages/Register.tsx';
import Dashboard from './pages/Dashboard.tsx';
import Workspace from './pages/Workspace.tsx';

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, isLoading } = useAuth();
  if (isLoading) return <div className="min-h-screen flex items-center justify-center bg-background"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div></div>;
  if (!isAuthenticated) return <Navigate to="/login" />;
  return <>{children}</>;
};

const PublicRoute = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, isLoading } = useAuth();
  if (isLoading) return <div className="min-h-screen flex items-center justify-center bg-background"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div></div>;
  if (isAuthenticated) return <Navigate to="/" />;
  return <>{children}</>;
};

function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={
        <PublicRoute>
          <Login />
        </PublicRoute>
      } />
      <Route path="/register" element={
        <PublicRoute>
          <Register />
        </PublicRoute>
      } />
      <Route path="/" element={
        <ProtectedRoute>
          <Dashboard />
        </ProtectedRoute>
      } />
      <Route path="/workspace/:workspaceId" element={
        <ProtectedRoute>
          <Workspace />
        </ProtectedRoute>
      } />
    </Routes>
  );
}

import { Linkedin } from 'lucide-react';

/* ... imports ... */

function App() {
  return (
    <AuthProvider>
      <Router>
        <div className="min-h-screen bg-background text-white font-sans flex flex-col">
          <div className="flex-1">
            <AppRoutes />
          </div>
          <footer className="py-6 border-t border-white/10 bg-black/20 backdrop-blur-sm">
            <div className="max-w-6xl mx-auto px-6 flex items-center justify-center gap-2 text-sm text-gray-400">
              <span>Made by Rendy Yusuf</span>
              <span>•</span>
              <a 
                href="https://www.linkedin.com/in/rendy-yusuf/" 
                target="_blank" 
                rel="noopener noreferrer"
                className="flex items-center gap-1 hover:text-primary transition-colors hover:underline"
              >
                <Linkedin className="w-3 h-3" />
                LinkedIn
              </a>
            </div>
          </footer>
        </div>
      </Router>
    </AuthProvider>
  );
}

export default App;
