import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { useRouter } from 'next/router';
import { AuthContext } from '@/contexts/AuthContext';
import AuditFrameworksPageContent from '../index';
import apiClient from '@/lib/axios';
import '@testing-library/jest-dom';

jest.mock('next/router', () => ({
  useRouter: jest.fn(() => ({ push: jest.fn() })),
}));
jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;
jest.mock('@/components/auth/WithAuth', () => (WrappedComponent: React.ComponentType) => (props: any) => <WrappedComponent {...props} />);

describe('AuditFrameworksPageContent', () => {
  const mockUser = { id: 'u1', name: 'Test User', email: 'u@e.com', role: 'admin', organization_id: 'org1' };
  const mockAuthContext = { isAuthenticated: true, user: mockUser, token: 'token', isLoading: false, login: jest.fn(), logout: jest.fn() };

  const mockFrameworks = [
    { id: 'fw1', name: 'NIST Test Framework', created_at: '', updated_at: '' },
    { id: 'fw2', name: 'ISO Test Framework', created_at: '', updated_at: '' },
  ];

  beforeEach(() => {
    mockedApiClient.get.mockReset();
  });

  const renderPage = () => render(
    <AuthContext.Provider value={mockAuthContext}>
      <AuditFrameworksPageContent />
    </AuthContext.Provider>
  );

  it('renders loading state initially', () => {
    mockedApiClient.get.mockReturnValue(new Promise(() => {})); // Promessa que nunca resolve
    renderPage();
    expect(screen.getByText(/Carregando frameworks.../i)).toBeInTheDocument();
  });

  it('fetches and displays frameworks', async () => {
    mockedApiClient.get.mockResolvedValue({ data: mockFrameworks });
    renderPage();
    expect(await screen.findByText('NIST Test Framework')).toBeInTheDocument();
    expect(screen.getByText('ISO Test Framework')).toBeInTheDocument();
    expect(mockedApiClient.get).toHaveBeenCalledWith('/audit/frameworks');
  });

  it('displays error message on API failure', async () => {
    mockedApiClient.get.mockRejectedValue({ response: { data: { error: 'API Error' } } });
    renderPage();
    expect(await screen.findByText(/Erro ao carregar frameworks: API Error/i)).toBeInTheDocument();
  });

  it('displays message when no frameworks are available', async () => {
    mockedApiClient.get.mockResolvedValue({ data: [] });
    renderPage();
    expect(await screen.findByText(/Nenhum framework de auditoria carregado ou disponível./i)).toBeInTheDocument();
  });

  it('framework cards link to correct detail page', async () => {
    mockedApiClient.get.mockResolvedValue({ data: mockFrameworks });
    renderPage();

    const nistLink = await screen.findByText('NIST Test Framework');
    // O link é o elemento pai 'a' do heading 'h2' que contém o texto.
    expect(nistLink.closest('a')).toHaveAttribute('href', '/admin/audit/frameworks/fw1');
  });
});
