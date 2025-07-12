import Head from 'next/head';
import { useRouter } from 'next/router';
import { useState } from 'react';
import SetupLayout from '@/components/layouts/SetupLayout';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import WelcomeStep from '@/components/setup/WelcomeStep';
import DatabaseStep from '@/components/setup/DatabaseStep';
import CompletionStep from '@/components/setup/CompletionStep';
import CliCommandStep from '@/components/setup/CliCommandStep'; // Novo componente

// Definir os tipos para as etapas do wizard
type SetupStep =
  | 'welcome'
  | 'db_config'
  | 'cli_command'
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

  const goToNextStep = () => {
    const stepOrder: SetupStep[] = ['welcome', 'db_config', 'cli_command', 'completion'];
    const currentIndex = stepOrder.indexOf(currentStep);
    if (currentIndex < stepOrder.length - 1) {
      setCurrentStep(stepOrder[currentIndex + 1]);
    }
  };

  const renderCurrentStep = () => {
    switch (currentStep) {
      case 'welcome':
        return <WelcomeStep onNext={goToNextStep} />;
      case 'db_config':
        return <DatabaseStep onVerifyAndContinue={goToNextStep} />;
      case 'cli_command':
        return <CliCommandStep onNext={goToNextStep} />;
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
