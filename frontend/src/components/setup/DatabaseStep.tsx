import React from 'react';
import { useTranslation } from 'next-i18next';

interface DatabaseStepProps {
  onVerifyAndContinue: () => void;
  isLoading: boolean;
  errorMessage?: string | null;
}

const DatabaseStep: React.FC<DatabaseStepProps> = ({ onVerifyAndContinue, isLoading, errorMessage }) => {
  const { t } = useTranslation('setupWizard');

  // As variáveis de ambiente agora são gerenciadas de forma mais inteligente.
  // O foco aqui é na verificação e no feedback de erro.
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-2xl font-bold text-gray-900 dark:text-white text-center">
          {t('steps.db_config.title_v2', 'Verificação do Sistema')}
        </h3>
        <p className="mt-3 text-sm text-gray-600 dark:text-gray-300">
          {t('steps.db_config.intro_paragraph_v2', 'Vamos verificar se o servidor backend e a conexão com o banco de dados estão prontos. Clique no botão abaixo para iniciar a verificação.')}
        </p>
      </div>

      {/* Exibição de Erro */}
      {errorMessage && (
        <div className="bg-red-50 dark:bg-red-700/30 border-l-4 border-red-400 dark:border-red-500 p-4 rounded-md">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400 dark:text-red-300" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800 dark:text-red-200">
                {t('steps.db_config.error_title', 'Falha na Verificação')}
              </h3>
              <div className="mt-2 text-sm text-red-700 dark:text-red-300">
                <p>{errorMessage}</p>
                <p className="mt-2">
                  {t('steps.db_config.error_instructions', 'Por favor, verifique se o Docker está em execução e se as variáveis no seu arquivo `.env` estão corretas (especialmente as de banco de dados). Após ajustar, reinicie a aplicação com `docker-compose down && docker-compose up -d` e tente novamente.')}
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="bg-blue-50 dark:bg-blue-700/30 border-l-4 border-blue-400 dark:border-blue-500 p-4 rounded-md">
          <div className="flex">
            <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-blue-400 dark:text-blue-300" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z" clipRule="evenodd" />
                </svg>
            </div>
            <div className="ml-3">
              <p className="text-sm text-blue-700 dark:text-blue-200">
                {t('steps.db_config.info_text', 'Esta etapa verifica se o frontend consegue se comunicar com o backend e se o backend consegue se conectar ao banco de dados. As chaves de segurança também são validadas neste momento.')}
              </p>
            </div>
          </div>
      </div>

      <div>
        <button
          type="button"
          onClick={onVerifyAndContinue}
          disabled={isLoading}
          className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary transition-colors disabled:opacity-50"
        >
          {isLoading ? t('steps.db_config.verifying_button', 'Verificando...') : t('steps.db_config.verify_button_v2', 'Verificar e Continuar')}
        </button>
      </div>
    </div>
  );
};

export default DatabaseStep;
