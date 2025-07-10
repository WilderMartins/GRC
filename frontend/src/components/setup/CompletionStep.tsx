import React from 'react';
import { useTranslation } from 'next-i18next';
import Link from 'next/link';
import { CheckCircleIcon } from '@heroicons/react/24/solid'; // Usar solid para o ícone de sucesso

interface CompletionStepProps {
  // Nenhuma prop de callback é estritamente necessária se usarmos Link direto.
  // onGoToLogin?: () => void;
}

const CompletionStep: React.FC<CompletionStepProps> = (/*{ onGoToLogin }*/) => {
  const { t } = useTranslation('setupWizard');

  return (
    <div className="space-y-6 text-center">
      <CheckCircleIcon className="w-16 h-16 text-green-500 mx-auto" />
      <div>
        <h3 className="text-2xl font-bold text-gray-900 dark:text-white">
          {t('steps.completed.title', 'Configuração Concluída!')}
        </h3>
        <p className="mt-3 text-sm text-gray-600 dark:text-gray-300">
          {t('steps.completed.message', 'O Phoenix GRC foi configurado com sucesso. Você agora pode prosseguir para a página de login.')}
        </p>
      </div>

      <div className="mt-8">
        <Link href="/auth/login" legacyBehavior>
          <a
            className="w-full flex justify-center items-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary transition-colors"
          >
            {t('steps.completed.go_to_login_button', 'Ir para a Página de Login')}
          </a>
        </Link>
      </div>
    </div>
  );
};

export default CompletionStep;
