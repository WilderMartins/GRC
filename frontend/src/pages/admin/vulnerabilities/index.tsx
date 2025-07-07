import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path se necessário

// Tipos (placeholder, idealmente de um arquivo compartilhado)
type VulnerabilitySeverity = "Baixo" | "Médio" | "Alto" | "Crítico";
type VulnerabilityStatus = "descoberta" | "em_correcao" | "corrigida";

interface Vulnerability {
  id: string;
  title: string;
  cve_id?: string;
  severity: VulnerabilitySeverity;
  status: VulnerabilityStatus;
  asset_affected: string;
  created_at: string;
}

interface PaginatedVulnerabilitiesResponse {
  items: Vulnerability[];
  total_items: number;
  total_pages: number;
  page: number;
  page_size: number;
}

const VulnerabilitiesPageContent = () => {
  const [vulnerabilities, setVulnerabilities] = useState<Vulnerability[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);

  const fetchVulnerabilities = async (page: number, size: number) => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await apiClient.get<PaginatedVulnerabilitiesResponse>('/vulnerabilities', {
        params: { page, page_size: size },
      });
      setVulnerabilities(response.data.items || []);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
      setCurrentPage(response.data.page);
      setPageSize(response.data.page_size);
    } catch (err: any) {
      console.error("Erro ao buscar vulnerabilidades:", err);
      setError(err.response?.data?.error || err.message || "Falha ao buscar vulnerabilidades.");
      setVulnerabilities([]);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchVulnerabilities(currentPage, pageSize);
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

  const handleDeleteVulnerability = async (vulnId: string, vulnTitle: string) => {
    if (window.confirm(`Tem certeza que deseja deletar a vulnerabilidade "${vulnTitle}"? Esta ação não pode ser desfeita.`)) {
      // Idealmente, ter um estado de loading específico para a deleção
      // setIsLoading(true); // ou um setLoadingDelete(true)
      try {
        await apiClient.delete(`/vulnerabilities/${vulnId}`);
        alert(`Vulnerabilidade "${vulnTitle}" deletada com sucesso.`); // Placeholder
        // Re-buscar para atualizar a lista
        if (vulnerabilities.length === 1 && currentPage > 1) {
          setCurrentPage(currentPage - 1); // Ir para página anterior se esta ficar vazia
        } else {
          fetchVulnerabilities(currentPage, pageSize); // Re-fetch a página atual
        }
      } catch (err: any) {
        console.error("Erro ao deletar vulnerabilidade:", err);
        setError(err.response?.data?.error || err.message || "Falha ao deletar vulnerabilidade.");
      } finally {
        // setIsLoading(false); // ou setLoadingDelete(false)
      }
    }
  };

  return (
    <AdminLayout title="Gestão de Vulnerabilidades - Phoenix GRC">
      <Head>
        <title>Gestão de Vulnerabilidades - Phoenix GRC</title>
      </Head>

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex flex-col sm:flex-row justify-between items-center mb-6 gap-3">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            Gestão de Vulnerabilidades
          </h1>
          <Link href="/admin/vulnerabilities/new" legacyBehavior>
            <a className="inline-flex items-center justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800">
              Adicionar Nova Vulnerabilidade
            </a>
          </Link>
        </div>

        {isLoading && <p className="text-center text-gray-500 dark:text-gray-400 py-4">Carregando vulnerabilidades...</p>}
        {error && <p className="text-center text-red-500 py-4">Erro ao carregar vulnerabilidades: {error}</p>}

        {!isLoading && !error && (
          <>
            {/* Tabela de Vulnerabilidades */}
            <div className="mt-8 flow-root">
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                      <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                          <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">Título</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">CVE ID</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Severidade</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Status</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">Ativo Afetado</th>
                      <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6">
                        <span className="sr-only">Ações</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                    {vulnerabilities.map((vuln) => (
                      <tr key={vuln.id}>
                        <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{vuln.title}</td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{vuln.cve_id || '-'}</td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{vuln.severity}</td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">
                           <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                vuln.status === 'descoberta' ? 'bg-orange-100 text-orange-800 dark:bg-orange-700 dark:text-orange-100' :
                                vuln.status === 'em_correcao' ? 'bg-blue-100 text-blue-800 dark:bg-blue-700 dark:text-blue-100' :
                                vuln.status === 'corrigida' ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' :
                                'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200'
                            }`}>
                                {vuln.status}
                            </span>
                        </td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{vuln.asset_affected}</td>
                        <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6 space-x-2">
                          <Link href={`/admin/vulnerabilities/edit/${vuln.id}`} legacyBehavior><a className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-200">Editar</a></Link>
                          <button
                            onClick={() => handleDeleteVulnerability(vuln.id, vuln.title)}
                            className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-200"
                            disabled={isLoading} // Desabilitar enquanto outra operação (como fetch) está em andamento
                          >
                            Deletar
                          </button>
                        </td>
                      </tr>
                    ))}
                    {vulnerabilities.length === 0 && (
                        <tr>
                            <td colSpan={6} className="text-center py-4 text-sm text-gray-500 dark:text-gray-400">
                                Nenhuma vulnerabilidade encontrada.
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

export default WithAuth(VulnerabilitiesPageContent);
