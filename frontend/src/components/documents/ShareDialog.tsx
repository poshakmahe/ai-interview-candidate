'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import Modal from '@/components/ui/Modal';
import Button from '@/components/ui/Button';
import Input from '@/components/ui/Input';
import { Document, ShareFormData } from '@/types';
import { useDocuments } from '@/hooks/useDocuments';

const shareSchema = z.object({
  email: z.string().email('Please enter a valid email address'),
  permission: z.enum(['view', 'edit']),
});

interface ShareDialogProps {
  document: Document;
  isOpen: boolean;
  onClose: () => void;
}

const ShareDialog = ({ document, isOpen, onClose }: ShareDialogProps) => {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const { shareDocument, error, clearError } = useDocuments();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<ShareFormData>({
    resolver: zodResolver(shareSchema),
    defaultValues: {
      permission: 'view',
    },
  });

  const onSubmit = async (data: ShareFormData) => {
    setIsSubmitting(true);
    setSuccess(false);
    clearError();

    try {
      await shareDocument(document.id, data.email, data.permission);
      setSuccess(true);
      reset();
    } catch (err) {
      // Error is handled by the store
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClose = () => {
    reset();
    setSuccess(false);
    clearError();
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Share Document">
      <div className="space-y-4">
        <p className="text-sm text-gray-600">
          Share <span className="font-medium">{document.name}</span> with another user.
        </p>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <Input
            label="Email Address"
            type="email"
            placeholder="user@example.com"
            error={errors.email?.message}
            {...register('email')}
          />

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Permission Level
            </label>
            <div className="flex space-x-4">
              <label className="flex items-center">
                <input
                  type="radio"
                  value="view"
                  {...register('permission')}
                  className="h-4 w-4 text-primary-600 focus:ring-primary-500"
                />
                <span className="ml-2 text-sm text-gray-700">View only</span>
              </label>
              <label className="flex items-center">
                <input
                  type="radio"
                  value="edit"
                  {...register('permission')}
                  className="h-4 w-4 text-primary-600 focus:ring-primary-500"
                />
                <span className="ml-2 text-sm text-gray-700">Can edit</span>
              </label>
            </div>
          </div>

          {error && (
            <p className="text-sm text-red-600">{error}</p>
          )}

          {success && (
            <p className="text-sm text-green-600">
              Document shared successfully!
            </p>
          )}

          <div className="flex justify-end space-x-3 pt-4">
            <Button type="button" variant="secondary" onClick={handleClose}>
              Close
            </Button>
            <Button type="submit" isLoading={isSubmitting}>
              Share
            </Button>
          </div>
        </form>
      </div>
    </Modal>
  );
};

export default ShareDialog;
