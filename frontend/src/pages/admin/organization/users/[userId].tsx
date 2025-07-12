import { useRouter } from 'next/router';
import Head from 'next/head';
import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import { useAuth } from '@/contexts/AuthContext';
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios';
import { User } from '@/types'; // Usando o tipo User principal
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetServerSideProps, InferGetServerSidePropsType } from 'next';
import Link from 'next/link';

// Definindo UserProfile como User por enquanto, pode ser expandido se a API retornar mais detalhes
type UserProfile = User;

type Props = {
    // Props from getServerSideProps
}

export const getServerSideProps: GetServerSideProps<Props> = async ({ locale }) => ({
    props: {
        ...(await serverSideTranslations(locale ?? 'pt', ['common', 'usersManagement'])),
    },
});

const UserProfilePageContent = (props: InferGetServerSidePropsType<typeof getServerSideProps>) => {
    const { t } = useTranslation(['usersManagement', 'common']);
    const router = useRouter();
    const { userId } = router.query;
    const { user: currentUser, isLoading: authLoading } = useAuth();

    const [userProfile, setUserProfile] = useState<UserProfile | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (userId && currentUser?.organization_id && !authLoading) {
            setIsLoading(true);
            setError(null);
            apiClient.get<UserProfile>(`/api/v1/organizations/${currentUser.organization_id}/users/${userId}`)
                .then(response => {
                    setUserProfile(response.data);
                })
                .catch(err => {
                    console.error("Erro ao buscar perfil do usuário:", err);
                    setError(err.response?.data?.error || t('view_user.error_loading_profile'));
                })
                .finally(() => {
                    setIsLoading(false);
                });
        } else if (!authLoading && (!userId || !currentUser?.organization_id)) {
            // Se não houver userId ou organizationId após o carregamento do auth, é um estado inválido para esta página.
            setIsLoading(false);
            setError(t('view_user.error_missing_params'));
        }
    }, [userId, currentUser?.organization_id, authLoading, t]);

    const pageTitle = userProfile ? t('view_user.page_title_with_name', { userName: userProfile.name }) : t('view_user.page_title');
    const appName = t('common:app_name');


    if (isLoading || authLoading) {
        return (
            <AdminLayout title={t('common:loading_ellipsis')}>
                <div className="p-6 text-center">{t('common:loading_ellipsis')}</div>
            </AdminLayout>
        );
    }

    if (error) {
        return (
            <AdminLayout title={t('view_user.error_page_title')}>
                <div className="p-6 text-center text-red-500">
                    <p>{error}</p>
                    <Link href="/admin/organization/users" legacyBehavior>
                        <a className="mt-4 inline-block text-brand-primary hover:underline">
                            {t('view_user.back_to_list_link')}
                        </a>
                    </Link>
                </div>
            </AdminLayout>
        );
    }

    if (!userProfile) {
        return (
            <AdminLayout title={t('view_user.error_user_not_found_title')}>
                <div className="p-6 text-center">
                    <p>{t('view_user.error_user_not_found')}</p>
                    <Link href="/admin/organization/users" legacyBehavior>
                        <a className="mt-4 inline-block text-brand-primary hover:underline">
                            {t('view_user.back_to_list_link')}
                        </a>
                    </Link>
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
                <div className="mb-6">
                    <Link href="/admin/organization/users" legacyBehavior>
                        <a className="text-sm text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70">
                            &larr; {t('view_user.back_to_list_link')}
                        </a>
                    </Link>
                </div>
                <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white mb-8">
                    {pageTitle}
                </h1>

                <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
                    <div className="px-4 py-5 sm:px-6">
                        <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">
                            {t('view_user.section_user_details_title')}
                        </h3>
                        <p className="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">
                            {t('view_user.section_user_details_subtitle')}
                        </p>
                    </div>
                    <div className="border-t border-gray-200 dark:border-gray-700 px-4 py-5 sm:p-0">
                        <dl className="sm:divide-y sm:divide-gray-200 dark:sm:divide-gray-700">
                            <div className="py-3 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_user.field_name')}</dt>
                                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">{userProfile.name}</dd>
                            </div>
                            <div className="py-3 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_user.field_email')}</dt>
                                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">{userProfile.email}</dd>
                            </div>
                            <div className="py-3 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_user.field_role')}</dt>
                                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">{userProfile.role}</dd>
                            </div>
                            <div className="py-3 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_user.field_status')}</dt>
                                <dd className="mt-1 text-sm sm:mt-0 sm:col-span-2">
                                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                        userProfile.is_active ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' : 'bg-red-100 text-red-800 dark:bg-red-700 dark:text-red-100'
                                    }`}>
                                        {userProfile.is_active ? t('users_list.status_active') : t('users_list.status_inactive')}
                                    </span>
                                </dd>
                            </div>
                             <div className="py-3 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_user.field_id')}</dt>
                                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">{userProfile.id}</dd>
                            </div>
                            <div className="py-3 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_user.field_org_id')}</dt>
                                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">{userProfile.organization_id}</dd>
                            </div>
                            <div className="py-3 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('view_user.field_2fa_status')}</dt>
                                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">
                                    {userProfile.is_totp_enabled ? t('common:yes') : t('common:no')}
                                </dd>
                            </div>
                            {/* Adicionar mais campos conforme necessário, ex: CreatedAt, UpdatedAt se vierem da API e forem úteis */}
                        </dl>
                    </div>
                </div>
                {/* Adicionar botões de ação aqui se a edição for feita nesta página no futuro */}
                {/* Ex: Editar Role, Ativar/Desativar (se não for feito via modal na lista) */}
            </div>
        </AdminLayout>
    );
};

export default WithAuth(UserProfilePageContent);
