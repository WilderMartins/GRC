import React, { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/router';
import apiClient from '@/lib/axios'; // Ajuste o path
import { useAuth } from '@/contexts/AuthContext'; // Ajuste o path
import { useNotifier } from '@/hooks/useNotifier'; // Para feedback
import {
    RiskStatus,
    RiskImpact,
    RiskProbability,
    RiskCategory,
    UserLookup
} from '@/types';

// Definições de tipos locais removidas

interface RiskFormData {
  title: string;
  description: string;
  category: RiskCategory; // Usar o tipo importado
  impact: RiskImpact | "";   // Usar o tipo importado
  probability: RiskProbability | ""; // Usar o tipo importado
  status: RiskStatus;     // Usar o tipo importado
  owner_id: string;
}

// UserLookup já foi importado de @/types

interface RiskFormProps {
  initialData?: RiskFormData & { id?: string }; // RiskFormData agora usa os tipos importados
  isEditing?: boolean;
  onSubmitSuccess?: () => void;
}

const RiskForm: React.FC<RiskFormProps> = ({ initialData, isEditing = false, onSubmitSuccess }) => {
  const router = useRouter();
  const { user, isLoading: authIsLoading } = useAuth();
  const notify = useNotifier();

  const [formData, setFormData] = useState<RiskFormData>({
    title: '',
    description: '',
    category: 'tecnologico',
    impact: '',
    probability: '',
    status: 'aberto',
    owner_id: '', // Será definido no useEffect ou pelo usuário
    ...(initialData || {}),
  });

  const [isLoading, setIsLoading] = useState(false); // Loading do submit do formulário
  const [formError, setFormError] = useState<string | null>(null); // Erros de validação do formulário

  const [organizationUsers, setOrganizationUsers] = useState<UserLookup[]>([]);
  const [isLoadingUsers, setIsLoadingUsers] = useState(true);
  const [usersError, setUsersError] = useState<string | null>(null);

  // Buscar usuários da organização para o select de Owner
  useEffect(() => {
    const fetchOrganizationUsers = async () => {
      if (!user || authIsLoading) return; // Precisa do usuário logado para o contexto da organização

      setIsLoadingUsers(true);
      setUsersError(null);
      try {
        // Usando o endpoint definido: GET /api/v1/users/organization-lookup
        // Este endpoint implicitamente usa a organização do usuário autenticado.
        const response = await apiClient.get<UserLookup[]>('/users/organization-lookup');
        setOrganizationUsers(response.data || []);

        // Se estiver criando um novo risco e o owner_id ainda não foi definido (ex: por initialData),
        // e o usuário logado estiver na lista, defina-o como padrão.
        if (!isEditing && user?.id && !formData.owner_id && response.data?.some(u => u.id === user.id)) {
            setFormData(prev => ({ ...prev, owner_id: user.id }));
        } else if (!isEditing && !formData.owner_id && response.data?.length > 0) {
            // Se não puder usar o usuário logado como padrão mas há usuários, não selecionar ninguém
            // ou selecionar o primeiro da lista, dependendo da preferência.
            // Por ora, deixaremos em branco para o usuário escolher.
        }

      } catch (err: any) {
        console.error("Erro ao buscar usuários da organização:", err);
        setUsersError("Falha ao carregar lista de proprietários.");
        // notify.error("Falha ao carregar lista de proprietários. Você pode precisar inserir o ID manualmente se souber.");
        setOrganizationUsers([]);
      } finally {
        setIsLoadingUsers(false);
      }
    };

    fetchOrganizationUsers();
  }, [user, authIsLoading, isEditing]); // Adicionado isEditing para reavaliar o owner_id padrão

 useEffect(() => {
    // Define o owner_id inicial com base no usuário logado ou initialData
    if (initialData) {
      setFormData(prev => ({ ...prev, ...initialData }));
    } else if (!isEditing && user && organizationUsers.length > 0) {
      // Se criando novo, e o usuário logado está na lista, use-o como padrão.
      // Se não, o primeiro da lista ou vazio.
      const loggedUserInList = organizationUsers.find(u => u.id === user.id);
      if (loggedUserInList) {
        setFormData(prev => ({ ...prev, owner_id: user.id }));
      } else if (organizationUsers.length > 0 && !prev.owner_id) {
        // setFormData(prev => ({ ...prev, owner_id: organizationUsers[0].id })); // Opcional: default para o primeiro
      }
    } else if (!isEditing && user && organizationUsers.length === 0 && !isLoadingUsers) {
        // Se criando, não há outros usuários, define o usuário logado como owner
        setFormData(prev => ({...prev, owner_id: user.id}));
    }
  }, [initialData, isEditing, user, organizationUsers, isLoadingUsers]);


  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsLoading(true);
    setFormError(null);

    if (!formData.impact || !formData.probability) {
        setFormError("Impacto e Probabilidade são obrigatórios.");
        setIsLoading(false);
        return;
    }
    if (!formData.owner_id) {
        setFormError("Proprietário do risco (Owner) é obrigatório.");
        setIsLoading(false);
        return;
    }

    try {
      if (isEditing && initialData?.id) {
        await apiClient.put(`/risks/${initialData.id}`, formData);
        notify.success('Risco atualizado com sucesso!');
      } else {
        await apiClient.post('/risks', formData);
        notify.success('Risco criado com sucesso!');
      }
      if (onSubmitSuccess) {
        onSubmitSuccess();
      } else {
        router.push('/admin/risks'); // Redirecionar para a lista por padrão
      }
    } catch (err: any) {
      console.error("Erro ao salvar risco:", err);
      const apiError = err.response?.data?.error || "Falha ao salvar risco.";
      setFormError(apiError); // Mostrar erro no formulário
      notify.error(apiError); // E também como toast
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {formError && <p className="text-red-500 bg-red-100 p-3 rounded-md">{formError}</p>}

      <div>
        <label htmlFor="title" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Título do Risco</label>
        <input type="text" name="title" id="title" value={formData.title} onChange={handleChange} required minLength={3} maxLength={255}
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>

      <div>
        <label htmlFor="description" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Descrição</label>
        <textarea name="description" id="description" value={formData.description} onChange={handleChange} rows={4}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label htmlFor="category" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Categoria</label>
          <select name="category" id="category" value={formData.category} onChange={handleChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
            <option value="tecnologico">Tecnológico</option>
            <option value="operacional">Operacional</option>
            <option value="legal">Legal</option>
          </select>
        </div>
        <div>
          <label htmlFor="status" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Status</label>
          <select name="status" id="status" value={formData.status} onChange={handleChange}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
            <option value="aberto">Aberto</option>
            <option value="em_andamento">Em Andamento</option>
            <option value="mitigado">Mitigado</option>
            <option value="aceito">Aceito</option>
          </select>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label htmlFor="impact" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Impacto</label>
          <select name="impact" id="impact" value={formData.impact} onChange={handleChange} required
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
            <option value="" disabled>Selecione o Impacto</option>
            <option value="Baixo">Baixo</option>
            <option value="Médio">Médio</option>
            <option value="Alto">Alto</option>
            <option value="Crítico">Crítico</option>
          </select>
        </div>
        <div>
          <label htmlFor="probability" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Probabilidade</label>
          <select name="probability" id="probability" value={formData.probability} onChange={handleChange} required
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
            <option value="" disabled>Selecione a Probabilidade</option>
            <option value="Baixo">Baixo</option>
            <option value="Médio">Médio</option>
            <option value="Alto">Alto</option>
            <option value="Crítico">Crítico</option>
          </select>
        </div>
      </div>

      <div>
        <label htmlFor="owner_id" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Proprietário do Risco (Owner)</label>
        {isLoadingUsers && <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">Carregando usuários...</p>}
        {usersError && <p className="text-sm text-red-500 dark:text-red-400 mt-1">{usersError} Por favor, insira o ID do proprietário manualmente ou tente recarregar.</p>}

        {!isLoadingUsers && !usersError && organizationUsers.length === 0 && (
             <p className="text-sm text-yellow-600 dark:text-yellow-400 mt-1">
                Nenhum outro usuário encontrado na organização. O risco será atribuído a você ({user?.name || 'usuário logado'}).
             </p>
        )}

        <select
            name="owner_id"
            id="owner_id"
            value={formData.owner_id}
            onChange={handleChange}
            required
            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2 disabled:opacity-50"
            disabled={isLoadingUsers || (usersError && organizationUsers.length === 0)}
        >
            <option value="" disabled>Selecione um proprietário</option>
            {organizationUsers.map(orgUser => (
                <option key={orgUser.id} value={orgUser.id}>
                    {orgUser.name} ({orgUser.id === user?.id ? 'Você' : orgUser.id.substring(0,8) + '...'})
                </option>
            ))}
            {/* Fallback se a lista de usuários não carregar, mas o usuário logado existe */}
            {usersError && organizationUsers.length === 0 && user && (
                 <option key={user.id} value={user.id}>
                    {user.name} (Você - fallback)
                </option>
            )}
        </select>
        { (usersError && organizationUsers.length > 0) &&
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                A lista de usuários pode estar incompleta devido a um erro. Selecione da lista ou insira o ID manualmente se necessário (não implementado).
            </p>
        }
      </div>

      <div className="flex justify-end space-x-3 pt-4">
        <button type="button" onClick={() => router.push('/admin/risks')}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 dark:bg-gray-600 dark:text-gray-200 dark:hover:bg-gray-500 rounded-md shadow-sm">
          Cancelar
        </button>
        <button type="submit" disabled={isLoading || isLoadingUsers}
                className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md shadow-sm disabled:opacity-50 flex items-center">
          {(isLoading || isLoadingUsers) && (
            <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          )}
          {isEditing ? 'Salvar Alterações' : 'Criar Risco'}
        </button>
      </div>
    </form>
  );
};

export default RiskForm;
