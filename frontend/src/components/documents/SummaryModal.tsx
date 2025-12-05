'use client';

import { useEffect, useState } from 'react';
import { Loader2, AlertCircle, Sparkles } from 'lucide-react';
import Modal from '@/components/ui/Modal';
import Button from '@/components/ui/Button';
import { Document } from '@/types';
import { useDocuments } from '@/hooks/useDocuments';

interface SummaryModalProps {
  isOpen: boolean;
  onClose: () => void;
  document: Document | null;
}

const SummaryModal = ({ isOpen, onClose, document }: SummaryModalProps) => {
  const [summary, setSummary] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { summarizeDocument, isSummarizing } = useDocuments();

  useEffect(() => {
    if (isOpen && document) {
      // Reset state when opening
      setSummary(null);
      setError(null);

      // Fetch summary
      summarizeDocument(document.id)
        .then((result) => {
          setSummary(result);
        })
        .catch((err) => {
          setError(err.message || 'Failed to generate summary');
        });
    }
  }, [isOpen, document, summarizeDocument]);

  const handleClose = () => {
    setSummary(null);
    setError(null);
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Document Summary" size="lg">
      <div className="space-y-4">
        {/* Document Info */}
        {document && (
          <div className="flex items-center space-x-2 pb-3 border-b border-gray-200">
            <Sparkles className="h-5 w-5 text-primary-500" />
            <span className="text-sm text-gray-600">
              Summarizing: <span className="font-medium text-gray-900">{document.name}</span>
            </span>
          </div>
        )}

        {/* Loading State */}
        {isSummarizing && (
          <div className="flex flex-col items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-primary-500 mb-4" />
            <p className="text-sm text-gray-600">Generating summary with AI...</p>
            <p className="text-xs text-gray-400 mt-1">This may take a few moments</p>
          </div>
        )}

        {/* Error State */}
        {error && !isSummarizing && (
          <div className="flex items-start space-x-3 p-4 bg-red-50 rounded-lg">
            <AlertCircle className="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-red-800">Failed to generate summary</p>
              <p className="text-sm text-red-600 mt-1">{error}</p>
            </div>
          </div>
        )}

        {/* Summary Content */}
        {summary && !isSummarizing && (
          <div className="prose prose-sm max-w-none">
            <div className="bg-gray-50 rounded-lg p-4">
              <div className="whitespace-pre-wrap text-gray-700 text-sm leading-relaxed">
                {summary}
              </div>
            </div>
          </div>
        )}

        {/* Actions */}
        <div className="flex justify-end pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={handleClose}>
            Close
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default SummaryModal;
