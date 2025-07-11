import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import SetupWizardPage from '../index';
import apiClient from '@/lib/axios';
import { useRouter } from 'next/router';
import { AuthProvider } from '@/contexts/AuthContext'; // Necessário se SetupWizardPage usar useAuth indiretamente
import { ThemeProvider } from '@/contexts/ThemeContext';
import { FeatureToggleProvider } from '@/contexts/FeatureToggleContext';

// Mock apiClient
jest.mock('@/lib/axios');
const mockedApiClientGet = apiClient.get as jest.Mock;

// Mock useRouter
const mockRouterPush = jest.fn();
jest.mock('next/router', () => ({
  useRouter: () => ({
    push: mockRouterPush,
    query: {},
    pathname: '/setup', // Simular estar na página de setup
    isReady: true,
  }),
}));

// Mock useTranslation
jest.mock('next-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, options?: { [key: string]: any } | string) => {
      if (typeof options === 'string') return options; // Retorna o valor default
      if (options && typeof options === 'object' && options.defaultValue) return options.defaultValue;
      return key;
    },
  }),
  serverSideTranslations: jest.fn().mockResolvedValue({}), // Mock para getStaticProps
}));

// Mock useNotifier
jest.mock('@/hooks/useNotifier', () => ({
  useNotifier: () => ({
    success: jest.fn(),
    error: jest.fn(),
    info: jest.fn(),
    warn: jest.fn(),
  }),
}));

// Fornecer um wrapper com todos os contextos necessários
const AllProviders: React.FC<{children: React.ReactNode}> = ({ children }) => {
  // Mock básico para AuthContext, pode ser mais elaborado se necessário
  const mockAuthContextValue = {
    isAuthenticated: false,
    user: null,
    token: null,
    branding: {},
    isLoading: false,
    login: jest.fn(() => Promise.resolve()),
    logout: jest.fn(),
    refreshBranding: jest.fn(() => Promise.resolve()),
    refreshUser: jest.fn(() => Promise.resolve()),
  };
  return (
    <AuthProvider value={mockAuthContextValue}>
      <ThemeProvider>
        <FeatureToggleProvider>
          {children}
        </FeatureToggleProvider>
      </ThemeProvider>
    </AuthProvider>
  );
};


describe('SetupWizardPage - Initial Status Checks', () => {
  beforeEach(() => {
    mockRouterPush.mockClear();
    mockedApiClientGet.mockClear();
  });

  it('shows loading state initially then welcome step if API returns a generic non-completed status', async () => {
    mockedApiClientGet.mockResolvedValue({ data: { status: 'some_initial_unmapped_status' } });
    render(<SetupWizardPage />, { wrapper: AllProviders });

    expect(screen.getByText('common:loading_status_check')).toBeInTheDocument(); // Verifica o estado de loading

    // Espera que o WelcomeStep seja renderizado (pelo seu título, por exemplo)
    expect(await screen.findByRole('heading', { name: 'steps.welcome.title' })).toBeInTheDocument();
    expect(screen.queryByText('common:loading_status_check')).not.toBeInTheDocument();
  });

  it('redirects to /auth/login if setup status is "completed"', async () => {
    mockedApiClientGet.mockResolvedValue({ data: { status: 'completed' } });
    render(<SetupWizardPage />, { wrapper: AllProviders });

    await waitFor(() => {
      expect(mockRouterPush).toHaveBeenCalledWith('/auth/login');
    });
    // Poderia verificar se "completed_redirect" é mostrado brevemente
    // expect(screen.getByText('steps.completed.redirecting')).toBeInTheDocument();
  });

  it('goes to db_config_check step if API status is "database_not_configured"', async () => {
    mockedApiClientGet.mockResolvedValue({ data: { status: 'database_not_configured' } });
    render(<SetupWizardPage />, { wrapper: AllProviders });
    expect(await screen.findByRole('heading', { name: 'steps.db_config.title' })).toBeInTheDocument();
  });

  it('goes to migrations step if API status is "db_configured_pending_migrations"', async () => {
    mockedApiClientGet.mockResolvedValue({ data: { status: 'db_configured_pending_migrations' } });
    render(<SetupWizardPage />, { wrapper: AllProviders });
    expect(await screen.findByRole('heading', { name: 'steps.migrations.title' })).toBeInTheDocument();
  });

  it('goes to admin_creation step if API status is "migrations_done_pending_admin"', async () => {
    mockedApiClientGet.mockResolvedValue({ data: { status: 'migrations_done_pending_admin' } });
    render(<SetupWizardPage />, { wrapper: AllProviders });
    expect(await screen.findByRole('heading', { name: 'steps.admin_creation.title' })).toBeInTheDocument();
  });

  it('shows server error message if API call fails', async () => {
    mockedApiClientGet.mockRejectedValue(new Error('Network Error'));
    render(<SetupWizardPage />, { wrapper: AllProviders });
    expect(await screen.findByRole('heading', { name: 'steps.error.title' })).toBeInTheDocument();
    expect(screen.getByText('steps.error_fetching_status')).toBeInTheDocument();
  });
});

describe('SetupWizardPage - Step Transitions', () => {
    beforeEach(() => {
        mockRouterPush.mockClear();
        mockedApiClientGet.mockClear();
    });

    it('moves from welcome to db_config_check when onNext is called from WelcomeStep', async () => {
        // 1. API inicial retorna um status que leva a 'welcome'
        mockedApiClientGet.mockResolvedValueOnce({ data: { status: 'some_initial_status_leading_to_welcome' } });
        render(<SetupWizardPage />, { wrapper: AllProviders });

        const welcomeTitle = await screen.findByRole('heading', { name: 'steps.welcome.title' });
        expect(welcomeTitle).toBeInTheDocument();

        // 2. Simular clique no botão "Próximo" do WelcomeStep
        //    Isso chama goToNextStep, que seta currentStep para 'loading_status' e refaz o fetch.
        //    A segunda chamada à API deve retornar 'database_not_configured'.
        mockedApiClientGet.mockResolvedValueOnce({ data: { status: 'database_not_configured' } });
        const nextButton = screen.getByRole('button', { name: 'steps.welcome.start_button' });
        fireEvent.click(nextButton);

        expect(await screen.findByText('common:loading_status_check')).toBeInTheDocument(); // Mostra loading
        expect(await screen.findByRole('heading', { name: 'steps.db_config.title' })).toBeInTheDocument(); // Depois vai para db_config
        expect(mockedApiClientGet).toHaveBeenCalledTimes(2); // Uma na carga inicial, outra no goToNextStep
    });

    // Testes similares para outras transições (db_config -> migrations, migrations -> admin_creation, etc.)
    // Esses testes dependerão da implementação dos handlers de ação (handleRunMigrations, handleAdminCreationSubmit)
    // e das respostas mockadas das APIs POST.
});
