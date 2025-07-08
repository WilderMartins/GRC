import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { useRouter } from 'next/router';
import { AuthContext } from '@/contexts/AuthContext';
import OrgWebhooksPageContent from '../index'; // O componente real
import apiClient from '@/lib/axios';
import '@testing-library/jest-dom';

// Mocks
jest.mock('next/router', () => ({
  useRouter: jest.fn(),
}));
jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

jest.mock('@/components/auth/WithAuth', () => (WrappedComponent: React.ComponentType) => (props: any) => <WrappedComponent {...props} />);
jest.mock('@/components/admin/WebhookForm', () => {
    // eslint-disable-next-line react/display-name
    return (props: any) => (
        <div data-testid="mock-webhook-form">
            <span>Formulário de Webhook (Mock)</span>
            <button onClick={props.onClose}>Fechar Mock Form</button>
            <button onClick={() => props.onSubmitSuccess({id: 'mock-webhook-saved'})}>Salvar Mock Webhook</button>
        </div>
    );
});


describe('OrgWebhooksPageContent', () => {
  const mockOrgId = 'org123';
  const mockUser = { id: 'admin1', name: 'Admin User', email: 'admin@example.com', role: 'admin', organization_id: mockOrgId };
  const mockAuthContextValue = {
    isAuthenticated: true, user: mockUser, token: 'fake-token', isLoading: false, login: jest.fn(), logout: jest.fn(),
  };

  const mockWebhooksAPI = [
    { id: 'wh1', name: 'Webhook Alpha', url: 'https://alpha.hook', event_types: 'risk_created', is_active: true },
    { id: 'wh2', name: 'Webhook Beta', url: 'https://beta.hook', event_types: 'risk_status_changed,risk_created', is_active: false },
  ];

  let mockRouterQuery: any;

  beforeEach(() => {
    mockRouterQuery = { orgId: mockOrgId };
    (useRouter as jest.Mock).mockReturnValue({ query: mockRouterQuery, isReady: true, push: jest.fn() });
    mockedApiClient.get.mockReset();
    mockedApiClient.delete.mockReset();
    window.alert = jest.fn();
    window.confirm = jest.fn(() => true);
  });

  const renderPage = () => render(
    <AuthContext.Provider value={mockAuthContextValue}>
      <OrgWebhooksPageContent />
    </AuthContext.Provider>
  );

  it('renders loading state and then fetches and displays webhooks', async () => {
    mockedApiClient.get.mockResolvedValue({ data: mockWebhooksAPI });
    renderPage();
    expect(screen.getByText(/Carregando webhooks.../i)).toBeInTheDocument();
    expect(await screen.findByText('Webhook Alpha')).toBeInTheDocument();
    expect(screen.getByText('Webhook Beta')).toBeInTheDocument();
    expect(screen.getByText('risk_created')).toBeInTheDocument(); // Do primeiro webhook
    // Verifica se ambos os eventos do segundo webhook são renderizados
    expect(screen.getByText('risk_status_changed')).toBeInTheDocument();

    expect(mockedApiClient.get).toHaveBeenCalledWith(`/organizations/${mockOrgId}/webhooks`);
  });

  it('opens modal to add new webhook and refetches on success', async () => {
    mockedApiClient.get.mockResolvedValue({ data: [] }); // Inicialmente sem webhooks

    renderPage();
    expect(await screen.findByText(/Nenhum webhook configurado./i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Adicionar Novo Webhook/i }));
    expect(await screen.findByTestId('mock-webhook-form')).toBeInTheDocument();

    // Simular submissão do form mockado
    fireEvent.click(screen.getByRole('button', { name: /Salvar Mock Webhook/i }));

    await waitFor(() => {
        expect(mockedApiClient.get).toHaveBeenCalledTimes(2); // 1 inicial, 1 após "salvar"
    });
  });

  it('calls delete API and refetches list on successful deletion', async () => {
    mockedApiClient.get.mockResolvedValueOnce({ data: mockWebhooksAPI }); // Fetch inicial
    mockedApiClient.delete.mockResolvedValue({}); // Mock da deleção
    mockedApiClient.get.mockResolvedValueOnce({ data: [mockWebhooksAPI[1]] }); // Fetch após deleção

    renderPage();
    expect(await screen.findByText('Webhook Alpha')).toBeInTheDocument();

    const deleteButtons = screen.getAllByRole('button', { name: /Deletar/i });
    fireEvent.click(deleteButtons[0]); // Deletar "Webhook Alpha"

    expect(window.confirm).toHaveBeenCalledWith('Tem certeza que deseja deletar o webhook "Webhook Alpha"? Esta ação não pode ser desfeita.');
    expect(mockedApiClient.delete).toHaveBeenCalledWith(`/organizations/${mockOrgId}/webhooks/wh1`);

    await waitFor(() => {
      expect(screen.queryByText('Webhook Alpha')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Webhook Beta')).toBeInTheDocument();
    expect(mockedApiClient.get).toHaveBeenCalledTimes(2);
  });

  it('shows error if orgId in URL does not match user orgId', async () => {
    mockRouterQuery = { orgId: 'anotherOrg123' }; // orgId diferente do usuário
     (useRouter as jest.Mock).mockReturnValue({ query: mockRouterQuery, isReady: true, push: jest.fn() });
    renderPage();
    expect(await screen.findByText(/Você não tem permissão para acessar as configurações desta organização./i)).toBeInTheDocument();
    expect(mockedApiClient.get).not.toHaveBeenCalled();
  });

});
