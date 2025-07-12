import { useRouter } from 'next/router';
import Head from 'next/head';
import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetServerSideProps, InferGetServerSidePropsType } from 'next';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios';
import { Vulnerability } from '@/types';

type Props = {}

export const getServerSideProps: GetServerSideProps<Props> = async ({ locale }) => {
  return {
    props: {
      ...(await serverSideTranslations(locale ?? 'pt', ['common', 'vulnerabilities'])),
    },
  };
};

const ViewVulnerabilityPageContent = (props: InferGetServerSidePropsType<typeof getServerSideProps>) => {
  const { t } = useTranslation(['vulnerabilities', 'common']);
  const router = useRouter();
  const { vulnId } = router.query;

  const [vulnerability, setVulnerability] = useState<Vulnerability | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (vulnId) {
      setIsLoading(true);
      setError(null);
      apiClient.get<Vulnerability>(`/api/v1/vulnerabilities/${vulnId}`)
        .then(response => {
          setVulnerability(response.data);
        })
        .catch(err => {
          console.error(t('view_page.error_loading_vuln_console'), err);
          setError(err.response?.data?.error || t('view_page.error_loading_vuln'));
        })
        .finally(() => {
          setIsLoading(false);
        });
    }
  }, [vulnId, t]);

  const pageTitle = t('view_page.page_title');
  const appName = t('common:app_name');

  if (isLoading) {
    return <AdminLayout title={t('common:loading_ellipsis')}><div className="p-6 text-center">{t('common:loading_ellipsis')}</div></AdminLayout>;
  }

  if (error) {
    return <AdminLayout title={t('common:error_page_title')}><div className="p-6 text-center text-red-500">{error}</div></AdminLayout>;
  }

  if (!vulnerability) {
    return <AdminLayout title={t('common:error_not_found_title')}><div className="p-6 text-center">{t('view_page.error_vuln_not_found')}</div></AdminLayout>;
  }

  return (
    <AdminLayout title={`${pageTitle}: ${vulnerability.title}`}>
      <Head>
        <title>{`${pageTitle}: ${vulnerability.title} - ${appName}`}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-6 flex justify-between items-center">
          <Link href="/admin/vulnerabilities" legacyBehavior>
            <a className="text-sm text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70">
              &larr; {t('common:back_to_list_link_generic', { list_name: t('list.page_title', {ns: 'vulnerabilities'}) })}
            </a>
          </Link>
          <Link href={`/admin/vulnerabilities/edit/${vulnerability.id}`} legacyBehavior>
            <a className="inline-flex items-center justify-center rounded-md border border-transparent bg-brand-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2">
              {t('common:action_edit')}
            </a>
          </Link>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow overflow-hidden sm:rounded-lg">
          <div className="px-4 py-5 sm:px-6">
            <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">
              {vulnerability.title}
            </h3>
            <p className="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">
              {t('view_page.details_subtitle')}
            </p>
          </div>
          <div className="border-t border-gray-200 dark:border-gray-700">
            <dl>
              <div className="bg-gray-50 dark:bg-gray-800 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_page.field_description')}</dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">{vulnerability.description || '-'}</dd>
              </div>
              <div className="bg-white dark:bg-gray-900 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_page.field_cve_id')}</dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">{vulnerability.cve_id || '-'}</dd>
              </div>
              <div className="bg-gray-50 dark:bg-gray-800 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_page.field_asset_affected')}</dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">{vulnerability.asset_affected || '-'}</dd>
              </div>
              <div className="bg-white dark:bg-gray-900 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_page.field_severity')}</dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">{t(`severity_options.${vulnerability.severity}`, {ns: 'vulnerabilities', defaultValue: vulnerability.severity})}</dd>
              </div>
              <div className="bg-gray-50 dark:bg-gray-800 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_page.field_status')}</dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        vulnerability.status === 'descoberta' ? 'bg-orange-100 text-orange-800 dark:bg-orange-700 dark:text-orange-100' :
                        vulnerability.status === 'em_correcao' ? 'bg-blue-100 text-blue-800 dark:bg-blue-700 dark:text-blue-100' :
                        vulnerability.status === 'corrigida' ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' :
                        'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200'
                    }`}>
                        {t(`status_options.${vulnerability.status}`, {ns: 'vulnerabilities', defaultValue: vulnerability.status})}
                    </span>
                </dd>
              </div>
               <div className="bg-white dark:bg-gray-900 px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_page.field_remediation')}</dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">{vulnerability.remediation_details || t('view_page.no_remediation_provided')}</dd>
              </div>
            </dl>
          </div>
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(ViewVulnerabilityPageContent);
