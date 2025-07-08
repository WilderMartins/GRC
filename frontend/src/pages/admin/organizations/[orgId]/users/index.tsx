import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import { useRouter } from 'next/router';
import { useAuth } from '@/contexts/AuthContext';
import { useEffect, useState, useCallback } from 'react';
import apiClient from '@/lib/axios';
import Link from 'next/link'; // Para futuros botões de "Convidar Usuário"

// Tipos (idealmente de um arquivo compartilhado/gerado)
type UserRole = "admin" | "manager" | "user";
interface User {
  id: string;
  name: string;
  email: string;
  role: UserRole;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}
interface PaginatedUsersResponse {
  items: User[];
  total_items: number;
  total_pages: number;
  page: number;
  page_size: number;
}

// --- Componente para o Modal de Edição de Role ---
interface EditRoleModalProps {
  userToEdit: User | null;
  onClose: () => void;
  onSuccess: () => void; // Para re-fetch da lista
  organizationId: string;
}

const EditRoleModal: React.FC<EditRoleModalProps> = ({ userToEdit, onClose, onSuccess, organizationId }) => {
    const [newRole, setNewRole] = useState<UserRole | ''>(userToEdit?.role || '');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    if (!userToEdit) return null;

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!newRole) {
            setError("Selecione uma nova role.");
            return;
        }
        setIsLoading(true);
        setError(null);
        try {
            await apiClient.put(`/organizations/${organizationId}/users/${userToEdit.id}/role`, { role: newRole });
            onSuccess();
            onClose();
        } catch (err: any) {
            setError(err.response?.data?.error || "Falha ao atualizar role.");
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm">
            <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-md">
                <h3 className="text-lg font-medium mb-4 dark:text-white">Alterar Role para {userToEdit.name}</h3>
                {error && <p className="text-red-500 text-sm mb-2">{error}</p>}
                <form onSubmit={handleSubmit}>
                    <select value={newRole} onChange={(e) => setNewRole(e.target.value as UserRole)}
                            className="w-full p-2 border rounded-md dark:bg-gray-700 dark:border-gray-600 dark:text-white">
                        <option value="" disabled>Selecione uma role</option>
                        <option value="user">User</option>
                        <option value="manager">Manager</option>
                        <option value="admin">Admin</option>
                    </select>
                    <div className="mt-4 flex justify-end space-x-2">
                        <button type="button" onClick={onClose} disabled={isLoading}
                                className="px-4 py-2 text-sm rounded-md text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-600 hover:bg-gray-200 dark:hover:bg-gray-500">Cancelar</button>
                        <button type="submit" disabled={isLoading || !newRole}
                                className="px-4 py-2 text-sm rounded-md text-white bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50">
                            {isLoading ? "Salvando..." : "Salvar Role"}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};


const OrgUsersPageContent = () => {
  const router = useRouter();
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
    if (!canAccess || !orgId || typeof orgId !== 'string') return;
    setIsLoadingData(true);
    setDataError(null);
    try {
      const response = await apiClient.get<PaginatedUsersResponse>(`/organizations/${orgId}/users`, {
        params: { page, page_size: size },
      });
      setUsers(response.data.items || []);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
      setCurrentPage(response.data.page);
      // setPageSize(response.data.page_size); // PageSize é controlado pelo estado local
    } catch (err: any) {
      setDataError(err.response?.data?.error || "Falha ao buscar usuários.");
      setUsers([]);
    } finally {
      setIsLoadingData(false);
    }
  }, [orgId, canAccess, pageSize]); // pageSize é dependência para re-fetch se ele mudar

  useEffect(() => {
    if (authIsLoading) return;
    if (!actingUser) { setPageError("Usuário não autenticado."); setCanAccess(false); return; }
    if (actingUser.organization_id !== orgId) {
      setPageError("Você não tem permissão para gerenciar usuários desta organização.");
      setCanAccess(false); return;
    }
    if (actingUser.role !== 'admin' && actingUser.role !== 'manager') {
      setPageError("Você não tem privilégios suficientes (requer Admin ou Manager).");
      setCanAccess(false); return;
    }
    setCanAccess(true);
    setPageError(null);
    fetchUsers(currentPage, pageSize);
  }, [orgId, actingUser, authIsLoading, fetchUsers, currentPage, pageSize]);


  const handleToggleUserStatus = async (userToToggle: User) => {
    if (!orgId || typeof orgId !== 'string') return;
    const newStatus = !userToToggle.is_active;
    if (window.confirm(`Tem certeza que deseja ${newStatus ? "ativar" : "desativar"} o usuário ${userToToggle.name}?`)) {
      try {
        await apiClient.put(`/organizations/${orgId}/users/${userToToggle.id}/status`, { is_active: newStatus });
        fetchUsers(currentPage, pageSize); // Re-fetch
      } catch (err: any) {
        alert(`Falha ao atualizar status: ${err.response?.data?.error || err.message}`);
      }
    }
  };

  const openEditRoleModal = (user: User) => {
    setUserToEditRole(user);
    setShowEditRoleModal(true);
  };


  if (authIsLoading || (!canAccess && !pageError) ) { // Adicionado !pageError para estado inicial antes do useEffect de acesso rodar
    return <AdminLayout title="Carregando..."><div className="p-6 text-center">Verificando permissões...</div></AdminLayout>;
  }
  if (!canAccess && pageError) {
    return <AdminLayout title="Acesso Negado"><div className="p-6 text-center text-red-500">{pageError}</div></AdminLayout>;
  }

  return (
    <AdminLayout title={`Gerenciar Usuários - Organização`}>
      <Head><title>Gerenciar Usuários - Phoenix GRC</title></Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="sm:flex sm:items-center sm:justify-between mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">Gerenciar Usuários</h1>
          {/* TODO: Botão Convidar Usuário */}
        </div>

        {isLoadingData && <p className="text-center">Carregando usuários...</p>}
        {dataError && <p className="text-center text-red-500">Erro: {dataError}</p>}
        {!isLoadingData && !dataError && (
          <>
            <div className="mt-8 flow-root">
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                    <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                            <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">Nome</th>
                            <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Email</th>
                            <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Role</th>
                            <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Status</th>
                            <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">Ações</span></th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                        {users.map((userItem) => (
                            <tr key={userItem.id}>
                                <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{userItem.name}</td>
                                <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{userItem.email}</td>
                                <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{userItem.role}</td>
                                <td className="whitespace-nowrap px-3 py-4 text-sm">
                                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                        userItem.is_active ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100'
                                                        : 'bg-red-100 text-red-800 dark:bg-red-700 dark:text-red-100'}`}>
                                        {userItem.is_active ? 'Ativo' : 'Inativo'}
                                    </span>
                                </td>
                                <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6 space-x-2">
                                    <button onClick={() => openEditRoleModal(userItem)} className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-200">Alterar Role</button>
                                    <button onClick={() => handleToggleUserStatus(userItem)} className={`font-medium ${userItem.is_active ? 'text-yellow-600 hover:text-yellow-900 dark:text-yellow-400 dark:hover:text-yellow-200' : 'text-green-600 hover:text-green-900 dark:text-green-400 dark:hover:text-green-200'}`}>
                                        {userItem.is_active ? 'Desativar' : 'Ativar'}
                                    </button>
                                </td>
                            </tr>
                        ))}
                        {users.length === 0 && (
                             <tr><td colSpan={5} className="text-center py-4 px-6 text-sm text-gray-500 dark:text-gray-400">Nenhum usuário encontrado nesta organização.</td></tr>
                        )}
                    </tbody>
                    </table>
                  </div>
                  {/* TODO: Controles de Paginação */}
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
        />
      )}
    </AdminLayout>
  );
};

export default WithAuth(OrgUsersPageContent);
