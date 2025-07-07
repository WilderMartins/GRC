import React from 'react';
import Head from 'next/head';
import { useAuth } from '../contexts/AuthContext'; // Ajuste o path se necessário
import WithAuth from '../components/auth/WithAuth'; // Ajuste o path se necessário
import Link from 'next/link';

const DashboardPage = () => {
  const { user, logout } = useAuth();

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
                <Link href="/dashboard">
                  <span className="font-bold text-xl text-indigo-600 dark:text-indigo-400">
                    Phoenix GRC
                  </span>
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
            <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
              Seu Dashboard
            </h1>

            {user && (
              <div className="mt-6 bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                <h2 className="text-lg font-medium text-gray-900 dark:text-white">
                  Informações do Usuário
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
              <p className="text-gray-700 dark:text-gray-300">
                Este é o seu dashboard principal. Funcionalidades específicas do seu papel serão exibidas aqui.
              </p>
              {user?.role === 'admin' || user?.role === 'manager' ? (
                <p className="mt-2 text-gray-700 dark:text-gray-300">
                  Você pode acessar o <Link href="/admin/dashboard"><span className="text-indigo-600 hover:underline dark:text-indigo-400">Painel Administrativo</span></Link> para mais opções.
                </p>
              ) : null}
            </div>
          </div>
        </main>
      </div>
    </>
  );
};

// Aplicar o HOC WithAuth para proteger esta página
export default WithAuth(DashboardPage);
