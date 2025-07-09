import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useEffect, useState, useCallback, useMemo } from 'react';
import apiClient from '@/lib/axios';
import PaginationControls from '@/components/common/PaginationControls';
import { useNotifier } from '@/hooks/useNotifier';
import { useDebounce } from '@/hooks/useDebounce'; // Suposição: um hook de debounce existe ou será criado
import {
    Vulnerability,
    VulnerabilitySeverity,
    VulnerabilityStatus,
    PaginatedResponse,
    SortOrder
} from '@/types';

// Definições de tipos locais removidas

const VulnerabilitiesPageContent = () => {
  const notify = useNotifier();
  const [vulnerabilities, setVulnerabilities] = useState<Vulnerability[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Paginação
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);

  // Filtros
  const [filterSeverity, setFilterSeverity] = useState<VulnerabilitySeverity>("");
  const [filterStatus, setFilterStatus] = useState<VulnerabilityStatus>("");
  const [searchAssetInput, setSearchAssetInput] = useState<string>("");
  const debouncedSearchAsset = useDebounce(searchAssetInput, 500); // Debounce para input de texto

  // Ordenação
  const [sortBy, setSortBy] = useState<string>('created_at'); // Default sort
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');

  const fetchVulnerabilities = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const params: any = {
        page: currentPage,
        page_size: pageSize,
        sort_by: sortBy,
        order: sortOrder,
      };
      if (filterSeverity) params.severity = filterSeverity;
      if (filterStatus) params.status = filterStatus;
      if (debouncedSearchAsset) params.asset_affected_like = debouncedSearchAsset;

      const response = await apiClient.get<PaginatedVulnerabilitiesResponse>('/vulnerabilities', { params });
      setVulnerabilities(response.data.items || []);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
      // setCurrentPage(response.data.page); // API retorna a página atual, já gerenciamos via estado
      // setPageSize(response.data.page_size); // Manter pageSize do estado local
    } catch (err: any) {
      console.error("Erro ao buscar vulnerabilidades:", err);
      setError(err.response?.data?.error || err.message || "Falha ao buscar vulnerabilidades.");
      setVulnerabilities([]);
      setTotalItems(0);
      setTotalPages(0);
    } finally {
      setIsLoading(false);
    }
  }, [currentPage, pageSize, sortBy, sortOrder, filterSeverity, filterStatus, debouncedSearchAsset]);

  useEffect(() => {
    fetchVulnerabilities();
  }, [fetchVulnerabilities]);

  // Resetar para primeira página ao mudar filtros ou ordenação (exceto paginação em si)
  useEffect(() => {
    if (currentPage !== 1) {
        setCurrentPage(1);
    }
    // Este efeito não deve chamar fetchVulnerabilities diretamente,
    // a mudança em currentPage irá disparar o useEffect acima que chama fetchVulnerabilities.
}, [filterSeverity, filterStatus, debouncedSearchAsset, sortBy, sortOrder]);


  const handlePageChange = (newPage: number) => {
    setCurrentPage(newPage);
  };

  const handleSort = (newSortBy: string) => {
    if (sortBy === newSortBy) {
      setSortOrder(prevOrder => prevOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(newSortBy);
      setSortOrder('asc');
    }
    // setCurrentPage(1); // Resetar para primeira página ao mudar ordenação - já tratado pelo useEffect acima
  };

  const clearFilters = () => {
    setFilterSeverity("");
    setFilterStatus("");
    setSearchAssetInput("");
    setSortBy('created_at');
    setSortOrder('desc');
    if (currentPage !== 1) setCurrentPage(1); // Resetar página se não for a primeira
    // A mudança de estados acima irá acionar o useEffect para rebuscar
  };

  const handleDeleteVulnerability = async (vulnId: string, vulnTitle: string) => {
    if (window.confirm(`Tem certeza que deseja deletar a vulnerabilidade "${vulnTitle}"? Esta ação não pode ser desfeita.`)) {
      setIsLoading(true); // Indicar loading para a ação de deleção
      try {
        await apiClient.delete(`/vulnerabilities/${vulnId}`);
        notify.success(`Vulnerabilidade "${vulnTitle}" deletada com sucesso.`);
        // Re-buscar para atualizar a lista.
        // Se for o último item da última página, pode ser necessário ajustar currentPage.
        if (vulnerabilities.length === 1 && currentPage > 1) {
            setCurrentPage(currentPage - 1); // Volta para a página anterior
        } else {
            fetchVulnerabilities(); // Re-fetch a página atual (ou a nova, se currentPage mudou)
        }
      } catch (err: any) {
        console.error("Erro ao deletar vulnerabilidade:", err);
        notify.error(err.response?.data?.error || "Falha ao deletar vulnerabilidade.");
      } finally {
        // setIsLoading(false); // O fetchVulnerabilities já tem seu próprio setIsLoading(false)
      }
    }
  };

  const TableHeader: React.FC<{ field: string; label: string }> = ({ field, label }) => (
    <th scope="col" className="py-3.5 px-3 text-left text-sm font-semibold text-gray-900 dark:text-white cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600"
        onClick={() => handleSort(field)}>
      {label}
      {sortBy === field && (sortOrder === 'asc' ? ' ▲' : ' ▼')}
    </th>
  );

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

        {/* Filtros UI */}
        <div className="my-6 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg shadow">
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4 items-end">
            <div>
              <label htmlFor="filterSeverity" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Severidade</label>
              <select id="filterSeverity" value={filterSeverity} onChange={(e) => setFilterSeverity(e.target.value as VulnerabilitySeverity)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">Todas</option>
                <option value="Crítico">Crítico</option>
                <option value="Alto">Alto</option>
                <option value="Médio">Médio</option>
                <option value="Baixo">Baixo</option>
              </select>
            </div>
            <div>
              <label htmlFor="filterStatus" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Status</label>
              <select id="filterStatus" value={filterStatus} onChange={(e) => setFilterStatus(e.target.value as VulnerabilityStatus)}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                <option value="">Todos</option>
                <option value="descoberta">Descoberta</option>
                <option value="em_correcao">Em Correção</option>
                <option value="corrigida">Corrigida</option>
              </select>
            </div>
            <div>
              <label htmlFor="searchAssetInput" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Ativo Afetado</label>
              <input type="text" id="searchAssetInput" value={searchAssetInput} onChange={(e) => setSearchAssetInput(e.target.value)}
                     placeholder="Buscar por ativo..."
                     className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md"/>
            </div>
            <div>
              <button onClick={clearFilters}
                      className="w-full inline-flex items-center justify-center rounded-md border border-gray-300 dark:border-gray-500 bg-white dark:bg-gray-600 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-100 shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2">
                Limpar Filtros
              </button>
            </div>
          </div>
        </div>

        {isLoading && <p className="text-center text-gray-500 dark:text-gray-400 py-4">Carregando vulnerabilidades...</p>}
        {error && <p className="text-center text-red-500 py-4">Erro ao carregar vulnerabilidades: {error}</p>}

        {!isLoading && !error && vulnerabilities.length === 0 && (
            <div className="text-center py-10">
                <p className="text-gray-500 dark:text-gray-400">Nenhuma vulnerabilidade encontrada com os filtros aplicados.</p>
            </div>
        )}

        {!isLoading && !error && vulnerabilities.length > 0 && (
          <>
            <div className="mt-8 flow-root">
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                      <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                          <TableHeader field="title" label="Título" />
                          <TableHeader field="cve_id" label="CVE ID" />
                          <TableHeader field="severity" label="Severidade" />
                          <TableHeader field="status" label="Status" />
                          <TableHeader field="asset_affected" label="Ativo Afetado" />
                          {/* <TableHeader field="created_at" label="Criado em" /> */}
                          <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">Ações</span></th>
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
                                disabled={isLoading}
                              >
                                Deletar
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
      </div>
    </AdminLayout>
  );
};

// Hook de debounce simples (deve ser movido para um arquivo de hooks se usado em mais lugares)
// const useDebounce = (value: string, delay: number): string => {
//   const [debouncedValue, setDebouncedValue] = useState(value);
//   useEffect(() => {
//     const handler = setTimeout(() => {
//       setDebouncedValue(value);
//     }, delay);
//     return () => {
//       clearTimeout(handler);
//     };
//   }, [value, delay]);
//   return debouncedValue;
// };


export default WithAuth(VulnerabilitiesPageContent);
