import AdminLayout from '@/components/layouts/AdminLayout'; // Ajuste o path se necessário

export default function AdminDashboardPage() {
  return (
    <AdminLayout title="Dashboard - Admin Phoenix GRC">
      <div className="container mx-auto px-4 py-8">
        <h1 className="text-3xl font-bold text-gray-800 dark:text-white mb-6">
          Dashboard Administrativo
        </h1>
        <p className="text-gray-600 dark:text-gray-300 mb-4">
          Bem-vindo ao painel de administração do Phoenix GRC.
        </p>

        {/* Placeholders para Widgets ou Cards de Informação */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md">
            <h2 className="text-xl font-semibold text-gray-700 dark:text-white mb-2">Usuários Ativos</h2>
            <p className="text-3xl font-bold text-indigo-600 dark:text-indigo-400">125</p> {/* Exemplo */}
          </div>
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md">
            <h2 className="text-xl font-semibold text-gray-700 dark:text-white mb-2">Riscos Registrados</h2>
            <p className="text-3xl font-bold text-indigo-600 dark:text-indigo-400">42</p> {/* Exemplo */}
          </div>
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md">
            <h2 className="text-xl font-semibold text-gray-700 dark:text-white mb-2">Frameworks Ativos</h2>
            <p className="text-3xl font-bold text-indigo-600 dark:text-indigo-400">3</p> {/* Exemplo */}
          </div>
        </div>

        <div className="mt-8 bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md">
          <h2 className="text-xl font-semibold text-gray-700 dark:text-white mb-4">Atividade Recente</h2>
          <ul className="space-y-3">
            <li className="text-gray-600 dark:text-gray-300">Novo risco "Vulnerabilidade X" criado por usuário@teste.com.</li>
            <li className="text-gray-600 dark:text-gray-300">Framework ISO 27001 atualizado.</li>
            <li className="text-gray-600 dark:text-gray-300">Usuário admin@phoenix.com logado.</li>
          </ul>
        </div>

      </div>
    </AdminLayout>
  );
}
