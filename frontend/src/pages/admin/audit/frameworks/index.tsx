import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path se necessário
import { AuditFramework } from '@/types';

// Definição local de AuditFramework removida

const AuditFrameworksPageContent = () => {
  const [frameworks, setFrameworks] = useState<AuditFramework[]>([]); // Usar o tipo importado
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchFrameworks = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const response = await apiClient.get<{ items: AuditFramework[] } | AuditFramework[]>('/audit/frameworks');
        // A API /audit/frameworks não está paginada no backend, então esperamos um array direto.
        // Se estivesse paginada, seria response.data.items
        if (Array.isArray(response.data)) {
            setFrameworks(response.data);
        } else if (response.data && Array.isArray((response.data as any).items)) { // Caso a API mude para paginada
            setFrameworks((response.data as any).items);
        } else {
            console.warn("Formato de resposta inesperado para frameworks:", response.data);
            setFrameworks([]);
        }
      } catch (err: any) {
        console.error("Erro ao buscar frameworks:", err);
        setError(err.response?.data?.error || err.message || "Falha ao buscar frameworks de auditoria.");
        setFrameworks([]);
      } finally {
        setIsLoading(false);
      }
    };

    fetchFrameworks();
  }, []);

  return (
    <AdminLayout title="Frameworks de Auditoria - Phoenix GRC">
      <Head>
        <title>Frameworks de Auditoria - Phoenix GRC</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            Frameworks de Auditoria e Conformidade
          </h1>
          {/* Botão para adicionar novo framework (se aplicável no futuro) */}
        </div>

        {isLoading && <p className="text-center text-gray-500 dark:text-gray-400 py-10">Carregando frameworks...</p>}
        {error && <p className="text-center text-red-500 py-10">Erro ao carregar frameworks: {error}</p>}

        {!isLoading && !error && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {frameworks.map((framework) => (
              <Link key={framework.id} href={`/admin/audit/frameworks/${framework.id}`} legacyBehavior>
                <a className="block p-6 bg-white dark:bg-gray-800 rounded-lg shadow-md hover:shadow-lg transition-shadow duration-200 ease-in-out">
                  <h2 className="text-xl font-semibold text-indigo-600 dark:text-indigo-400 mb-2">{framework.name}</h2>
                  <p className="text-gray-600 dark:text-gray-400 text-sm">
                    Clique para ver os controles e realizar avaliações para este framework.
                  </p>
                  {/* TODO: Adicionar contagem de controles ou progresso de conformidade aqui */}
                </a>
              </Link>
            ))}
            {frameworks.length === 0 && (
              <p className="text-gray-500 dark:text-gray-400 col-span-full text-center py-10">
                Nenhum framework de auditoria carregado ou disponível.
              </p>
            )}
          </div>
        )}
      </div>
    </AdminLayout>
  );
};

export default WithAuth(AuditFrameworksPageContent);
