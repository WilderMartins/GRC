import React, { useState, useEffect, FormEvent } from 'react';
import apiClient from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';
import { useTranslation } from 'next-i18next';
import { WebhookConfiguration } from '@/types';

// Definir os tipos de evento possíveis que o backend suporta.
// Idealmente, isso viria de uma constante compartilhada ou da API.
const AVAILABLE_EVENT_TYPES = [
  { id: 'risk_created', labelKey: 'form.event_type_risk_created' },
  { id: 'risk_updated', labelKey: 'form.event_type_risk_updated' },
  { id: 'risk_deleted', labelKey: 'form.event_type_risk_deleted' },
  { id: 'risk_status_changed', labelKey: 'form.event_type_risk_status_changed' },
  { id: 'vulnerability_created', labelKey: 'form.event_type_vulnerability_created' },
  { id: 'vulnerability_updated', labelKey: 'form.event_type_vulnerability_updated' },
  // Adicionar mais tipos de evento conforme necessário
];

interface WebhookFormData {
  name: string;
  url: string;
  event_types: string[]; // Array de strings para os checkboxes
  is_active: boolean;
  secret?: string; // Opcional, se o backend suportar segredos de webhook
}

interface WebhookFormProps {
  organizationId: string;
  initialData?: WebhookConfiguration; // Para edição
  isEditing?: boolean;
  onSubmitSuccess: () => void;
}

const WebhookForm: React.FC<WebhookFormProps> = ({
  organizationId,
  initialData,
  isEditing = false,
  onSubmitSuccess,
}) => {
  const { t } = useTranslation(['webhooks', 'common']);
  const notify = useNotifier();

  const [formData, setFormData] = useState<WebhookFormData>({
    name: '',
    url: '',
    event_types: [],
    is_active: true,
    secret: '', // Inicializar
    ...(initialData ? {
      ...initialData,
      // A API retorna event_types como string JSON, precisamos parsear para o formulário
      event_types: initialData.event_types && typeof initialData.event_types === 'string'
                   ? JSON.parse(initialData.event_types)
                   : (Array.isArray(initialData.event_types_list) ? initialData.event_types_list : []),
      secret: initialData.secret || '', // Lidar com o segredo
    } : {}),
  });

  const [isLoading, setIsLoading] = useState(false);
  const [isSendingTest, setIsSendingTest] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const handleSendTest = async () => {
    setIsSendingTest(true);
    notify.info(t('form.sending_test_notification'));
    try {
      await apiClient.post(`/organizations/${organizationId}/webhooks/${initialData?.id}/test`);
      notify.success(t('form.send_test_success_message'));
    } catch (err: any) {
      console.error("Erro ao enviar webhook de teste:", err);
      notify.error(t('form.send_test_error_message', { message: err.response?.data?.error || t('common:unknown_error') }));
    } finally {
      setIsSendingTest(false);
    }
  };

  useEffect(() => {
    if (isEditing && initialData) {
      setFormData({
        name: initialData.name || '',
        url: initialData.url || '',
        event_types: initialData.event_types_list || (initialData.event_types && typeof initialData.event_types === 'string' ? JSON.parse(initialData.event_types) : []),
        is_active: initialData.is_active === undefined ? true : initialData.is_active,
        secret: initialData.secret || '',
      });
    }
  }, [initialData, isEditing]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    if (type === 'checkbox' && name !== 'event_type_option') { // Checkbox geral (is_active)
      setFormData(prev => ({ ...prev, [name]: (e.target as HTMLInputElement).checked }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
    }
  };

  const handleEventTypeChange = (eventId: string) => {
    setFormData(prev => {
      const newEventTypes = prev.event_types.includes(eventId)
        ? prev.event_types.filter(et => et !== eventId)
        : [...prev.event_types, eventId];
      return { ...prev, event_types: newEventTypes };
    });
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsLoading(true);
    setFormError(null);

    if (formData.event_types.length === 0) {
        setFormError(t('form.error_event_types_required'));
        setIsLoading(false);
        return;
    }

    const payload = {
      name: formData.name,
      url: formData.url,
      event_types: JSON.stringify(formData.event_types), // API espera string JSON
      is_active: formData.is_active,
      secret: formData.secret || undefined, // Enviar undefined se vazio para não enviar chave vazia
    };

    try {
      if (isEditing && initialData?.id) {
        await apiClient.put(`/organizations/${organizationId}/webhooks/${initialData.id}`, payload);
        notify.success(t('form.update_success_message'));
      } else {
        await apiClient.post(`/organizations/${organizationId}/webhooks`, payload);
        notify.success(t('form.create_success_message'));
      }
      onSubmitSuccess();
    } catch (err: any) {
      console.error(t('form.save_error_console'), err);
      const apiError = err.response?.data?.error || err.response?.data?.message || t('common:unknown_error');
      setFormError(apiError);
      notify.error(t('form.save_error_message', { message: apiError }));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {formError && <p className="text-sm text-red-500 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md">{formError}</p>}

      <div>
        <label htmlFor="name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.name_label')}</label>
        <input type="text" name="name" id="name" value={formData.name} onChange={handleInputChange} required
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>

      <div>
        <label htmlFor="url" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.url_label')}</label>
        <input type="url" name="url" id="url" value={formData.url} onChange={handleInputChange} required
               placeholder="https://example.com/webhook-receiver"
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>

      <div>
        <label htmlFor="secret" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('form.secret_label')} <span className="text-xs text-gray-500 dark:text-gray-400">({t('common:optional')})</span>
        </label>
        <input type="password" name="secret" id="secret" value={formData.secret} onChange={handleInputChange}
               placeholder={t('form.secret_placeholder')}
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{t('form.secret_help')}</p>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.event_types_label')}</label>
        <div className="mt-2 space-y-2 p-3 border border-gray-300 dark:border-gray-600 rounded-md max-h-60 overflow-y-auto">
          {AVAILABLE_EVENT_TYPES.map(eventType => (
            <div key={eventType.id} className="flex items-start">
              <div className="flex items-center h-5">
                <input
                  id={`event_type_${eventType.id}`}
                  name="event_type_option" // Nome comum para grupo de checkboxes
                  type="checkbox"
                  checked={formData.event_types.includes(eventType.id)}
                  onChange={() => handleEventTypeChange(eventType.id)}
                  className="focus:ring-brand-primary h-4 w-4 text-brand-primary border-gray-300 rounded"
                />
              </div>
              <div className="ml-3 text-sm">
                <label htmlFor={`event_type_${eventType.id}`} className="font-medium text-gray-700 dark:text-gray-300">
                  {t(eventType.labelKey)}
                </label>
              </div>
            </div>
          ))}
        </div>
        {formError && formData.event_types.length === 0 && <p className="text-xs text-red-500 mt-1">{t('form.error_event_types_required')}</p>}
      </div>

      <div className="flex items-start">
        <div className="flex items-center h-5">
            <input id="is_active" name="is_active" type="checkbox" checked={formData.is_active} onChange={handleInputChange}
                    className="focus:ring-brand-primary h-4 w-4 text-brand-primary border-gray-300 rounded"/>
        </div>
        <div className="ml-3 text-sm">
            <label htmlFor="is_active" className="font-medium text-gray-700 dark:text-gray-300">{t('form.is_active_label')}</label>
        </div>
      </div>

      <div className="flex justify-between items-center pt-4 border-t border-gray-200 dark:border-gray-700 mt-6">
        <div>
          {isEditing && (
            <button
              type="button"
              onClick={handleSendTest}
              disabled={isLoading || isSendingTest}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary disabled:opacity-50"
            >
              {isSendingTest ? t('form.sending_test_button') : t('form.send_test_button')}
            </button>
          )}
        </div>
        <div className="flex space-x-3">
          <button type="button" onClick={() => router.back()}
                  className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary disabled:opacity-50"
                  disabled={isLoading}>
            {t('common:cancel_button')}
          </button>
          <button type="submit" disabled={isLoading || formData.event_types.length === 0}
                  className="px-4 py-2 text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 rounded-md shadow-sm disabled:opacity-50 flex items-center">
          {isLoading && (
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

export default WebhookForm;
