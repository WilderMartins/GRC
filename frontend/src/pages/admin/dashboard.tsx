import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import { useAuth } from '@/contexts/AuthContext';
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios';
import Link from 'next/link'; // Mantido para o caso de links de atividade recente
import StatCard from '@/components/common/StatCard';
import { AdminStatistics, ActivityLog } from '@/types';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import Head from 'next/head'; // Adicionado Head para o título da página

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'dashboard', 'auth'])), // Adicionado 'auth' se user.name for usado em saudações
  },
});

const AdminDashboardContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['dashboard', 'common']); // Especificar namespaces
  const { user } = useAuth();

  const [statistics, setStatistics] = useState<AdminStatistics | null>(null);
  const [recentActivity, setRecentActivity] = useState<ActivityLog[]>([]);

  const [isLoadingStats, setIsLoadingStats] = useState(true);
  const [statsError, setStatsError] = useState<string | null>(null);

  const [isLoadingActivity, setIsLoadingActivity] = useState(true);
  const [activityError, setActivityError] = useState<string | null>(null);

  useEffect(() => {
    const fetchDashboardData = async () => {
      setIsLoadingStats(true);
      setStatsError(null);
      try {
        const statsResponse = await apiClient.get<AdminStatistics>('/admin/dashboard/statistics');
        setStatistics(statsResponse.data);
      } catch (err: any) {
        console.error("Erro ao buscar estatísticas do admin:", err);
        const apiError = err.response?.data?.error || t('common:unknown_error');
        setStatsError(t('admin.error_loading_statistics', { message: apiError }));
      } finally {
        setIsLoadingStats(false);
      }

      setIsLoadingActivity(true);
      setActivityError(null);
      try {
        const activityResponse = await apiClient.get<ActivityLog[]>('/admin/dashboard/recent-activity?limit=5');
        setRecentActivity(activityResponse.data || []);
      } catch (err: any) {
        console.error("Erro ao buscar atividade recente:", err);
        const apiError = err.response?.data?.error || t('common:unknown_error');
        setActivityError(t('admin.error_loading_activities', { message: apiError }));
      } finally {
        setIsLoadingActivity(false);
      }
    };

    if (user) {
      fetchDashboardData();
    }
  }, [user, t]); // Adicionado t como dependência para as mensagens de erro

  const formatTimestamp = (timestamp: string) => {
    try {
      return new Date(timestamp).toLocaleString(t('common:locale_date_time_format') || 'pt-BR', { // Usar locale para formatação
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch (e) {
      return timestamp;
    }
  };

  const pageTitle = t('admin.page_title');
  const appName = t('common:app_name');

  return (
    <AdminLayout title={`${pageTitle} - ${appName}`}>
      <Head>
        <title>{`${pageTitle} - ${appName}`}</title>
      </Head>
      <div className="container mx-auto px-4 py-8">
        <h1 className="text-3xl font-bold text-gray-800 dark:text-white mb-6">
          {t('admin.header')}
        </h1>
        <p className="text-gray-600 dark:text-gray-300 mb-8">
          {t('admin.welcome_message', { userName: user?.name || user?.email || t('common:guest_user') })}
        </p>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <StatCard
            title={t('admin.stats_active_users')}
            value={statistics?.active_users_count ?? '-'}
            isLoading={isLoadingStats}
            error={statsError && statistics?.active_users_count === undefined ? t('common:error_loading_specific') : null}
          />
          <StatCard
            title={t('admin.stats_total_risks')}
            value={statistics?.total_risks_count ?? '-'}
            isLoading={isLoadingStats}
            error={statsError && statistics?.total_risks_count === undefined ? t('common:error_loading_specific') : null}
          />
          <StatCard
            title={t('admin.stats_active_frameworks')}
            value={statistics?.active_frameworks_count ?? '-'}
            isLoading={isLoadingStats}
            error={statsError && statistics?.active_frameworks_count === undefined ? t('common:error_loading_specific') : null}
          />
          <StatCard
            title={t('admin.stats_open_vulnerabilities')}
            value={statistics?.open_vulnerabilities_count ?? '-'}
            isLoading={isLoadingStats}
            error={statsError && statistics?.open_vulnerabilities_count === undefined ? t('common:error_loading_specific') : null}
          />
        </div>
        {statsError && !statistics && (
             <p className="text-sm text-red-500 dark:text-red-400 mb-6 text-center">{statsError}</p>
        )}

        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md">
          <h2 className="text-xl font-semibold text-gray-700 dark:text-white mb-4">{t('admin.recent_activity_header')}</h2>
          {isLoadingActivity && <p className="text-gray-500 dark:text-gray-400">{t('admin.loading_activities')}</p>}
          {activityError && <p className="text-red-500 dark:text-red-400">{activityError}</p>}
          {!isLoadingActivity && !activityError && recentActivity.length === 0 && (
            <p className="text-gray-500 dark:text-gray-400">{t('admin.no_recent_activity')}</p>
          )}
          {!isLoadingActivity && !activityError && recentActivity.length > 0 && (
            <ul className="space-y-4">
              {recentActivity.map((activity) => (
                <li key={activity.id} className="border-b border-gray-200 dark:border-gray-700 pb-3 last:border-b-0">
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="font-semibold text-brand-primary dark:text-brand-primary">{activity.actor_name}</span>
                      <span className="text-gray-600 dark:text-gray-300"> {activity.action_description}</span>
                    </div>
                    <span className="text-xs text-gray-400 dark:text-gray-500 whitespace-nowrap ml-2">
                      {formatTimestamp(activity.timestamp)}
                    </span>
                  </div>
                  {activity.target_link && (
                    <Link href={activity.target_link} legacyBehavior>
                      <a className="text-sm text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70 mt-1 inline-block transition-colors">
                        {t('admin.view_details_link')}
                      </a>
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(AdminDashboardContent);
