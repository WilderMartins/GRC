import { useState, useCallback, useEffect } from 'react';
import apiClient from '@/lib/axios';
import { PaginatedResponse } from '@/types';
import { useDebounce } from './useDebounce';

/**
 * Hook customizado para gerenciar dados paginados de uma API.
 * Encapsula a lógica de busca, paginação, filtros e ordenação.
 * @param T - O tipo dos itens nos dados paginados.
 * @param TFilter - O tipo do objeto de filtros.
 */
interface UsePaginatedDataParams<TFilter> {
  endpoint: string;
  initialPageSize?: number;
  initialSortBy?: string;
  initialSortOrder?: 'asc' | 'desc';
  initialFilters?: TFilter;
  debounceDelay?: number;
}

const usePaginatedData = <T, TFilter extends Record<string, any>>({
  endpoint,
  initialPageSize = 10,
  initialSortBy = 'created_at',
  initialSortOrder = 'desc',
  initialFilters = {} as TFilter,
  debounceDelay = 500,
}: UsePaginatedDataParams<TFilter>) => {
  const [data, setData] = useState<T[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Estados de paginação
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(initialPageSize);
  const [totalPages, setTotalPages] = useState(0);
  const [totalItems, setTotalItems] = useState(0);

  // Estados de filtro e ordenação
  const [filters, setFilters] = useState<TFilter>(initialFilters);
  const [sortBy, setSortBy] = useState(initialSortBy);
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>(initialSortOrder);

  // Debounce nos filtros para evitar chamadas de API em cada digitação, melhorando a performance.
  const debouncedFilters = useDebounce(filters, debounceDelay);

  const fetchData = useCallback(async (page = currentPage, size = pageSize) => {
    setIsLoading(true);
    setError(null);
    try {
      const params = {
        page,
        page_size: size,
        sort_by: sortBy,
        order: sortOrder,
        ...debouncedFilters,
      };

      // Remove chaves de filtro que são strings vazias, nulas ou indefinidas para não poluir a URL da API.
      Object.keys(params).forEach(key => {
        const paramKey = key as keyof typeof params;
        if (params[paramKey] === '' || params[paramKey] === null || params[paramKey] === undefined) {
          delete params[paramKey];
        }
      });

      const response = await apiClient.get<PaginatedResponse<T>>(endpoint, { params });
      setData(response.data.items || []);
      setTotalItems(response.data.total_items);
      setTotalPages(response.data.total_pages);
    } catch (err: any) {
      console.error(`Error fetching data from ${endpoint}:`, err);
      setError(err.response?.data?.error || `Failed to load data from ${endpoint}`);
      setData([]);
      setTotalItems(0);
      setTotalPages(0);
    } finally {
      setIsLoading(false);
    }
  }, [endpoint, currentPage, pageSize, sortBy, sortOrder, debouncedFilters]);

  // Efeito para buscar dados quando qualquer parâmetro de busca muda
  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Efeito para resetar a página para 1 quando os filtros mudam
  useEffect(() => {
    if (currentPage !== 1) {
      setCurrentPage(1);
    }
  }, [debouncedFilters, sortBy, sortOrder]);


  // Funções para manipulação externa
  const handlePageChange = (newPage: number) => {
    setCurrentPage(newPage);
  };

  const handleSort = (newSortBy: string) => {
    if (sortBy === newSortBy) {
      setSortOrder(prev => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortBy(newSortBy);
      setSortOrder('asc');
    }
  };

  const handleFilterChange = (newFilters: Partial<TFilter>) => {
    setFilters(prev => ({ ...prev, ...newFilters }));
  };

  const resetFilters = () => {
    setFilters(initialFilters);
    setSortBy(initialSortBy);
    setSortOrder(initialSortOrder);
    if(currentPage !== 1) {
        setCurrentPage(1);
    }
  };

  return {
    data,
    isLoading,
    error,
    currentPage,
    totalPages,
    totalItems,
    pageSize,
    filters,
    sortBy,
    sortOrder,
    handlePageChange,
    handleSort,
    handleFilterChange,
    resetFilters,
    fetchData, // Expor para re-fetch manual
  };
};

export default usePaginatedData;
