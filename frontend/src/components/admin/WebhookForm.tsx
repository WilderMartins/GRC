import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';

// Tipos (devem ser consistentes com os modelos do backend)
const availableEventTypes = [
  { id: 'risk_created', label: 'Risco Criado' },
  { id: 'risk_status_changed', label: 'Status do Risco Alterado' },
  // Adicionar outros tipos de evento aqui conforme o backend evolui
] as const; // as const para inferir tipos literais

type WebhookEventType = typeof availableEventTypes[number]['id'];

interface WebhookFormData {
  name: string;
  url: string;
  event_types: WebhookEventType[]; // Array de strings para os checkboxes/multiselect
  is_active: boolean;
}

// Tipo para o Webhook como ele vem da API ou está na lista (com event_types como string)
interface WebhookConfigForForm {
    id?: string;
    name: string;
    url: string;
    event_types: string; // string separada por vírgulas
    is_active: boolean;
}

interface WebhookFormProps {
  organizationId: string; // Necessário para construir a URL da API
  initialData?: WebhookConfigForForm;
  isEditing?: boolean;
  onSubmitSuccess: (webhookData: any) => void;
}

const WebhookForm: React.FC<WebhookFormProps> = ({
  organizationId,
  initialData,
import { useTranslation } from 'next-i18next'; // Importar

// ...

const WebhookForm: React.FC<WebhookFormProps> = ({
  organizationId,
  initialData,
  isEditing = false,
  onSubmitSuccess,
}) => {
  const { t } = useTranslation(['webhooks', 'common']); // Adicionar hook
  const router = useRouter();
import { useTranslation } from 'next-i18next';

// ... (outras definições)

const WebhookForm: React.FC<WebhookFormProps> = ({
  organizationId,
  initialData,
  isEditing = false,
  onSubmitSuccess,
}) => {
  const router = useRouter();
  const { t } = useTranslation(['webhooks', 'common']);
  const [formData, setFormData] = useState<WebhookFormData>({
    name: '',
    url: '',
    event_types: [],
    is_active: true,
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (initialData) {
      setFormData({
        name: initialData.name || '',
        url: initialData.url || '',
        event_types: initialData.event_types ? initialData.event_types.split(',') as WebhookEventType[] : [],
        is_active: initialData.is_active === undefined ? true : initialData.is_active,
      });
    } else {
      setFormData({ name: '', url: '', event_types: [], is_active: true });
    }
  }, [initialData]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    if (type === 'checkbox' && name === 'is_active') {
        const { checked } = e.target as HTMLInputElement;
        setFormData(prev => ({ ...prev, [name]: checked }));
    } else {
        setFormData(prev => ({ ...prev, [name]: value }));
    }
  };

  const handleEventTypeChange = (eventType: WebhookEventType) => {
    setFormData(prev => {
      const newEventTypes = prev.event_types.includes(eventType)
        ? prev.event_types.filter(et => et !== eventType)
        : [...prev.event_types, eventType];
      return { ...prev, event_types: newEventTypes };
    });
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    if (formData.event_types.length === 0) {
        setError("Selecione pelo menos um tipo de evento.");
        setIsLoading(false);
        return;
    }

    // O backend espera 'event_types' como uma string separada por vírgulas no payload JSON
    // mas o modelo WebhookPayload no handler do backend espera um []string.
    // A API do backend para criar/atualizar webhooks espera um JSON com "event_types": ["type1", "type2"]
    // Portanto, formData.event_types já está no formato correto para o payload.
    const payload = {
      name: formData.name,
      url: formData.url,
      event_types: formData.event_types, // Enviar como array de strings
      is_active: formData.is_active,
    };

    try {
      let response;
      if (isEditing && initialData?.id) {
        response = await apiClient.put(`/api/v1/organizations/${organizationId}/webhooks/${initialData.id}`, payload);
      } else {
        response = await apiClient.post(`/api/v1/organizations/${organizationId}/webhooks`, payload);
      }
      onSubmitSuccess(response.data);
      // A prop onClose foi removida para adequar o uso em página inteira
    } catch (err: any) {
      console.error("Erro ao salvar webhook:", err);
      setError(err.response?.data?.error || err.message || "Falha ao salvar webhook.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-white">
        {isEditing ? 'Editar' : 'Adicionar Novo'} Webhook
      </h3>
      {error && <p className="text-sm text-red-600 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md">{error}</p>}

      <div>
        <label htmlFor="name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Nome do Webhook</label>
        <input type="text" name="name" id="name" value={formData.name} onChange={handleInputChange} required minLength={3} maxLength={100}
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white"/>
      </div>

      <div>
        <label htmlFor="url" className="block text-sm font-medium text-gray-700 dark:text-gray-300">URL do Webhook</label>
        <input type="url" name="url" id="url" value={formData.url} onChange={handleInputChange} required maxLength={2048}
               placeholder="https://your-webhook-url.com/..."
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white"/>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Tipos de Evento</label>
        <div className="mt-2 space-y-2">
            {availableEventTypes.map(eventTypeOption => (
                <div key={eventTypeOption.id} className="flex items-center">
                    <input
                        id={`event-${eventTypeOption.id}`}
                        name="event_types"
                        type="checkbox"
                        value={eventTypeOption.id}
                        checked={formData.event_types.includes(eventTypeOption.id)}
                        onChange={() => handleEventTypeChange(eventTypeOption.id)}
                        className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 dark:border-gray-600"
                    />
                    <label htmlFor={`event-${eventTypeOption.id}`} className="ml-2 block text-sm text-gray-900 dark:text-gray-300">
                        {eventTypeOption.label}
                    </label>
                </div>
            ))}
        </div>
         {formData.event_types.length === 0 && <p className="mt-1 text-xs text-red-500">Selecione pelo menos um tipo de evento.</p>}
      </div>

      <div className="flex items-center">
        <input type="checkbox" name="is_active" id="is_active_webhook" checked={formData.is_active} onChange={handleInputChange}
               className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 dark:border-gray-600"/>
        <label htmlFor="is_active_webhook" className="ml-2 block text-sm text-gray-900 dark:text-gray-300">Ativo</label>
      </div>

      <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700 mt-6">
        <button type="button" onClick={() => router.push('/admin/organization/webhooks')} disabled={isLoading}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 disabled:opacity-50 transition-colors">
          Cancelar
        </button>
        <button type="submit" disabled={isLoading || formData.event_types.length === 0}
                className="px-4 py-2 text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 rounded-md shadow-sm disabled:opacity-50 flex items-center transition-colors">
          {isLoading && (
            <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          )}
          {isEditing ? 'Salvar Alterações' : 'Adicionar Webhook'}
        </button>
      </div>
    </form>
  );
};

export default WebhookForm;
