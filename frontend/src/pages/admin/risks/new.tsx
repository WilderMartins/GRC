import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import RiskForm from '@/components/risks/RiskForm'; // Importar o formulário
import { useRouter } from 'next/router';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'risks'])),
  },
});

const NewRiskPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['risks', 'common']);
  const router = useRouter();
  // const notify = useNotifier(); // Descomentar se for usar notify.success aqui

  const handleSuccess = () => {
    // notify.success(t('create_page.success_notification', { ns: 'risks' })); // Exemplo com i18n
    alert(t('create_page.success_alert_placeholder', { ns: 'risks' })); // Placeholder com i18n
    router.push('/admin/risks');
  };

  const pageTitle = t('create_page.page_title', { ns: 'risks' });
  const appName = t('common:app_name');

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
          <Link href="/admin/risks" legacyBehavior>
            <a className="text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70">
              &larr; {t('common:back_to_list_link_generic', { list_name: t('list.page_title_plural', { ns: 'risks' }) })}
            </a>
          </Link>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          <RiskForm onSubmitSuccess={handleSuccess} isEditing={false} />
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(NewRiskPageContent);
