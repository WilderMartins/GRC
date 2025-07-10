import React from 'react';
import { useTranslation } from 'next-i18next';

interface DatabaseStepProps {
  onVerifyAndContinue: () => void;
  errorMessage?: string | null;
}

const DatabaseStep: React.FC<DatabaseStepProps> = ({ onVerifyAndContinue, errorMessage }) => {
  const { t } = useTranslation('setupWizard');

  const envVars = [
    { name: 'POSTGRES_HOST', example: 'db (ou localhost)', description: t('steps.db_config.var_host_desc', 'O endereço do seu servidor PostgreSQL.') },
    { name: 'POSTGRES_PORT', example: '5432', description: t('steps.db_config.var_port_desc', 'A porta do seu servidor PostgreSQL.') },
    { name: 'POSTGRES_USER', example: 'admin', description: t('steps.db_config.var_user_desc', 'O nome de usuário para conectar ao banco.') },
    { name: 'POSTGRES_PASSWORD', example: 'password123', description: t('steps.db_config.var_password_desc', 'A senha para o usuário do banco.') },
    { name: 'POSTGRES_DB', example: 'phoenix_grc_dev', description: t('steps.db_config.var_dbname_desc', 'O nome do banco de dados a ser usado/criado.') },
    { name: 'POSTGRES_SSLMODE', example: 'disable (ou require, etc.)', description: t('steps.db_config.var_sslmode_desc', 'Modo SSL para a conexão.') },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-2xl font-bold text-gray-900 dark:text-white text-center">
          {t('steps.db_config.title', 'Configuração do Banco de Dados')}
        </h3>
        <p className="mt-3 text-sm text-gray-600 dark:text-gray-300">
          {t('steps.db_config.intro_paragraph', 'O Phoenix GRC requer um banco de dados PostgreSQL para armazenar seus dados. Por favor, configure as variáveis de ambiente no seu arquivo `.env` na raiz do projeto.')}
        </p>
        {errorMessage && (
          <p className="mt-4 text-sm text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900/30 p-3 rounded-md">
            {t('steps.db_config.previous_error_message', 'Tentativa anterior falhou:')} {errorMessage}
          </p>
        )}
      </div>

      <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg space-y-3">
        <p className="text-sm text-gray-700 dark:text-gray-200">
          {t('steps.db_config.env_instructions_1', 'Se o arquivo `.env` não existir, copie `.env.example` para `.env` na raiz do seu projeto.')}
        </p>
        <p className="text-sm font-semibold text-gray-800 dark:text-white">
          {t('steps.db_config.env_instructions_2', 'Certifique-se de que as seguintes variáveis estão corretamente configuradas:')}
        </p>
        <ul className="list-none space-y-2 text-sm">
          {envVars.map(v => (
            <li key={v.name} className="p-2 bg-white dark:bg-gray-800 rounded shadow-sm">
              <code className="font-mono text-brand-primary dark:text-brand-primary">{v.name}</code>:
              <span className="text-gray-600 dark:text-gray-300"> {v.description} (Ex: <code>{v.example}</code>)</span>
            </li>
          ))}
        </ul>
      </div>

      <div className="bg-yellow-50 dark:bg-yellow-700/30 border-l-4 border-yellow-400 dark:border-yellow-500 p-4 rounded-md">
        <div className="flex">
          <div className="flex-shrink-0">
            <svg className="h-5 w-5 text-yellow-400 dark:text-yellow-300" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 6a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 6zm0 9a1 1 0 100-2 1 1 0 000 2z" clipRule="evenodd" />
            </svg>
          </div>
          <div className="ml-3">
            <p className="text-sm text-yellow-700 dark:text-yellow-200">
              <strong>{t('steps.db_config.important_restart_title', 'Importante:')}</strong> {t('steps.db_config.important_restart_instruction', 'Após salvar as alterações no arquivo `.env`, você DEVE reiniciar os containers da aplicação (especialmente o backend) para que as novas configurações sejam carregadas.')}
               <span className="block mt-1">{t('steps.db_config.docker_compose_restart_tip', 'Ex: `docker-compose down && docker-compose up -d --build` (se estiver usando Docker).')}</span>
            </p>
          </div>
        </div>
      </div>

      <div>
        <button
          type="button"
          onClick={onVerifyAndContinue}
          className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary transition-colors"
        >
          {t('steps.db_config.verify_button', 'Já configurei e reiniciei. Verificar e Continuar')}
        </button>
      </div>
    </div>
  );
};

export default DatabaseStep;
