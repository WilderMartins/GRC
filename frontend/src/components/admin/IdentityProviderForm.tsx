import React, { useState, useEffect } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path
import { useAuth } from '@/contexts/AuthContext'; // Para obter organization_id

// Tipos (devem ser consistentes com os usados na página de listagem e no backend)
type IdentityProviderType = 'saml' | 'oauth2_google' | 'oauth2_github' | '';

interface IdentityProviderFormData {
  name: string;
  provider_type: IdentityProviderType;
  is_active: boolean;
  config_json_string: string; // Para o textarea, será parseado/stringificado
  attribute_mapping_json_string: string; // Para o textarea
}

// Tipo para o IdP como ele é no estado da página de listagem (com JSONs parseados)
interface IdentityProviderForForm {
    id?: string;
    name: string;
    provider_type: IdentityProviderType;
    is_active: boolean;
    config_json_parsed: Record<string, any>;
    attribute_mapping_json_parsed?: Record<string, any>;
}


interface IdentityProviderFormProps {
  initialData?: IdentityProviderForForm;
  isEditing?: boolean;
  onClose: () => void;
  onSubmitSuccess: (idpData: any) => void;
}

const IdentityProviderForm: React.FC<IdentityProviderFormProps> = ({
  initialData,
  isEditing = false,
  onClose,
  onSubmitSuccess,
}) => {
  const { user } = useAuth();
  const [formData, setFormData] = useState<IdentityProviderFormData>({
    name: '',
    provider_type: '',
    is_active: true,
    config_json_string: '',
    attribute_mapping_json_string: '',
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (initialData) {
      setFormData({
        name: initialData.name || '',
        provider_type: initialData.provider_type || '',
        is_active: initialData.is_active === undefined ? true : initialData.is_active,
        config_json_string: initialData.config_json_parsed ? JSON.stringify(initialData.config_json_parsed, null, 2) : '',
        attribute_mapping_json_string: initialData.attribute_mapping_json_parsed ? JSON.stringify(initialData.attribute_mapping_json_parsed, null, 2) : '',
      });
    } else {
      // Reset para valores padrão para um novo formulário
      setFormData({
        name: '',
        provider_type: '',
        is_active: true,
        config_json_string: '',
        attribute_mapping_json_string: '',
      });
    }
  }, [initialData]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    if (type === 'checkbox') {
        const { checked } = e.target as HTMLInputElement;
        setFormData(prev => ({ ...prev, [name]: checked }));
    } else {
        setFormData(prev => ({ ...prev, [name]: value }));
    }
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    if (!formData.provider_type) {
        setError("O campo Tipo de Provedor é obrigatório.");
        setIsLoading(false);
        return;
    }

    let configJsonParsed, attributeMappingJsonParsed;
    try {
        configJsonParsed = JSON.parse(formData.config_json_string || '{}');
    } catch (jsonErr) {
        setError("Configuração JSON inválida: " + jsonErr.message);
        setIsLoading(false);
        return;
    }
    if (formData.attribute_mapping_json_string) {
        try {
            attributeMappingJsonParsed = JSON.parse(formData.attribute_mapping_json_string);
        } catch (jsonErr) {
            setError("Mapeamento de Atributos JSON inválido: " + jsonErr.message);
            setIsLoading(false);
            return;
        }
    }


    const payload = {
      name: formData.name,
      provider_type: formData.provider_type,
      is_active: formData.is_active,
      config_json: configJsonParsed, // Enviar como objeto JSON
      attribute_mapping_json: attributeMappingJsonParsed, // Enviar como objeto JSON ou undefined
    };

    try {
      let response;
      if (isEditing && initialData?.id) {
        response = await apiClient.put(`/organizations/${user?.organization_id}/identity-providers/${initialData.id}`, payload);
      } else {
        response = await apiClient.post(`/organizations/${user?.organization_id}/identity-providers`, payload);
      }
      onSubmitSuccess(response.data);
      onClose();
    } catch (err: any) {
      console.error("Erro ao salvar provedor de identidade:", err);
      setError(err.response?.data?.error || err.message || "Falha ao salvar provedor de identidade.");
    } finally {
      setIsLoading(false);
    }
  };

  const getConfigPlaceholder = (type: IdentityProviderType): string => {
    switch(type) {
        case 'saml':
            return JSON.stringify({
                idp_entity_id: "https://idp.example.com/entityid",
                idp_sso_url: "https://idp.example.com/sso",
                idp_x509_cert: "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
                sign_request: true, // opcional
                want_assertions_signed: true // opcional
            }, null, 2);
        case 'oauth2_google':
            return JSON.stringify({
                client_id: "YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com",
                client_secret: "YOUR_GOOGLE_CLIENT_SECRET",
                scopes: ["email", "profile"] // opcional
            }, null, 2);
        case 'oauth2_github':
             return JSON.stringify({
                client_id: "YOUR_GITHUB_CLIENT_ID",
                client_secret: "YOUR_GITHUB_CLIENT_SECRET",
                scopes: ["read:user", "user:email"] // opcional
            }, null, 2);
        default:
            return '{}';
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-white">
        {isEditing ? 'Editar' : 'Adicionar Novo'} Provedor de Identidade
      </h3>
      {error && <p className="text-sm text-red-600 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md">{error}</p>}

      <div>
        <label htmlFor="name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Nome do Provedor</label>
        <input type="text" name="name" id="name" value={formData.name} onChange={handleChange} required minLength={3} maxLength={100}
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white"/>
      </div>

      <div>
        <label htmlFor="provider_type" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Tipo de Provedor</label>
        <select name="provider_type" id="provider_type" value={formData.provider_type} onChange={handleChange} required
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white">
          <option value="" disabled>Selecione um Tipo</option>
          <option value="saml">SAML 2.0</option>
          <option value="oauth2_google">OAuth2 (Google)</option>
          <option value="oauth2_github">OAuth2 (GitHub)</option>
          {/* Adicionar outros tipos aqui conforme backend/models.IdentityProviderType */}
        </select>
      </div>

      <div>
        <label htmlFor="config_json_string" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Configuração JSON</label>
        <textarea name="config_json_string" id="config_json_string" rows={8} value={formData.config_json_string} onChange={handleChange} required
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 font-mono text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  placeholder={getConfigPlaceholder(formData.provider_type)}></textarea>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            A estrutura do JSON varia conforme o tipo de provedor. Veja exemplos no README ou documentação da API.
        </p>
      </div>

      <div>
        <label htmlFor="attribute_mapping_json_string" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Mapeamento de Atributos JSON (Opcional)</label>
        <textarea name="attribute_mapping_json_string" id="attribute_mapping_json_string" rows={4} value={formData.attribute_mapping_json_string} onChange={handleChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 font-mono text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  placeholder={JSON.stringify({"email": "emailAttributeFromIdP", "name": "displayNameFromIdP"}, null, 2)}></textarea>
      </div>

      <div className="flex items-center">
        <input type="checkbox" name="is_active" id="is_active" checked={formData.is_active} onChange={handleChange}
               className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 dark:border-gray-600"/>
        <label htmlFor="is_active" className="ml-2 block text-sm text-gray-900 dark:text-gray-300">Ativo</label>
      </div>

      <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700 mt-6">
        <button type="button" onClick={onClose} disabled={isLoading}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 disabled:opacity-50">
          Cancelar
        </button>
        <button type="submit" disabled={isLoading}
                className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md shadow-sm disabled:opacity-50 flex items-center">
          {isLoading && (
            <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          )}
          {isEditing ? 'Salvar Alterações' : 'Adicionar Provedor'}
        </button>
      </div>
    </form>
  );
};

export default IdentityProviderForm;
