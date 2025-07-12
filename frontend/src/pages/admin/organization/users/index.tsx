import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import { useState, useEffect, useCallback } from 'react';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';
import { useNotifier } from '@/hooks/useNotifier';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import { User, PaginatedResponse, UserRole } from '@/types';
import PaginationControls from '@/components/common/PaginationControls';
import EditUserRoleModal from '@/components/admin/organization/EditUserRoleModal';
import InviteUserModal from '@/components/admin/organization/InviteUserModal'; // Importar o modal

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'organizationSettings', 'usersManagement'])),
  },
});

// Usar User diretamente, já que não há campos adicionais específicos para esta listagem por enquanto.
// Se OrganizationUser fosse necessário, definiria aqui.

const OrganizationUsersPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['usersManagement', 'common', 'organizationSettings']);
  const { user: currentUser, isLoading: authLoading } = useAuth();
  const notify = useNotifier();

  const [users, setUsers] = useState<User[]>([]); // Alterado para User[]
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);

  // Estados para o modal de edição de role
  const [showEditRoleModal, setShowEditRoleModal] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [showInviteUserModal, setShowInviteUserModal] = useState(false); // Estado para o modal de convite

  const fetchUsers = useCallback(async () => {
    if (!currentUser?.organization_id) return;

    setIsLoading(true);
    setError(null);
    try {
      const params = { page: currentPage, page_size: pageSize };
      const response = await apiClient.get<PaginatedResponse<User>>( // Alterado para User
        `/api/v1/organizations/${currentUser.organization_id}/users`,
        { params }
      );
      setUsers(response.data.items || []);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
    } catch (err: any) {
      console.error("Erro ao buscar usuários da organização:", err);
      setError(err.response?.data?.error || t('common:error_loading_list_general', { list_name: t('users_list.title') }));
    } finally {
      setIsLoading(false);
    }
  }, [currentUser?.organization_id, currentPage, pageSize, t]);

  useEffect(() => {
    if (!authLoading && currentUser?.organization_id) {
      fetchUsers();
    }
  }, [authLoading, currentUser?.organization_id, fetchUsers]);

  const handlePageChange = (newPage: number) => {
    setCurrentPage(newPage);
  };

  // A lógica de handleUpdateUserRole será movida para o modal ou chamada por ele
  // Esta função pode ser simplificada para apenas re-fetch ou ser o callback onRoleUpdated
  const onRoleUpdateSuccess = () => {
    fetchUsers(); // Re-fetch a lista após a atualização
  };

  const handleUpdateUserStatus = async (userId: string, isActive: boolean) => {
    if (!currentUser?.organization_id) return;
    if (currentUser?.id === userId && !isActive) {
        notify.error(t('users_list.error_cannot_deactivate_self'));
        return;
    }
    // Adicionar um estado de loading específico para esta ação se desejado
    try {
      await apiClient.put(`/api/v1/organizations/${currentUser.organization_id}/users/${userId}/status`, { is_active: isActive });
      notify.success(isActive ? t('users_list.success_user_activated') : t('users_list.success_user_deactivated'));
      fetchUsers();
    } catch (err: any) {
      notify.error(t('users_list.error_updating_status', { message: err.response?.data?.error || t('common:unknown_error')}));
    }
  };

  const handleOpenEditRoleModal = (userToEdit: User) => {
    setEditingUser(userToEdit);
    setShowEditRoleModal(true);
  };

  const handleCloseEditRoleModal = () => {
    setEditingUser(null);
    setShowEditRoleModal(false);
  };

  const handleOpenInviteUserModal = () => {
    setShowInviteUserModal(true);
  };

  const handleCloseInviteUserModal = () => {
    setShowInviteUserModal(false);
  };

  const onUserInviteSuccess = () => {
    fetchUsers(); // Re-fetch a lista após convidar/criar usuário
  };

  const pageTitle = t('users_list.page_title');
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
            <button
              type="button"
              onClick={handleOpenInviteUserModal}
              className="inline-flex items-center justify-center rounded-md border border-transparent bg-brand-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 sm:w-auto"
            >
              {t('users_list.invite_user_button')}
            </button>
          </div>
        </div>

        {isLoading && <p className="text-center py-4">{t('common:loading_ellipsis')}</p>}
        {error && <p className="text-center text-red-500 py-4">{error}</p>}

        {!isLoading && !error && users.length === 0 && (
          <div className="text-center py-10">
            <p className="text-gray-500 dark:text-gray-400">{t('users_list.no_users_found')}</p>
          </div>
        )}

        {!isLoading && !error && users.length > 0 && (
          <>
            <div className="mt-8 flow-root">
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                      <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                          <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">{t('users_list.header_name')}</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('users_list.header_email')}</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('users_list.header_role')}</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('users_list.header_status')}</th>
                          <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">{t('users_list.header_actions')}</span></th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                        {users.map((person) => (
                          <tr key={person.id}>
                            <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{person.name}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{person.email}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{person.role}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm">
                              <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                person.is_active ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' : 'bg-red-100 text-red-800 dark:bg-red-700 dark:text-red-100'
                              }`}>
                                {person.is_active ? t('users_list.status_active') : t('users_list.status_inactive')}
                              </span>
                            </td>
                            <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6 space-x-2">
                              <button
                                onClick={() => handleOpenEditRoleModal(person)}
                                className="font-medium text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 rounded-sm"
                                title={t('users_list.action_edit_role_title')}
                                disabled={currentUser?.id === person.id} // Não permitir editar a própria role diretamente na lista (geralmente feito em perfil)
                              >
                                {t('users_list.action_edit_role', 'Editar Role')}
                              </button>
                              <button onClick={() => handleUpdateUserStatus(person.id, !person.is_active)}
                                      disabled={currentUser?.id === person.id && person.is_active}
                                      className={`font-medium ${person.is_active ? 'text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300' : 'text-green-600 hover:text-green-800 dark:text-green-400 dark:hover:text-green-300'} focus:outline-none focus:ring-2 ${person.is_active ? 'focus:ring-red-500' : 'focus:ring-green-500'} focus:ring-offset-2 rounded-sm disabled:opacity-50 disabled:cursor-not-allowed`}>
                                {person.is_active ? t('users_list.action_deactivate') : t('users_list.action_activate')}
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
        {/* Modais aqui */}
        {showEditRoleModal && editingUser && currentUser?.organization_id && (
          <EditUserRoleModal
            isOpen={showEditRoleModal}
            onClose={handleCloseEditRoleModal}
            user={editingUser}
            organizationId={currentUser.organization_id}
            onRoleUpdated={onRoleUpdateSuccess}
          />
        )}
        {showInviteUserModal && currentUser?.organization_id && (
          <InviteUserModal
            isOpen={showInviteUserModal}
            onClose={handleCloseInviteUserModal}
            organizationId={currentUser.organization_id}
            onUserInvited={onUserInviteSuccess}
          />
        )}
      </div>
    </AdminLayout>
  );
};

export default WithAuth(OrganizationUsersPageContent);
