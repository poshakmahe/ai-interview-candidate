import { create } from 'zustand';
import { Document, PaginatedResponse } from '@/types';
import api from '@/services/api';

interface DocumentsState {
  documents: Document[];
  sharedDocuments: Document[];
  selectedDocument: Document | null;
  isLoading: boolean;
  error: string | null;
  pagination: {
    page: number;
    perPage: number;
    total: number;
    totalPages: number;
  };
  sharedPagination: {
    page: number;
    perPage: number;
    total: number;
    totalPages: number;
  };

  fetchDocuments: (page?: number, signal?: AbortSignal) => Promise<void>;
  fetchSharedDocuments: (page?: number, signal?: AbortSignal) => Promise<void>;
  fetchDocument: (id: string) => Promise<void>;
  uploadDocument: (file: File, name?: string) => Promise<Document>;
  renameDocument: (id: string, name: string) => Promise<void>;
  deleteDocument: (id: string) => Promise<void>;
  shareDocument: (id: string, email: string, permission: 'view' | 'edit') => Promise<void>;
  downloadDocument: (id: string, filename: string) => Promise<void>;
  clearError: () => void;
  setSelectedDocument: (document: Document | null) => void;
}

export const useDocuments = create<DocumentsState>((set, get) => ({
  documents: [],
  sharedDocuments: [],
  selectedDocument: null,
  isLoading: false,
  error: null,
  pagination: {
    page: 1,
    perPage: 20,
    total: 0,
    totalPages: 0,
  },
  sharedPagination: {
    page: 1,
    perPage: 20,
    total: 0,
    totalPages: 0,
  },

  fetchDocuments: async (page = 1, signal?: AbortSignal) => {
    set({ isLoading: true, error: null });
    try {
      const response = await api.listDocuments(page, get().pagination.perPage, signal);
      set({
        documents: response.data || [],
        pagination: {
          page: response.page,
          perPage: response.per_page,
          total: response.total,
          totalPages: response.total_pages,
        },
        isLoading: false,
      });
    } catch (error: any) {
      // Don't set error if request was aborted
      if (error.name === 'AbortError' || error.name === 'CanceledError') {
        return;
      }
      set({ error: error.response?.data?.message || 'Failed to fetch documents', isLoading: false });
    }
  },

  fetchSharedDocuments: async (page = 1, signal?: AbortSignal) => {
    set({ isLoading: true, error: null });
    try {
      const response = await api.listSharedDocuments(page, get().sharedPagination.perPage, signal);
      set({
        sharedDocuments: response.data || [],
        sharedPagination: {
          page: response.page,
          perPage: response.per_page,
          total: response.total,
          totalPages: response.total_pages,
        },
        isLoading: false,
      });
    } catch (error: any) {
      // Don't set error if request was aborted
      if (error.name === 'AbortError' || error.name === 'CanceledError') {
        return;
      }
      set({ error: error.response?.data?.message || 'Failed to fetch shared documents', isLoading: false });
    }
  },

  fetchDocument: async (id: string) => {
    set({ isLoading: true, error: null });
    try {
      const document = await api.getDocument(id);
      set({ selectedDocument: document, isLoading: false });
    } catch (error: any) {
      set({ error: error.response?.data?.message || 'Failed to fetch document', isLoading: false });
    }
  },

  uploadDocument: async (file: File, name?: string) => {
    set({ isLoading: true, error: null });
    try {
      const document = await api.uploadDocument(file, name);
      set((state) => ({
        documents: [document, ...state.documents],
        isLoading: false,
      }));
      return document;
    } catch (error: any) {
      set({ error: error.response?.data?.message || 'Failed to upload document', isLoading: false });
      throw error;
    }
  },

  renameDocument: async (id: string, name: string) => {
    set({ isLoading: true, error: null });
    try {
      await api.renameDocument(id, name);
      set((state) => ({
        documents: state.documents.map((doc) =>
          doc.id === id ? { ...doc, name } : doc
        ),
        selectedDocument:
          state.selectedDocument?.id === id
            ? { ...state.selectedDocument, name }
            : state.selectedDocument,
        isLoading: false,
      }));
    } catch (error: any) {
      set({ error: error.response?.data?.message || 'Failed to rename document', isLoading: false });
      throw error;
    }
  },

  deleteDocument: async (id: string) => {
    set({ isLoading: true, error: null });
    try {
      await api.deleteDocument(id);
      set((state) => ({
        documents: state.documents.filter((doc) => doc.id !== id),
        selectedDocument: state.selectedDocument?.id === id ? null : state.selectedDocument,
        isLoading: false,
      }));
    } catch (error: any) {
      set({ error: error.response?.data?.message || 'Failed to delete document', isLoading: false });
      throw error;
    }
  },

  shareDocument: async (id: string, email: string, permission: 'view' | 'edit') => {
    set({ isLoading: true, error: null });
    try {
      await api.shareDocument(id, email, permission);
      set({ isLoading: false });
    } catch (error: any) {
      set({ error: error.response?.data?.message || 'Failed to share document', isLoading: false });
      throw error;
    }
  },

  downloadDocument: async (id: string, filename: string) => {
    try {
      const blob = await api.downloadDocument(id);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error: any) {
      set({ error: error.response?.data?.message || 'Failed to download document' });
      throw error;
    }
  },

  clearError: () => set({ error: null }),

  setSelectedDocument: (document: Document | null) => set({ selectedDocument: document }),
}));
