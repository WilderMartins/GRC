import Head from 'next/head';
import { useRouter } from 'next/router';
import { useState } from 'react';
import SetupLayout from '@/components/layouts/SetupLayout';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import WelcomeStep from '@/components/setup/WelcomeStep';
import DatabaseStep from '@/components/setup/DatabaseStep';
import axios from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';
import CompletionStep from '@/components/setup/CompletionStep';
import AdminUserStep from '@/components/setup/AdminUserStep';
import CompletionStep from '@/components/setup/CompletionStep';

// Definir os tipos para as etapas do wizard
type SetupStep =
  | 'welcome'
  | 'db_config'
  | 'admin_user'
  | 'completion';

type Props = {};

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'setupWizard'])),
  },
});

const SetupWizardPage = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['setupWizard', 'common']);
  const [currentStep, setCurrentStep] = useState<SetupStep>('welcome');
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const { showError } = useNotifier();

  const handleVerifySystem = async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const response = await axios.get('/api/public/setup-status');
      const status = response.data.status;

      if (status === 'setup_complete') {
        // Se o setup já estiver completo, redirecionar ou mostrar uma mensagem
        showError(t('notifications.setup_already_complete'));
        // Idealmente, redirecionar para a página de login
        // router.push('/auth/login');
        setCurrentStep('completion'); // Ou ir para a tela de conclusão
      } else if (status === 'database_not_connected' || status === 'database_not_configured') {
        setErrorMessage(response.data.message);
      } else {
        // Status como 'migrations_not_run', 'setup_pending_org', etc. são OK para prosseguir
        setCurrentStep('admin_user');
      }
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        setErrorMessage(error.response.data.message || t('notifications.generic_error'));
      } else {
        setErrorMessage(t('notifications.generic_error'));
      }
    } finally {
      setIsLoading(false);
    }
  };

  const goToCompletion = () => {
    setCurrentStep('completion');
  }

  const renderCurrentStep = () => {
    switch (currentStep) {
      case 'welcome':
        return <WelcomeStep onNext={() => setCurrentStep('db_config')} />;
      case 'db_config':
        return (
          <DatabaseStep
            onVerifyAndContinue={handleVerifySystem}
            isLoading={isLoading}
            errorMessage={errorMessage}
          />
        );
      case 'admin_user':
        return <AdminUserStep onSetupComplete={goToCompletion} />;
      case 'completion':
        return <CompletionStep />;
      default:
        return <div className="text-center py-10"><p>{t('steps.unknown_step')}: {currentStep}</p></div>;
    }
  };

  const pageDisplayTitle = t(`steps.${currentStep}.page_title`, t('page_title_default'));

  return (
    <SetupLayout title={`${pageDisplayTitle} - ${t('common:app_name')}`} pageTitle={pageDisplayTitle}>
      {renderCurrentStep()}
    </SetupLayout>
  );
};

export default SetupWizardPage;
