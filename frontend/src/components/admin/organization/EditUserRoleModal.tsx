import React, { useState, useEffect, FormEvent } from 'react';
import apiClient from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';
import { useTranslation } from 'next-i18next';
import { User, UserRole } from '@/types'; // Supondo que UserRole é um enum/tipo

interface EditUserRoleModalProps {
  isOpen: boolean;
  onClose: () => void;
  user: User | null; // Usuário sendo editado
  organizationId: string;
  onRoleUpdated: () => void; // Callback para atualizar a lista na página pai
}

const EditUserRoleModal: React.FC<EditUserRoleModalProps> = ({
  isOpen,
  onClose,
  user,
  organizationId,
  onRoleUpdated,
}) => {
  const { t } = useTranslation(['usersManagement', 'common']);
  const notify = useNotifier();
  const [selectedRole, setSelectedRole] = useState<UserRole | string>('');
  const [isLoading, setIsLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    if (user) {
      setSelectedRole(user.role);
    }
  }, [user]);

  if (!isOpen || !user) {
    return null;
  }

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!selectedRole) {
      setFormError(t('edit_role_modal.error_role_required'));
      return;
    }
    setIsLoading(true);
    setFormError(null);

    try {
      await apiClient.put(`/api/v1/organizations/${organizationId}/users/${user.id}/role`, { role: selectedRole });
      notify.success(t('edit_role_modal.success_role_updated', { userName: user.name }));
      onRoleUpdated();
      onClose();
    } catch (err: any) {
      console.error("Erro ao atualizar role:", err);
      const apiError = err.response?.data?.error || t('common:unknown_error');
      setFormError(apiError); // Mostrar erro no modal
      notify.error(t('edit_role_modal.error_updating_role', { message: apiError }));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm">
      <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-md">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
          {t('edit_role_modal.title', { userName: user.name })}
        </h2>

        <p className="text-sm text-gray-600 dark:text-gray-400 mb-1">
            {t('edit_role_modal.user_label')}: <span className="font-medium">{user.name} ({user.email})</span>
        </p>

        {formError && <p className="text-sm text-red-500 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md mb-4">{formError}</p>}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="role-select" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('edit_role_modal.role_label')}
            </label>
            <select
              id="role-select"
              name="role"
              value={selectedRole}
              onChange={(e) => setSelectedRole(e.target.value as UserRole)}
              required
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"
            >
              <option value="" disabled>{t('edit_role_modal.select_role_placeholder')}</option>
              <option value="admin">{t('roles.admin', { ns: 'common' })}</option>
              <option value="manager">{t('roles.manager', { ns: 'common' })}</option>
              <option value="user">{t('roles.user', { ns: 'common' })}</option>
            </select>
          </div>

          <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700 mt-6">
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary disabled:opacity-50"
            >
              {t('common:cancel_button')}
            </button>
            <button
              type="submit"
              disabled={isLoading || !selectedRole || selectedRole === user.role}
              className="px-4 py-2 text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 rounded-md shadow-sm disabled:opacity-50 flex items-center"
            >
              {isLoading && (
                <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              )}
              {t('common:save_changes_button')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default EditUserRoleModal;
