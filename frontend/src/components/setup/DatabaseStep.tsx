import React, { useState } from 'react';
import { useTranslation } from 'next-i18next';
import axios from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';

interface DatabaseStepProps {
  onVerifyAndContinue: () => void;
  isLoading: boolean;
  errorMessage?: string | null;
}

const DatabaseStep: React.FC<DatabaseStepProps> = ({ onVerifyAndContinue, isLoading, errorMessage }) => {
  const { t } = useTranslation('setupWizard');
  const { showError, showSuccess } = useNotifier();
  const [dbConfig, setDbConfig] = useState({
    host: 'db',
    port: '5432',
    user: 'admin',
    password: 'password123',
    dbname: 'phoenix_grc_dev',
  });
  const [isTesting, setIsTesting] = useState(false);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setDbConfig(prev => ({ ...prev, [name]: value }));
  };

  const handleTestConnection = async () => {
    setIsTesting(true);
    try {
      await axios.post('/api/public/setup/test-db', dbConfig);
      showSuccess(t('notifications.db_connection_successful'));
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        showError(error.response.data.error || t('notifications.db_connection_failed'));
      } else {
        showError(t('notifications.db_connection_failed'));
      }
    } finally {
      setIsTesting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-2xl font-bold text-gray-900 dark:text-white text-center">
          {t('steps.db_config.title_v2', 'Configuração do Banco de Dados')}
        </h3>
        <p className="mt-3 text-sm text-gray-600 dark:text-gray-300">
          {t('steps.db_config.intro_paragraph_v2', 'Insira os detalhes de conexão do seu banco de dados PostgreSQL.')}
        </p>
      </div>

      <div className="grid grid-cols-1 gap-y-6 sm:grid-cols-2 sm:gap-x-8">
        <div>
          <label htmlFor="host" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('steps.db_config.form.host_label')}
          </label>
          <input
            type="text"
            name="host"
            id="host"
            value={dbConfig.host}
            onChange={handleInputChange}
            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white"
          />
        </div>
        <div>
          <label htmlFor="port" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('steps.db_config.form.port_label')}
          </label>
          <input
            type="text"
            name="port"
            id="port"
            value={dbConfig.port}
            onChange={handleInputChange}
            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white"
          />
        </div>
        <div>
          <label htmlFor="user" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('steps.db_config.form.user_label')}
          </label>
          <input
            type="text"
            name="user"
            id="user"
            value={dbConfig.user}
            onChange={handleInputChange}
            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white"
          />
        </div>
        <div>
          <label htmlFor="password" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('steps.db_config.form.password_label')}
          </label>
          <input
            type="password"
            name="password"
            id="password"
            value={dbConfig.password}
            onChange={handleInputChange}
            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white"
          />
        </div>
        <div className="sm:col-span-2">
          <label htmlFor="dbname" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t('steps.db_config.form.dbname_label')}
          </label>
          <input
            type="text"
            name="dbname"
            id="dbname"
            value={dbConfig.dbname}
            onChange={handleInputChange}
            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary sm:text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white"
          />
        </div>
      </div>

      {errorMessage && (
        <div className="bg-red-50 dark:bg-red-700/30 border-l-4 border-red-400 dark:border-red-500 p-4 rounded-md">
          <p className="text-sm text-red-700 dark:text-red-300">{errorMessage}</p>
        </div>
      )}

      <div className="flex justify-end space-x-4">
        <button
          type="button"
          onClick={handleTestConnection}
          disabled={isTesting || isLoading}
          className="inline-flex items-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary dark:bg-gray-700 dark:text-gray-200 dark:border-gray-600 dark:hover:bg-gray-600"
        >
          {isTesting ? t('steps.db_config.testing_button') : t('steps.db_config.test_button')}
        </button>
        <button
          type="button"
          onClick={onVerifyAndContinue}
          disabled={isLoading || isTesting}
          className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary transition-colors disabled:opacity-50"
        >
          {isLoading ? t('steps.db_config.verifying_button') : t('steps.db_config.verify_button_v2')}
        </button>
      </div>
    </div>
  );
};

export default DatabaseStep;
