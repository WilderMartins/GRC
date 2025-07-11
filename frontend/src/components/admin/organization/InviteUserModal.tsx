import React, { useState, FormEvent } from 'react';
import apiClient from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';
import { useTranslation } from 'next-i18next';
import { UserRole } from '@/types';

interface InviteUserModalProps {
  isOpen: boolean;
  onClose: () => void;
  organizationId: string;
  onUserInvited: () => void; // Callback para atualizar a lista na página pai
}

interface NewUserFormData {
  name: string;
  email: string;
  role: UserRole | string;
}

const InviteUserModal: React.FC<InviteUserModalProps> = ({
  isOpen,
  onClose,
  organizationId,
  onUserInvited,
}) => {
  const { t } = useTranslation(['usersManagement', 'common']);
  const notify = useNotifier();
  const [formData, setFormData] = useState<NewUserFormData>({
    name: '',
    email: '',
    role: 'user', // Default role
  });
  const [isLoading, setIsLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  if (!isOpen) {
    return null;
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!formData.name.trim() || !formData.email.trim() || !formData.role) {
      setFormError(t('invite_user_modal.error_all_fields_required'));
      return;
    }
    // Adicionar validação de email se desejado

    setIsLoading(true);
    setFormError(null);

    try {
      // Assumindo endpoint POST /organizations/{orgId}/users para criar/convidar
      // O backend pode enviar um email para o usuário definir a senha.
      await apiClient.post(`/organizations/${organizationId}/users`, formData);
      notify.success(t('invite_user_modal.success_user_invited', { email: formData.email }));
      onUserInvited();
      onClose();
      // Resetar formulário para a próxima vez
      setFormData({ name: '', email: '', role: 'user' });
    } catch (err: any) {
      console.error("Erro ao convidar usuário:", err);
      const apiError = err.response?.data?.error || t('common:unknown_error');
      setFormError(apiError);
      notify.error(t('invite_user_modal.error_inviting_user', { message: apiError }));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm">
      <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-lg">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-6">
          {t('invite_user_modal.title')}
        </h2>

        {formError && <p className="text-sm text-red-500 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md mb-4">{formError}</p>}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('invite_user_modal.name_label')}
            </label>
            <input
              type="text"
              name="name"
              id="name"
              value={formData.name}
              onChange={handleChange}
              required
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"
              placeholder={t('invite_user_modal.name_placeholder')}
            />
          </div>
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('invite_user_modal.email_label')}
            </label>
            <input
              type="email"
              name="email"
              id="email"
              value={formData.email}
              onChange={handleChange}
              required
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"
              placeholder={t('invite_user_modal.email_placeholder')}
            />
          </div>
          <div>
            <label htmlFor="role" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('invite_user_modal.role_label')}
            </label>
            <select
              id="role"
              name="role"
              value={formData.role}
              onChange={handleChange}
              required
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"
            >
              <option value="user">{t('roles.user', { ns: 'common' })}</option>
              <option value="manager">{t('roles.manager', { ns: 'common' })}</option>
              <option value="admin">{t('roles.admin', { ns: 'common' })}</option>
            </select>
          </div>

          <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700 mt-6">
            <button
              type="button"
              onClick={() => {
                onClose();
                setFormData({ name: '', email: '', role: 'user' }); // Resetar form ao cancelar
                setFormError(null);
              }}
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary disabled:opacity-50"
            >
              {t('common:cancel_button')}
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 rounded-md shadow-sm disabled:opacity-50 flex items-center"
            >
              {isLoading && (
                <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              )}
              {t('invite_user_modal.submit_button')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default InviteUserModal;
