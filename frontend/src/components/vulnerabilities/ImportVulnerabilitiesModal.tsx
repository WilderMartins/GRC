import { useState, useRef } from 'react';
import { useTranslation } from 'next-i18next';
import apiClient from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';

interface ImportVulnerabilitiesModalProps {
  isOpen: boolean;
  onClose: () => void;
  onImportSuccess: () => void;
}

const ImportVulnerabilitiesModal: React.FC<ImportVulnerabilitiesModalProps> = ({ isOpen, onClose, onImportSuccess }) => {
  const { t } = useTranslation(['vulnerabilities', 'common']);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isImporting, setIsImporting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const notify = useNotifier();

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.files && event.target.files[0]) {
      setSelectedFile(event.target.files[0]);
    }
  };

  const handleImport = async () => {
    if (!selectedFile) {
      notify.error(t('import_modal.no_file_selected_error'));
      return;
    }

    setIsImporting(true);
    const formData = new FormData();
    formData.append('file', selectedFile);

    try {
      const response = await apiClient.post('/api/v1/vulnerabilities/import-csv', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      notify.success(t('import_modal.import_success_message', { created: response.data.created_count, updated: response.data.updated_count }));
      onImportSuccess();
      handleClose();
    } catch (err: any) {
      notify.error(t('import_modal.import_error_message', { message: err.response?.data?.error || t('common:unknown_error') }));
    } finally {
      setIsImporting(false);
    }
  };

  const handleClose = () => {
    setSelectedFile(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 transition-opacity">
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-lg mx-4">
        <div className="p-6">
          <h3 className="text-xl font-semibold text-gray-900 dark:text-white">{t('import_modal.title')}</h3>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-300">
            {t('import_modal.description')}
          </p>
          <div className="mt-4">
            <p className="text-sm font-medium text-gray-700 dark:text-gray-200">{t('import_modal.csv_format_label')}</p>
            <ul className="list-disc list-inside mt-1 text-xs text-gray-500 dark:text-gray-400 space-y-1">
              <li>{t('import_modal.csv_header_note')} <strong>title, severity, asset_affected</strong></li>
              <li>{t('import_modal.csv_optional_note')} description, cveid, status</li>
            </ul>
          </div>

          <div className="mt-6">
            <label htmlFor="file-upload" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
              {t('import_modal.file_upload_label')}
            </label>
            <div className="mt-2 flex items-center justify-center w-full">
              <label className="flex flex-col w-full h-32 border-4 border-dashed hover:bg-gray-100 dark:hover:bg-gray-700 hover:border-gray-300 dark:hover:border-gray-500 rounded-lg cursor-pointer">
                <div className="flex flex-col items-center justify-center pt-7">
                  <svg className="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M7 16a4 4 0 01-4-4V7a4 4 0 014-4h.586a1 1 0 01.707.293l2 2a1 1 0 001.414 0l2-2A1 1 0 0116.414 3H17a4 4 0 014 4v5a4 4 0 01-4 4H7z" /></svg>
                  <p className="pt-1 text-sm tracking-wider text-gray-400">{selectedFile ? selectedFile.name : t('import_modal.select_file_placeholder')}</p>
                </div>
                <input id="file-upload" ref={fileInputRef} type="file" className="opacity-0" accept=".csv" onChange={handleFileChange} />
              </label>
            </div>
          </div>
        </div>
        <div className="bg-gray-50 dark:bg-gray-700/50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
          <button
            type="button"
            onClick={handleImport}
            disabled={!selectedFile || isImporting}
            className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-brand-primary text-base font-medium text-white hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary sm:ml-3 sm:w-auto sm:text-sm disabled:opacity-50"
          >
            {isImporting ? t('import_modal.importing_button') : t('import_modal.import_button')}
          </button>
          <button
            type="button"
            onClick={handleClose}
            className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 dark:border-gray-500 shadow-sm px-4 py-2 bg-white dark:bg-gray-600 text-base font-medium text-gray-700 dark:text-gray-100 hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:w-auto sm:text-sm"
          >
            {t('common:cancel_button')}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ImportVulnerabilitiesModal;
