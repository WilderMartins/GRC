import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import { useAuth } from '@/contexts/AuthContext';
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios';
import Link from 'next/link';
import { useNotifier } from '@/hooks/useNotifier';
import StatCard from '@/components/common/StatCard'; // Importar o StatCard comum
import { AdminStatistics, ActivityLog } from '@/types';

// Definições de tipos locais removidas

const AdminDashboardContent = () => {
  const { user } = useAuth();
  const notify = useNotifier();

  const [statistics, setStatistics] = useState<AdminStatistics | null>(null);
  const [recentActivity, setRecentActivity] = useState<ActivityLog[]>([]);

  const [isLoadingStats, setIsLoadingStats] = useState(true);
  const [statsError, setStatsError] = useState<string | null>(null);

  const [isLoadingActivity, setIsLoadingActivity] = useState(true);
  const [activityError, setActivityError] = useState<string | null>(null);

  useEffect(() => {
    const fetchDashboardData = async () => {
      // Fetch Statistics
      setIsLoadingStats(true);
      setStatsError(null);
      try {
        const statsResponse = await apiClient.get<AdminStatistics>('/admin/dashboard/statistics');
        setStatistics(statsResponse.data);
      } catch (err: any) {
        console.error("Erro ao buscar estatísticas do admin:", err);
        setStatsError(err.response?.data?.error || "Falha ao carregar estatísticas.");
        // notify.error("Falha ao carregar estatísticas do dashboard.");
      } finally {
        setIsLoadingStats(false);
      }

      // Fetch Recent Activity
      setIsLoadingActivity(true);
      setActivityError(null);
      try {
        const activityResponse = await apiClient.get<ActivityLog[]>('/admin/dashboard/recent-activity?limit=5');
        setRecentActivity(activityResponse.data || []);
      } catch (err: any) {
        console.error("Erro ao buscar atividade recente:", err);
        setActivityError(err.response?.data?.error || "Falha ao carregar atividade recente.");
        // notify.error("Falha ao carregar atividade recente.");
      } finally {
        setIsLoadingActivity(false);
      }
    };

    if (user) { // Garante que o usuário (admin) está carregado/autenticado
      fetchDashboardData();
    }
  }, [user, notify]); // Adicionado notify, embora não usado diretamente no fetch, é boa prática se fosse

  const formatTimestamp = (timestamp: string) => {
    try {
      return new Date(timestamp).toLocaleString('pt-BR', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch (e) {
      return timestamp; // Retorna o original se houver erro de formatação
    }
  };

  return (
    <AdminLayout title="Dashboard - Admin Phoenix GRC">
      <div className="container mx-auto px-4 py-8">
        <h1 className="text-3xl font-bold text-gray-800 dark:text-white mb-6">
          Dashboard Administrativo
        </h1>
        <p className="text-gray-600 dark:text-gray-300 mb-8">
          Bem-vindo ao painel de administração do Phoenix GRC, {user?.name || user?.email}.
        </p>

        {/* Cards de Estatísticas */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <StatCard
            title="Usuários Ativos"
            value={statistics?.active_users_count ?? '-'}
            isLoading={isLoadingStats}
            error={statsError && !statistics?.active_users_count ? "Erro" : null} // Mostrar erro no card se a stat específica falhou
          />
          <StatCard
            title="Riscos Totais"
            value={statistics?.total_risks_count ?? '-'}
            isLoading={isLoadingStats}
            error={statsError && !statistics?.total_risks_count ? "Erro" : null}
          />
          <StatCard
            title="Frameworks Ativos"
            value={statistics?.active_frameworks_count ?? '-'}
            isLoading={isLoadingStats}
            error={statsError && !statistics?.active_frameworks_count ? "Erro" : null}
          />
          <StatCard
            title="Vulnerabilidades Abertas"
            value={statistics?.open_vulnerabilities_count ?? '-'}
            isLoading={isLoadingStats}
            error={statsError && !statistics?.open_vulnerabilities_count ? "Erro" : null}
          />
        </div>
        {statsError && !statistics && ( // Erro geral para o bloco de estatísticas se tudo falhar
             <p className="text-sm text-red-500 dark:text-red-400 mb-6 text-center">Não foi possível carregar as estatísticas: {statsError}</p>
        )}


        {/* Atividade Recente */}
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md">
          <h2 className="text-xl font-semibold text-gray-700 dark:text-white mb-4">Atividade Recente</h2>
          {isLoadingActivity && <p className="text-gray-500 dark:text-gray-400">Carregando atividades...</p>}
          {activityError && <p className="text-red-500 dark:text-red-400">Erro ao carregar atividades: {activityError}</p>}
          {!isLoadingActivity && !activityError && recentActivity.length === 0 && (
            <p className="text-gray-500 dark:text-gray-400">Nenhuma atividade recente encontrada.</p>
          )}
          {!isLoadingActivity && !activityError && recentActivity.length > 0 && (
            <ul className="space-y-4">
              {recentActivity.map((activity) => (
                <li key={activity.id} className="border-b border-gray-200 dark:border-gray-700 pb-3 last:border-b-0">
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="font-semibold text-indigo-600 dark:text-indigo-400">{activity.actor_name}</span>
                      <span className="text-gray-600 dark:text-gray-300"> {activity.action_description}</span>
                    </div>
                    <span className="text-xs text-gray-400 dark:text-gray-500 whitespace-nowrap ml-2">
                      {formatTimestamp(activity.timestamp)}
                    </span>
                  </div>
                  {activity.target_link && (
                    <Link href={activity.target_link} legacyBehavior>
                      <a className="text-sm text-indigo-500 hover:underline dark:text-indigo-300 mt-1 inline-block">
                        Ver detalhes
                      </a>
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(AdminDashboardContent);
