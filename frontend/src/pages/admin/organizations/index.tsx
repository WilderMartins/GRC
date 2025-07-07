import AdminLayout from '@/components/layouts/AdminLayout';

export default function AdminOrganizationsPage() {
  return (
    <AdminLayout title="Gerenciar Organizações - Admin Phoenix GRC">
      <div className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold text-gray-800 dark:text-white">
            Gerenciar Organizações
          </h1>
          <button
            // onClick={handleAddNewOrganization} // TODO
            className="bg-indigo-600 hover:bg-indigo-700 text-white font-bold py-2 px-4 rounded-md shadow-sm transition duration-150 ease-in-out"
          >
            Adicionar Nova Organização (TODO)
          </button>
        </div>
        <div className="bg-white dark:bg-gray-800 shadow-md rounded-lg p-6">
          <p className="text-gray-600 dark:text-gray-300">
            A funcionalidade de gerenciamento de organizações (CRUD, configurações de whitelabel, etc.) será implementada aqui.
          </p>
          {/* TODO: Tabela de organizações, formulário para adicionar/editar */}
        </div>
      </div>
    </AdminLayout>
  );
}
