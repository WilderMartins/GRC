import Head from 'next/head';
import { useRouter } from 'next/router';
import { useEffect, useState, useCallback } from 'react';
import SetupLayout from '@/components/layouts/SetupLayout';
import apiClient from '@/lib/axios';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';

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

  const [currentStep, setCurrentStep] = useState<SetupStep>('loading_status');
  const [apiError, setApiError] = useState<string | null>(null);
  // const [stepData, setStepData] = useState<any>({}); // Para passar dados entre etapas, se necessário

  const determineNextStep = useCallback((apiStatus: string) => {
    switch (apiStatus) {
      case 'database_not_configured':
        // No nosso plano, esta etapa instrui a editar o .env e re-verificar.
        // O frontend não coleta credenciais de DB.
        // Então, se o backend não consegue conectar ao DB, o wizard mostra 'db_config_check'.
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
        router.push('/auth/login'); // Redireciona se já completou
        break;
      default:
        // Se um status desconhecido vier, ou se quisermos começar pelo 'welcome'
        // se o DB estiver configurado mas outras etapas não claras.
        // Por segurança, se o status não for reconhecido como incompleto, pode indicar erro ou já completo.
        // Para este exemplo, vamos assumir que qualquer status não listado explicitamente como incompleto
        // ou é um erro (tratado no catch) ou deve levar ao login se não for um erro de API.
        // No entanto, a API /setup/status deve ser clara.
        // Se a API está ok, mas status é inesperado, tratar como erro de lógica.
        setApiError(t('steps.error_unexpected_status', { status: apiStatus }));
        setCurrentStep('server_error');
        break;
    }
  }, [router, t]);

  useEffect(() => {
    const fetchSetupStatus = async () => {
      setCurrentStep('loading_status'); // Garante que estamos em loading ao re-verificar
      setApiError(null);
      try {
        const response = await apiClient.get<SetupStatusResponse>('/setup/status'); // API Hipotética
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
    };
    fetchSetupStatus();
  }, [determineNextStep, t]); // Adicionado t

  // Funções de navegação (serão usadas pelos componentes de etapa)
  const goToNextStep = (nextLogicalStep?: SetupStep) => {
    // Esta função pode se tornar mais complexa, determinando o próximo passo
    // com base no passo atual e no sucesso da ação do passo.
    // Por enquanto, pode ser um placeholder ou ser chamada com o próximo passo explícito.
    // Ou, melhor ainda, cada etapa, ao concluir, chama fetchSetupStatus novamente para que o backend dite o próximo passo.
    console.log("goToNextStep called. Next logical step might be:", nextLogicalStep);
    // Para forçar a reavaliação do estado:
    const fetchSetupStatusAgain = async () => {
        setCurrentStep('loading_status');
        setApiError(null);
        try {
            const response = await apiClient.get<SetupStatusResponse>('/setup/status');
            determineNextStep(response.data.status);
        } catch (err: any) {
            setApiError(err.response?.data?.message || err.message || t('steps.error_fetching_status'));
            setCurrentStep('server_error');
        }
    };
    fetchSetupStatusAgain();
  };

  const renderCurrentStep = () => {
    switch (currentStep) {
      case 'loading_status':
        return <div className="text-center"><p>{t('common:loading_ellipsis')}</p></div>;
      case 'welcome':
        // return <WelcomeStep onNext={() => goToNextStep('db_config_check')} />;
        return <div><h3 className="text-xl font-semibold mb-4">{t('steps.welcome.title')}</h3><p>{t('steps.welcome.description')}</p><button onClick={() => goToNextStep()} className="mt-4 px-4 py-2 bg-brand-primary text-white rounded hover:bg-brand-primary/90">{t('steps.welcome.start_button')}</button></div>;
      case 'db_config_check':
        // return <DatabaseStep onNext={() => goToNextStep('migrations')} onVerify={goToNextStep} />; // onVerify pode chamar goToNextStep para refazer o status check
         return <div><h3 className="text-xl font-semibold mb-4">{t('steps.db_config.title')}</h3><p className="text-sm mb-4">{t('steps.db_config.instructions_env')}</p><p className="text-xs mb-4">{t('steps.db_config.instructions_env_detail')}</p><button onClick={() => goToNextStep()} className="mt-4 px-4 py-2 bg-brand-primary text-white rounded hover:bg-brand-primary/90">{t('steps.db_config.verify_button')}</button></div>;
      case 'migrations':
        // return <MigrationsStep onNext={() => goToNextStep('admin_creation')} />;
        return <div><h3 className="text-xl font-semibold mb-4">{t('steps.migrations.title')}</h3><p>{t('steps.migrations.description')}</p><button onClick={() => { /* TODO: apiClient.post('/setup/run-migrations').then(goToNextStep) */ console.log("TODO: Run migrations"); goToNextStep();}} className="mt-4 px-4 py-2 bg-brand-primary text-white rounded hover:bg-brand-primary/90">{t('steps.migrations.run_button')}</button></div>;
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
