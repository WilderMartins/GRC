import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { useRouter } from 'next/router';
import { AuthContext } from '@/contexts/AuthContext';
import FrameworkDetailPageContent from '../index'; // Ajuste para o caminho correto
import apiClient from '@/lib/axios';
import '@testing-library/jest-dom';

// Mocks
jest.mock('next/router', () => ({
  useRouter: jest.fn(),
}));
jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;
jest.mock('@/components/auth/WithAuth', () => (WrappedComponent: React.ComponentType) => (props: any) => <WrappedComponent {...props} />);
jest.mock('@/components/audit/AssessmentForm', () => () => <div data-testid="mock-assessment-form">Mock Assessment Form</div>);


describe('FrameworkDetailPageContent', () => {
  const mockUser = { id: 'u1', name: 'Test User', email: 'u@e.com', role: 'admin', organization_id: 'org123' };
  const mockAuthContext = { isAuthenticated: true, user: mockUser, token: 'token', isLoading: false, login: jest.fn(), logout: jest.fn() };
  const mockFrameworkId = 'fw-test-id';

  const mockFrameworksAPI = [{ id: mockFrameworkId, name: 'Test Framework Detailed' }];
  const mockControlsAPI = [
    { id: 'ctrl1', control_id: 'CTRL-01', description: 'Control 1 Desc', family: 'Family A' },
    { id: 'ctrl2', control_id: 'CTRL-02', description: 'Control 2 Desc', family: 'Family B' },
  ];
  const mockAssessmentsAPI = [
    { id: 'assess1', audit_control_id: 'ctrl1', status: 'conforme', score: 100, evidence_url: 'http://evi.dence/1' },
  ];

  let mockRouterQuery: any;

  beforeEach(() => {
    mockRouterQuery = { frameworkId: mockFrameworkId };
    (useRouter as jest.Mock).mockReturnValue({ query: mockRouterQuery, isReady: true, push: jest.fn() });
    mockedApiClient.get.mockReset();
    window.alert = jest.fn();
  });

  const renderPage = () => render(
    <AuthContext.Provider value={mockAuthContext}>
      <FrameworkDetailPageContent />
    </AuthContext.Provider>
  );

  it('renders loading state initially', () => {
    mockedApiClient.get.mockImplementation((url: string) => {
      if (url.includes('/audit/frameworks')) return new Promise(() => {}); // NIST, ISO etc.
      if (url.includes(`/controls`)) return new Promise(() => {}); // Controls for frameworkId
      if (url.includes(`/assessments`)) return new Promise(() => {}); // Assessments for orgId and frameworkId
      return Promise.reject(new Error("Unknown API call in loading test"));
    });
    renderPage();
    expect(screen.getByText(/Carregando dados do framework.../i)).toBeInTheDocument();
  });

  it('fetches and displays framework name, controls, and assessments', async () => {
    mockedApiClient.get.mockImplementation((url: string) => {
      if (url === '/audit/frameworks') return Promise.resolve({ data: mockFrameworksAPI });
      if (url === `/audit/frameworks/${mockFrameworkId}/controls`) return Promise.resolve({ data: mockControlsAPI });
      if (url === `/audit/organizations/${mockUser.organization_id}/frameworks/${mockFrameworkId}/assessments`) return Promise.resolve({ data: mockAssessmentsAPI });
      return Promise.reject(new Error(`Unexpected API call: ${url}`));
    });

    renderPage();

    expect(await screen.findByText('Test Framework Detailed')).toBeInTheDocument();
    expect(screen.getByText('CTRL-01')).toBeInTheDocument();
    expect(screen.getByText('Control 1 Desc')).toBeInTheDocument();
    expect(screen.getByText('CTRL-02')).toBeInTheDocument(); // Control without assessment

    // Check assessment status for CTRL-01
    expect(screen.getByText('conforme')).toBeInTheDocument();
    expect(screen.getByText('100')).toBeInTheDocument();

    // Check "Não Avaliado" for CTRL-02
    const control2Row = screen.getByText('CTRL-02').closest('tr');
    expect(control2Row).toHaveTextContent('Não Avaliado');

    // Check evidence link for CTRL-01
    const evidenceLink = screen.getByRole('link', { name: /Ver Evidência/i });
    expect(evidenceLink).toBeInTheDocument();
    expect(evidenceLink).toHaveAttribute('href', 'http://evi.dence/1');


    expect(mockedApiClient.get).toHaveBeenCalledWith('/audit/frameworks');
    expect(mockedApiClient.get).toHaveBeenCalledWith(`/audit/frameworks/${mockFrameworkId}/controls`);
    expect(mockedApiClient.get).toHaveBeenCalledWith(`/audit/organizations/${mockUser.organization_id}/frameworks/${mockFrameworkId}/assessments`);
  });

  it('displays error message on API failure', async () => {
    mockedApiClient.get.mockRejectedValue({ response: { data: { error: 'Failed to load details' } } });
    renderPage();
    expect(await screen.findByText(/Erro: Failed to load details/i)).toBeInTheDocument();
  });

  it('opens assessment modal when "Avaliar" button is clicked', async () => {
    mockedApiClient.get.mockImplementation((url: string) => {
      if (url === '/audit/frameworks') return Promise.resolve({ data: mockFrameworksAPI });
      if (url === `/audit/frameworks/${mockFrameworkId}/controls`) return Promise.resolve({ data: mockControlsAPI });
      if (url === `/audit/organizations/${mockUser.organization_id}/frameworks/${mockFrameworkId}/assessments`) return Promise.resolve({ data: [] }); // No assessments initially
      return Promise.reject(new Error(`Unexpected API call: ${url}`));
    });
    renderPage();

    // Wait for controls to load
    expect(await screen.findByText('CTRL-01')).toBeInTheDocument();

    // Find the "Avaliar" button for the first control (CTRL-01)
    const assessButtons = screen.getAllByRole('button', { name: /Avaliar/i });
    fireEvent.click(assessButtons[0]);

    // Check if the mocked AssessmentForm is now visible
    expect(screen.getByTestId('mock-assessment-form')).toBeInTheDocument();
  });

});
