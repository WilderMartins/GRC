import React, { useEffect, useState } from 'react';
import Head from 'next/head';
import { useAuth } from '../contexts/AuthContext';
import WithAuth from '../components/auth/WithAuth';
import Link from 'next/link';
import apiClient from '@/lib/axios';
// import { useNotifier } from '@/hooks/useNotifier'; // Notifier não está sendo usado ativamente aqui para erros de fetch
import StatCard from '@/components/common/StatCard';
import { UserDashboardSummary } from '@/types';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'dashboard', 'auth'])), // Incluindo 'auth' para consistência e se user.name for usado
  },
});


const DashboardPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['dashboard', 'common']);
  const { user, logout } = useAuth();
  // const notify = useNotifier(); // Mantido se formos adicionar notificações de erro aqui

  const [summary, setSummary] = useState<UserDashboardSummary | null>(null);
  const [isLoadingSummary, setIsLoadingSummary] = useState(true);
  const [summaryError, setSummaryError] = useState<string | null>(null);

  useEffect(() => {
    const fetchUserSummary = async () => {
      if (!user) return;
      setIsLoadingSummary(true);
      setSummaryError(null);
      try {
        const response = await apiClient.get<UserDashboardSummary>('/me/dashboard/summary');
        setSummary(response.data);
      } catch (err: any) {
        console.error("Erro ao buscar resumo do dashboard do usuário:", err);
        const apiError = err.response?.data?.error || t('common:unknown_error');
        setSummaryError(t('user.error_loading_summary', { message: apiError }));
      } finally {
        setIsLoadingSummary(false);
      }
    };
    fetchUserSummary();
  }, [user, t]); // Adicionado t como dependência

  const pageTitle = t('user.page_title');
  const appName = t('common:app_name');

  return (
    <>
      <Head>
        <title>{`${pageTitle} - ${appName}`}</title>
      </Head>
      <div className="min-h-screen bg-gray-100 dark:bg-gray-900">
        <header className="bg-white dark:bg-gray-800 shadow">
          <div className="container mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex h-16 items-center justify-between">
              <div className="flex items-center">
                <Link href="/dashboard" legacyBehavior>
                  <a className="font-bold text-xl text-indigo-600 dark:text-indigo-400">
                    {appName}
                  </a>
                </Link>
              </div>
              <div className="flex items-center">
                {user && (
                  <span className="text-gray-700 dark:text-gray-300 mr-4">
                    {t('user.welcome_greeting_header', { userName: user.name || user.email || t('common:guest_user')})}
                  </span>
                )}
                <button
                  onClick={logout}
                  className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
                >
                  {t('common:logout_button')}
                </button>
              </div>
            </div>
          </div>
        </header>

        <main className="py-10">
          <div className="container mx-auto px-4 sm:px-6 lg:px-8">
            <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white mb-8">
              {t('user.header')}
            </h1>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
              <StatCard
                title={t('user.summary_assigned_risks_open')}
                value={summary?.assigned_risks_open_count ?? '-'}
                isLoading={isLoadingSummary}
                linkTo="/admin/risks" // TODO: Link para riscos do usuário, não admin
                error={summaryError && summary?.assigned_risks_open_count === undefined ? t('common:error_loading_specific') : null}
              />
              <StatCard
                title={t('user.summary_assigned_vulnerabilities_open')}
                value={summary?.assigned_vulnerabilities_open_count ?? '-'}
                isLoading={isLoadingSummary}
                linkTo="/admin/vulnerabilities" // TODO: Link para vulnerabilidades do usuário
                error={summaryError && summary?.assigned_vulnerabilities_open_count === undefined ? t('common:error_loading_specific') : null}
              />
              <StatCard
                title={t('user.summary_pending_approval_tasks')}
                value={summary?.pending_approval_tasks_count ?? '-'}
                isLoading={isLoadingSummary}
                // linkTo="/approvals" // TODO: Link para página de aprovações do usuário
                error={summaryError && summary?.pending_approval_tasks_count === undefined ? t('common:error_loading_specific') : null}
              />
            </div>
            {summaryError && !summary && (
                 <p className="text-sm text-red-500 dark:text-red-400 mb-6 text-center">{summaryError}</p>
            )}

            {user && (
              <div className="mt-6 bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                <h2 className="text-lg font-medium text-gray-900 dark:text-white">
                  {t('user.profile_info_header')}
                </h2>
                <dl className="mt-5 grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-2">
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('user.profile_name_label')}</dt>
                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{user.name}</dd>
                  </div>
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('user.profile_email_label')}</dt>
                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{user.email}</dd>
                  </div>
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('user.profile_role_label')}</dt>
                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{user.role}</dd>
                  </div>
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('user.profile_org_id_label')}</dt>
                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{user.organization_id}</dd>
                  </div>
                </dl>
              </div>
            )}

            <div className="mt-8">
              {user?.role === 'admin' || user?.role === 'manager' ? (
                <p className="text-gray-700 dark:text-gray-300">
                  {t('user.admin_panel_link_text', {
                      adminDashboardLink: (
                          <Link href="/admin/dashboard">
                            <span className="text-indigo-600 hover:underline dark:text-indigo-400">
                                {t('user.admin_dashboard_link_name')}
                            </span>
                          </Link>
                      )
                  })}
                </p>
              ) : (
                <p className="text-gray-700 dark:text-gray-300">
                  {t('user.user_navigation_prompt')}
                </p>
              )}
            </div>
          </div>
        </main>
      </div>
    </>
  );
};

const DashboardPage = WithAuth(DashboardPageContent);
export default DashboardPage;
