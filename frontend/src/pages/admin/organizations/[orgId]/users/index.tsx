import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import { useRouter } from 'next/router';
import { useAuth } from '@/contexts/AuthContext';
import { useEffect, useState, useCallback } from 'react';
import apiClient from '@/lib/axios';
// import Link from 'next/link'; // Removido, pois o botão "Convidar Usuário" é TODO
import PaginationControls from '@/components/common/PaginationControls';
import { User, UserRole, PaginatedResponse } from '@/types';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetServerSideProps, InferGetServerSidePropsType } from 'next';
import { useNotifier } from '@/hooks/useNotifier';


type Props = {
  // Props from getServerSideProps
}

export const getServerSideProps: GetServerSideProps<Props> = async ({ locale }) => {
  return {
    props: {
      ...(await serverSideTranslations(locale ?? 'pt', ['common', 'orgUsers'])),
    },
  };
};

// --- Componente para o Modal de Edição de Role ---
interface EditRoleModalProps {
  userToEdit: User | null;
  onClose: () => void;
  onSuccess: () => void;
  organizationId: string;
  // Passar 't' como prop ou chamar useTranslation dentro do modal
  t: (key: string, options?: any) => string;
}

const EditRoleModal: React.FC<EditRoleModalProps> = ({ userToEdit, onClose, onSuccess, organizationId, t }) => {
    const notify = useNotifier(); // Notifier para erros da API no modal
    const [newRole, setNewRole] = useState<UserRole | ''>(userToEdit?.role as UserRole || '');
    const [isLoadingModal, setIsLoadingModal] = useState(false); // Renomeado para evitar conflito
    const [modalError, setModalError] = useState<string | null>(null); // Renomeado

    if (!userToEdit) return null;

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!newRole) {
            setModalError(t('edit_role_modal.error_select_role'));
            return;
        }
        setIsLoadingModal(true);
        setModalError(null);
        try {
            await apiClient.put(`/organizations/${organizationId}/users/${userToEdit.id}/role`, { role: newRole });
            notify.success(t('common:update_success_generic'));
            onSuccess();
            onClose();
        } catch (err: any) {
            const apiError = err.response?.data?.error || t('common:unknown_error');
            notify.error(t('edit_role_modal.error_update_failed', { message: apiError }));
            setModalError(apiError); // Opcional: mostrar também no modal
        } finally {
            setIsLoadingModal(false);
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm">
            <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-md">
                <h3 className="text-lg font-medium mb-4 dark:text-white">{t('edit_role_modal.title_prefix')} {userToEdit.name}</h3>
                {modalError && <p className="text-red-500 text-sm mb-2">{modalError}</p>}
                <form onSubmit={handleSubmit}>
                    <select value={newRole} onChange={(e) => setNewRole(e.target.value as UserRole)}
                            className="w-full p-2 border rounded-md dark:bg-gray-700 dark:border-gray-600 dark:text-white">
                        <option value="" disabled>{t('edit_role_modal.select_role_label')}</option>
                        <option value="user">{t('edit_role_modal.option_user')}</option>
                        <option value="manager">{t('edit_role_modal.option_manager')}</option>
                        <option value="admin">{t('edit_role_modal.option_admin')}</option>
                    </select>
                    <div className="mt-4 flex justify-end space-x-2">
                        <button type="button" onClick={onClose} disabled={isLoadingModal}
                                className="px-4 py-2 text-sm rounded-md text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-600 hover:bg-gray-200 dark:hover:bg-gray-500">
                                    {t('common:cancel_button')}
                        </button>
                        <button type="submit" disabled={isLoadingModal || !newRole}
                                className="px-4 py-2 text-sm rounded-md text-white bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50">
                            {isLoadingModal ? t('common:saving_button') : t('edit_role_modal.save_role_button')}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};


const OrgUsersPageContent = (props: InferGetServerSidePropsType<typeof getServerSideProps>) => {
  const { t } = useTranslation(['orgUsers', 'common']);
  const router = useRouter();
  const notify = useNotifier();
  const { orgId } = router.query;
  const { user: actingUser, isLoading: authIsLoading } = useAuth();

  const [canAccess, setCanAccess] = useState(false);
  const [pageError, setPageError] = useState<string | null>(null);

  const [users, setUsers] = useState<User[]>([]);
  const [isLoadingData, setIsLoadingData] = useState(true);
  const [dataError, setDataError] = useState<string | null>(null);

  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);

  const [showEditRoleModal, setShowEditRoleModal] = useState(false);
  const [userToEditRole, setUserToEditRole] = useState<User | null>(null);

  const fetchUsers = useCallback(async (page: number, size: number) => {
    if (!canAccess || !orgId || typeof orgId !== 'string') {
        setIsLoadingData(false);
        return;
    }
    setIsLoadingData(true);
    setDataError(null);
    try {
      const response = await apiClient.get<PaginatedResponse<User>>(`/organizations/${orgId}/users`, {
        params: { page, page_size: size },
      });
      setUsers(response.data.items || []);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
      setCurrentPage(response.data.page);
    } catch (err: any) {
      const apiError = err.response?.data?.error || t('common:unknown_error');
      setDataError(t('list.error_loading_users', { message: apiError }));
      setUsers([]);
    } finally {
      setIsLoadingData(false);
    }
  }, [orgId, canAccess, t]);

  useEffect(() => {
    if (authIsLoading || !router.isReady) return;

    if (!actingUser) {
      setPageError(t('common:error_unauthenticated'));
      setCanAccess(false);
      setIsLoadingData(false);
      return;
    }
    if (typeof orgId === 'string') {
        if (actingUser.organization_id !== orgId) {
            setPageError(t('list.access_denied_message'));
            setCanAccess(false);
            setIsLoadingData(false);
            return;
        }
        if (actingUser.role !== 'admin' && actingUser.role !== 'manager') {
            setPageError(t('list.insufficient_privileges_message'));
            setCanAccess(false);
            setIsLoadingData(false);
            return;
        }
        setCanAccess(true);
        setPageError(null);
    } else { // orgId is not ready or not a string
        setPageError(t('common:error_invalid_org_id'));
        setCanAccess(false);
        setIsLoadingData(false);
    }
  }, [orgId, actingUser, authIsLoading, router.isReady, t]);

  useEffect(() => {
    if (canAccess && typeof orgId === 'string') {
      fetchUsers(currentPage, pageSize);
    } else if (!authIsLoading && !actingUser) {
        setIsLoadingData(false);
    }
  }, [canAccess, currentPage, pageSize, fetchUsers, orgId, authIsLoading, actingUser]);

  const handleToggleUserStatus = async (userToToggle: User) => {
    if (!orgId || typeof orgId !== 'string') return;
    const newStatus = !userToToggle.is_active;
    const confirmMessage = newStatus
        ? t('confirmations.activate_user_message', { userName: userToToggle.name })
        : t('confirmations.deactivate_user_message', { userName: userToToggle.name });

    if (window.confirm(confirmMessage)) {
      try {
        setIsLoadingData(true); // Indicate general loading as it affects the list
        await apiClient.put(`/organizations/${orgId}/users/${userToToggle.id}/status`, { is_active: newStatus });
        notify.success(t('common:update_success_generic'));
        fetchUsers(currentPage, pageSize);
      } catch (err: any) {
        const apiError = err.response?.data?.error || t('common:unknown_error');
        notify.error(t('confirmations.update_status_failure_alert', { message: apiError }));
        setIsLoadingData(false); // Reset loading on error if fetchUsers isn't called
      }
    }
  };

  const openEditRoleModal = (user: User) => {
    setUserToEditRole(user);
    setShowEditRoleModal(true);
  };

  const handlePageChange = (newPage: number) => {
    if (newPage >= 1 && newPage <= totalPages && newPage !== currentPage) {
      setCurrentPage(newPage);
    }
  };

  const pageTitle = t('list.page_title');
  const appName = t('common:app_name');

  if (authIsLoading || (!router.isReady && !pageError && !canAccess)) {
    return <AdminLayout title={t('common:loading_ellipsis')}><div className="p-6 text-center">{t('list.loading_permissions')}</div></AdminLayout>;
  }
  if (!canAccess && pageError) {
    return <AdminLayout title={t('common:access_denied')}><div className="p-6 text-center text-red-500">{pageError}</div></AdminLayout>;
  }
  if (!canAccess && !pageError && !authIsLoading) { // Should be caught by above, but as a fallback
      return <AdminLayout title={t('common:loading_ellipsis')}><div className="p-6 text-center">{t('list.loading_data_org')}</div></AdminLayout>;
  }

  return (
    <AdminLayout title={`${pageTitle} - ${appName}`}>
      <Head><title>{`${pageTitle} - ${appName}`}</title></Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="sm:flex sm:items-center sm:justify-between mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">{t('list.header')}</h1>
          {/* <Link href={`/admin/organizations/${orgId}/users/invite`} legacyBehavior> // TODO: Implement invite user
            <a className="inline-flex items-center justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800">
              {t('list.invite_user_button')}
            </a>
          </Link> */}
        </div>

        {isLoadingData && users.length === 0 && <p className="text-center py-4">{t('list.loading_users')}</p>}
        {dataError && <p className="text-center text-red-500 py-4">{dataError}</p>}

        {!isLoadingData && users.length === 0 && !dataError && (
            <div className="text-center py-10">
                <p className="text-gray-500 dark:text-gray-400">{t('list.no_users_found')}</p>
                {/* TODO: Adicionar botão "Convidar primeiro usuário" se apropriado */}
            </div>
        )}

        {users.length > 0 && !dataError && (
          <>
            <div className="mt-8 flow-root">
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                    <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                            <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">{t('table.header_name')}</th>
                            <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('table.header_email')}</th>
                            <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('table.header_role')}</th>
                            <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('table.header_status')}</th>
                            <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">{t('table.header_actions')}</span></th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                        {users.map((userItem) => (
                            <tr key={userItem.id}>
                                <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{userItem.name}</td>
                                <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{userItem.email}</td>
                                <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{t(`edit_role_modal.option_${userItem.role.toLowerCase()}`, {defaultValue: userItem.role})}</td>
                                <td className="whitespace-nowrap px-3 py-4 text-sm">
                                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                        userItem.is_active ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100'
                                                        : 'bg-red-100 text-red-800 dark:bg-red-700 dark:text-red-100'}`}>
                                        {userItem.is_active ? t('table.status_active') : t('table.status_inactive')}
                                    </span>
                                </td>
                                <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6 space-x-2">
                                    <button onClick={() => openEditRoleModal(userItem)} className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-200">{t('table.action_change_role')}</button>
                                    <button
                                        onClick={() => handleToggleUserStatus(userItem)}
                                        className={`font-medium ${userItem.is_active ?
                                            'text-yellow-600 hover:text-yellow-900 dark:text-yellow-400 dark:hover:text-yellow-200'
                                            : 'text-green-600 hover:text-green-900 dark:text-green-400 dark:hover:text-green-200'}`}
                                        disabled={isLoadingData && users.length > 0}
                                    >
                                        {userItem.is_active ? t('table.action_deactivate') : t('table.action_activate')}
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
                    isLoading={isLoadingData && users.length > 0}
                  />
                </div>
              </div>
            </div>
          </>
        )}
      </div>
      {showEditRoleModal && userToEditRole && orgId && typeof orgId === 'string' && (
        <EditRoleModal
            userToEdit={userToEditRole}
            organizationId={orgId}
            onClose={() => {setShowEditRoleModal(false); setUserToEditRole(null);}}
            onSuccess={() => {setShowEditRoleModal(false); setUserToEditRole(null); fetchUsers(currentPage, pageSize);}}
            t={t} // Passar a função t para o modal
        />
      )}
    </AdminLayout>
  );
};

export default WithAuth(OrgUsersPageContent);
