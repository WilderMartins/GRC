import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useState, useEffect, useCallback } from 'react';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';
import { useNotifier } from '@/hooks/useNotifier';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import { IdentityProvider, PaginatedResponse, IdentityProviderType } from '@/types';
import PaginationControls from '@/components/common/PaginationControls';

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'organizationSettings', 'idp'])),
  },
});

const IdentityProvidersPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['idp', 'common']);
  const { user: currentUser, isLoading: authLoading } = useAuth();
  const notify = useNotifier();

  const [idps, setIdps] = useState<IdentityProvider[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);

  const fetchIdps = useCallback(async () => {
    if (!currentUser?.organization_id) return;

    setIsLoading(true);
    setError(null);
    try {
      const params = { page: currentPage, page_size: pageSize };
      const response = await apiClient.get<PaginatedResponse<IdentityProvider>>(
        `/organizations/${currentUser.organization_id}/identity-providers`,
        { params }
      );
      setIdps(response.data.items || []);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
    } catch (err: any) {
      console.error(t('list.error_loading_idps_console'), err);
      setError(err.response?.data?.error || t('common:error_loading_list_general', { list_name: t('list.title') }));
    } finally {
      setIsLoading(false);
    }
  }, [currentUser?.organization_id, currentPage, pageSize, t]);

  useEffect(() => {
    if (!authLoading && currentUser?.organization_id) {
      fetchIdps();
    }
  }, [authLoading, currentUser?.organization_id, fetchIdps]);

  const handlePageChange = (newPage: number) => {
    setCurrentPage(newPage);
  };

  const handleDeleteIdp = async (idpId: string, idpName: string) => {
    if (!currentUser?.organization_id) return;
    if (window.confirm(t('list.confirm_delete_message', { idpName }))) {
      // Idealmente, ter um estado de loading para o botão de delete específico
      try {
        await apiClient.delete(`/organizations/${currentUser.organization_id}/identity-providers/${idpId}`);
        notify.success(t('list.delete_success_message', { idpName }));
        fetchIdps(); // Re-fetch
      } catch (err: any) {
        notify.error(t('list.delete_error_message', { message: err.response?.data?.error || t('common:unknown_error') }));
      }
    }
  };

  const getProviderTypeDisplay = (type: IdentityProviderType | string) => {
    switch(type) {
        case IdentityProviderType.SAML: return t('types.saml');
        case IdentityProviderType.OAUTH2_GOOGLE: return t('types.oauth2_google');
        case IdentityProviderType.OAUTH2_GITHUB: return t('types.oauth2_github');
        default: return type;
    }
  }


  const pageTitle = t('list.page_title');
  const appName = t('common:app_name');

  return (
    <AdminLayout title={`${pageTitle} - ${appName}`}>
      <Head>
        <title>{`${pageTitle} - ${appName}`}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="sm:flex sm:items-center sm:justify-between mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            {pageTitle}
          </h1>
          <div className="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
            <Link href="/admin/organization/identity-providers/new" legacyBehavior>
              <a className="inline-flex items-center justify-center rounded-md border border-transparent bg-brand-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 sm:w-auto">
                {t('list.add_new_button')}
              </a>
            </Link>
          </div>
        </div>

        {isLoading && <p className="text-center py-4">{t('common:loading_ellipsis')}</p>}
        {error && <p className="text-center text-red-500 py-4">{error}</p>}

        {!isLoading && !error && idps.length === 0 && (
          <div className="text-center py-10">
            <p className="text-gray-500 dark:text-gray-400">{t('list.no_idps_found')}</p>
          </div>
        )}

        {!isLoading && !error && idps.length > 0 && (
          <>
            <div className="mt-8 flow-root">
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                      <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                          <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">{t('list.header_name')}</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('list.header_type')}</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('list.header_status')}</th>
                          <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">{t('list.header_actions')}</span></th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                        {idps.map((idp) => (
                          <tr key={idp.id}>
                            <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{idp.name}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{getProviderTypeDisplay(idp.provider_type)}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm">
                              <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                idp.is_active ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' : 'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200'
                              }`}>
                                {idp.is_active ? t('common:status_active') : t('common:status_inactive')}
                              </span>
                            </td>
                            <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6 space-x-2">
                              <Link href={`/admin/organization/identity-providers/edit/${idp.id}`} legacyBehavior>
                                <a className="font-medium text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 rounded-sm">
                                  {t('common:action_edit')}
                                </a>
                              </Link>
                              <button
                                onClick={() => handleDeleteIdp(idp.id, idp.name)}
                                className="font-medium text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 rounded-sm"
                              >
                                {t('common:action_delete')}
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                   <PaginationControls
                        currentPage={currentPage}
                        totalPages={totalPages}
                        totalItems={totalItems}
                        pageSize={pageSize}
                        onPageChange={handlePageChange}
                        isLoading={isLoading}
                    />
                </div>
              </div>
            </div>
          </>
        )}
      </div>
    </AdminLayout>
  );
};

export default WithAuth(IdentityProvidersPageContent);
