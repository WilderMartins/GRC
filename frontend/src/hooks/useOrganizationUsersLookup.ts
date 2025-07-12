import { useState, useEffect, useCallback } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o caminho se necessário
import { UserLookupResponse } from '@/types'; // Supondo que você tenha esse tipo definido

// Definindo o tipo aqui se não existir em @/types
// interface UserLookupResponse {
//   id: string;
//   name: string;
// }

interface UseOrganizationUsersLookupReturn {
  users: UserLookupResponse[];
  isLoading: boolean;
  error: string | null;
  fetchUsers: () => Promise<void>; // Permitir recarregar manualmente se necessário
}

const useOrganizationUsersLookup = (): UseOrganizationUsersLookupReturn => {
  const [users, setUsers] = useState<UserLookupResponse[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false); // Iniciar como false, carregar sob demanda ou no mount
  const [error, setError] = useState<string | null>(null);

  const fetchUsers = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await apiClient.get<UserLookupResponse[]>('/api/v1/users/organization-lookup');
      setUsers(response.data || []);
    } catch (err: any) {
      console.error('Error fetching organization users lookup:', err);
      const errorMessage = err.response?.data?.error || 'Failed to load users';
      setError(errorMessage);
      setUsers([]); // Limpar usuários em caso de erro
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Opcional: Carregar usuários automaticamente quando o hook é montado.
  // Descomente se este for o comportamento desejado por padrão.
  // useEffect(() => {
  //   fetchUsers();
  // }, [fetchUsers]);

  return { users, isLoading, error, fetchUsers };
};

export default useOrganizationUsersLookup;
