import Head from 'next/head';
import { useRouter } from 'next/router';
import { useEffect, useState, useCallback } from 'react';
import SetupLayout from '@/components/layouts/SetupLayout';
import apiClient from '@/lib/axios';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import WelcomeStep from '@/components/setup/WelcomeStep';
import DatabaseStep from '@/components/setup/DatabaseStep';
import MigrationsStep from '@/components/setup/MigrationsStep'; // Importar MigrationsStep
import { useNotifier } from '@/hooks/useNotifier'; // Importar useNotifier

// Definir os tipos para as etapas do wizard e status da API
type SetupStep =
  | 'loading_status'
  | 'welcome'
  | 'db_config_check'
  | 'migrations'
  | 'admin_creation'
  | 'completed_redirect' // Estado intermediário antes do redirect
  | 'server_error';

interface SetupStatusResponse {
  status: 'database_not_configured' | 'db_configured_pending_migrations' | 'migrations_done_pending_admin' | 'completed' | string; // string para outros status futuros
  message?: string;
}

type Props = {};

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'setupWizard'])),
  },
});

const SetupWizardPage = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['setupWizard', 'common']);
  const router = useRouter();
  const notify = useNotifier(); // Adicionar notifier

  const [currentStep, setCurrentStep] = useState<SetupStep>('loading_status'); // Iniciar em loading
  const [apiError, setApiError] = useState<string | null>(null);
  const [isProcessingMigrations, setIsProcessingMigrations] = useState(false);

  const determineNextStep = useCallback((apiStatus: string) => {
    switch (apiStatus) {
      case 'database_not_configured':
        setCurrentStep('db_config_check');
        break;
      case 'db_configured_pending_migrations':
        setCurrentStep('migrations');
        break;
      case 'migrations_done_pending_admin':
        setCurrentStep('admin_creation');
        break;
      case 'completed':
        setCurrentStep('completed_redirect');
        router.push('/auth/login');
        break;
      default:
        // Se status não é conhecido e não é 'completed', consideramos um estado inicial válido para 'welcome'.
        // Se a API de status falhar, o catch em fetchSetupStatus definirá 'server_error'.
        setCurrentStep('welcome');
        break;
    }
  }, [router, t]);

  const fetchSetupStatus = useCallback(async () => {
    // Não resetar currentStep para 'loading_status' aqui,
    // pois queremos que o loader seja específico da ação de goToNextStep
    // ou do carregamento inicial da página.
    setApiError(null);
    try {
      const response = await apiClient.get<SetupStatusResponse>('/setup/status');
      if (response.data && response.data.status) {
        determineNextStep(response.data.status);
      } else {
        throw new Error('Invalid response from /setup/status');
      }
    } catch (err: any) {
      console.error("Error fetching setup status:", err);
      setApiError(err.response?.data?.message || err.message || t('steps.error_fetching_status'));
      setCurrentStep('server_error');
    }
  }, [determineNextStep, t]);


  useEffect(() => {
    // Fetch inicial do status
    setCurrentStep('loading_status'); // Mostrar loading na primeira carga
    fetchSetupStatus().finally(() => {
        // Se após o fetch inicial o currentStep ainda for loading_status
        // (porque determineNextStep o setou para 'welcome' que é o default e não queremos loader para welcome)
        // precisamos garantir que não fique em loading_status.
        // A lógica de determineNextStep já deve setar para 'welcome' ou outra etapa.
        // Este finally pode não ser necessário se determineNextStep sempre setar um estado final.
        // No entanto, para garantir, se após tudo currentStep for 'loading_status', vá para 'welcome'.
        // Esta lógica foi simplificada em determineNextStep.
    });
  }, [fetchSetupStatus]);


  const goToNextStep = useCallback(() => {
    setCurrentStep('loading_status'); // Mostra loader e dispara o useEffect abaixo
  }, []);

  // Este useEffect reage à mudança de currentStep para 'loading_status' (causada por goToNextStep)
  // ou à montagem inicial se currentStep começar como 'loading_status'.
  useEffect(() => {
    if (currentStep === 'loading_status') {
      fetchSetupStatus();
    }
  }, [currentStep, fetchSetupStatus]);


  const handleRunMigrations = async () => {
    setIsProcessingMigrations(true);
    setApiError(null);
    try {
      await apiClient.post('/setup/run-migrations');
      notify.success(t('steps.migrations.success_message'));
      goToNextStep(); // Dispara a re-verificação de status para avançar
    } catch (err: any) {
      const errorMsg = err.response?.data?.message || err.message || t('steps.migrations.error_running');
      setApiError(errorMsg); // Erro será exibido pelo MigrationsStep
      notify.error(errorMsg);
      // Permanece na etapa de migrações para o usuário tentar novamente ou ver o erro
      setCurrentStep('migrations'); // Garante que fica na etapa de migração se houve erro
    } finally {
      setIsProcessingMigrations(false);
    }
  };

  const renderCurrentStep = () => {
    if (currentStep === 'loading_status' && !apiError) {
      return <div className="text-center py-10"><p>{t('common:loading_status_check', 'Verificando status da configuração...')}</p></div>;
    }
    if (currentStep === 'server_error' && apiError) {
      return <div className="text-center text-red-500 dark:text-red-300 py-10"><h3 className="text-xl font-semibold mb-2">{t('steps.error.title')}</h3><p>{apiError}</p></div>;
    }

    switch (currentStep) {
      case 'welcome':
        return <WelcomeStep onNext={goToNextStep} />;
      case 'db_config_check':
        return <DatabaseStep onVerifyAndContinue={goToNextStep} errorMessage={apiError} />;
      case 'migrations':
        return <MigrationsStep
                  onRunMigrations={handleRunMigrations}
                  isLoading={isProcessingMigrations}
                  errorMessage={apiError}
                />;
      case 'admin_creation':
        // return <AdminCreationStep onNext={() => goToNextStep('completed_redirect')} />;
        return <div><h3 className="text-xl font-semibold mb-4">{t('steps.admin_creation.title')}</h3><p>{t('steps.admin_creation.description')}</p><button onClick={() => { /* TODO: Submit admin form, then goToNextStep */ console.log("TODO: Create admin"); goToNextStep(); }} className="mt-4 px-4 py-2 bg-brand-primary text-white rounded hover:bg-brand-primary/90">{t('steps.admin_creation.create_button')}</button></div>;
      case 'completed_redirect':
        return <div className="text-center"><p>{t('steps.completed.redirecting')}</p></div>;
      case 'server_error':
        return <div className="text-center text-red-500 dark:text-red-300"><h3 className="text-xl font-semibold mb-2">{t('steps.error.title')}</h3><p>{apiError || t('steps.error.generic')}</p></div>;
      default:
        return <div className="text-center"><p>{t('steps.unknown_step')}: {currentStep}</p></div>;
    }
  };

  const pageDisplayTitle = t(`steps.${currentStep}.page_title`, t('page_title_default'));


  return (
    <SetupLayout title={`${pageDisplayTitle} - ${t('common:app_name')}`} pageTitle={currentStep !== 'loading_status' && currentStep !== 'completed_redirect' ? pageDisplayTitle : undefined}>
      {renderCurrentStep()}
    </SetupLayout>
  );
};

export default SetupWizardPage;
