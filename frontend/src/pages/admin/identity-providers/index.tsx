import AdminLayout from '@/components/layouts/AdminLayout';
import { useState } from 'react'; // Para gerenciar estado do modal/formulário

// Tipos simulados (devem corresponder aos modelos do backend no futuro)
type IdentityProviderType = 'saml' | 'oauth2_google' | 'oauth2_github';

interface IdentityProvider {
  id: string;
  organization_id: string;
  provider_type: IdentityProviderType;
  name: string;
  is_active: boolean;
  config_json: Record<string, any>; // Simplificado
  attribute_mapping_json?: Record<string, any>; // Simplificado
  created_at: string;
  updated_at: string;
}

// Mock de dados de provedores (substituir por chamadas de API)
const mockIdentityProviders: IdentityProvider[] = [
  {
    id: 'google-uuid-123',
    organization_id: 'org-uuid-abc',
    provider_type: 'oauth2_google',
    name: 'Login com Google (Corporativo)',
    is_active: true,
    config_json: { client_id: 'xxxx.apps.googleusercontent.com', client_secret: 'YYYYY' },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'saml-okta-uuid-456',
    organization_id: 'org-uuid-abc',
    provider_type: 'saml',
    name: 'Okta SAML SSO',
    is_active: true,
    config_json: { idp_entity_id: 'http://www.okta.com/exk123', idp_sso_url: 'https://org.okta.com/app/...' },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 'github-uuid-789',
    organization_id: 'org-uuid-abc',
    provider_type: 'oauth2_github',
    name: 'Login com GitHub',
    is_active: false,
    config_json: { client_id: 'gh-client-id', client_secret: 'gh-client-secret' },
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
];

export default function IdentityProvidersPage() {
  const [identityProviders, setIdentityProviders] = useState<IdentityProvider[]>(mockIdentityProviders);
  const [showModal, setShowModal] = useState(false);
  const [editingProvider, setEditingProvider] = useState<IdentityProvider | null>(null);

  // TODO: Funções para buscar, criar, atualizar, deletar provedores via API

  const handleAddNewProvider = () => {
    setEditingProvider(null); // Limpar formulário para novo provedor
    setShowModal(true);
  };

  const handleEditProvider = (provider: IdentityProvider) => {
    setEditingProvider(provider);
    setShowModal(true);
  };

  const handleDeleteProvider = (providerId: string) => {
    if (window.confirm("Tem certeza que deseja remover este provedor de identidade?")) {
      // TODO: Chamar API para deletar
      setIdentityProviders(prev => prev.filter(p => p.id !== providerId));
      alert(`Provedor ${providerId} deletado (simulação).`);
    }
  };

  const handleSaveProvider = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    // TODO: Extrair dados do formulário
    // TODO: Chamar API para criar ou atualizar
    const formData = new FormData(event.currentTarget);
    const name = formData.get('name') as string;
    const type = formData.get('provider_type') as IdentityProviderType;

    alert(`Salvando provedor: ${name} do tipo ${type} (simulação).`);
    setShowModal(false);
    // Atualizar lista local após salvar (simulação)
  };

  return (
    <AdminLayout title="Provedores de Identidade - Admin Phoenix GRC">
      <div className="container mx-auto px-4 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold text-gray-800 dark:text-white">
            Gerenciar Provedores de Identidade (SSO/Social Login)
          </h1>
          <button
            onClick={handleAddNewProvider}
            className="bg-indigo-600 hover:bg-indigo-700 text-white font-bold py-2 px-4 rounded-md shadow-sm transition duration-150 ease-in-out"
          >
            Adicionar Novo Provedor
          </button>
        </div>

        {/* Tabela de Provedores de Identidade */}
        <div className="bg-white dark:bg-gray-800 shadow-md rounded-lg overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-700">
              <tr>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Nome</th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Tipo</th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Status</th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Ações</th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {identityProviders.length === 0 && (
                <tr><td colSpan={4} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">Nenhum provedor de identidade configurado.</td></tr>
              )}
              {identityProviders.map((provider) => (
                <tr key={provider.id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">{provider.name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-300">{provider.provider_type}</td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                      provider.is_active ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' : 'bg-red-100 text-red-800 dark:bg-red-700 dark:text-red-100'
                    }`}>
                      {provider.is_active ? 'Ativo' : 'Inativo'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                    <button onClick={() => handleEditProvider(provider)} className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-200">Editar</button>
                    <button onClick={() => handleDeleteProvider(provider.id)} className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-200">Deletar</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Modal para Adicionar/Editar Provedor */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 transition-opacity duration-300 ease-in-out">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
            <h2 className="text-2xl font-bold text-gray-800 dark:text-white mb-6">
              {editingProvider ? 'Editar' : 'Adicionar Novo'} Provedor de Identidade
            </h2>
            <form onSubmit={handleSaveProvider} className="space-y-4">
              <div>
                <label htmlFor="name" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Nome do Provedor</label>
                <input type="text" name="name" id="name" defaultValue={editingProvider?.name} required className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white"/>
              </div>
              <div>
                <label htmlFor="provider_type" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Tipo</label>
                <select name="provider_type" id="provider_type" defaultValue={editingProvider?.provider_type} required className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 dark:bg-gray-700 dark:border-gray-600 dark:text-white">
                  <option value="saml">SAML 2.0</option>
                  <option value="oauth2_google">OAuth2 (Google)</option>
                  <option value="oauth2_github">OAuth2 (GitHub)</option>
                  {/* Adicionar outros tipos aqui */}
                </select>
              </div>
              <div>
                <label htmlFor="config_json" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Configuração JSON</label>
                <textarea name="config_json" id="config_json" rows={5} defaultValue={editingProvider ? JSON.stringify(editingProvider.config_json, null, 2) : ''} required className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 font-mono text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white" placeholder='Ex: {"client_id": "...", "client_secret": "..."}'></textarea>
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">Para SAML: idp_entity_id, idp_sso_url, idp_x509_cert. Para OAuth2: client_id, client_secret, scopes (opcional).</p>
              </div>
               <div>
                <label htmlFor="attribute_mapping_json" className="block text-sm font-medium text-gray-700 dark:text-gray-200">Mapeamento de Atributos JSON (Opcional)</label>
                <textarea name="attribute_mapping_json" id="attribute_mapping_json" rows={3} defaultValue={editingProvider && editingProvider.attribute_mapping_json ? JSON.stringify(editingProvider.attribute_mapping_json, null, 2) : ''} className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 font-mono text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white" placeholder='Ex: {"email": "userPrincipalName"}'></textarea>
              </div>
              <div className="flex items-center">
                <input type="checkbox" name="is_active" id="is_active" defaultChecked={editingProvider ? editingProvider.is_active : true} className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"/>
                <label htmlFor="is_active" className="ml-2 block text-sm text-gray-900 dark:text-gray-200">Ativo</label>
              </div>
              <div className="flex justify-end space-x-3 pt-4">
                <button type="button" onClick={() => setShowModal(false)} className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 dark:bg-gray-600 dark:text-gray-200 dark:hover:bg-gray-500 rounded-md shadow-sm">Cancelar</button>
                <button type="submit" className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md shadow-sm">Salvar Provedor</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </AdminLayout>
  );
}
