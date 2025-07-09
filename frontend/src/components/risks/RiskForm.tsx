import React, { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/router';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';
import { useNotifier } from '@/hooks/useNotifier';
import {
    RiskStatus,
    RiskImpact,
    RiskProbability,
    RiskCategory,
    UserLookup
} from '@/types';
import { useTranslation } from 'next-i18next'; // Importar useTranslation

interface RiskFormData {
  title: string;
  description: string;
  category: RiskCategory;
  impact: RiskImpact | "";
  probability: RiskProbability | "";
  status: RiskStatus;
  owner_id: string;
}

interface RiskFormProps {
  initialData?: RiskFormData & { id?: string };
  isEditing?: boolean;
  onSubmitSuccess?: () => void;
}

const RiskForm: React.FC<RiskFormProps> = ({ initialData, isEditing = false, onSubmitSuccess }) => {
  const { t } = useTranslation(['risks', 'common']); // Adicionar hook de tradução
  const router = useRouter();
  const { user, isLoading: authIsLoading } = useAuth();
  const notify = useNotifier();

  const [formData, setFormData] = useState<RiskFormData>({
    title: '',
    description: '',
    category: 'tecnologico',
    impact: '',
    probability: '',
    status: 'aberto',
    owner_id: '',
    ...(initialData || {}),
  });

  const [isLoading, setIsLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const [organizationUsers, setOrganizationUsers] = useState<UserLookup[]>([]);
  const [isLoadingUsers, setIsLoadingUsers] = useState(true);
  const [usersError, setUsersError] = useState<string | null>(null);

  useEffect(() => {
    const fetchOrganizationUsers = async () => {
      if (!user || authIsLoading) return;

      setIsLoadingUsers(true);
      setUsersError(null);
      try {
        const response = await apiClient.get<UserLookup[]>('/users/organization-lookup');
        setOrganizationUsers(response.data || []);

        if (!isEditing && user?.id && !formData.owner_id && response.data?.some(u => u.id === user.id)) {
            setFormData(prev => ({ ...prev, owner_id: user.id }));
        } else if (!isEditing && !formData.owner_id && response.data?.length > 0) {
            // No default owner if logged-in user is not in the list or no specific logic for first user
        }

      } catch (err: any) {
        console.error(t('form.error_loading_owners_console'), err); // Log traduzido
        setUsersError(t('form.error_loading_owners'));
        setOrganizationUsers([]);
      } finally {
        setIsLoadingUsers(false);
      }
    };

    fetchOrganizationUsers();
  }, [user, authIsLoading, isEditing, formData.owner_id, t]); // Adicionado t

 useEffect(() => {
    if (initialData) {
      setFormData(prev => ({ ...prev, ...initialData }));
    } else if (!isEditing && user && organizationUsers.length > 0) {
      const loggedUserInList = organizationUsers.find(u => u.id === user.id);
      if (loggedUserInList && !formData.owner_id) { // Apenas se owner_id não estiver já preenchido
        setFormData(prev => ({ ...prev, owner_id: user.id }));
      }
    } else if (!isEditing && user && organizationUsers.length === 0 && !isLoadingUsers && !formData.owner_id) {
        setFormData(prev => ({...prev, owner_id: user.id}));
    }
  }, [initialData, isEditing, user, organizationUsers, isLoadingUsers, formData.owner_id]);


  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value as any })); // as any para os tipos de enum
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsLoading(true);
    setFormError(null);

    if (!formData.impact || !formData.probability) {
        setFormError(t('form.error_impact_probability_required'));
        setIsLoading(false);
        return;
    }
    if (!formData.owner_id) {
        setFormError(t('form.error_owner_required'));
        setIsLoading(false);
        return;
    }

    try {
      if (isEditing && initialData?.id) {
        await apiClient.put(`/risks/${initialData.id}`, formData);
        notify.success(t('form.update_success_message'));
      } else {
        await apiClient.post('/risks', formData);
        notify.success(t('form.create_success_message'));
      }
      if (onSubmitSuccess) {
        onSubmitSuccess();
      } else {
        router.push('/admin/risks');
      }
    } catch (err: any) {
      console.error(t('form.save_error_console'), err); // Log traduzido
      const apiError = err.response?.data?.error || t('common:unknown_error');
      setFormError(apiError);
      notify.error(t('form.save_error_message', { message: apiError }));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {formError && <p className="text-red-500 bg-red-100 p-3 rounded-md">{formError}</p>}

      <div>
        <label htmlFor="title" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.field_title_label')}</label>
        <input type="text" name="title" id="title" value={formData.title} onChange={handleChange} required minLength={3} maxLength={255}
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>

      <div>
        <label htmlFor="description" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.field_description_label')}</label>
        <textarea name="description" id="description" value={formData.description} onChange={handleChange} rows={4}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label htmlFor="category" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.field_category_label')}</label>
          <select name="category" id="category" value={formData.category} onChange={handleChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
            <option value="" disabled>{t('common_placeholders.select_category')}</option>
            <option value="tecnologico">{t('form.option_category_tech')}</option>
            <option value="operacional">{t('form.option_category_op')}</option>
            <option value="legal">{t('form.option_category_legal')}</option>
          </select>
        </div>
        <div>
          <label htmlFor="status" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.field_status_label')}</label>
          <select name="status" id="status" value={formData.status} onChange={handleChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
            {/* <option value="" disabled>{t('common_placeholders.select_status')}</option> */}
            <option value="aberto">{t('form.option_status_open')}</option>
            <option value="em_andamento">{t('form.option_status_in_progress')}</option>
            <option value="mitigado">{t('form.option_status_mitigated')}</option>
            <option value="aceito">{t('form.option_status_accepted')}</option>
          </select>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label htmlFor="impact" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.field_impact_label')}</label>
          <select name="impact" id="impact" value={formData.impact} onChange={handleChange} required
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
            <option value="" disabled>{t('form.option_select_impact')}</option>
            <option value="Baixo">{t('form.option_impact_low')}</option>
            <option value="Médio">{t('form.option_impact_medium')}</option>
            <option value="Alto">{t('form.option_impact_high')}</option>
            <option value="Crítico">{t('form.option_impact_critical')}</option>
          </select>
        </div>
        <div>
          <label htmlFor="probability" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.field_probability_label')}</label>
          <select name="probability" id="probability" value={formData.probability} onChange={handleChange} required
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
            <option value="" disabled>{t('form.option_select_probability')}</option>
            <option value="Baixo">{t('form.option_probability_low')}</option>
            <option value="Médio">{t('form.option_probability_medium')}</option>
            <option value="Alto">{t('form.option_probability_high')}</option>
            <option value="Crítico">{t('form.option_probability_critical')}</option>
          </select>
        </div>
      </div>

      <div>
        <label htmlFor="owner_id" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.field_owner_label')}</label>
        {isLoadingUsers && <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{t('form.loading_owners')}</p>}
        {usersError && <p className="text-sm text-red-500 dark:text-red-400 mt-1">{t('form.error_loading_owners_manual_entry')}</p>}

        {!isLoadingUsers && !usersError && organizationUsers.length === 0 && user && (
             <p className="text-sm text-yellow-600 dark:text-yellow-400 mt-1">
                {t('form.no_other_users_found', { userName: user.name || 'usuário logado' })}
             </p>
        )}

        <select
            name="owner_id"
            id="owner_id"
            value={formData.owner_id}
            onChange={handleChange}
            required
            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2 disabled:opacity-50"
            disabled={isLoadingUsers || (usersError && organizationUsers.length === 0 && !user)} // Desabilitar se erro E não há fallback para user logado
        >
            <option value="" disabled>{t('form.select_owner_placeholder')}</option>
            {organizationUsers.map(orgUser => (
                <option key={orgUser.id} value={orgUser.id}>
                    {orgUser.name} {orgUser.id === user?.id ? t('form.owner_you_suffix') : ''}
                </option>
            ))}
            {usersError && organizationUsers.length === 0 && user && (
                 <option key={user.id} value={user.id}>
                    {user.name} {t('form.owner_fallback_suffix')}
                </option>
            )}
        </select>
        { (usersError && organizationUsers.length > 0) && // Se houve erro mas alguns usuários carregaram
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {t('form.owner_list_incomplete_warning')}
            </p>
        }
      </div>

      <div className="flex justify-end space-x-3 pt-4">
        <button type="button" onClick={() => router.push('/admin/risks')}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 dark:bg-gray-600 dark:text-gray-200 dark:hover:bg-gray-500 rounded-md shadow-sm">
          {t('common:cancel_button')}
        </button>
        <button type="submit" disabled={isLoading || isLoadingUsers}
                className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md shadow-sm disabled:opacity-50 flex items-center">
          {(isLoading || isLoadingUsers) && (
            <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          )}
          {isEditing ? t('common:save_changes_button') : t('common:create_button')}
        </button>
      </div>
    </form>
  );
};

export default RiskForm;
