import React, { useState, useEffect, FormEvent } from 'react';
import apiClient from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';
import { useTranslation } from 'next-i18next';
import {
  IdentityProvider,
  IdentityProviderType,
  IdentityProviderConfigSaml,
  IdentityProviderConfigOAuth2,
  AttributeMapping,
} from '@/types';

// Interface para os dados do formulário, combinando todos os campos possíveis
interface IdpFormData {
  name: string;
  provider_type: IdentityProviderType | '';
  is_active: boolean;
  // SAML
  idp_entity_id: string;
  idp_sso_url: string;
  idp_x509_cert: string;
  sign_request: boolean;
  want_assertions_signed: boolean;
  // OAuth2
  client_id: string;
  client_secret: string;
  scopes: string; // Comma-separated string for input
  // Attribute Mapping
  map_email: string;
  map_name: string;
}

interface IdentityProviderFormProps {
  organizationId: string;
  initialData?: IdentityProvider; // Para edição
  isEditing?: boolean;
  onSubmitSuccess: () => void;
}

const IdentityProviderForm: React.FC<IdentityProviderFormProps> = ({
  organizationId,
  initialData,
  isEditing = false,
  onSubmitSuccess,
}) => {
  const { t } = useTranslation(['idp', 'common']);
  const notify = useNotifier();

  const [formData, setFormData] = useState<IdpFormData>({
    name: '',
    provider_type: '',
    is_active: true,
    idp_entity_id: '',
    idp_sso_url: '',
    idp_x509_cert: '',
    sign_request: false,
    want_assertions_signed: true,
    client_id: '',
    client_secret: '',
    scopes: 'email,profile', // Default scopes for OAuth2
    map_email: '',
    map_name: '',
  });

  const [isLoading, setIsLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    if (isEditing && initialData) {
      let config: Partial<IdpFormData> = {};
      let mapping: Partial<IdpFormData> = {};

      try {
        if (initialData.config_json) {
          const parsedConfig = JSON.parse(initialData.config_json);
          if (initialData.provider_type === IdentityProviderType.SAML) {
            const samlConfig = parsedConfig as IdentityProviderConfigSaml;
            config = {
              idp_entity_id: samlConfig.idp_entity_id || '',
              idp_sso_url: samlConfig.idp_sso_url || '',
              idp_x509_cert: samlConfig.idp_x509_cert || '',
              sign_request: samlConfig.sign_request === undefined ? false : samlConfig.sign_request,
              want_assertions_signed: samlConfig.want_assertions_signed === undefined ? true : samlConfig.want_assertions_signed,
            };
          } else if (initialData.provider_type === IdentityProviderType.OAUTH2_GOOGLE || initialData.provider_type === IdentityProviderType.OAUTH2_GITHUB) {
            const oauthConfig = parsedConfig as IdentityProviderConfigOAuth2;
            config = {
              client_id: oauthConfig.client_id || '',
              client_secret: oauthConfig.client_secret || '', // Cuidado com a exibição/edição de segredos
              scopes: (oauthConfig.scopes || ['email', 'profile']).join(','),
            };
          }
        }
      } catch (e) { console.error("Error parsing config_json", e); }

      try {
        if (initialData.attribute_mapping_json) {
          const parsedMapping = JSON.parse(initialData.attribute_mapping_json) as AttributeMapping;
          mapping = {
            map_email: parsedMapping.email || '',
            map_name: parsedMapping.name || '',
          };
        }
      } catch (e) { console.error("Error parsing attribute_mapping_json", e); }

      setFormData({
        name: initialData.name || '',
        provider_type: initialData.provider_type as IdentityProviderType || '',
        is_active: initialData.is_active === undefined ? true : initialData.is_active,
        ...config,
        ...mapping,
      } as IdpFormData); // Type assertion to satisfy all potential fields
    }
  }, [initialData, isEditing]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    const { name, value, type } = e.target;
    if (type === 'checkbox') {
      setFormData(prev => ({ ...prev, [name]: (e.target as HTMLInputElement).checked }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
    }
    // Reset client_secret if provider type changes from OAuth2 to prevent accidental exposure or saving
    if (name === 'provider_type' && formData.provider_type?.startsWith('oauth2_') && !value.startsWith('oauth2_')) {
        setFormData(prev => ({...prev, client_secret: ''}));
    }
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsLoading(true);
    setFormError(null);

    if (!formData.provider_type) {
      setFormError(t('form.error_provider_type_required'));
      setIsLoading(false);
      return;
    }

    let config_json: any = {};
    if (formData.provider_type === IdentityProviderType.SAML) {
      config_json = {
        idp_entity_id: formData.idp_entity_id,
        idp_sso_url: formData.idp_sso_url,
        idp_x509_cert: formData.idp_x509_cert,
        sign_request: formData.sign_request,
        want_assertions_signed: formData.want_assertions_signed,
      };
    } else if (formData.provider_type === IdentityProviderType.OAUTH2_GOOGLE || formData.provider_type === IdentityProviderType.OAUTH2_GITHUB) {
      config_json = {
        client_id: formData.client_id,
        client_secret: formData.client_secret, // Enviar apenas se fornecido/alterado
        scopes: formData.scopes.split(',').map(s => s.trim()).filter(s => s),
      };
      if (!isEditing && !formData.client_secret) { // Se criando e client_secret vazio
        setFormError(t('form.error_client_secret_required_on_create'));
        setIsLoading(false);
        return;
      }
       if (isEditing && !formData.client_secret && initialData?.config_json) {
        // Se editando e client_secret foi apagado, não enviar para manter o antigo.
        // Se o backend espera um valor para apagar, esta lógica precisa mudar.
        // Ou, ter um campo "Alterar Client Secret"
        delete config_json.client_secret;
      }
    }

    let attribute_mapping_json: any = {};
    if (formData.map_email || formData.map_name) {
        if(formData.map_email) attribute_mapping_json.email = formData.map_email;
        if(formData.map_name) attribute_mapping_json.name = formData.map_name;
    }


    const payload: Partial<IdentityProvider> = {
      name: formData.name,
      provider_type: formData.provider_type,
      is_active: formData.is_active,
      config_json: JSON.stringify(config_json),
      attribute_mapping_json: Object.keys(attribute_mapping_json).length > 0 ? JSON.stringify(attribute_mapping_json) : undefined,
    };

    // Não enviar client_secret vazio na edição se não foi alterado
    if (isEditing && formData.provider_type?.startsWith('oauth2_') && !formData.client_secret) {
        const currentConfig = initialData?.config_json ? JSON.parse(initialData.config_json) : {};
        if (currentConfig.client_secret) {
            // Se existia um client_secret e o campo está vazio, não o inclua no payload para PUT,
            // assumindo que o backend mantém o valor existente se não for fornecido.
            // Se o backend apaga o segredo se não for fornecido, esta lógica está incorreta.
            // Para maior clareza, seria melhor ter um campo "Alterar Client Secret"
            if (payload.config_json) {
                const tempConfig = JSON.parse(payload.config_json);
                delete tempConfig.client_secret;
                payload.config_json = JSON.stringify(tempConfig);
            }
        }
    }


    try {
      if (isEditing && initialData?.id) {
        await apiClient.put(`/organizations/${organizationId}/identity-providers/${initialData.id}`, payload);
        notify.success(t('form.update_success_message'));
      } else {
        await apiClient.post(`/organizations/${organizationId}/identity-providers`, payload);
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

  const renderSAMLFields = () => (
    <>
      <div>
        <label htmlFor="idp_entity_id" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.saml_entity_id_label')}</label>
        <input type="text" name="idp_entity_id" id="idp_entity_id" value={formData.idp_entity_id} onChange={handleChange} required
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>
      <div>
        <label htmlFor="idp_sso_url" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.saml_sso_url_label')}</label>
        <input type="url" name="idp_sso_url" id="idp_sso_url" value={formData.idp_sso_url} onChange={handleChange} required
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>
      <div>
        <label htmlFor="idp_x509_cert" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.saml_x509_cert_label')}</label>
        <textarea name="idp_x509_cert" id="idp_x509_cert" value={formData.idp_x509_cert} onChange={handleChange} required rows={6}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2 font-mono text-xs"/>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{t('form.saml_x509_cert_help')}</p>
      </div>
      <div className="flex items-start space-x-4">
        <div className="flex items-center h-5">
            <input id="sign_request" name="sign_request" type="checkbox" checked={formData.sign_request} onChange={handleChange}
                    className="focus:ring-brand-primary h-4 w-4 text-brand-primary border-gray-300 rounded"/>
        </div>
        <div className="text-sm">
            <label htmlFor="sign_request" className="font-medium text-gray-700 dark:text-gray-300">{t('form.saml_sign_request_label')}</label>
            <p className="text-gray-500 dark:text-gray-400 text-xs">{t('form.saml_sign_request_help')}</p>
        </div>
      </div>
       <div className="flex items-start space-x-4">
        <div className="flex items-center h-5">
            <input id="want_assertions_signed" name="want_assertions_signed" type="checkbox" checked={formData.want_assertions_signed} onChange={handleChange}
                    className="focus:ring-brand-primary h-4 w-4 text-brand-primary border-gray-300 rounded"/>
        </div>
        <div className="text-sm">
            <label htmlFor="want_assertions_signed" className="font-medium text-gray-700 dark:text-gray-300">{t('form.saml_want_assertions_signed_label')}</label>
             <p className="text-gray-500 dark:text-gray-400 text-xs">{t('form.saml_want_assertions_signed_help')}</p>
        </div>
      </div>
    </>
  );

  const renderOAuth2Fields = () => (
    <>
      <div>
        <label htmlFor="client_id" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.oauth_client_id_label')}</label>
        <input type="text" name="client_id" id="client_id" value={formData.client_id} onChange={handleChange} required
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>
      <div>
        <label htmlFor="client_secret" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {isEditing && initialData?.config_json && JSON.parse(initialData.config_json)?.client_secret
                ? t('form.oauth_client_secret_edit_label')
                : t('form.oauth_client_secret_label')}
        </label>
        <input type="password" name="client_secret" id="client_secret" value={formData.client_secret} onChange={handleChange}
               required={!isEditing || (isEditing && !initialData?.config_json) || (isEditing && initialData.config_json && !JSON.parse(initialData.config_json)?.client_secret) } // Required se criando ou se não havia segredo antes
               placeholder={isEditing && initialData?.config_json && JSON.parse(initialData.config_json)?.client_secret ? t('form.oauth_client_secret_edit_placeholder') : ''}
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
         {isEditing && initialData?.config_json && JSON.parse(initialData.config_json)?.client_secret && (
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{t('form.oauth_client_secret_edit_help')}</p>
        )}
      </div>
      <div>
        <label htmlFor="scopes" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.oauth_scopes_label')}</label>
        <input type="text" name="scopes" id="scopes" value={formData.scopes} onChange={handleChange}
               placeholder="email,profile"
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{t('form.oauth_scopes_help')}</p>
      </div>
    </>
  );

  const renderAttributeMappingFields = () => (
    <div className="mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
        <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-1">{t('form.mapping_title')}</h3>
        <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">{t('form.mapping_description')}</p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
                <label htmlFor="map_email" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.mapping_email_label')}</label>
                <input type="text" name="map_email" id="map_email" value={formData.map_email} onChange={handleChange}
                        placeholder={t('form.mapping_email_placeholder')}
                        className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
            </div>
            <div>
                <label htmlFor="map_name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.mapping_name_label')}</label>
                <input type="text" name="map_name" id="map_name" value={formData.map_name} onChange={handleChange}
                        placeholder={t('form.mapping_name_placeholder')}
                        className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
            </div>
        </div>
    </div>
  );

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {formError && <p className="text-sm text-red-500 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md">{formError}</p>}

      <div>
        <label htmlFor="name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.name_label')}</label>
        <input type="text" name="name" id="name" value={formData.name} onChange={handleChange} required
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>

      <div>
        <label htmlFor="provider_type" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('form.provider_type_label')}</label>
        <select name="provider_type" id="provider_type" value={formData.provider_type} onChange={handleChange} required
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
          <option value="" disabled>{t('form.provider_type_placeholder')}</option>
          <option value={IdentityProviderType.SAML}>{t('types.saml')}</option>
          <option value={IdentityProviderType.OAUTH2_GOOGLE}>{t('types.oauth2_google')}</option>
          <option value={IdentityProviderType.OAUTH2_GITHUB}>{t('types.oauth2_github')}</option>
        </select>
      </div>

      <div className="flex items-start">
        <div className="flex items-center h-5">
            <input id="is_active" name="is_active" type="checkbox" checked={formData.is_active} onChange={handleChange}
                    className="focus:ring-brand-primary h-4 w-4 text-brand-primary border-gray-300 rounded"/>
        </div>
        <div className="ml-3 text-sm">
            <label htmlFor="is_active" className="font-medium text-gray-700 dark:text-gray-300">{t('form.is_active_label')}</label>
        </div>
      </div>

      {/* Campos Condicionais */}
      {formData.provider_type === IdentityProviderType.SAML && renderSAMLFields()}
      {(formData.provider_type === IdentityProviderType.OAUTH2_GOOGLE || formData.provider_type === IdentityProviderType.OAUTH2_GITHUB) && renderOAuth2Fields()}

      {/* Campos de Mapeamento de Atributos (Sempre visíveis se provider_type selecionado) */}
      {formData.provider_type && renderAttributeMappingFields()}


      <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700 mt-6">
        <button type="button" onClick={onSubmitSuccess} // Ou router.back() dependendo do contexto
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary disabled:opacity-50"
                disabled={isLoading}>
          {t('common:cancel_button')}
        </button>
        <button type="submit" disabled={isLoading || !formData.provider_type}
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

export default IdentityProviderForm;
