import Head from 'next/head';
import React, { ReactNode } from 'react';
// import Image from 'next/image'; // Se for usar next/image para o logo

// Caminho para o logo padrão, assumindo que está em public/logos/
const PHOENIX_DEFAULT_LOGO_PATH = '/logos/phoenix-grc-logo-default.svg';

interface SetupLayoutProps {
  children: ReactNode;
  title?: string; // Título da aba do navegador
  pageTitle?: string; // Título exibido na página (opcional)
}

const SetupLayout: React.FC<SetupLayoutProps> = ({
  children,
  title = 'Configuração - Phoenix GRC',
  pageTitle,
}) => {
  return (
    <>
      <Head>
        <title>{title}</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900 py-12 px-4 sm:px-6 lg:px-8">
        <div className="w-full max-w-md space-y-8">
          <div>
            <img
              className="mx-auto h-12 w-auto"
              src={PHOENIX_DEFAULT_LOGO_PATH} // Usando <img> diretamente para SVGs em public
              alt="Phoenix GRC Logo"
            />
            {pageTitle && (
              <h2 className="mt-6 text-center text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
                {pageTitle}
              </h2>
            )}
          </div>
          <div className="bg-white dark:bg-gray-800 shadow-xl rounded-lg p-8 sm:p-10">
            {children}
          </div>
          <p className="mt-4 text-center text-xs text-gray-500 dark:text-gray-400">
            &copy; {new Date().getFullYear()} Phoenix GRC. Todos os direitos reservados.
          </p>
        </div>
      </div>
    </>
  );
};

export default SetupLayout;
