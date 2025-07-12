import React, { useState } from 'react';
import { useTranslation } from 'next-i18next';
import axios from 'axios';
import { useNotifier } from '@/hooks/useNotifier';

interface AdminUserStepProps {
  onSetupComplete: () => void;
}

const AdminUserStep: React.FC<AdminUserStepProps> = ({ onSetupComplete }) => {
  const { t } = useTranslation('setupWizard');
  const { showSuccess, showError } = useNotifier();
  const [isLoading, setIsLoading] = useState(false);
  const [formData, setFormData] = useState({
    organization_name: '',
    admin_name: '',
    admin_email: '',
    admin_password: '',
    confirm_password: '',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
    // Clear error when user starts typing
    if (errors[name]) {
      setErrors((prev) => ({ ...prev, [name]: '' }));
    }
  };

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};
    if (!formData.organization_name) newErrors.organization_name = t('steps.admin_user.errors.org_name_required');
    if (!formData.admin_name) newErrors.admin_name = t('steps.admin_user.errors.admin_name_required');
    if (!formData.admin_email) {
      newErrors.admin_email = t('steps.admin_user.errors.email_required');
    } else if (!/\S+@\S+\.\S+/.test(formData.admin_email)) {
      newErrors.admin_email = t('steps.admin_user.errors.email_invalid');
    }
    if (!formData.admin_password) {
      newErrors.admin_password = t('steps.admin_user.errors.password_required');
    } else if (formData.admin_password.length < 8) {
      newErrors.admin_password = t('steps.admin_user.errors.password_too_short');
    }
    if (formData.admin_password !== formData.confirm_password) {
      newErrors.confirm_password = t('steps.admin_user.errors.passwords_do_not_match');
    }
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validateForm()) {
      return;
    }
    setIsLoading(true);
    try {
      const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL || '';
      const payload = {
        organization_name: formData.organization_name,
        admin_name: formData.admin_name,
        admin_email: formData.admin_email,
        admin_password: formData.admin_password,
      };
      await axios.post(`${apiBaseUrl}/auth/setup`, payload);
      showSuccess(t('steps.admin_user.notifications.setup_successful'));
      onSetupComplete();
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        showError(error.response.data.error || t('steps.admin_user.errors.generic_error'));
      } else {
        showError(t('steps.admin_user.errors.generic_error'));
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-2xl font-bold text-gray-900 dark:text-white text-center">
          {t('steps.admin_user.title', 'Criar Conta de Administrador')}
        </h3>
        <p className="mt-3 text-sm text-gray-600 dark:text-gray-300">
          {t('steps.admin_user.intro_paragraph', 'Preencha os detalhes abaixo para criar a sua organização e a conta de administrador principal.')}
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Organization Name */}
        <div>
          <label htmlFor="organization_name" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
            {t('steps.admin_user.labels.organization_name')}
          </label>
          <input
            type="text"
            name="organization_name"
            id="organization_name"
            value={formData.organization_name}
            onChange={handleInputChange}
            className={`mt-1 block w-full px-3 py-2 border ${errors.organization_name ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-md shadow-sm focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm dark:bg-gray-700 dark:text-white`}
          />
          {errors.organization_name && <p className="mt-2 text-sm text-red-600 dark:text-red-400">{errors.organization_name}</p>}
        </div>

        {/* Admin Name */}
        <div>
          <label htmlFor="admin_name" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
            {t('steps.admin_user.labels.admin_name')}
          </label>
          <input
            type="text"
            name="admin_name"
            id="admin_name"
            value={formData.admin_name}
            onChange={handleInputChange}
            className={`mt-1 block w-full px-3 py-2 border ${errors.admin_name ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-md shadow-sm focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm dark:bg-gray-700 dark:text-white`}
          />
          {errors.admin_name && <p className="mt-2 text-sm text-red-600 dark:text-red-400">{errors.admin_name}</p>}
        </div>

        {/* Admin Email */}
        <div>
          <label htmlFor="admin_email" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
            {t('steps.admin_user.labels.admin_email')}
          </label>
          <input
            type="email"
            name="admin_email"
            id="admin_email"
            value={formData.admin_email}
            onChange={handleInputChange}
            className={`mt-1 block w-full px-3 py-2 border ${errors.admin_email ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-md shadow-sm focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm dark:bg-gray-700 dark:text-white`}
          />
          {errors.admin_email && <p className="mt-2 text-sm text-red-600 dark:text-red-400">{errors.admin_email}</p>}
        </div>

        {/* Admin Password */}
        <div>
          <label htmlFor="admin_password" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
            {t('steps.admin_user.labels.admin_password')}
          </label>
          <input
            type="password"
            name="admin_password"
            id="admin_password"
            value={formData.admin_password}
            onChange={handleInputChange}
            className={`mt-1 block w-full px-3 py-2 border ${errors.admin_password ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-md shadow-sm focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm dark:bg-gray-700 dark:text-white`}
          />
          {errors.admin_password && <p className="mt-2 text-sm text-red-600 dark:text-red-400">{errors.admin_password}</p>}
        </div>

        {/* Confirm Password */}
        <div>
          <label htmlFor="confirm_password" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
            {t('steps.admin_user.labels.confirm_password')}
          </label>
          <input
            type="password"
            name="confirm_password"
            id="confirm_password"
            value={formData.confirm_password}
            onChange={handleInputChange}
            className={`mt-1 block w-full px-3 py-2 border ${errors.confirm_password ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'} rounded-md shadow-sm focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm dark:bg-gray-700 dark:text-white`}
          />
          {errors.confirm_password && <p className="mt-2 text-sm text-red-600 dark:text-red-400">{errors.confirm_password}</p>}
        </div>

        <div>
          <button
            type="submit"
            disabled={isLoading}
            className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary transition-colors disabled:opacity-50"
          >
            {isLoading ? t('steps.admin_user.buttons.loading', 'Criando...') : t('steps.admin_user.buttons.submit', 'Criar e Finalizar Setup')}
          </button>
        </div>
      </form>
    </div>
  );
};

export default AdminUserStep;
