import axios, { AxiosError, AxiosInstance } from 'axios';
import { AuthResponse, Document, PaginatedResponse, User, ErrorResponse } from '@/types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

class ApiService {
  private client: AxiosInstance;
  private token: string | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add auth token to requests
    this.client.interceptors.request.use((config) => {
      if (this.token) {
        config.headers.Authorization = `Bearer ${this.token}`;
      }
      return config;
    });

    // Handle errors
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError<ErrorResponse>) => {
        if (error.response?.status === 401) {
          this.clearToken();
          if (typeof window !== 'undefined') {
            window.location.href = '/login';
          }
        }
        return Promise.reject(error);
      }
    );

    // Load token from localStorage
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('token');
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('token', token);
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('token');
    }
  }

  getToken(): string | null {
    return this.token;
  }

  // Auth endpoints
  async register(email: string, password: string, name: string): Promise<AuthResponse> {
    const response = await this.client.post<AuthResponse>('/auth/register', {
      email,
      password,
      name,
    });
    this.setToken(response.data.token);
    return response.data;
  }

  async login(email: string, password: string): Promise<AuthResponse> {
    const response = await this.client.post<AuthResponse>('/auth/login', {
      email,
      password,
    });
    this.setToken(response.data.token);
    return response.data;
  }

  async getCurrentUser(): Promise<User> {
    const response = await this.client.get<User>('/auth/me');
    return response.data;
  }

  logout() {
    this.clearToken();
  }

  // Document endpoints
  async listDocuments(page = 1, perPage = 20, signal?: AbortSignal): Promise<PaginatedResponse<Document>> {
    const response = await this.client.get<PaginatedResponse<Document>>('/documents', {
      params: { page, per_page: perPage },
      signal,
    });
    return response.data;
  }

  async getDocument(id: string): Promise<Document> {
    const response = await this.client.get<Document>(`/documents/${id}`);
    return response.data;
  }

  async uploadDocument(file: File, name?: string): Promise<Document> {
    const formData = new FormData();
    formData.append('file', file);
    if (name) {
      formData.append('name', name);
    }

    const response = await this.client.post<Document>('/documents', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    return response.data;
  }

  async renameDocument(id: string, name: string): Promise<void> {
    await this.client.patch(`/documents/${id}`, { name });
  }

  async deleteDocument(id: string): Promise<void> {
    await this.client.delete(`/documents/${id}`);
  }

  async downloadDocument(id: string): Promise<Blob> {
    const response = await this.client.get(`/documents/${id}/download`, {
      responseType: 'blob',
    });
    return response.data;
  }

  async shareDocument(id: string, email: string, permission: 'view' | 'edit'): Promise<void> {
    await this.client.post(`/documents/${id}/share`, { email, permission });
  }

  async listSharedDocuments(page = 1, perPage = 20, signal?: AbortSignal): Promise<PaginatedResponse<Document>> {
    const response = await this.client.get<PaginatedResponse<Document>>('/shared', {
      params: { page, per_page: perPage },
      signal,
    });
    return response.data;
  }

  // Health check
  async healthCheck(): Promise<boolean> {
    try {
      await this.client.get('/health');
      return true;
    } catch {
      return false;
    }
  }

  // Summarize document
  async summarizeDocument(documentId: string): Promise<{ summary: string }> {
    // First download the document content
    const blob = await this.downloadDocument(documentId);
    const text = await this.extractTextFromBlob(blob);

    // Call the summarize API route
    const response = await axios.post<{ summary: string }>('/api/summarize', {
      text,
      documentId,
    });
    return response.data;
  }

  private async extractTextFromBlob(blob: Blob): Promise<string> {
    const mimeType = blob.type;

    // Handle plain text files
    if (mimeType.startsWith('text/') || mimeType === 'application/json') {
      return await blob.text();
    }

    // For PDF and DOCX, we'll send the blob to a text extraction endpoint
    // Since we can't easily extract text client-side without heavy libraries
    // We'll handle this in the summarize API route instead
    if (mimeType === 'application/pdf' ||
        mimeType === 'application/vnd.openxmlformats-officedocument.wordprocessingml.document') {
      // Convert blob to base64 for sending to API
      const arrayBuffer = await blob.arrayBuffer();
      const base64 = btoa(
        new Uint8Array(arrayBuffer).reduce((data, byte) => data + String.fromCharCode(byte), '')
      );
      return `__BINARY_DOCUMENT__:${mimeType}:${base64}`;
    }

    // Default: try to read as text
    return await blob.text();
  }
}

export const api = new ApiService();
export default api;
