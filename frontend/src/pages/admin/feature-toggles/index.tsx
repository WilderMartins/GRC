import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import { useAuth } from '@/contexts/AuthContext';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import { useEffect, useState, useCallback } from 'react';
import apiClient from '@/lib/axios';
import { FeatureToggle } from '@/types';
import { Switch } from '@headlessui/react';
import { useNotifier } from '@/hooks/useNotifier';


type Props = {}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'featureToggles'])),
  },
});

const FeatureTogglesPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['featureToggles', 'common']);
  const { user, isLoading: authIsLoading } = useAuth();
  const notify = useNotifier();

  const [featureToggles, setFeatureToggles] = useState<FeatureToggle[]>([]);
  const [isLoadingData, setIsLoadingData] = useState(true);
  const [dataError, setDataError] = useState<string | null>(null);

  const canManageFeatureToggles = user?.role === 'admin';

  const fetchFeatureToggles = useCallback(async () => {
    if (!canManageFeatureToggles) {
      setIsLoadingData(false);
      setDataError(t('common:error_insufficient_permissions'));
      return;
    }
    setIsLoadingData(true);
    setDataError(null);
    try {
      const response = await apiClient.get<FeatureToggle[]>('/feature-toggles'); // API Hipotética
      setFeatureToggles(response.data || []);
    } catch (err: any) {
      const apiError = err.response?.data?.error || t('common:unknown_error');
      setDataError(t('error_loading_toggles', { message: apiError }));
      notify.error(t('error_loading_toggles', { message: apiError }));
      setFeatureToggles([]);
    } finally {
      setIsLoadingData(false);
    }
  }, [canManageFeatureToggles, t, notify]);

  useEffect(() => {
    if (!authIsLoading && user) {
      fetchFeatureToggles();
    } else if (!authIsLoading && !user) {
      setIsLoadingData(false);
      setDataError(t('common:error_unauthenticated_action'));
    }
  }, [authIsLoading, user, fetchFeatureToggles]);

  const handleToggleChange = async (key: string, newIsActive: boolean) => {
    // Otimisticamente atualiza a UI
    setFeatureToggles(currentToggles =>
      currentToggles.map(toggle =>
        toggle.key === key ? { ...toggle, is_active: newIsActive } : toggle
      )
    );

    try {
      await apiClient.put(`/feature-toggles/${key}`, { is_active: newIsActive }); // API Hipotética
      notify.success(t('update_success', { toggleKey: key }));
      // Opcional: Re-fetch para garantir consistência, embora a atualização otimista já tenha sido feita.
      // await fetchFeatureToggles();
    } catch (err: any) {
      const apiError = err.response?.data?.error || t('common:unknown_error');
      notify.error(t('update_error', { toggleKey: key, message: apiError }));
      // Reverter a mudança otimista em caso de erro
      setFeatureToggles(currentToggles =>
        currentToggles.map(toggle =>
          toggle.key === key ? { ...toggle, is_active: !newIsActive } : toggle
        )
      );
    }
  };

  const pageTitle = t('page_title');
  const appName = t('common:app_name');

  if (authIsLoading) {
    return <AdminLayout title={t('common:loading_ellipsis')}><div className="p-6 text-center">{t('common:loading_user_data')}</div></AdminLayout>;
  }

  if (!canManageFeatureToggles && !authIsLoading) { // Verifica permissão após auth carregar
    return (
      <AdminLayout title={t('common:access_denied')}>
        <div className="p-6 text-center text-red-500 dark:text-red-300">
          {t('common:error_insufficient_permissions')}
        </div>
      </AdminLayout>
    );
  }

  return (
    <AdminLayout title={`${pageTitle} - ${appName}`}>
      <Head>
        <title>{`${pageTitle} - ${appName}`}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            {t('header')}
          </h1>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
          {isLoadingData && (
            <p className="p-6 text-center text-gray-500 dark:text-gray-400">
              {t('table_placeholder_loading')}
            </p>
          )}
          {dataError && !isLoadingData && ( // Exibir erro apenas se não estiver carregando
            <p className="p-6 text-center text-red-500 dark:text-red-300">
              {dataError}
            </p>
          )}
          {!isLoadingData && !dataError && featureToggles.length === 0 && (
            <p className="p-6 text-center text-gray-500 dark:text-gray-400">
              {t('no_toggles_found')}
            </p>
          )}
          {!isLoadingData && !dataError && featureToggles.length > 0 && (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-700">
                  <tr>
                    <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">{t('table_header_key')}</th>
                    <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white min-w-[300px]">{t('table_header_description')}</th>
                    <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('table_header_status')}</th>
                    <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('table_header_action_enable_disable')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                  {featureToggles.map((toggle) => (
                    <tr key={toggle.key}>
                      <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-mono text-gray-700 dark:text-gray-300 sm:pl-6">{toggle.key}</td>
                      <td className="px-3 py-4 text-sm text-gray-500 dark:text-gray-400">{toggle.description}</td>
                      <td className="whitespace-nowrap px-3 py-4 text-sm">
                        <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full
                          ${toggle.is_active
                            ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100'
                            : 'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200'}`}>
                          {toggle.is_active ? t('status_active') : t('status_inactive')}
                        </span>
                      </td>
                      <td className="whitespace-nowrap px-3 py-4 text-sm">
                        {toggle.read_only ? (
                          <span className="italic text-xs text-gray-400 dark:text-gray-500">{t('read_only_label')}</span>
                        ) : (
                          <Switch
                            checked={toggle.is_active}
                            onChange={(newIsActive) => handleToggleChange(toggle.key, newIsActive)}
                            className={`${
                              toggle.is_active ? 'bg-brand-primary' : 'bg-gray-300 dark:bg-gray-600'
                            } relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 dark:focus:ring-offset-gray-800`}
                          >
                            <span className="sr-only">{t('table_header_action_enable_disable')}</span>
                            <span
                              aria-hidden="true"
                              className={`${
                                toggle.is_active ? 'translate-x-5' : 'translate-x-0'
                              } inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out`}
                            />
                          </Switch>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(FeatureTogglesPageContent);
