'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Plus, FileText, RefreshCw } from 'lucide-react';
import Header from '@/components/layout/Header';
import Button from '@/components/ui/Button';
import Card from '@/components/ui/Card';
import Modal from '@/components/ui/Modal';
import Input from '@/components/ui/Input';
import DocumentCard from '@/components/documents/DocumentCard';
import UploadForm from '@/components/documents/UploadForm';
import ShareDialog from '@/components/documents/ShareDialog';
import SummaryModal from '@/components/documents/SummaryModal';
import { useAuth, useIsAuthenticated } from '@/hooks/useAuth';
import { useDocuments } from '@/hooks/useDocuments';
import { Document } from '@/types';

export default function DashboardPage() {
  const router = useRouter();
  const { isLoading: authLoading, fetchUser } = useAuth();
  const isAuthenticated = useIsAuthenticated();
  const {
    documents,
    pagination,
    isLoading,
    error,
    fetchDocuments,
    renameDocument,
  } = useDocuments();

  const [showUploadModal, setShowUploadModal] = useState(false);
  const [showShareDialog, setShowShareDialog] = useState(false);
  const [showRenameModal, setShowRenameModal] = useState(false);
  const [showSummaryModal, setShowSummaryModal] = useState(false);
  const [selectedDocument, setSelectedDocument] = useState<Document | null>(null);
  const [newName, setNewName] = useState('');

  useEffect(() => {
    fetchUser();
  }, [fetchUser]);

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push('/login');
    }
  }, [isAuthenticated, authLoading, router]);

  useEffect(() => {
    if (!isAuthenticated) return;

    const abortController = new AbortController();
    fetchDocuments(1, abortController.signal);

    return () => {
      abortController.abort();
    };
  }, [isAuthenticated]);

  const handleUploadSuccess = () => {
    setShowUploadModal(false);
    fetchDocuments();
  };

  const handleShare = (doc: Document) => {
    setSelectedDocument(doc);
    setShowShareDialog(true);
  };

  const handleRename = (doc: Document) => {
    setSelectedDocument(doc);
    setNewName(doc.name);
    setShowRenameModal(true);
  };

  const handleSummarize = (doc: Document) => {
    setSelectedDocument(doc);
    setShowSummaryModal(true);
  };

  const handleRenameSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDocument || !newName.trim()) return;

    try {
      await renameDocument(selectedDocument.id, newName.trim());
      setShowRenameModal(false);
      setSelectedDocument(null);
      setNewName('');
    } catch (err) {
      // Error is handled by the store
    }
  };

  if (authLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  return (
    <>
      <Header />
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Page Header */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">My Documents</h1>
            <p className="text-gray-600 mt-1">
              {pagination.total} document{pagination.total !== 1 ? 's' : ''} stored securely
            </p>
          </div>
          <div className="flex items-center space-x-3">
            <Button
              variant="secondary"
              onClick={() => fetchDocuments()}
              disabled={isLoading}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
            <Button onClick={() => setShowUploadModal(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Upload Document
            </Button>
          </div>
        </div>

        {/* Error Message */}
        {error && (
          <Card className="mb-6 bg-red-50 border-red-200">
            <p className="text-red-600">{error}</p>
          </Card>
        )}

        {/* Documents Grid */}
        {documents.length > 0 ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {documents.map((doc) => (
              <DocumentCard
                key={doc.id}
                document={doc}
                onShare={() => handleShare(doc)}
                onRename={() => handleRename(doc)}
                onSummarize={() => handleSummarize(doc)}
              />
            ))}
          </div>
        ) : (
          <Card className="text-center py-12">
            <FileText className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              No documents yet
            </h3>
            <p className="text-gray-600 mb-4">
              Upload your first document to get started.
            </p>
            <Button onClick={() => setShowUploadModal(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Upload Document
            </Button>
          </Card>
        )}

        {/* Pagination */}
        {pagination.totalPages > 1 && (
          <div className="flex justify-center mt-8 space-x-2">
            {Array.from({ length: pagination.totalPages }, (_, i) => i + 1).map((page) => (
              <Button
                key={page}
                variant={page === pagination.page ? 'primary' : 'secondary'}
                size="sm"
                onClick={() => fetchDocuments(page)}
              >
                {page}
              </Button>
            ))}
          </div>
        )}

        {/* Upload Modal */}
        <Modal
          isOpen={showUploadModal}
          onClose={() => setShowUploadModal(false)}
          title="Upload Document"
        >
          <UploadForm
            onSuccess={handleUploadSuccess}
            onCancel={() => setShowUploadModal(false)}
          />
        </Modal>

        {/* Share Dialog */}
        {selectedDocument && (
          <ShareDialog
            document={selectedDocument}
            isOpen={showShareDialog}
            onClose={() => {
              setShowShareDialog(false);
              setSelectedDocument(null);
            }}
          />
        )}

        {/* Rename Modal */}
        <Modal
          isOpen={showRenameModal}
          onClose={() => {
            setShowRenameModal(false);
            setSelectedDocument(null);
            setNewName('');
          }}
          title="Rename Document"
          size="sm"
        >
          <form onSubmit={handleRenameSubmit} className="space-y-4">
            <Input
              label="Document Name"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="Enter new name"
            />
            <div className="flex justify-end space-x-3">
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  setShowRenameModal(false);
                  setSelectedDocument(null);
                  setNewName('');
                }}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={!newName.trim()}>
                Rename
              </Button>
            </div>
          </form>
        </Modal>

        {/* Summary Modal */}
        <SummaryModal
          isOpen={showSummaryModal}
          onClose={() => {
            setShowSummaryModal(false);
            setSelectedDocument(null);
          }}
          document={selectedDocument}
        />
      </main>
    </>
  );
}
