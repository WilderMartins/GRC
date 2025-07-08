import React, { useState, useCallback } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path

interface BulkUploadErrorDetail {
  line_number: number;
  errors: string[];
}

interface BulkUploadRisksResponse {
  successfully_imported: number;
  failed_rows?: BulkUploadErrorDetail[];
  general_error?: string;
}

interface RiskBulkUploadModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUploadSuccess: () => void; // Para re-fetch da lista de riscos na página pai
}

const RiskBulkUploadModal: React.FC<RiskBulkUploadModalProps> = ({ isOpen, onClose, onUploadSuccess }) => {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadResult, setUploadResult] = useState<BulkUploadRisksResponse | null>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.files && event.target.files[0]) {
      if (event.target.files[0].type === 'text/csv' || event.target.files[0].name.endsWith('.csv')) {
        setSelectedFile(event.target.files[0]);
        setUploadError(null); // Limpar erro anterior ao selecionar novo arquivo
        setUploadResult(null); // Limpar resultado anterior
      } else {
        setSelectedFile(null);
        setUploadError("Formato de arquivo inválido. Por favor, selecione um arquivo .csv");
      }
    } else {
      setSelectedFile(null);
    }
  };

  const handleUploadSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedFile) {
      setUploadError("Nenhum arquivo selecionado.");
      return;
    }

    setIsUploading(true);
    setUploadError(null);
    setUploadResult(null);

    const formData = new FormData();
    formData.append('file', selectedFile);

    try {
      const response = await apiClient.post<BulkUploadRisksResponse>('/risks/bulk-upload-csv', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      setUploadResult(response.data);
      if (response.data.successfully_imported > 0) {
        onUploadSuccess(); // Notifica o pai para re-buscar os riscos
      }
      // Não fechar o modal automaticamente para o usuário ver o resultado.
    } catch (err: any) {
      console.error("Erro no upload do CSV:", err);
      const errorData = err.response?.data;
      if (errorData && (errorData.failed_rows || errorData.general_error)) {
        setUploadResult(errorData); // A API pode retornar detalhes de erro no corpo
      } else {
        setUploadError(err.response?.data?.error || err.message || "Falha ao enviar arquivo CSV.");
      }
    } finally {
      setIsUploading(false);
      // Resetar o input do arquivo para permitir novo upload do mesmo arquivo se necessário
      const fileInput = document.getElementById('csv-file-input') as HTMLInputElement;
      if (fileInput) {
        fileInput.value = '';
      }
      setSelectedFile(null); // Limpar o estado do arquivo selecionado
    }
  };

  const handleCloseModal = () => {
    setSelectedFile(null);
    setUploadResult(null);
    setUploadError(null);
    setIsUploading(false);
    onClose();
  }

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm">
      <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-lg">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-xl font-semibold text-gray-900 dark:text-white">Importar Riscos via CSV</h3>
          <button onClick={handleCloseModal} className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300">
            <span className="sr-only">Fechar</span>
            <svg className="h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleUploadSubmit} className="space-y-4">
          <div>
            <label htmlFor="csv-file-input" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Selecione o arquivo CSV
            </label>
            <input
              type="file"
              id="csv-file-input"
              name="file"
              accept=".csv,text/csv"
              onChange={handleFileChange}
              className="mt-1 block w-full text-sm text-gray-900 border border-gray-300 rounded-lg cursor-pointer bg-gray-50 dark:text-gray-400 focus:outline-none dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400"
            />
            {selectedFile && <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">Arquivo selecionado: {selectedFile.name}</p>}
          </div>

          <div className="flex justify-end space-x-3">
            <button type="button" onClick={handleCloseModal} disabled={isUploading}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 disabled:opacity-50">
              Cancelar
            </button>
            <button type="submit" disabled={!selectedFile || isUploading}
                    className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md shadow-sm disabled:opacity-50 flex items-center">
              {isUploading && (
                <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              )}
              {isUploading ? 'Enviando...' : 'Enviar Arquivo'}
            </button>
          </div>
        </form>

        {uploadError && <p className="mt-4 text-sm text-red-600 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md">{uploadError}</p>}

        {uploadResult && (
          <div className="mt-6 border-t border-gray-200 dark:border-gray-700 pt-4">
            <h4 className="text-md font-semibold text-gray-800 dark:text-white mb-2">Resultado da Importação:</h4>
            {uploadResult.general_error && (
                <p className="text-sm text-red-600 bg-red-100 dark:bg-red-900 dark:text-red-200 p-2 rounded-md">Erro Geral: {uploadResult.general_error}</p>
            )}
            <p className="text-sm text-green-600 dark:text-green-400">
              Riscos importados com sucesso: {uploadResult.successfully_imported}
            </p>
            {uploadResult.failed_rows && uploadResult.failed_rows.length > 0 && (
              <div className="mt-2">
                <p className="text-sm text-red-600 dark:text-red-400">
                  Linhas com erro ({uploadResult.failed_rows.length}):
                </p>
                <ul className="list-disc list-inside text-xs text-red-500 dark:text-red-300 max-h-40 overflow-y-auto bg-red-50 dark:bg-red-900 p-2 rounded-md">
                  {uploadResult.failed_rows.map((rowError, index) => (
                    <li key={index}>
                      Linha {rowError.line_number}: {rowError.errors.join(', ')}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default RiskBulkUploadModal;
