'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Share2, RefreshCw } from 'lucide-react';
import Header from '@/components/layout/Header';
import Button from '@/components/ui/Button';
import Card from '@/components/ui/Card';
import DocumentCard from '@/components/documents/DocumentCard';
import SummaryModal from '@/components/documents/SummaryModal';
import { useAuth, useIsAuthenticated } from '@/hooks/useAuth';
import { useDocuments } from '@/hooks/useDocuments';
import { Document } from '@/types';

export default function SharedDocumentsPage() {
  const router = useRouter();
  const { isLoading: authLoading, fetchUser } = useAuth();
  const isAuthenticated = useIsAuthenticated();
  const {
    sharedDocuments,
    sharedPagination,
    isLoading,
    error,
    fetchSharedDocuments,
  } = useDocuments();

  const [showSummaryModal, setShowSummaryModal] = useState(false);
  const [selectedDocument, setSelectedDocument] = useState<Document | null>(null);

  const handleSummarize = (doc: Document) => {
    setSelectedDocument(doc);
    setShowSummaryModal(true);
  };

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
    fetchSharedDocuments(1, abortController.signal);

    return () => {
      abortController.abort();
    };
  }, [isAuthenticated]);

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
            <h1 className="text-2xl font-bold text-gray-900">Shared with Me</h1>
            <p className="text-gray-600 mt-1">
              {sharedPagination.total} document{sharedPagination.total !== 1 ? 's' : ''} shared with you
            </p>
          </div>
          <Button
            variant="secondary"
            onClick={() => fetchSharedDocuments()}
            disabled={isLoading}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        </div>

        {/* Error Message */}
        {error && (
          <Card className="mb-6 bg-red-50 border-red-200">
            <p className="text-red-600">{error}</p>
          </Card>
        )}

        {/* Documents Grid */}
        {sharedDocuments.length > 0 ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {sharedDocuments.map((doc) => (
              <DocumentCard
                key={doc.id}
                document={doc}
                showOwner={true}
                onSummarize={() => handleSummarize(doc)}
              />
            ))}
          </div>
        ) : (
          <Card className="text-center py-12">
            <Share2 className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              No shared documents
            </h3>
            <p className="text-gray-600">
              Documents shared with you will appear here.
            </p>
          </Card>
        )}

        {/* Pagination */}
        {sharedPagination.totalPages > 1 && (
          <div className="flex justify-center mt-8 space-x-2">
            {Array.from({ length: sharedPagination.totalPages }, (_, i) => i + 1).map((page) => (
              <Button
                key={page}
                variant={page === sharedPagination.page ? 'primary' : 'secondary'}
                size="sm"
                onClick={() => fetchSharedDocuments(page)}
              >
                {page}
              </Button>
            ))}
          </div>
        )}

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
