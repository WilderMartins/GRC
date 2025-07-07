import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import apiClient from '@/lib/axios'; // Ajuste o path
import { useAuth } from '@/contexts/AuthContext'; // Ajuste o path

// Tipos (podem vir de um arquivo compartilhado no futuro)
type RiskStatus = "aberto" | "em_andamento" | "mitigado" | "aceito";
type RiskImpact = "Baixo" | "Médio" | "Alto" | "Crítico";
type RiskProbability = "Baixo" | "Médio" | "Alto" | "Crítico";
type RiskCategory = "tecnologico" | "operacional" | "legal" | ""; // Adicionar mais se necessário

interface RiskFormData {
  title: string;
  description: string;
  category: RiskCategory;
  impact: RiskImpact | ""; // Permitir string vazia para valor inicial do select
  probability: RiskProbability | ""; // Permitir string vazia
  status: RiskStatus;
  owner_id: string; // UUID do proprietário
}

interface RiskFormProps {
  initialData?: RiskFormData & { id?: string }; // Para edição
  isEditing?: boolean;
  onSubmitSuccess?: () => void; // Callback para sucesso
}

const RiskForm: React.FC<RiskFormProps> = ({ initialData, isEditing = false, onSubmitSuccess }) => {
  const router = useRouter();
  const { user } = useAuth(); // Para pegar o ID do usuário logado como owner padrão
  const [formData, setFormData] = useState<RiskFormData>({
    title: '',
    description: '',
    category: 'tecnologico', // Default
    impact: '', // Default para select
    probability: '', // Default para select
    status: 'aberto', // Default
    owner_id: user?.id || '', // Default para usuário logado, se disponível
    ...(initialData || {}),
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // TODO: Carregar lista de usuários para o select de OwnerID (adiado)

  useEffect(() => {
    if (initialData) {
      setFormData({ ...initialData });
    }
    if (!isEditing && user && !initialData?.owner_id) { // Se criando e não há owner_id inicial
        setFormData(prev => ({...prev, owner_id: user.id}));
    }
  }, [initialData, isEditing, user]);


  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    if (!formData.impact || !formData.probability) {
        setError("Impacto e Probabilidade são obrigatórios.");
        setIsLoading(false);
        return;
    }
    if (!formData.owner_id) {
        // Tenta usar o usuário logado se owner_id não foi preenchido (ex: se o campo não estiver visível/editável)
        if (user?.id) {
            formData.owner_id = user.id;
        } else {
            setError("Proprietário do risco (Owner ID) é obrigatório.");
            setIsLoading(false);
            return;
        }
    }


    try {
      if (isEditing && initialData?.id) {
        await apiClient.put(`/risks/${initialData.id}`, formData);
        // alert('Risco atualizado com sucesso!'); // Substituir por notificação melhor
      } else {
        await apiClient.post('/risks', formData);
        // alert('Risco criado com sucesso!'); // Substituir por notificação melhor
      }
      if (onSubmitSuccess) {
        onSubmitSuccess();
      } else {
        router.push('/admin/risks'); // Redirecionar para a lista por padrão
      }
    } catch (err: any) {
      console.error("Erro ao salvar risco:", err);
      setError(err.response?.data?.error || err.message || "Falha ao salvar risco.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {error && <p className="text-red-500 bg-red-100 p-3 rounded-md">{error}</p>}

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
            {/* Adicionar outras categorias conforme definido em models.RiskCategory */}
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
        <label htmlFor="owner_id" className="block text-sm font-medium text-gray-700 dark:text-gray-300">ID do Proprietário (Owner)</label>
        <input type="text" name="owner_id" id="owner_id" value={formData.owner_id} onChange={handleChange} required
               placeholder="UUID do usuário proprietário"
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
        {/* TODO: Substituir por um select/autocomplete de usuários da organização */}
         <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Por enquanto, insira o UUID do usuário. Se deixado em branco ao criar, será atribuído ao usuário logado.
        </p>
      </div>

      <div className="flex justify-end space-x-3">
        <button type="button" onClick={() => router.push('/admin/risks')}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 dark:bg-gray-600 dark:text-gray-200 dark:hover:bg-gray-500 rounded-md shadow-sm">
          Cancelar
        </button>
        <button type="submit" disabled={isLoading}
                className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md shadow-sm disabled:opacity-50 flex items-center">
          {isLoading && (
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
