import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { useRouter } from 'next/router';
import { AuthContext } from '@/contexts/AuthContext';
import IdentityProvidersPageContent from '../index'; // O componente real, não o default exportado com WithAuth
import apiClient from '@/lib/axios';
import '@testing-library/jest-dom';

// Mocks
jest.mock('next/router', () => ({
  useRouter: jest.fn(() => ({ push: jest.fn(), query: {} })),
}));
jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

jest.mock('@/components/auth/WithAuth', () => (WrappedComponent: React.ComponentType) => (props: any) => <WrappedComponent {...props} />);
// Mock IdentityProviderForm para simplificar o teste da página de listagem
jest.mock('@/components/admin/IdentityProviderForm', () => {
    // eslint-disable-next-line react/display-name
    return (props: any) => (
        <div data-testid="mock-idp-form">
            <span>Formulário de IdP (Mock)</span>
            <button onClick={props.onClose}>Fechar Mock Form</button>
            {/* Simular um submit de sucesso para testar o callback */}
            <button onClick={() => props.onSubmitSuccess({id: 'mock-idp-saved'})}>Salvar Mock IdP</button>
        </div>
    );
});


describe('IdentityProvidersPageContent', () => {
  const mockUser = { id: 'admin1', name: 'Admin User', email: 'admin@example.com', role: 'admin', organization_id: 'org123' };
  const mockAuthContextValue = {
    isAuthenticated: true, user: mockUser, token: 'fake-token', isLoading: false, login: jest.fn(), logout: jest.fn(),
  };

  const mockIdpsAPI = [
    { id: 'idp1', name: 'Okta Test', provider_type: 'saml', is_active: true, config_json: '{"entity_id":"okta"}', attribute_mapping_json: '{}' },
    { id: 'idp2', name: 'Google Test', provider_type: 'oauth2_google', is_active: false, config_json: '{"client_id":"google"}', attribute_mapping_json: '{}' },
  ];

  beforeEach(() => {
    (useRouter as jest.Mock).mockReturnValue({ push: jest.fn(), query: {} });
    mockedApiClient.get.mockReset();
    mockedApiClient.post.mockReset();
    mockedApiClient.put.mockReset();
    mockedApiClient.delete.mockReset();
    window.alert = jest.fn();
    window.confirm = jest.fn(() => true); // Default confirm to true
  });

  const renderPage = () => render(
    <AuthContext.Provider value={mockAuthContextValue}>
      <IdentityProvidersPageContent />
    </AuthContext.Provider>
  );

  it('renders loading state and then fetches and displays IdPs', async () => {
    mockedApiClient.get.mockResolvedValue({ data: mockIdpsAPI });
    renderPage();
    expect(screen.getByText(/Carregando provedores.../i)).toBeInTheDocument();
    expect(await screen.findByText('Okta Test')).toBeInTheDocument();
    expect(screen.getByText('Google Test')).toBeInTheDocument();
    expect(screen.getByText('saml')).toBeInTheDocument();
    expect(screen.getByText('oauth2_google')).toBeInTheDocument();
    expect(mockedApiClient.get).toHaveBeenCalledWith(`/organizations/${mockUser.organization_id}/identity-providers`);
  });

  it('opens modal to add new IdP and calls create API on form submit', async () => {
    mockedApiClient.get.mockResolvedValue({ data: [] }); // Sem IdPs inicialmente
    mockedApiClient.post.mockResolvedValue({ data: { id: 'new-idp', name: 'New SAML IdP' }}); // Resposta do POST

    renderPage();
    expect(await screen.findByText(/Nenhum provedor de identidade configurado./i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Adicionar Novo Provedor/i }));
    expect(await screen.findByTestId('mock-idp-form')).toBeInTheDocument();

    // Simular submissão do form mockado
    fireEvent.click(screen.getByRole('button', { name: /Salvar Mock IdP/i }));

    // O IdentityProviderForm real chamaria apiClient.post, que é mockado.
    // Aqui, o callback onSubmitSuccess (que é handleSaveProvider) é chamado.
    // handleSaveProvider então chama fetchIdentityProviders.
    await waitFor(() => {
        expect(mockedApiClient.get).toHaveBeenCalledTimes(2); // 1 inicial, 1 após "salvar"
    });
  });

  it('opens modal with data to edit IdP', async () => {
    mockedApiClient.get.mockResolvedValue({ data: mockIdpsAPI });
    renderPage();
    expect(await screen.findByText('Okta Test')).toBeInTheDocument();

    const editButtons = screen.getAllByRole('button', { name: /Editar/i });
    fireEvent.click(editButtons[0]); // Editar o primeiro (Okta Test)

    expect(await screen.findByTestId('mock-idp-form')).toBeInTheDocument();
    // No IdentityProviderForm real, os initialData seriam passados e preencheriam os campos.
    // Nosso mock do form não faz isso, mas podemos verificar que o modal abriu.
  });

  it('calls delete API and refetches list on successful deletion', async () => {
    mockedApiClient.get.mockResolvedValueOnce({ data: mockIdpsAPI }); // Fetch inicial
    mockedApiClient.delete.mockResolvedValue({}); // Mock da deleção
    mockedApiClient.get.mockResolvedValueOnce({ data: [mockIdpsAPI[1]] }); // Fetch após deleção

    renderPage();
    expect(await screen.findByText('Okta Test')).toBeInTheDocument();

    const deleteButtons = screen.getAllByRole('button', { name: /Deletar/i });
    fireEvent.click(deleteButtons[0]); // Deletar "Okta Test"

    expect(window.confirm).toHaveBeenCalledWith('Tem certeza que deseja remover o provedor de identidade "Okta Test"? Esta ação não pode ser desfeita.');
    expect(mockedApiClient.delete).toHaveBeenCalledWith(`/organizations/${mockUser.organization_id}/identity-providers/idp1`);

    await waitFor(() => {
      expect(screen.queryByText('Okta Test')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Google Test')).toBeInTheDocument();
    expect(mockedApiClient.get).toHaveBeenCalledTimes(2);
  });
});
