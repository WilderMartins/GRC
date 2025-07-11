import React from 'react';
import { useTranslation } from 'next-i18next';

interface PasswordStrengthIndicatorProps {
  isValid: boolean;
  textKey: string;
  showIcon?: boolean; // Prop para controlar a visibilidade do ícone
}

const PasswordStrengthIndicator: React.FC<PasswordStrengthIndicatorProps> = ({ isValid, textKey, showIcon = true }) => {
  const { t } = useTranslation('auth'); // Assumindo que as chaves de tradução estão em 'auth.json' ou similar

  return (
    <div className={`flex items-center text-sm ${isValid ? 'text-green-600 dark:text-green-400' : 'text-gray-500 dark:text-gray-400'}`}>
      {showIcon && (
        isValid ? (
          <svg className="w-4 h-4 mr-2 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
          </svg>
        ) : (
          <svg className="w-4 h-4 mr-2 flex-shrink-0" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm0-2a6 6 0 100-12 6 6 0 000 12zM9 9a1 1 0 011-1h.01a1 1 0 110 2H10a1 1 0 01-1-1zm.01-3.01a.99.99 0 000 1.98h-.01a.99.99 0 000-1.98h.01z" clipRule="evenodd" />
          </svg> // Ícone de círculo ou "não marcado"
        )
      )}
      <span>{t(textKey)}</span>
    </div>
  );
};

export default PasswordStrengthIndicator;
