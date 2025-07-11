import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios';
import { AuditFramework } from '@/types';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'audit'])),
  },
});

const AuditFrameworksPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['audit', 'common']);
  const [frameworks, setFrameworks] = useState<AuditFramework[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchFrameworks = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const response = await apiClient.get<{ items: AuditFramework[] } | AuditFramework[]>('/audit/frameworks');
        if (Array.isArray(response.data)) {
            setFrameworks(response.data);
        } else if (response.data && Array.isArray((response.data as any).items)) {
            setFrameworks((response.data as any).items);
        } else {
            console.warn("Formato de resposta inesperado para frameworks:", response.data);
            setFrameworks([]);
        }
      } catch (err: any) {
        console.error("Erro ao buscar frameworks:", err);
        const apiError = err.response?.data?.error || err.message || t('common:unknown_error');
        setError(t('frameworks_list.error_loading_frameworks', { message: apiError }));
        setFrameworks([]);
      } finally {
        setIsLoading(false);
      }
    };

    fetchFrameworks();
  }, [t]); // Adicionado t como dependência para mensagens de erro

  const pageTitle = t('frameworks_list.page_title');
  const appName = t('common:app_name');

  return (
    <AdminLayout title={`${pageTitle} - ${appName}`}>
      <Head>
        <title>{`${pageTitle} - ${appName}`}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            {t('frameworks_list.header')}
          </h1>
          {/* Botão para adicionar novo framework (se aplicável no futuro) */}
        </div>

        {isLoading && <p className="text-center text-gray-500 dark:text-gray-400 py-10">{t('frameworks_list.loading_frameworks')}</p>}
        {error && <p className="text-center text-red-500 py-10">{error}</p>}

        {!isLoading && !error && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {frameworks.map((framework) => (
              <Link key={framework.id} href={`/admin/audit/frameworks/${framework.id}`} legacyBehavior>
                <a className="block p-6 bg-white dark:bg-gray-800 rounded-lg shadow-md hover:shadow-lg transition-shadow duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-brand-primary">
                  <h2 className="text-xl font-semibold text-brand-primary dark:text-brand-primary mb-2">{framework.name}</h2>
                  <p className="text-gray-600 dark:text-gray-400 text-sm">
                    {t('frameworks_list.card_description')}
                  </p>
                  {/* TODO: Adicionar contagem de controles ou progresso de conformidade aqui */}
                </a>
              </Link>
            ))}
            {frameworks.length === 0 && (
              <p className="text-gray-500 dark:text-gray-400 col-span-full text-center py-10">
                {t('frameworks_list.no_frameworks_found')}
              </p>
            )}
          </div>
        )}
      </div>
    </AdminLayout>
  );
};

export default WithAuth(AuditFrameworksPageContent);
