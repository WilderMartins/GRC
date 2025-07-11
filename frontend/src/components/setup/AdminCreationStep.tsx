import React, { useState, FormEvent } from 'react';
import { useTranslation } from 'next-i18next';

export interface AdminCreationFormData {
  organizationName: string;
  adminName: string;
  adminEmail: string;
  adminPassword: string;
  adminPasswordConfirm: string;
}

interface AdminCreationStepProps {
  onSubmitAdminForm: (data: AdminCreationFormData) => Promise<void>;
  isLoading: boolean;
  errorMessage?: string | null;
}

const AdminCreationStep: React.FC<AdminCreationStepProps> = ({
  onSubmitAdminForm,
  isLoading,
  errorMessage,
}) => {
  const { t } = useTranslation('setupWizard');
  const [formData, setFormData] = useState<AdminCreationFormData>({
    organizationName: '',
    adminName: '',
    adminEmail: '',
    adminPassword: '',
    adminPasswordConfirm: '',
  });
  const [formErrors, setFormErrors] = useState<Partial<AdminCreationFormData>>({});

  const validateForm = (): boolean => {
    const errors: Partial<AdminCreationFormData> = {};
    if (!formData.organizationName.trim()) {
      errors.organizationName = t('steps.admin_creation.validation.org_name_required', 'Nome da organização é obrigatório.');
    }
    if (!formData.adminName.trim()) {
      errors.adminName = t('steps.admin_creation.validation.admin_name_required', 'Nome do administrador é obrigatório.');
    }
    if (!formData.adminEmail.trim()) {
      errors.adminEmail = t('steps.admin_creation.validation.email_required', 'Email é obrigatório.');
    } else if (!/\S+@\S+\.\S+/.test(formData.adminEmail)) {
      errors.adminEmail = t('steps.admin_creation.validation.email_invalid', 'Formato de email inválido.');
    }
    if (!formData.adminPassword) {
      errors.adminPassword = t('steps.admin_creation.validation.password_required', 'Senha é obrigatória.');
    } else if (formData.adminPassword.length < 8) { // Exemplo de requisito de força
        errors.adminPassword = t('steps.admin_creation.validation.password_min_length', 'Senha deve ter pelo menos 8 caracteres.');
    }
    if (formData.adminPassword !== formData.adminPasswordConfirm) {
      errors.adminPasswordConfirm = t('steps.admin_creation.validation.passwords_no_match', 'As senhas não coincidem.');
    }
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    // Limpar erro do campo ao digitar
    if (formErrors[name as keyof AdminCreationFormData]) {
        setFormErrors(prev => ({...prev, [name]: undefined }));
    }
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (validateForm()) {
      await onSubmitAdminForm(formData);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div>
        <h3 className="text-2xl font-bold text-gray-900 dark:text-white text-center">
          {t('steps.admin_creation.title', 'Criar Conta de Administrador')}
        </h3>
        <p className="mt-2 text-sm text-gray-600 dark:text-gray-300 text-center">
          {t('steps.admin_creation.intro_paragraph', 'Por favor, forneça os detalhes para a sua organização e a conta de administrador principal.')}
        </p>
      </div>

      {errorMessage && (
        <p className="text-sm text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900/30 p-3 rounded-md text-center">
          {t('steps.admin_creation.api_error_prefix', 'Erro ao criar administrador:')} {errorMessage}
        </p>
      )}

      {/* Organização */}
      <fieldset className="space-y-4 border p-4 rounded-md dark:border-gray-700">
        <legend className="text-lg font-medium text-gray-900 dark:text-white px-2">{t('steps.admin_creation.org_section_title', 'Detalhes da Organização')}</legend>
        <div>
          <label htmlFor="organizationName" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('steps.admin_creation.org_name_label', 'Nome da Organização')}
          </label>
          <input
            type="text"
            name="organizationName"
            id="organizationName"
            value={formData.organizationName}
            onChange={handleChange}
            required
            className={`mt-1 block w-full rounded-md shadow-sm p-2 dark:bg-gray-700 dark:text-white ${formErrors.organizationName ? 'border-red-500 dark:border-red-400 focus:ring-red-500 focus:border-red-500' : 'border-gray-300 dark:border-gray-600 focus:ring-brand-primary focus:border-brand-primary'}`}
          />
          {formErrors.organizationName && <p className="mt-1 text-xs text-red-500 dark:text-red-400">{formErrors.organizationName}</p>}
        </div>
      </fieldset>

      {/* Administrador */}
      <fieldset className="space-y-4 border p-4 rounded-md dark:border-gray-700">
        <legend className="text-lg font-medium text-gray-900 dark:text-white px-2">{t('steps.admin_creation.admin_section_title', 'Detalhes do Administrador')}</legend>
        <div>
          <label htmlFor="adminName" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('steps.admin_creation.admin_name_label', 'Nome Completo')}
          </label>
          <input type="text" name="adminName" id="adminName" value={formData.adminName} onChange={handleChange} required
                 className={`mt-1 block w-full rounded-md shadow-sm p-2 dark:bg-gray-700 dark:text-white ${formErrors.adminName ? 'border-red-500 dark:border-red-400 focus:ring-red-500 focus:border-red-500' : 'border-gray-300 dark:border-gray-600 focus:ring-brand-primary focus:border-brand-primary'}`}
          />
          {formErrors.adminName && <p className="mt-1 text-xs text-red-500 dark:text-red-400">{formErrors.adminName}</p>}
        </div>
        <div>
          <label htmlFor="adminEmail" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('steps.admin_creation.admin_email_label', 'Email')}
          </label>
          <input type="email" name="adminEmail" id="adminEmail" value={formData.adminEmail} onChange={handleChange} required
                 className={`mt-1 block w-full rounded-md shadow-sm p-2 dark:bg-gray-700 dark:text-white ${formErrors.adminEmail ? 'border-red-500 dark:border-red-400 focus:ring-red-500 focus:border-red-500' : 'border-gray-300 dark:border-gray-600 focus:ring-brand-primary focus:border-brand-primary'}`}
           />
          {formErrors.adminEmail && <p className="mt-1 text-xs text-red-500 dark:text-red-400">{formErrors.adminEmail}</p>}
        </div>
        <div>
          <label htmlFor="adminPassword" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('steps.admin_creation.password_label', 'Senha')}
          </label>
          <input type="password" name="adminPassword" id="adminPassword" value={formData.adminPassword} onChange={handleChange} required
                 className={`mt-1 block w-full rounded-md shadow-sm p-2 dark:bg-gray-700 dark:text-white ${formErrors.adminPassword ? 'border-red-500 dark:border-red-400 focus:ring-red-500 focus:border-red-500' : 'border-gray-300 dark:border-gray-600 focus:ring-brand-primary focus:border-brand-primary'}`}
          />
          {formErrors.adminPassword && <p className="mt-1 text-xs text-red-500 dark:text-red-400">{formErrors.adminPassword}</p>}
        </div>
        <div>
          <label htmlFor="adminPasswordConfirm" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('steps.admin_creation.password_confirm_label', 'Confirmar Senha')}
          </label>
          <input type="password" name="adminPasswordConfirm" id="adminPasswordConfirm" value={formData.adminPasswordConfirm} onChange={handleChange} required
                 className={`mt-1 block w-full rounded-md shadow-sm p-2 dark:bg-gray-700 dark:text-white ${formErrors.adminPasswordConfirm ? 'border-red-500 dark:border-red-400 focus:ring-red-500 focus:border-red-500' : 'border-gray-300 dark:border-gray-600 focus:ring-brand-primary focus:border-brand-primary'}`}
          />
          {formErrors.adminPasswordConfirm && <p className="mt-1 text-xs text-red-500 dark:text-red-400">{formErrors.adminPasswordConfirm}</p>}
        </div>
      </fieldset>

      <div className="mt-8">
        <button
          type="submit"
          disabled={isLoading}
          className="w-full flex justify-center items-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary disabled:opacity-70 transition-colors"
        >
          {isLoading ? (
            <>
              <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              {t('steps.admin_creation.loading_button', 'Criando Administrador...')}
            </>
          ) : (
            t('steps.admin_creation.create_button', 'Criar Administrador e Finalizar Configuração')
          )}
        </button>
      </div>
    </form>
  );
};

export default AdminCreationStep;
