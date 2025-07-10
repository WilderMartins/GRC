import React from 'react';
import { useTranslation } from 'next-i18next';

interface MigrationsStepProps {
  onRunMigrations: () => Promise<void>;
  isLoading: boolean;
  errorMessage?: string | null;
}

const MigrationsStep: React.FC<MigrationsStepProps> = ({
  onRunMigrations,
  isLoading,
  errorMessage
}) => {
  const { t } = useTranslation('setupWizard');

  return (
    <div className="space-y-6 text-center">
      <div>
        <h3 className="text-2xl font-bold text-gray-900 dark:text-white">
          {t('steps.migrations.title', 'Executar Migrações do Banco de Dados')}
        </h3>
        <p className="mt-3 text-sm text-gray-600 dark:text-gray-300">
          {t('steps.migrations.intro_paragraph', 'O próximo passo é preparar a estrutura do seu banco de dados. Isso criará todas as tabelas necessárias para a aplicação Phoenix GRC funcionar.')}
        </p>
        {errorMessage && (
          <p className="mt-4 text-sm text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900/30 p-3 rounded-md">
            {t('steps.migrations.previous_error_message', 'Tentativa anterior falhou:')} {errorMessage}
          </p>
        )}
      </div>

      <div className="mt-6">
        <button
          type="button"
          onClick={onRunMigrations}
          disabled={isLoading}
          className="w-full flex justify-center items-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary disabled:opacity-70 transition-colors"
        >
          {isLoading ? (
            <>
              <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              {t('steps.migrations.loading_button', 'Executando Migrações...')}
            </>
          ) : (
            t('steps.migrations.run_button', 'Executar Migrações')
          )}
        </button>
      </div>

      <p className="mt-4 text-xs text-gray-500 dark:text-gray-400">
        {t('steps.migrations.info_idempotent', 'Esta operação é segura para ser executada múltiplas vezes, se necessário.')}
      </p>
    </div>
  );
};

export default MigrationsStep;
