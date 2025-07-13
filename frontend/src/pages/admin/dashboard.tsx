import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import { useAuth } from '@/contexts/AuthContext';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import Head from 'next/head';
import RiskMatrixChart from '@/components/dashboard/RiskMatrixChart';
import VulnerabilitySummary from '@/components/dashboard/VulnerabilitySummary';
import ComplianceGauge from '@/components/dashboard/ComplianceGauge';
import RecentActivityFeed from '@/components/dashboard/RecentActivityFeed';

type Props = {}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'dashboard'])),
  },
});

const AdminDashboardContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['dashboard', 'common']);
  const { user } = useAuth();

  const pageTitle = t('page_title');
  const appName = t('common:app_name');

  return (
    <AdminLayout title={`${pageTitle} - ${appName}`}>
      <Head>
        <title>{`${pageTitle} - ${appName}`}</title>
      </Head>
      <div className="container mx-auto px-4 py-8">
        <h1 className="text-3xl font-bold text-gray-800 dark:text-white mb-2">
          {t('header')}
        </h1>
        <p className="text-gray-600 dark:text-gray-300 mb-8">
          {t('welcome_message', { userName: user?.name || user?.email || t('common:guest_user') })}
        </p>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Coluna Principal (2/3 da largura) */}
          <div className="lg:col-span-2 space-y-6">
            <RiskMatrixChart />
            <VulnerabilitySummary />
          </div>

          {/* Coluna Lateral (1/3 da largura) */}
          <div className="space-y-6">
            <ComplianceGauge />
            <RecentActivityFeed />
          </div>
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(AdminDashboardContent);
