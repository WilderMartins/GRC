import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { useRouter } from 'next/router';
import { AuthContext } from '@/contexts/AuthContext'; // Ajuste o path
import VulnerabilitiesPageContent from '../index'; // O componente real
import apiClient from '@/lib/axios';
import '@testing-library/jest-dom';

// Mock Next.js Router
jest.mock('next/router', () => ({
  useRouter: jest.fn(() => ({
    push: jest.fn(),
    query: {},
  })),
}));

// Mock apiClient (axios)
jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

// Mock do WithAuth
jest.mock('@/components/auth/WithAuth', () => (WrappedComponent: React.ComponentType) => {
  // eslint-disable-next-line react/display-name
  return (props: any) => <WrappedComponent {...props} />;
});

describe('VulnerabilitiesPageContent', () => {
  const mockUser = { id: 'admin1', name: 'Admin User', email: 'admin@example.com', role: 'admin', organization_id: 'org1' };
  const mockAuthContextValue = {
    isAuthenticated: true,
    user: mockUser,
    token: 'fake-token',
    isLoading: false,
    login: jest.fn(),
    logout: jest.fn(),
  };

  const mockVulnerabilities = [
    { id: 'vuln1', title: 'Vuln Teste 1', cve_id: 'CVE-001', severity: 'Alto', status: 'descoberta', asset_affected: 'Servidor A' },
    { id: 'vuln2', title: 'Vuln Teste 2', severity: 'Médio', status: 'em_correcao', asset_affected: 'Desktop B' },
  ];

  const mockPaginatedResponse = {
    items: mockVulnerabilities,
    total_items: 2,
    total_pages: 1,
    page: 1,
    page_size: 10,
  };

  beforeEach(() => {
    (useRouter as jest.Mock).mockImplementation(() => ({ push: jest.fn(), query: {} }));
    mockedApiClient.get.mockReset();
    mockedApiClient.delete.mockReset();
    window.alert = jest.fn();
    window.confirm = jest.fn(() => true);
  });

  const renderVulnerabilitiesPage = () => {
    return render(
      <AuthContext.Provider value={mockAuthContextValue}>
        <VulnerabilitiesPageContent />
      </AuthContext.Provider>
    );
  };

  it('renders page title and add new vulnerability button', async () => {
    mockedApiClient.get.mockResolvedValue({ data: { ...mockPaginatedResponse, items: [] } });
    renderVulnerabilitiesPage();
    expect(screen.getByRole('heading', { name: /Gestão de Vulnerabilidades/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Adicionar Nova Vulnerabilidade/i })).toBeInTheDocument();
    await waitFor(() => expect(mockedApiClient.get).toHaveBeenCalledTimes(1));
  });

  it('fetches and displays vulnerabilities in a table', async () => {
    mockedApiClient.get.mockResolvedValue({ data: mockPaginatedResponse });
    renderVulnerabilitiesPage();

    expect(screen.getByText(/Carregando vulnerabilidades.../i)).toBeInTheDocument();
    expect(await screen.findByText('Vuln Teste 1')).toBeInTheDocument();
    expect(screen.getByText('Vuln Teste 2')).toBeInTheDocument();
    expect(screen.getByText('CVE-001')).toBeInTheDocument();
    expect(screen.getByText('Alto')).toBeInTheDocument();

    const editButtons = screen.getAllByRole('link', { name: /Editar/i });
    expect(editButtons.length).toBe(mockVulnerabilities.length);
    const deleteButtons = screen.getAllByRole('button', { name: /Deletar/i });
    expect(deleteButtons.length).toBe(mockVulnerabilities.length);

    expect(mockedApiClient.get).toHaveBeenCalledWith('/vulnerabilities', { params: { page: 1, page_size: 10 } });
  });

  it('calls delete API and refetches vulnerabilities on successful deletion', async () => {
    mockedApiClient.get.mockResolvedValueOnce({ data: mockPaginatedResponse }); // Fetch inicial
    mockedApiClient.delete.mockResolvedValue({}); // Mock da deleção
    mockedApiClient.get.mockResolvedValueOnce({ data: { ...mockPaginatedResponse, items: [mockVulnerabilities[1]] } }); // Fetch após deleção

    renderVulnerabilitiesPage();

    expect(await screen.findByText('Vuln Teste 1')).toBeInTheDocument();

    const deleteButtons = screen.getAllByRole('button', { name: /Deletar/i });
    fireEvent.click(deleteButtons[0]);

    expect(window.confirm).toHaveBeenCalledWith('Tem certeza que deseja deletar a vulnerabilidade "Vuln Teste 1"? Esta ação não pode ser desfeita.');
    expect(mockedApiClient.delete).toHaveBeenCalledWith('/vulnerabilities/vuln1');

    await waitFor(() => {
      expect(screen.queryByText('Vuln Teste 1')).not.toBeInTheDocument();
    });
    expect(screen.getByText('Vuln Teste 2')).toBeInTheDocument();
    expect(window.alert).toHaveBeenCalledWith('Vulnerabilidade "Vuln Teste 1" deletada com sucesso.');
    expect(mockedApiClient.get).toHaveBeenCalledTimes(2);
  });

  // TODO: Testar funcionalidade de paginação
});
