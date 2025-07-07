import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { useRouter } from 'next/router';
import { AuthContext } from '@/contexts/AuthContext'; // Ajuste o path
import RisksPageContent from '../index'; // O componente real, não o default exportado com WithAuth
import apiClient from '@/lib/axios'; // Para mockar apiClient
import '@testing-library/jest-dom';

// Mock Next.js Router
jest.mock('next/router', () => ({
  useRouter: jest.fn(() => ({
    push: jest.fn(),
    query: {}, // Mock query params se necessário para filtros/paginaçao inicial
  })),
}));

// Mock apiClient (axios)
jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

// Mock do WithAuth para não interferir nos testes do conteúdo da página
jest.mock('@/components/auth/WithAuth', () => (WrappedComponent: React.ComponentType) => {
  // eslint-disable-next-line react/display-name
  return (props: any) => <WrappedComponent {...props} />;
});


describe('RisksPageContent', () => {
  const mockUser = { id: 'admin1', name: 'Admin User', email: 'admin@example.com', role: 'admin', organization_id: 'org1' };
  const mockAuthContextValue = {
    isAuthenticated: true,
    user: mockUser,
    token: 'fake-token',
    isLoading: false,
    login: jest.fn(),
    logout: jest.fn(),
  };

  const mockRisks = [
    { id: 'risk1', title: 'Risco Teste 1', category: 'tecnologico', impact: 'Alto', probability: 'Médio', status: 'aberto', owner_id: 'user1', owner: { name: 'Owner 1' } },
    { id: 'risk2', title: 'Risco Teste 2', category: 'operacional', impact: 'Baixo', probability: 'Baixo', status: 'mitigado', owner_id: 'user2', owner: { name: 'Owner 2' }  },
  ];

  const mockPaginatedResponse = {
    items: mockRisks,
    total_items: 2,
    total_pages: 1,
    page: 1,
    page_size: 10,
  };

  beforeEach(() => {
    // Reset mocks antes de cada teste
    (useRouter as jest.Mock).mockImplementation(() => ({
        push: jest.fn(),
        query: {},
    }));
    mockedApiClient.get.mockReset();
    mockedApiClient.delete.mockReset(); // Se for testar delete
    window.alert = jest.fn(); // Mock window.alert
    window.confirm = jest.fn(() => true); // Mock window.confirm para retornar true por padrão
  });

  const renderRisksPage = () => {
    return render(
      <AuthContext.Provider value={mockAuthContextValue}>
        <RisksPageContent />
      </AuthContext.Provider>
    );
  };

  it('renders page title and add new risk button', async () => {
    mockedApiClient.get.mockResolvedValue({ data: { ...mockPaginatedResponse, items: [] } }); // Retorna lista vazia inicialmente
    renderRisksPage();
    expect(screen.getByRole('heading', { name: /Gestão de Riscos/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Adicionar Novo Risco/i })).toBeInTheDocument();
    await waitFor(() => expect(mockedApiClient.get).toHaveBeenCalledTimes(1)); // Espera a chamada inicial
  });

  it('fetches and displays risks in a table', async () => {
    mockedApiClient.get.mockResolvedValue({ data: mockPaginatedResponse });
    renderRisksPage();

    expect(screen.getByText(/Carregando riscos.../i)).toBeInTheDocument(); // Estado inicial de loading

    // Espera pelos riscos serem renderizados
    expect(await screen.findByText('Risco Teste 1')).toBeInTheDocument();
    expect(screen.getByText('Risco Teste 2')).toBeInTheDocument();
    expect(screen.getByText('tecnologico')).toBeInTheDocument();
    expect(screen.getByText('Alto')).toBeInTheDocument(); // Impacto do Risco 1
    expect(screen.getByText('Owner 1')).toBeInTheDocument(); // Nome do Owner

    // Verifica se os botões de ação estão lá para cada risco
    const editButtons = screen.getAllByRole('link', { name: /Editar/i });
    expect(editButtons.length).toBe(mockRisks.length);
    const deleteButtons = screen.getAllByRole('button', { name: /Deletar/i });
    expect(deleteButtons.length).toBe(mockRisks.length);

    expect(mockedApiClient.get).toHaveBeenCalledWith('/risks', { params: { page: 1, page_size: 10 } });
  });

  it('displays message when no risks are found', async () => {
    mockedApiClient.get.mockResolvedValue({ data: { ...mockPaginatedResponse, items: [], total_items: 0, total_pages: 0 } });
    renderRisksPage();
    expect(await screen.findByText(/Nenhum risco encontrado./i)).toBeInTheDocument();
  });

  it('handles API error when fetching risks', async () => {
    const errorMessage = "Falha na API";
    mockedApiClient.get.mockRejectedValue({ response: { data: { error: errorMessage } } });
    renderRisksPage();
    expect(await screen.findByText(`Erro ao carregar riscos: ${errorMessage}`)).toBeInTheDocument();
  });

  it('navigates to new risk page when "Adicionar Novo Risco" is clicked', async () => {
    mockedApiClient.get.mockResolvedValue({ data: { ...mockPaginatedResponse, items: [] } });
    const mockRouter = { push: jest.fn(), query: {} };
    (useRouter as jest.Mock).mockReturnValue(mockRouter);

    renderRisksPage();
    await waitFor(() => expect(mockedApiClient.get).toHaveBeenCalled()); // Garante que o fetch inicial ocorreu

    const addButton = screen.getByRole('link', { name: /Adicionar Novo Risco/i });
    fireEvent.click(addButton);
    expect(mockRouter.push).toHaveBeenCalledWith('/admin/risks/new');
  });

  it('calls delete API and refetches risks on successful deletion', async () => {
    mockedApiClient.get.mockResolvedValueOnce({ data: mockPaginatedResponse }); // Fetch inicial
    mockedApiClient.delete.mockResolvedValue({}); // Mock da deleção
    mockedApiClient.get.mockResolvedValueOnce({ data: { ...mockPaginatedResponse, items: [mockRisks[1]] } }); // Fetch após deleção

    renderRisksPage();

    // Espera o primeiro risco aparecer
    expect(await screen.findByText('Risco Teste 1')).toBeInTheDocument();

    const deleteButtons = screen.getAllByRole('button', { name: /Deletar/i });
    fireEvent.click(deleteButtons[0]); // Clica em deletar o primeiro risco

    expect(window.confirm).toHaveBeenCalledWith('Tem certeza que deseja deletar o risco "Risco Teste 1"? Esta ação não pode ser desfeita.');
    expect(mockedApiClient.delete).toHaveBeenCalledWith('/risks/risk1');

    // Espera a UI atualizar (Risco Teste 1 removido)
    await waitFor(() => {
      expect(screen.queryByText('Risco Teste 1')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Risco Teste 2')).toBeInTheDocument(); // O segundo risco ainda deve estar lá
    expect(window.alert).toHaveBeenCalledWith('Risco "Risco Teste 1" deletado com sucesso.');
    expect(mockedApiClient.get).toHaveBeenCalledTimes(2); // Chamada inicial + chamada após deleção
  });

  // TODO: Testar funcionalidade de paginação (clicar nos botões "Anterior"/"Próxima")
});
