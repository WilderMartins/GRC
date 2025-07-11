import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import WebhookForm from '@/components/admin/organization/WebhookForm';
import { useRouter } from 'next/router';
import { useAuth } from '@/contexts/AuthContext';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'organizationSettings', 'webhooks'])),
  },
});

const NewWebhookPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['webhooks', 'common']);
  const router = useRouter();
  const { user: currentUser, isLoading: authLoading } = useAuth();

  const handleSuccess = () => {
    // Notificação de sucesso já é tratada pelo WebhookForm
    router.push('/admin/organization/webhooks');
  };

  const pageTitle = t('form.add_page_title'); // Ex: "Adicionar Novo Webhook"
  const appName = t('common:app_name');

  if (authLoading || !currentUser?.organization_id) {
    return (
      <AdminLayout title={t('common:loading_ellipsis')}>
        <div className="text-center p-10">{t('common:loading_ellipsis')}</div>
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
            {pageTitle}
          </h1>
          <Link href="/admin/organization/webhooks" legacyBehavior>
            <a className="text-brand-primary hover:text-brand-primary/80 dark:text-brand-primary dark:hover:text-brand-primary/70 transition-colors">
              &larr; {t('form.back_to_list_link')}
            </a>
          </Link>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          <WebhookForm
            organizationId={currentUser.organization_id}
            onSubmitSuccess={handleSuccess}
            isEditing={false}
          />
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(NewWebhookPageContent);
