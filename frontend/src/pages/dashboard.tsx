import React, { useEffect, useState } from 'react';
import Head from 'next/head';
import { useAuth } from '../contexts/AuthContext';
import WithAuth from '../components/auth/WithAuth';
import Link from 'next/link';
import apiClient from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';

interface UserDashboardSummary {
  assigned_risks_open_count?: number;
  assigned_vulnerabilities_open_count?: number;
  pending_approval_tasks_count?: number;
}

// Reutilizando o StatCard do admin/dashboard.tsx, idealmente seria um componente comum
const StatCard: React.FC<{ title: string; value: number | string; isLoading: boolean; linkTo?: string; error?: string | null }> = ({ title, value, isLoading, linkTo, error }) => {
  const cardContent = (
    <>
      <h2 className="text-xl font-semibold text-gray-700 dark:text-white mb-2 truncate">{title}</h2>
      {isLoading && <p className="text-2xl font-bold text-gray-500 dark:text-gray-400 animate-pulse">Carregando...</p>}
      {error && !isLoading && <p className="text-sm font-bold text-red-500 dark:text-red-400">Erro</p>}
      {!isLoading && !error && <p className="text-3xl font-bold text-indigo-600 dark:text-indigo-400">{value}</p>}
    </>
  );

  if (linkTo) {
    return (
      <Link href={linkTo} legacyBehavior>
        <a className="block bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md hover:shadow-lg transition-shadow duration-150">
          {cardContent}
        </a>
      </Link>
    );
  }
  return (
    <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md">
      {cardContent}
    </div>
  );
};


const DashboardPageContent = () => {
  const { user, logout } = useAuth();
  const notify = useNotifier();

  const [summary, setSummary] = useState<UserDashboardSummary | null>(null);
  const [isLoadingSummary, setIsLoadingSummary] = useState(true);
  const [summaryError, setSummaryError] = useState<string | null>(null);

  useEffect(() => {
    const fetchUserSummary = async () => {
      if (!user) return;
      setIsLoadingSummary(true);
      setSummaryError(null);
      try {
        const response = await apiClient.get<UserDashboardSummary>('/me/dashboard/summary');
        setSummary(response.data);
      } catch (err: any) {
        console.error("Erro ao buscar resumo do dashboard do usuário:", err);
        setSummaryError(err.response?.data?.error || "Falha ao carregar resumo do dashboard.");
        // notify.error("Falha ao carregar dados do seu dashboard."); // Opcional: notificar erro
      } finally {
        setIsLoadingSummary(false);
      }
    };
    fetchUserSummary();
  }, [user, notify]); // Adicionado notify, embora não usado no fetch, por consistência

  return (
    <>
      <Head>
        <title>Dashboard - Phoenix GRC</title>
      </Head>
      <div className="min-h-screen bg-gray-100 dark:bg-gray-900">
        {/* Header Simples */}
        <header className="bg-white dark:bg-gray-800 shadow">
          <div className="container mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex h-16 items-center justify-between">
              <div className="flex items-center">
                <Link href="/dashboard" legacyBehavior>
                  <a className="font-bold text-xl text-indigo-600 dark:text-indigo-400">
                    Phoenix GRC
                  </a>
                </Link>
              </div>
              <div className="flex items-center">
                {user && (
                  <span className="text-gray-700 dark:text-gray-300 mr-4">
                    Olá, {user.name || user.email}!
                  </span>
                )}
                <button
                  onClick={logout}
                  className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
                >
                  Logout
                </button>
              </div>
            </div>
          </div>
        </header>

        {/* Conteúdo Principal */}
        <main className="py-10">
          <div className="container mx-auto px-4 sm:px-6 lg:px-8">
            <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white mb-8">
              Seu Dashboard
            </h1>

            {/* Cards de Resumo do Usuário */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
              <StatCard
                title="Riscos Abertos Atribuídos"
                value={summary?.assigned_risks_open_count ?? '-'}
                isLoading={isLoadingSummary}
                linkTo="/admin/risks" // Exemplo, ajustar o link correto
                error={summaryError && summary?.assigned_risks_open_count === undefined ? "Erro" : null}
              />
              <StatCard
                title="Vulnerabilidades Abertas Atribuídas"
                value={summary?.assigned_vulnerabilities_open_count ?? '-'}
                isLoading={isLoadingSummary}
                linkTo="/admin/vulnerabilities" // Exemplo
                error={summaryError && summary?.assigned_vulnerabilities_open_count === undefined ? "Erro" : null}
              />
              <StatCard
                title="Tarefas de Aprovação Pendentes"
                value={summary?.pending_approval_tasks_count ?? '-'}
                isLoading={isLoadingSummary}
                // linkTo="/approvals" // Exemplo, se houver uma página de aprovações
                error={summaryError && summary?.pending_approval_tasks_count === undefined ? "Erro" : null}
              />
            </div>
            {summaryError && !summary && ( // Erro geral para o bloco de resumo se tudo falhar
                 <p className="text-sm text-red-500 dark:text-red-400 mb-6 text-center">Não foi possível carregar seu resumo: {summaryError}</p>
            )}


            {/* Informações do Usuário (Mantido) */}
            {user && (
              <div className="mt-6 bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                <h2 className="text-lg font-medium text-gray-900 dark:text-white">
                  Informações do Perfil
                </h2>
                <dl className="mt-5 grid grid-cols-1 gap-x-4 gap-y-8 sm:grid-cols-2">
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Nome</dt>
                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{user.name}</dd>
                  </div>
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Email</dt>
                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{user.email}</dd>
                  </div>
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Role</dt>
                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{user.role}</dd>
                  </div>
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Organization ID</dt>
                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{user.organization_id}</dd>
                  </div>
                </dl>
              </div>
            )}

            <div className="mt-8">
              {user?.role === 'admin' || user?.role === 'manager' ? (
                <p className="text-gray-700 dark:text-gray-300">
                  Você também pode acessar o <Link href="/admin/dashboard"><span className="text-indigo-600 hover:underline dark:text-indigo-400">Painel Administrativo</span></Link> para gerenciamento da plataforma.
                </p>
              ) : (
                <p className="text-gray-700 dark:text-gray-300">
                  Use os menus de navegação para acessar as funcionalidades do sistema.
                </p>
              )}
            </div>
          </div>
        </main>
      </div>
    </>
  );
};

const DashboardPage = WithAuth(DashboardPageContent);
export default DashboardPage;
