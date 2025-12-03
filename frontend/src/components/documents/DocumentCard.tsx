'use client';

import { useState } from 'react';
import { format } from 'date-fns';
import { FileText, Download, Trash2, Share2, Edit3, MoreVertical } from 'lucide-react';
import { Document } from '@/types';
import Button from '@/components/ui/Button';
import { useDocuments } from '@/hooks/useDocuments';

interface DocumentCardProps {
  document: Document;
  onShare?: () => void;
  onRename?: () => void;
  showOwner?: boolean;
}

const DocumentCard = ({ document, onShare, onRename, showOwner = false }: DocumentCardProps) => {
  const [showMenu, setShowMenu] = useState(false);
  const { deleteDocument, downloadDocument, isLoading } = useDocuments();

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const handleDownload = async () => {
    try {
      await downloadDocument(document.id, document.original_name);
    } catch (error) {
      console.error('Download failed:', error);
    }
  };

  const handleDelete = async () => {
    if (confirm('Are you sure you want to delete this document?')) {
      try {
        await deleteDocument(document.id);
      } catch (error) {
        console.error('Delete failed:', error);
      }
    }
    setShowMenu(false);
  };

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between">
        <div className="flex items-start space-x-3 flex-1 min-w-0">
          <div className="flex-shrink-0">
            <FileText className="h-10 w-10 text-primary-500" />
          </div>
          <div className="flex-1 min-w-0">
            <h3 className="text-sm font-medium text-gray-900 truncate">
              {document.name}
            </h3>
            <p className="text-xs text-gray-500 truncate">
              {document.original_name}
            </p>
            <div className="flex items-center space-x-2 mt-1">
              <span className="text-xs text-gray-400">
                {formatFileSize(document.size)}
              </span>
              <span className="text-xs text-gray-300">•</span>
              <span className="text-xs text-gray-400">
                {format(new Date(document.created_at), 'MMM d, yyyy')}
              </span>
            </div>
            {showOwner && document.owner_name && (
              <p className="text-xs text-gray-500 mt-1">
                Shared by: {document.owner_name}
              </p>
            )}
          </div>
        </div>

        {/* Actions Menu */}
        <div className="relative flex-shrink-0">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowMenu(!showMenu)}
            className="p-1"
          >
            <MoreVertical className="h-4 w-4" />
          </Button>

          {showMenu && (
            <>
              <div
                className="fixed inset-0 z-10"
                onClick={() => setShowMenu(false)}
              />
              <div className="absolute right-0 mt-1 w-48 bg-white rounded-md shadow-lg border border-gray-200 z-20">
                <div className="py-1">
                  <button
                    onClick={handleDownload}
                    className="flex items-center w-full px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                  >
                    <Download className="h-4 w-4 mr-2" />
                    Download
                  </button>
                  {onRename && (
                    <button
                      onClick={() => {
                        onRename();
                        setShowMenu(false);
                      }}
                      className="flex items-center w-full px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                    >
                      <Edit3 className="h-4 w-4 mr-2" />
                      Rename
                    </button>
                  )}
                  {onShare && (
                    <button
                      onClick={() => {
                        onShare();
                        setShowMenu(false);
                      }}
                      className="flex items-center w-full px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                    >
                      <Share2 className="h-4 w-4 mr-2" />
                      Share
                    </button>
                  )}
                  {!showOwner && (
                    <button
                      onClick={handleDelete}
                      disabled={isLoading}
                      className="flex items-center w-full px-4 py-2 text-sm text-red-600 hover:bg-red-50"
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      Delete
                    </button>
                  )}
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Encryption Badge */}
      {document.is_encrypted && (
        <div className="mt-3 flex items-center">
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
            Encrypted ({document.encryption_algo})
          </span>
        </div>
      )}
    </div>
  );
};

export default DocumentCard;
