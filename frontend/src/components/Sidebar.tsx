import { Plus, MessageSquare, Trash2 } from 'lucide-react';
import { motion } from 'framer-motion';
import clsx from 'clsx';
import { useState } from 'react';

interface Session {
  id: string;
  title: string;
  created_at: string;
}

interface SidebarProps {
  sessions: Session[];
  currentSessionId: string | null;
  onSelectSession: (id: string) => void;
  onNewChat: () => void;
  onDeleteSession: (id: string) => void;
  isLoading: boolean;
}

const Sidebar = ({ 
  sessions, 
  currentSessionId, 
  onSelectSession, 
  onNewChat, 
  onDeleteSession,
  isLoading 
}: SidebarProps) => {
  const [hoveredSession, setHoveredSession] = useState<string | null>(null);

  // Group sessions by date could be a nice addition later

  return (
    <div className="w-64 bg-[#0d1117] border-r border-white/5 flex flex-col h-full shrink-0">
      {/* New Chat Button */}
      <div className="p-4">
        <button
          onClick={onNewChat}
          className="w-full flex items-center gap-2 justify-center bg-primary hover:bg-blue-600 text-white p-3 rounded-xl transition-all font-medium shadow-lg shadow-primary/20"
        >
          <Plus className="w-5 h-5" />
          New Chat
        </button>
      </div>

      {/* Sessions List */}
      <div className="flex-1 overflow-y-auto px-2 pb-4 scrollbar-thin scrollbar-thumb-white/10">
        <div className="space-y-1">
          {isLoading && sessions.length === 0 ? (
             <div className="p-4 text-center text-xs text-gray-500 animate-pulse">Loading history...</div>
          ) : sessions.length === 0 ? (
            <div className="p-4 text-center text-xs text-gray-500">No chat history</div>
          ) : (
            sessions.map((session) => (
              <motion.button
                key={session.id}
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                onClick={() => onSelectSession(session.id)}
                onMouseEnter={() => setHoveredSession(session.id)}
                onMouseLeave={() => setHoveredSession(null)}
                className={clsx(
                  "w-full text-left p-3 rounded-lg text-sm transition-all group relative flex items-center gap-3",
                  currentSessionId === session.id 
                    ? "bg-white/10 text-white" 
                    : "text-gray-400 hover:bg-white/5 hover:text-gray-200"
                )}
              >
                <MessageSquare className="w-4 h-4 shrink-0 opacity-70" />
                <span className="truncate flex-1">{session.title}</span>
                
                {/* Delete button (visible on hover) */}
                {(hoveredSession === session.id || currentSessionId === session.id) && (
                  <div 
                    onClick={(e) => {
                      e.stopPropagation();
                      onDeleteSession(session.id);
                    }}
                    className="opacity-0 group-hover:opacity-100 p-1 hover:bg-red-500/20 hover:text-red-400 rounded transition-all"
                  >
                    <Trash2 className="w-3 h-3" />
                  </div>
                )}
              </motion.button>
            ))
          )}
        </div>
      </div>

      {/* User / Settings Footer (Optional) */}
      <div className="p-4 border-t border-white/5 flex items-center gap-3 text-sm text-gray-400">
         {/* Could put user info here */}
      </div>
    </div>
  );
};

export default Sidebar;
