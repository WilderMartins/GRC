import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link'; // Para o botão "Adicionar Novo Risco" se levar a outra página

import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path se necessário

// Tipos do backend (idealmente compartilhados ou gerados)
type RiskStatus = "aberto" | "em_andamento" | "mitigado" | "aceito";
type RiskImpact = "Baixo" | "Médio" | "Alto" | "Crítico";
type RiskProbability = "Baixo" | "Médio" | "Alto" | "Crítico";

interface RiskOwner { // Supondo que o preload de Owner retorne pelo menos isso
    id: string;
    name: string;
    email: string;
}
interface Risk {
  id: string;
  organization_id: string;
  title: string;
  description: string;
  category: string;
  impact: RiskImpact;
  probability: RiskProbability;
  status: RiskStatus;
  owner_id: string;
  owner?: RiskOwner; // GORM Preload pode popular isso
  created_at: string;
  updated_at: string;
}

interface PaginatedRisksResponse {
  items: Risk[];
  total_items: number;
  total_pages: number;
  page: number;
  page_size: number;
}

const RisksPageContent = () => {
  const [risks, setRisks] = useState<Risk[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10); // Pode ser configurável
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);

  const fetchRisks = async (page: number, size: number) => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await apiClient.get<PaginatedRisksResponse>('/risks', {
        params: { page, page_size: size },
      });
      setRisks(response.data.items || []);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
      setCurrentPage(response.data.page);
      setPageSize(response.data.page_size);
    } catch (err: any) {
      console.error("Erro ao buscar riscos:", err);
      setError(err.response?.data?.error || err.message || "Falha ao buscar riscos.");
      setRisks([]); // Limpar riscos em caso de erro
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchRisks(currentPage, pageSize);
  }, [currentPage, pageSize]);


  const handlePreviousPage = () => {
    if (currentPage > 1) {
      setCurrentPage(currentPage - 1);
    }
  };

  const handleNextPage = () => {
    if (currentPage < totalPages) {
      setCurrentPage(currentPage + 1);
    }
  };

  const handleDeleteRisk = async (riskId: string, riskTitle: string) => {
    if (window.confirm(`Tem certeza que deseja deletar o risco "${riskTitle}"? Esta ação não pode ser desfeita.`)) {
      // Idealmente, ter um estado de loading específico para a deleção da linha
      // Para simplificar, vamos reusar o isLoading geral ou adicionar um novo se necessário.
      // setIsLoading(true); // Ou um setLoadingDelete(true)
      try {
        await apiClient.delete(`/risks/${riskId}`);
        alert(`Risco "${riskTitle}" deletado com sucesso.`); // Placeholder para notificação melhor
        // Re-buscar os riscos para atualizar a lista
        // Se estiver na última página e ela ficar vazia, ajustar currentPage
        if (risks.length === 1 && currentPage > 1) {
            setCurrentPage(currentPage - 1);
        } else {
            fetchRisks(currentPage, pageSize); // Re-fetch a página atual
        }
      } catch (err: any) {
        console.error("Erro ao deletar risco:", err);
        setError(err.response?.data?.error || err.message || "Falha ao deletar risco.");
      } finally {
        // setIsLoading(false); // Ou setLoadingDelete(false)
      }
    }
  };


  return (
    <AdminLayout title="Gestão de Riscos - Phoenix GRC">
      <Head>
        <title>Gestão de Riscos - Phoenix GRC</title>
      </Head>

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex flex-col sm:flex-row justify-between items-center mb-6 gap-3">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            Gestão de Riscos
          </h1>
          {/* O botão pode abrir um Modal ou navegar para uma nova página /admin/risks/new */}
          <Link href="/admin/risks/new" legacyBehavior>
            <a className="inline-flex items-center justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800">
              Adicionar Novo Risco
            </a>
          </Link>
        </div>

        {/* TODO: Adicionar filtros e busca aqui */}

        {/* Tabela de Riscos (Placeholder) */}
        {/* TODO: Adicionar filtros e busca aqui */}

        {isLoading && <p className="text-center text-gray-500 dark:text-gray-400 py-4">Carregando riscos...</p>}
        {error && <p className="text-center text-red-500 py-4">Erro ao carregar riscos: {error}</p>}

        {!isLoading && !error && (
          <>
            <div className="mt-8 flow-root">
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                      <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                          <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">Título</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Categoria</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Impacto</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Probabilidade</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Status</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Proprietário</th>
                      <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6">
                        <span className="sr-only">Ações</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                    {risks.map((risk) => (
                      <tr key={risk.id}>
                        <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{risk.title}</td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{risk.category}</td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{risk.impact}</td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{risk.probability}</td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">
                           <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                risk.status === 'aberto' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-700 dark:text-yellow-100' :
                                risk.status === 'em_andamento' ? 'bg-blue-100 text-blue-800 dark:bg-blue-700 dark:text-blue-100' :
                                risk.status === 'mitigado' ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' :
                                risk.status === 'aceito' ? 'bg-purple-100 text-purple-800 dark:bg-purple-700 dark:text-purple-100' :
                                'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200'
                            }`}>
                                {risk.status}
                            </span>
                        </td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{risk.owner?.name || risk.owner_id}</td>
                        <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6 space-x-2">
                          <Link href={`/admin/risks/edit/${risk.id}`} legacyBehavior><a className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-200">Editar</a></Link>
                          <button
                            onClick={() => handleDeleteRisk(risk.id, risk.title)}
                            className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-200"
                            disabled={isLoading} // Desabilitar enquanto outra operação (como fetch) está em andamento
                          >
                            Deletar
                          </button>
                        </td>
                      </tr>
                    ))}
                    {risks.length === 0 && (
                        <tr>
                            <td colSpan={7} className="text-center py-4 text-sm text-gray-500 dark:text-gray-400">
                                Nenhum risco encontrado.
                            </td>
                        </tr>
                    )}
                  </tbody>
                </table>
              </div>
               {/* Controles de Paginação */}
              {totalPages > 0 && (
                <nav
                  className="flex items-center justify-between border-t border-gray-200 bg-white dark:bg-gray-800 px-4 py-3 sm:px-6"
                  aria-label="Paginação"
                >
                  <div className="hidden sm:block">
                    <p className="text-sm text-gray-700 dark:text-gray-300">
                      Mostrando <span className="font-medium">{(currentPage - 1) * pageSize + 1}</span>
                      {' '}a <span className="font-medium">{Math.min(currentPage * pageSize, totalItems)}</span>
                      {' '}de <span className="font-medium">{totalItems}</span> resultados
                    </p>
                  </div>
                  <div className="flex flex-1 justify-between sm:justify-end">
                    <button
                      onClick={handlePreviousPage}
                      disabled={currentPage <= 1 || isLoading}
                      className="relative inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
                    >
                      Anterior
                    </button>
                    <button
                      onClick={handleNextPage}
                      disabled={currentPage >= totalPages || isLoading}
                      className="relative ml-3 inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
                    >
                      Próxima
                    </button>
                  </div>
                </nav>
              )}
            </div>
          </div>
        </div>
        </>
        )}
      </div>
    </AdminLayout>
  );
};

export default WithAuth(RisksPageContent);
