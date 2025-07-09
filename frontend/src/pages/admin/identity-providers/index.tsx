import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import { useState, useEffect, useCallback } from 'react';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';
import IdentityProviderForm from '@/components/admin/IdentityProviderForm'; // Importar o formulário
import {
    IdentityProvider,
    // IdentityProviderType // IdentityProvider já inclui provider_type, então IdentityProviderType não é diretamente usado aqui.
                           // Mas o tipo IdentityProvider de @/types usa o enum IdentityProviderType.
} from '@/types';

// Definições de tipos locais (IdentityProviderType, IdentityProviderAPIResponse, IdentityProvider) removidas


// Anteriormente 'IdentityProvidersPage', agora 'IdentityProvidersPageContent' para o HOC
const IdentityProvidersPageContent = () => {
  const { user, isLoading: authIsLoading } = useAuth();
  const [identityProviders, setIdentityProviders] = useState<IdentityProvider[]>([]); // Usa IdentityProvider de @/types
  const [isLoading, setIsLoading] = useState(true); // Loading da lista de IdPs
  const [error, setError] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [editingProvider, setEditingProvider] = useState<IdentityProvider | null>(null);

  const fetchIdentityProviders = useCallback(async () => {
    if (!user?.organization_id || authIsLoading) {
      // Se authIsLoading é true, user pode ser null temporariamente.
      // Se !authIsLoading e user ainda é null ou não tem organization_id, então há um problema.
      if (!authIsLoading && !user?.organization_id) {
          setError("Organização do usuário não encontrada para buscar provedores.");
          setIsLoading(false);
      }
      return;
    }
    setIsLoading(true);
    setError(null);
    try {
      const response = await apiClient.get<IdentityProviderAPIResponse[]>(`/organizations/${user.organization_id}/identity-providers`);
      const parsedProviders = (response.data || []).map(p => ({
        ...p,
        config_json_parsed: JSON.parse(p.config_json || '{}'),
        attribute_mapping_json_parsed: p.attribute_mapping_json ? JSON.parse(p.attribute_mapping_json) : undefined,
      }));
      setIdentityProviders(parsedProviders);
    } catch (err: any) {
      console.error("Erro ao buscar provedores de identidade:", err);
      setError(err.response?.data?.error || err.message || "Falha ao buscar provedores de identidade.");
      setIdentityProviders([]);
    } finally {
      setIsLoading(false);
    }
  }, [user?.organization_id, authIsLoading]);

  useEffect(() => {
    // Apenas busca se o auth não estiver carregando e o usuário estiver disponível
    if (!authIsLoading && user) {
        fetchIdentityProviders();
    } else if (!authIsLoading && !user) {
        // Se o auth carregou e não há usuário, provavelmente não está autenticado
        setError("Usuário não autenticado.");
        setIsLoading(false);
    }
    // Não adicionar fetchIdentityProviders diretamente aqui se ele depende de 'user' que está no mesmo escopo
    // A chamada está dentro de um if que já verifica 'user'
  }, [user, authIsLoading, fetchIdentityProviders]);


  const handleAddNewProvider = () => {
    setEditingProvider(null); // Limpar formulário para novo provedor
    setShowModal(true);
  };

  const handleEditProvider = (provider: IdentityProvider) => {
    setEditingProvider(provider);
    setShowModal(true);
  };

  const handleDeleteProvider = async (providerId: string) => {
    // Obter o nome para a mensagem de confirmação
    const providerToDelete = identityProviders.find(p => p.id === providerId);
    const providerName = providerToDelete ? providerToDelete.name : `ID ${providerId.substring(0,8)}`;

    if (window.confirm(`Tem certeza que deseja remover o provedor de identidade "${providerName}"? Esta ação não pode ser desfeita.`)) {
      setIsLoading(true); // Pode-se usar um estado de loading específico para a deleção
      setError(null);
      try {
        await apiClient.delete(`/organizations/${user?.organization_id}/identity-providers/${providerId}`);
        // alert(`Provedor "${providerName}" deletado com sucesso.`); // Usar notificação melhor no futuro
        fetchIdentityProviders(); // Re-busca a lista para refletir a remoção
      } catch (err: any) {
        console.error("Erro ao deletar provedor de identidade:", err);
        setError(err.response?.data?.error || err.message || "Falha ao deletar provedor de identidade.");
      } finally {
        setIsLoading(false);
      }
    }
  };

  const handleSaveProvider = () => { // Não é mais um handler de evento de form, mas um callback de sucesso
    fetchIdentityProviders(); // Re-busca a lista para refletir as mudanças
    setShowModal(false);
    setEditingProvider(null);
    // O IdentityProviderForm já lida com o alerta de sucesso ou erro da submissão.
  };

  return (
    <AdminLayout title="Provedores de Identidade - Admin Phoenix GRC">
      <div className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold text-gray-800 dark:text-white">
            Gerenciar Provedores de Identidade (SSO/Social Login)
          </h1>
          <button
            onClick={handleAddNewProvider}
            className="bg-indigo-600 hover:bg-indigo-700 text-white font-bold py-2 px-4 rounded-md shadow-sm transition duration-150 ease-in-out"
          >
            Adicionar Novo Provedor
          </button>
        </div>

        {isLoading && <p className="text-center text-gray-500 dark:text-gray-400 py-4">Carregando provedores...</p>}
        {error && <p className="text-center text-red-500 py-4">Erro ao carregar provedores: {error}</p>}

        {!isLoading && !error && (
          <>
            {/* Tabela de Provedores de Identidade */}
            <div className="bg-white dark:bg-gray-800 shadow-md rounded-lg overflow-x-auto mt-6">
              <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-700">
                  <tr>
                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Nome</th>
                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Tipo</th>
                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Status</th>
                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Ações</th>
                  </tr>
                </thead>
                <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                  {identityProviders.length === 0 && (
                    <tr><td colSpan={4} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">Nenhum provedor de identidade configurado.</td></tr>
                  )}
                  {identityProviders.map((provider) => (
                    <tr key={provider.id}>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">{provider.name}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">{provider.provider_type}</td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                          provider.is_active ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' : 'bg-red-100 text-red-800 dark:bg-red-700 dark:text-red-100'
                        }`}>
                          {provider.is_active ? 'Ativo' : 'Inativo'}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                        <button onClick={() => handleEditProvider(provider)} className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-200">Editar</button>
                        <button onClick={() => handleDeleteProvider(provider.id)} className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-200">Deletar</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
      </div>

      {/* Modal para Adicionar/Editar Provedor */}
      {showModal && (
        // O modal em si não precisa mudar, apenas o conteúdo dele que agora é o IdentityProviderForm
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
            <IdentityProviderForm
              initialData={editingProvider || undefined} // Passar undefined se não estiver editando
              isEditing={!!editingProvider}
              onClose={() => {
                setShowModal(false);
                setEditingProvider(null);
              }}
              onSubmitSuccess={handleSaveProvider} // Reutiliza handleSaveProvider para re-fetch
            />
          </div>
        </div>
      )}
    </AdminLayout>
  );
}

// Envolver o componente da página com WithAuth
const IdentityProvidersPageWithAuth = WithAuth(IdentityProvidersPage);

export default IdentityProvidersPageWithAuth;
