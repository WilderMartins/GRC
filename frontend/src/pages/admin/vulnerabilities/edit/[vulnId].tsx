import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import VulnerabilityForm from '@/components/vulnerabilities/VulnerabilityForm';
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios';
import { Vulnerability, VulnerabilitySeverity, VulnerabilityStatus } from '@/types';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetServerSideProps, InferGetServerSidePropsType } from 'next';

type Props = {
  // Props from getServerSideProps
}

export const getServerSideProps: GetServerSideProps<Props> = async ({ locale, params }) => {
  // const { vulnId } = params; // vulnId está disponível aqui se necessário para pré-carregar dados
  // Mas vamos manter a busca de dados no lado do cliente por enquanto para consistência com outras páginas de edição.
  return {
    props: {
      ...(await serverSideTranslations(locale ?? 'pt', ['common', 'vulnerabilities'])),
    },
  };
};

const EditVulnerabilityPageContent = (props: InferGetServerSidePropsType<typeof getServerSideProps>) => {
  const { t } = useTranslation(['vulnerabilities', 'common']);
  const router = useRouter();
  const { vulnId } = router.query;
  const [initialData, setInitialData] = useState<Vulnerability | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (vulnId && typeof vulnId === 'string') {
      setIsLoading(true);
      setError(null);
      apiClient.get(`/vulnerabilities/${vulnId}`)
        .then(response => {
          setInitialData(response.data);
        })
        .catch(err => {
          console.error("Erro ao buscar dados da vulnerabilidade:", err);
          setError(err.response?.data?.error || err.message || t('common:error_loading_data_entity', {entity: t('common:vulnerability_singular')}));
        })
        .finally(() => setIsLoading(false));
    } else if (router.isReady && !vulnId) { // Checar router.isReady antes de assumir que vulnId está ausente
        setError(t('common:error_invalid_id', {entity: t('common:vulnerability_singular')}));
        setIsLoading(false);
    }
  }, [vulnId, router.isReady, t]);

  const handleSuccess = () => {
    router.push('/admin/vulnerabilities');
  };

  const pageTitleBase = t('form.edit_page_title');
  const appName = t('common:app_name');
  const dynamicPageTitle = initialData?.title ? `${pageTitleBase}: ${initialData.title} - ${appName}` : `${pageTitleBase} - ${appName}`;

  if (isLoading && !initialData && !error) { // Mostrar loading apenas se não houver erro ainda e não houver dados
    return <AdminLayout title={t('common:loading_data')}><div className="text-center p-10">{t('common:loading_data_entity', {entity: t('common:vulnerability_singular')})}</div></AdminLayout>;
  }

  if (error) {
    return <AdminLayout title={t('common:error_page_title')}><div className="text-center p-10 text-red-500">{error}</div></AdminLayout>;
  }

  if (!initialData && !isLoading) { // Se não está carregando, não houve erro, mas não há dados
     return <AdminLayout title={t('common:error_not_found_title')}><div className="text-center p-10">{t('common:error_entity_not_found', {entity: t('common:vulnerability_singular')})}</div></AdminLayout>;
  }

  return (
    <AdminLayout title={dynamicPageTitle}>
      <Head>
        <title>{dynamicPageTitle}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            {t('form.edit_page_header_prefix')} <span className="text-indigo-600 dark:text-indigo-400">{initialData?.title}</span>
          </h1>
          <Link href="/admin/vulnerabilities" legacyBehavior>
            <a className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-200">
              &larr; {t('form.back_to_list_link')}
            </a>
          </Link>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          {initialData && (
            <VulnerabilityForm
              initialData={{
                id: initialData.id,
                title: initialData.title,
                description: initialData.description,
                cve_id: initialData.cve_id || '',
                severity: initialData.severity as VulnerabilitySeverity,
                status: initialData.status as VulnerabilityStatus,
                asset_affected: initialData.asset_affected,
              }}
              isEditing={true}
              onSubmitSuccess={handleSuccess}
            />
          )}
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(EditVulnerabilityPageContent);
