import Head from 'next/head';
import Link from 'next/link';
import Image from 'next/image'; // Importar Image do Next.js
import { ReactNode } from 'react';
import { useAuth } from '@/contexts/AuthContext'; // Importar useAuth

type AdminLayoutProps = {
  children: ReactNode;
  title?: string;
};

// Define o caminho para o logo padrão do Phoenix GRC
const PHOENIX_DEFAULT_LOGO_PATH = '/logos/phoenix-grc-logo-default.svg'; // Exemplo de caminho, ajuste conforme necessário

export default function AdminLayout({ children, title = 'Painel Administrativo - Phoenix GRC' }: AdminLayoutProps) {
  const { user, logout, branding, isLoading: authIsLoading } = useAuth(); // Obter branding e isLoading

  // Define o logo a ser usado: o da organização ou o padrão.
  // Considera também um estado de carregamento para evitar piscar o logo padrão rapidamente.
  const logoToDisplay = !authIsLoading && branding.logoUrl ? branding.logoUrl : PHOENIX_DEFAULT_LOGO_PATH;
  const organizationName = user?.organization?.name || 'Phoenix GRC'; // Assume que user.organization.name pode existir

  return (
    <>
      <Head>
        <title>{title}</title>
      </Head>
      <div className="flex h-screen bg-gray-100 dark:bg-gray-900">
        {/* Sidebar */}
        <aside className="w-64 bg-white p-6 shadow-md dark:bg-gray-800 hidden md:block">
          <div className="mb-8 text-center">
            <Link href="/admin/dashboard" passHref>
              <div className="cursor-pointer inline-block"> {/* Div para o cursor pointer */}
                {authIsLoading ? (
                  <div className="h-10 w-auto animate-pulse bg-gray-300 dark:bg-gray-700 rounded"></div> // Placeholder enquanto carrega
                ) : branding.logoUrl ? (
                  <img
                    src={branding.logoUrl}
                    alt={`${organizationName} Logo`}
                    className="h-10 w-auto mx-auto object-contain" // Ajuste de altura e centralização
                  />
                ) : (
                  // Fallback para o logo padrão SVG ou texto
                  <img
                    src={PHOENIX_DEFAULT_LOGO_PATH}
                    alt="Phoenix GRC Default Logo"
                    className="h-10 w-auto mx-auto object-contain"
                  />
                )}
              </div>
            </Link>
          </div>
          <nav>
            <ul className="space-y-2">
              <li>
                <Link href="/admin/dashboard">
                  <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-brand-primary hover:text-white dark:text-gray-300 dark:hover:bg-brand-primary dark:hover:text-gray-100">
                    Dashboard
                  </span>
                </Link>
              </li>
              <li>
                <Link href="/admin/audit/frameworks">
                  <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-brand-primary hover:text-white dark:text-gray-300 dark:hover:bg-brand-primary dark:hover:text-gray-100">
                    Auditoria & Conformidade
                  </span>
                </Link>
              </li>
              <li>
                <Link href="/admin/risks">
                  <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-brand-primary hover:text-white dark:text-gray-300 dark:hover:bg-brand-primary dark:hover:text-gray-100">
                    Gestão de Riscos
                  </span>
                </Link>
              </li>
              <li>
                <Link href="/admin/vulnerabilities">
                  <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-brand-primary hover:text-white dark:text-gray-300 dark:hover:bg-brand-primary dark:hover:text-gray-100">
                    Gestão de Vulnerabilidades
                  </span>
                </Link>
              </li>
              {user?.organization_id && ( // Apenas mostrar se orgId estiver disponível
                <>
                  <li>
                    <Link href={`/admin/organizations/${user.organization_id}/identity-providers`}>
                      <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-brand-primary hover:text-white dark:text-gray-300 dark:hover:bg-brand-primary dark:hover:text-gray-100">
                        Provedores de Identidade
                      </span>
                    </Link>
                  </li>
                  <li>
                    <Link href={`/admin/organizations/${user.organization_id}/webhooks`}>
                      <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-brand-primary hover:text-white dark:text-gray-300 dark:hover:bg-brand-primary dark:hover:text-gray-100">
                        Webhooks
                      </span>
                    </Link>
                  </li>
                </>
              )}
              {/* Link para Organizações pode ser mais genérico ou removido se a gestão for sempre contextual */}
              {/* <li>
                <Link href="/admin/organizations">
                  <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-indigo-500 hover:text-white dark:text-gray-300 dark:hover:bg-indigo-600">
                    Organizações (Geral)
                  </span>
                </Link>
              </li> */}
              {/* Adicionar mais links de navegação aqui */}
              <li className="mt-auto"> {/* Empurrar para o final */}
                <button
                  onClick={logout}
                  className="w-full text-left block rounded-md px-4 py-2 text-gray-700 hover:bg-red-500 hover:text-white dark:text-gray-300 dark:hover:bg-red-600"
                >
                  Logout
                </button>
              </li>
            </ul>
          </nav>
        </aside>

        {/* Main Content Area */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* Header (Placeholder) */}
          <header className="bg-white shadow-sm dark:bg-gray-800 p-4">
            <div className="flex items-center justify-between">
              <h1 className="text-xl font-semibold text-gray-800 dark:text-white">
                {title}
              </h1>
              <div className="flex items-center space-x-3">
                <span className="text-gray-600 dark:text-gray-400">{user?.name || user?.email}</span>
                <button
                  onClick={logout}
                  title="Logout"
                  className="p-2 rounded-full hover:bg-gray-200 dark:hover:bg-gray-700">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-6 h-6 text-gray-600 dark:text-gray-400">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0 0 13.5 3h-6a2.25 2.25 0 0 0-2.25 2.25v13.5A2.25 2.25 0 0 0 7.5 21h6a2.25 2.25 0 0 0 2.25-2.25V15m3 0 3-3m0 0-3-3m3 3H9" />
                  </svg>
                </button>
              </div>
            </div>
          </header>

          {/* Page Content */}
          <main className="flex-1 overflow-y-auto p-6">
            {children}
          </main>
        </div>
      </div>
    </>
  );
}
