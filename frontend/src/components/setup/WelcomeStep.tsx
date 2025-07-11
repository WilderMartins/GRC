import React from 'react';
import { useTranslation } from 'next-i18next';

interface WelcomeStepProps {
  onNext: () => void;
}

const WelcomeStep: React.FC<WelcomeStepProps> = ({ onNext }) => {
  const { t } = useTranslation('setupWizard');

  return (
    <div className="space-y-6 text-center">
      <div>
        <h3 className="text-2xl font-bold text-gray-900 dark:text-white">
          {t('steps.welcome.title', 'Bem-vindo à Configuração do Phoenix GRC!')}
        </h3>
        <p className="mt-3 text-sm text-gray-600 dark:text-gray-300">
          {t('steps.welcome.intro_paragraph', 'Este assistente irá guiá-lo através do processo de configuração inicial da sua instância Phoenix GRC.')}
        </p>
      </div>

      <div className="text-left bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
        <h4 className="font-semibold text-gray-800 dark:text-white mb-2">
            {t('steps.welcome.steps_overview_title', 'O processo cobrirá:')}
        </h4>
        <ul className="list-disc list-inside space-y-1 text-sm text-gray-600 dark:text-gray-300">
          <li>{t('steps.welcome.step1_env_config', 'Configuração da conexão com o Banco de Dados (via arquivo .env)')}</li>
          <li>{t('steps.welcome.step2_migrations', 'Execução das migrações do banco de dados')}</li>
          <li>{t('steps.welcome.step3_admin_user', 'Criação da sua organização e do primeiro usuário administrador')}</li>
        </ul>
      </div>

      <p className="text-sm text-gray-500 dark:text-gray-400">
        {t('steps.welcome.ready_to_start', 'Pronto para começar?')}
      </p>

      <div>
        <button
          type="button"
          onClick={onNext}
          className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary transition-colors"
        >
          {t('steps.welcome.start_button', 'Iniciar Configuração')}
        </button>
      </div>
    </div>
  );
};

export default WelcomeStep;
