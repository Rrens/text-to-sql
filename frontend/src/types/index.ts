export interface User {
  id: string;
  email: string;
  llm_config?: Record<string, any>;
}

export interface Workspace {
  id: string;
  name: string;
  created_at: string;
}

export interface Connection {
  id: string;
  name: string;
  database_type: string;
  host: string;
  database: string;
}
