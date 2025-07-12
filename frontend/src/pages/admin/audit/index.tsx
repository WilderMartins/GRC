import Head from 'next/head';
import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios';
import { AuditFramework } from '@/types';

type Props = {}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => {
  return {
    props: {
      ...(await serverSideTranslations(locale ?? 'pt', ['common', 'audit'])),
    },
  };
};

const AuditFrameworksPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['audit', 'common']);
  const [frameworks, setFrameworks] = useState<AuditFramework[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setIsLoading(true);
    setError(null);
    apiClient.get<AuditFramework[]>('/api/v1/audit/frameworks')
      .then(response => {
        setFrameworks(response.data);
      })
      .catch(err => {
        console.error(t('framework_list.error_loading_console'), err);
        setError(err.response?.data?.error || t('framework_list.error_loading'));
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [t]);

  const pageTitle = t('framework_list.page_title');
  const appName = t('common:app_name');

  return (
    <AdminLayout title={`${pageTitle} - ${appName}`}>
      <Head>
        <title>{`${pageTitle} - ${appName}`}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white mb-8">
          {pageTitle}
        </h1>

        {isLoading && <p className="text-center">{t('common:loading_ellipsis')}</p>}
        {error && <p className="text-center text-red-500">{error}</p>}

        {!isLoading && !error && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {frameworks.map(fw => (
              <Link key={fw.ID} href={`/admin/audit/frameworks/${fw.ID}`} legacyBehavior>
                <a className="block p-6 bg-white dark:bg-gray-800 rounded-lg shadow hover:shadow-lg transition-shadow duration-200">
                  <h2 className="text-xl font-semibold text-brand-primary dark:text-brand-primary mb-2">{fw.Name}</h2>
                  <p className="text-gray-600 dark:text-gray-400 text-sm">{fw.Description}</p>
                   <div className="mt-4 text-xs text-gray-400 dark:text-gray-500">
                        <span>Version: {fw.Version || 'N/A'}</span>
                    </div>
                </a>
              </Link>
            ))}
             {frameworks.length === 0 && (
                <p className="text-center text-gray-500 col-span-full">{t('framework_list.no_frameworks_found')}</p>
            )}
          </div>
        )}
      </div>
    </AdminLayout>
  );
};

export default WithAuth(AuditFrameworksPageContent);
