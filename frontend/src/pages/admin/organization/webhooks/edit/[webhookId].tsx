import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import WebhookForm from '@/components/admin/organization/WebhookForm';
import { useEffect, useState, useCallback } from 'react';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetServerSideProps, InferGetServerSidePropsType } from 'next';
import { WebhookConfiguration } from '@/types';

type Props = {}

export const getServerSideProps: GetServerSideProps<Props> = async ({ locale }) => {
  return {
    props: {
      ...(await serverSideTranslations(locale ?? 'pt', ['common', 'organizationSettings', 'webhooks'])),
    },
  };
};

const EditWebhookPageContent = (props: InferGetServerSidePropsType<typeof getServerSideProps>) => {
  const { t } = useTranslation(['webhooks', 'common']);
  const router = useRouter();
  const { user: currentUser, isLoading: authLoading } = useAuth();
  const { webhookId } = router.query;

  const [initialData, setInitialData] = useState<WebhookConfiguration | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchInitialData = useCallback(async (currentWebhookId: string) => {
    if (!currentUser?.organization_id) return;
    setIsLoading(true);
    setError(null);
    try {
      const response = await apiClient.get<WebhookConfiguration>(
        `/organizations/${currentUser.organization_id}/webhooks/${currentWebhookId}`
      );
      setInitialData(response.data);
    } catch (err: any) {
      console.error(t('form.error_loading_webhook_data_console'), err);
      setError(err.response?.data?.error || t('common:error_loading_data_entity', { entity_name: t('common_item_names.webhook', {ns: 'common'}) }));
    } finally {
      setIsLoading(false);
    }
  }, [currentUser?.organization_id, t]);

  useEffect(() => {
    if (router.isReady && webhookId && typeof webhookId === 'string' && !authLoading && currentUser?.organization_id) {
      fetchInitialData(webhookId);
    } else if (router.isReady && !webhookId) {
        setError(t('common:error_invalid_id', { entity_name: t('common_item_names.webhook', {ns: 'common'}) }));
        setIsLoading(false);
    }
  }, [webhookId, router.isReady, fetchInitialData, authLoading, currentUser?.organization_id, t]);

  const handleSuccess = () => {
    router.push('/admin/organization/webhooks');
  };

  const pageTitleBase = t('form.edit_page_title');
  const appName = t('common:app_name');
  const dynamicPageTitle = initialData?.name ? `${pageTitleBase}: ${initialData.name} - ${appName}` : `${pageTitleBase} - ${appName}`;

  if (isLoading || authLoading) {
    return (
      <AdminLayout title={t('common:loading_ellipsis')}>
        <div className="text-center p-10">{t('common:loading_data_entity', { entity_name: t('common_item_names.webhook', {ns: 'common'})})}</div>
      </AdminLayout>
    );
  }

  if (error) {
    return <AdminLayout title={t('common:error_page_title')}><div className="text-center p-10 text-red-500">{error}</div></AdminLayout>;
  }

  if (!initialData) {
     return <AdminLayout title={t('common:error_not_found_title')}><div className="text-center p-10">{t('common:error_entity_not_found', {entity_name: t('common_item_names.webhook', {ns: 'common'})})}</div></AdminLayout>;
  }

  return (
    <AdminLayout title={dynamicPageTitle}>
      <Head>
        <title>{dynamicPageTitle}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            {pageTitleBase}: <span className="text-brand-primary dark:text-brand-primary">{initialData.name}</span>
          </h1>
          <Link href="/admin/organization/webhooks" legacyBehavior>
            <a className="text-brand-primary hover:text-brand-primary/80 dark:text-brand-primary dark:hover:text-brand-primary/70 transition-colors">
              &larr; {t('form.back_to_list_link')}
            </a>
          </Link>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          <WebhookForm
            organizationId={currentUser!.organization_id!}
            initialData={initialData}
            isEditing={true}
            onSubmitSuccess={handleSuccess}
          />
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(EditWebhookPageContent);
