import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import FeatureTogglesPageContent from '../index'; // Ajustar path se necessário
import { AuthProvider, AuthContextType } from '@/contexts/AuthContext';
import { FeatureToggleProvider, FeatureToggleContextType } from '@/contexts/FeatureToggleContext';
import apiClient from '@/lib/axios';
import { FeatureToggle } from '@/types';
import { I18nextProvider } from 'react-i18next';
import i18n from '@/../test/i18n-test-config'; // Supondo uma config de i18n para testes

jest.mock('@/lib/axios');
const mockedApiClientGet = apiClient.get as jest.Mock;
const mockedApiClientPut = apiClient.put as jest.Mock;

// Mock useRouter
jest.mock('next/router', () => ({
  useRouter: () => ({
    push: jest.fn(),
    query: {},
    pathname: '/admin/feature-toggles',
    isReady: true,
  }),
}));

// Mock useNotifier
const mockNotifySuccess = jest.fn();
const mockNotifyError = jest.fn();
jest.mock('@/hooks/useNotifier', () => ({
  useNotifier: () => ({
    success: mockNotifySuccess,
    error: mockNotifyError,
  }),
}));

const mockAdminUser = { id: 'admin1', name: 'Admin User', email: 'admin@example.com', role: 'admin', organization_id: 'org1' };
const mockNonAdminUser = { ...mockAdminUser, role: 'user' };

const mockFeatureToggles: FeatureToggle[] = [
  { key: 'featureOne', description: 'Description for One', is_active: true, read_only: false },
  { key: 'featureTwo', description: 'Description for Two', is_active: false, read_only: false },
  { key: 'featureReadOnly', description: 'Read Only Feature', is_active: true, read_only: true },
];

const renderPage = (
    user: AuthContextType['user'],
    initialToggles?: FeatureToggle[]
) => {
  const authContextValue: AuthContextType = {
    isAuthenticated: !!user, user, token: user ? 'fake-token' : null, branding: {}, isLoading: false,
    login: jest.fn(() => Promise.resolve()), logout: jest.fn(),
    refreshBranding: jest.fn(() => Promise.resolve()), refreshUser: jest.fn(() => Promise.resolve()),
  };

  // FeatureToggleProvider mock - o provider real faz fetch, aqui controlamos os toggles iniciais
  const featureToggleContextValue: FeatureToggleContextType = {
    toggles: initialToggles ? initialToggles.reduce((acc, t) => { acc[t.key] = t; return acc; }, {} as Record<string, FeatureToggle>) : {},
    isLoadingToggles: false,
    isFeatureEnabled: (key: string) => !!initialToggles?.find(t => t.key === key)?.is_active,
    getFeatureToggle: (key: string) => initialToggles?.find(t => t.key === key),
    refreshToggles: jest.fn(() => Promise.resolve()),
  };

  // Mock da API para a busca inicial de feature toggles pela página
  if (user?.role === 'admin') { // Apenas mockar se o admin for renderizar a tabela
      mockedApiClientGet.mockResolvedValue({ data: initialToggles || [] });
  }


  return render(
    <AuthProvider value={authContextValue}>
      <FeatureToggleProvider value={featureToggleContextValue}> {/* Pode ser necessário mockar o provider para controlar os toggles ou deixar o provider real fazer o fetch mockado */}
        <I18nextProvider i18n={i18n}>
          <FeatureTogglesPageContent />
        </I18nextProvider>
      </FeatureToggleProvider>
    </AuthProvider>
  );
};


describe('FeatureTogglesPageContent', () => {
  beforeEach(() => {
    mockedApiClientGet.mockClear();
    mockedApiClientPut.mockClear();
    mockNotifySuccess.mockClear();
    mockNotifyError.mockClear();
  });

  it('shows insufficient permissions message for non-admin users', () => {
    renderPage(mockNonAdminUser);
    expect(screen.getByText('common:error_insufficient_permissions')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('shows loading state initially for admin users', () => {
    mockedApiClientGet.mockImplementation(() => new Promise(() => {})); // Promessa que nunca resolve para simular loading
    renderPage(mockAdminUser);
    expect(screen.getByText('featureToggles:table_placeholder_loading')).toBeInTheDocument();
  });

  it('fetches and displays feature toggles for admin users', async () => {
    renderPage(mockAdminUser, mockFeatureToggles);

    expect(await screen.findByRole('table')).toBeInTheDocument();
    expect(screen.getByText('featureOne')).toBeInTheDocument();
    expect(screen.getByText('Description for Two')).toBeInTheDocument();
    expect(screen.getByText('featureToggles:status_active')).toBeInTheDocument(); // Para featureOne
    // O Switch para featureOne deve estar checado
  });

  it('displays read-only text for read-only toggles', async () => {
    renderPage(mockAdminUser, mockFeatureToggles);
    expect(await screen.findByText('featureReadOnly')).toBeInTheDocument();
    // O elemento que contém "Read Only" pode ser um span ou td específico.
    const readOnlyRow = screen.getByText('featureReadOnly').closest('tr');
    expect(readOnlyRow).toHaveTextContent('featureToggles:read_only_label');
    expect(readOnlyRow?.querySelector('button[role="switch"]')).toBeNull(); // Não deve haver switch
  });

  it('allows toggling a non-read-only feature and calls API (optimistic update)', async () => {
    renderPage(mockAdminUser, mockFeatureToggles);
    const featureOneRow = (await screen.findByText('featureOne')).closest('tr');
    const toggleSwitch = featureOneRow?.querySelector('button[role="switch"]') as HTMLElement;

    expect(toggleSwitch).toBeInTheDocument();
    // Supondo que 'featureOne' is_active: true inicialmente, o switch está checado.
    // fireEvent.click(toggleSwitch); // Para desativar

    // Para testar a mudança de false para true para 'featureTwo'
    const featureTwoRow = screen.getByText('featureTwo').closest('tr');
    const toggleSwitchTwo = featureTwoRow?.querySelector('button[role="switch"]') as HTMLElement;

    mockedApiClientPut.mockResolvedValue({ data: { ...mockFeatureToggles.find(t=>t.key==='featureTwo'), is_active: true } });

    fireEvent.click(toggleSwitchTwo); // Ativa featureTwo

    // Verificar atualização otimista (difícil de testar diretamente o estado interno sem expor)
    // Mas podemos verificar se a API foi chamada.
    await waitFor(() => {
      expect(mockedApiClientPut).toHaveBeenCalledWith('/feature-toggles/featureTwo', { is_active: true });
    });
    expect(mockNotifySuccess).toHaveBeenCalledWith('featureToggles:update_success');

    // Se a API falhasse, testar a reversão (mais complexo)
  });

  it('handles API error when fetching toggles', async () => {
    mockedApiClientGet.mockRejectedValue({ response: { data: { error: 'API Error' } } });
    renderPage(mockAdminUser);
    expect(await screen.findByText(/featureToggles:error_loading_toggles/i)).toBeInTheDocument();
    expect(screen.getByText(/API Error/i)).toBeInTheDocument();
  });
});
