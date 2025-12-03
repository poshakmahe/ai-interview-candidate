export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface Document {
  id: string;
  owner_id: string;
  name: string;
  original_name: string;
  size: number;
  mime_type: string;
  encryption_algo: string;
  is_encrypted: boolean;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
  owner_name?: string;
}

export interface DocumentShare {
  id: string;
  document_id: string;
  shared_by_id: string;
  shared_with_id: string;
  permission: 'view' | 'edit';
  expires_at?: string;
  created_at: string;
}

export interface SharedUserInfo {
  email: string;
  name: string;
  permission: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface ErrorResponse {
  error: string;
  message?: string;
}

export interface LoginFormData {
  email: string;
  password: string;
}

export interface RegisterFormData {
  email: string;
  password: string;
  name: string;
}

export interface ShareFormData {
  email: string;
  permission: 'view' | 'edit';
}
