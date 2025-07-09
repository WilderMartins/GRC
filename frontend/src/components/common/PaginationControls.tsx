import React from 'react';

interface PaginationControlsProps {
  currentPage: number;
  totalPages: number;
  totalItems: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  isLoading?: boolean; // Para desabilitar controles durante o carregamento de dados
}

const PaginationControls: React.FC<PaginationControlsProps> = ({
  currentPage,
  totalPages,
  totalItems,
  pageSize,
  onPageChange,
  isLoading = false,
}) => {
  const handlePreviousPage = () => {
    if (currentPage > 1) {
      onPageChange(currentPage - 1);
    }
  };

  const handleNextPage = () => {
    if (currentPage < totalPages) {
      onPageChange(currentPage + 1);
    }
  };

  if (totalPages <= 0) { // Não renderizar nada se não houver páginas (ou apenas uma, dependendo da lógica)
    // Ou se totalItems for 0. Se totalItems > 0 mas totalPages é 0 ou 1, também não precisa de controles.
    // Simplificando: se não há páginas ou apenas uma, não mostrar.
    return null;
  }

  const startItem = totalItems > 0 ? (currentPage - 1) * pageSize + 1 : 0;
  const endItem = Math.min(currentPage * pageSize, totalItems);

  return (
    <nav
      className="flex items-center justify-between border-t border-gray-200 bg-white dark:bg-gray-800 px-4 py-3 sm:px-6 mt-4"
      aria-label="Paginação"
    >
      <div className="hidden sm:block">
        {totalItems > 0 ? (
            <p className="text-sm text-gray-700 dark:text-gray-300">
            Mostrando <span className="font-medium">{startItem}</span>
            {' '}a <span className="font-medium">{endItem}</span>
            {' '}de <span className="font-medium">{totalItems}</span> resultados
            </p>
        ) : (
            <p className="text-sm text-gray-700 dark:text-gray-300">Nenhum resultado encontrado.</p>
        )}
      </div>
      <div className="flex flex-1 justify-between sm:justify-end">
        <button
          onClick={handlePreviousPage}
          disabled={currentPage <= 1 || isLoading}
          className="relative inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
        >
          Anterior
        </button>
        <button
          onClick={handleNextPage}
          disabled={currentPage >= totalPages || isLoading}
          className="relative ml-3 inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
        >
          Próxima
        </button>
      </div>
    </nav>
  );
};

export default PaginationControls;
