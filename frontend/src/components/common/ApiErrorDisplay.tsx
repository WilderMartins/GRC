import React from 'react';
import { useTranslation } from 'next-i18next';

interface ApiErrorDisplayProps {
  error: string | null;
  className?: string;
}

const ApiErrorDisplay: React.FC<ApiErrorDisplayProps> = ({ error, className = '' }) => {
  const { t } = useTranslation('common');

  if (!error) {
    return null;
  }

  return (
    <div className={`bg-red-50 dark:bg-red-700/30 border-l-4 border-red-400 dark:border-red-500 p-4 rounded-md ${className}`}>
      <div className="flex">
        <div className="flex-shrink-0">
          <svg className="h-5 w-5 text-red-400 dark:text-red-300" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
          </svg>
        </div>
        <div className="ml-3">
          <h3 className="text-sm font-medium text-red-800 dark:text-red-200">
            {t('error_api_title')}
          </h3>
          <div className="mt-2 text-sm text-red-700 dark:text-red-300">
            <p>{error}</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ApiErrorDisplay;
