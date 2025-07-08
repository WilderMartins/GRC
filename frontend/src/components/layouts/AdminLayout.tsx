import Head from 'next/head';
import Link from 'next/link';
import { ReactNode } from 'react';
import { useAuth } from '@/contexts/AuthContext'; // Importar useAuth

type AdminLayoutProps = {
  children: ReactNode;
  title?: string;
};

export default function AdminLayout({ children, title = 'Painel Administrativo - Phoenix GRC' }: AdminLayoutProps) {
  const { user, logout } = useAuth(); // Obter usuário para orgId e função de logout

  return (
    <>
      <Head>
        <title>{title}</title>
      </Head>
      <div className="flex h-screen bg-gray-100 dark:bg-gray-900">
        {/* Sidebar (Placeholder) */}
        <aside className="w-64 bg-white p-6 shadow-md dark:bg-gray-800 hidden md:block">
          <div className="mb-8 text-center">
            <Link href="/admin/dashboard">
              <span className="text-2xl font-bold text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-500">
                Phoenix GRC
              </span>
            </Link>
          </div>
          <nav>
            <ul className="space-y-2">
              <li>
                <Link href="/admin/dashboard">
                  <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-indigo-500 hover:text-white dark:text-gray-300 dark:hover:bg-indigo-600">
                    Dashboard
                  </span>
                </Link>
              </li>
              <li>
                <Link href="/admin/audit/frameworks">
                  <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-indigo-500 hover:text-white dark:text-gray-300 dark:hover:bg-indigo-600">
                    Auditoria & Conformidade
                  </span>
                </Link>
              </li>
              <li>
                <Link href="/admin/risks">
                  <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-indigo-500 hover:text-white dark:text-gray-300 dark:hover:bg-indigo-600">
                    Gestão de Riscos
                  </span>
                </Link>
              </li>
              <li>
                <Link href="/admin/vulnerabilities">
                  <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-indigo-500 hover:text-white dark:text-gray-300 dark:hover:bg-indigo-600">
                    Gestão de Vulnerabilidades
                  </span>
                </Link>
              </li>
              {user?.organization_id && ( // Apenas mostrar se orgId estiver disponível
                <>
                  <li>
                    <Link href={`/admin/organizations/${user.organization_id}/identity-providers`}>
                      <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-indigo-500 hover:text-white dark:text-gray-300 dark:hover:bg-indigo-600">
                        Provedores de Identidade
                      </span>
                    </Link>
                  </li>
                  <li>
                    <Link href={`/admin/organizations/${user.organization_id}/webhooks`}>
                      <span className="block rounded-md px-4 py-2 text-gray-700 hover:bg-indigo-500 hover:text-white dark:text-gray-300 dark:hover:bg-indigo-600">
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
