import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import LoginPage from '../login'; // A página que estamos testando
import { AuthProvider } from '../../../contexts/AuthContext'; // Para envolver o componente
import apiClient from '../../../lib/axios'; // Para mockar chamadas de API

// Mocking next/router
jest.mock('next/router', () => ({
  useRouter() {
    return {
      route: '/auth/login',
      pathname: '/auth/login',
      query: '',
      asPath: '/auth/login',
      push: jest.fn(), // Mock da função push
    };
  },
}));

// Mocking next-i18next
// O HOC appWithTranslation e serverSideTranslations são mais complexos de mockar diretamente
// para testes unitários de página. Em vez disso, mockamos useTranslation.
// Para serverSideTranslations, geralmente não o testamos no teste unitário do componente,
// mas confiamos que ele funciona conforme a documentação do next-i18next.
// A página LoginPage espera props de getStaticProps, então precisamos simular isso.
jest.mock('next-i18next', () => ({
  useTranslation: () => {
    return {
      t: (key: string, options?: { message?: string }) => {
        // Retornar chaves com placeholders para interpolação, se necessário
        if (options?.message) {
          return `${key.replace('{{message}}', options.message)}`;
        }
        // Mapear chaves para textos de teste simples
        const translations: Record<string, string> = {
          'common:app_name': 'Phoenix GRC Test',
          'common:app_title_login': 'Login - Phoenix GRC Test',
          'common:loading_options': 'Loading options...',
          'common:unknown_error': 'An unknown error occurred.',
          'login.title': 'Login Test',
          'login.welcome_message': 'Welcome back Test!',
          'login.email_label': 'Email Address Test',
          'login.email_placeholder': 'you@example.com',
          'login.password_label': 'Password Test',
          'login.password_placeholder': 'Your password',
          'login.remember_me_label': 'Remember me Test',
          'login.forgot_password_link': 'Forgot your password? Test',
          'login.submit_button': 'Sign In Test',
          'login.no_account_prompt': "Don't have an account? Test",
          'login.register_link': 'Register here Test',
          'login.sso_divider_text': 'OR Test',
          'login.error_loading_sso': 'Failed to load SSO options Test. Traditional login is still available.',
          'login.error_login_failed': 'Login failed: {message}', // Suporta interpolação
          'login.error_unexpected_response': 'Login failed: Unexpected response from server Test.',
          'login.error_email_password_required': 'Email and password are required Test.',
        };
        return translations[key] || key; // Retorna a chave se não houver tradução mockada
      },
      i18n: {
        language: 'pt', // Idioma padrão para o teste
        changeLanguage: jest.fn(),
      },
    };
  },
  // serverSideTranslations: jest.fn().mockResolvedValue({ _nextI18Next: { initialLocale: 'pt', userConfig: { i18n: { defaultLocale: 'pt', locales: ['pt', 'en', 'es']}} } }),
}));

// Mock apiClient (axios instance)
jest.mock('../../../lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

// Mock AuthContext
const mockLogin = jest.fn();
const mockUseAuth = jest.fn(() => ({
  isAuthenticated: false,
  user: null,
  login: mockLogin,
  logout: jest.fn(),
  isLoading: false,
}));

jest.mock('../../../contexts/AuthContext', () => ({
  useAuth: () => mockUseAuth(),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));


// Helper para renderizar com AuthProvider, pois LoginPage usa useAuth
// E next-i18next geralmente é configurado em _app.tsx, mas para testes unitários de página
// precisamos garantir que o contexto de tradução esteja disponível.
// O mock de useTranslation acima já lida com isso para os textos.
const renderLoginPage = () => {
  // A página LoginPage espera props de getStaticProps.
  // Como getStaticProps é chamado no lado do servidor/build, precisamos mockar as props que ela passaria.
  // Para este teste, as props de serverSideTranslations são internas ao next-i18next e não precisam ser
  // explicitamente passadas se useTranslation está mockado corretamente.
  // Passamos um objeto vazio como props por enquanto.
  return render(
    <AuthProvider>
      <LoginPage />
    </AuthProvider>
  );
};


describe('LoginPage', () => {
  beforeEach(() => {
    mockLogin.mockClear();
    mockedApiClient.get.mockClear();
    mockedApiClient.post.mockClear();
    // Reset window.location mocks if any were set by tests
    // @ts-ignore
    delete window.location;
    // @ts-ignore
    window.location = { href: '' } as Location;
  });

  test('renders initial loading state for SSO and login form', async () => {
    mockedApiClient.get.mockResolvedValueOnce({ data: [] }); // Para /api/public/auth/identity-providers
    renderLoginPage();

    expect(screen.getByText('login.title')).toBeInTheDocument(); // Traduzido
    expect(screen.getByLabelText('login.email_label')).toBeInTheDocument();
    expect(screen.getByLabelText('login.password_label')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'login.submit_button' })).toBeInTheDocument();
    expect(screen.getByText('common:loading_options')).toBeInTheDocument();

    await waitFor(() => expect(screen.queryByText('common:loading_options')).not.toBeInTheDocument());
  });

  describe('SSO Providers', () => {
    test('renders SSO provider buttons on successful fetch', async () => {
      const mockProviders = [
        { id: 'google1', name: 'Login com Google Test', type: 'oauth2_google', login_url: 'https://google.com/login' },
        { id: 'saml1', name: 'Login SAML Test', type: 'saml', login_url: 'https://saml.com/login' },
      ];
      mockedApiClient.get.mockResolvedValueOnce({ data: mockProviders });
      renderLoginPage();

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Login com Google Test' })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Login SAML Test' })).toBeInTheDocument();
      });
      expect(screen.queryByText('common:loading_options')).not.toBeInTheDocument();
      expect(screen.queryByText('login.error_loading_sso')).not.toBeInTheDocument();
    });

    test('handles click on SSO provider button by redirecting', async () => {
      const mockProviders = [
        { id: 'google1', name: 'Login com Google Test', type: 'oauth2_google', login_url: 'https://google.com/login' },
      ];
      mockedApiClient.get.mockResolvedValueOnce({ data: mockProviders });
      renderLoginPage();

      const googleButton = await screen.findByRole('button', { name: 'Login com Google Test' });
      fireEvent.click(googleButton);
      expect(window.location.href).toBe('https://google.com/login');
    });

    test('shows error message if fetching SSO providers fails', async () => {
      mockedApiClient.get.mockRejectedValueOnce(new Error('Network Error'));
      renderLoginPage();

      await waitFor(() => {
        expect(screen.getByText('login.error_loading_sso')).toBeInTheDocument();
      });
      expect(screen.queryByText('common:loading_options')).not.toBeInTheDocument();
    });
  });

  describe('Traditional Login', () => {
    test('shows validation error if email or password is not provided', async () => {
      mockedApiClient.get.mockResolvedValueOnce({ data: [] }); // Mock SSO fetch
      renderLoginPage();
      await waitFor(() => expect(screen.queryByText('common:loading_options')).not.toBeInTheDocument());

      const submitButton = screen.getByRole('button', { name: 'login.submit_button' });
      fireEvent.click(submitButton);

      expect(await screen.findByText('login.error_email_password_required')).toBeInTheDocument();
      expect(mockLogin).not.toHaveBeenCalled();
    });

    test('calls authContext.login on successful traditional login', async () => {
      mockedApiClient.get.mockResolvedValueOnce({ data: [] }); // Mock SSO fetch
      mockedApiClient.post.mockResolvedValueOnce({
        data: {
          token: 'fake-jwt-token',
          user_id: 'user123',
          name: 'Test User',
          email: 'test@example.com',
          role: 'user',
          organization_id: 'org123'
        }
      });
      renderLoginPage();
      await waitFor(() => expect(screen.queryByText('common:loading_options')).not.toBeInTheDocument());

      fireEvent.change(screen.getByLabelText('login.email_label'), { target: { value: 'test@example.com' } });
      fireEvent.change(screen.getByLabelText('login.password_label'), { target: { value: 'password123' } });
      fireEvent.click(screen.getByRole('button', { name: 'login.submit_button' }));

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledWith(
          expect.objectContaining({
            id: 'user123',
            name: 'Test User',
            email: 'test@example.com',
            role: 'user',
            organization_id: 'org123'
          }),
          'fake-jwt-token'
        );
      });
    });

    test('shows API error message on failed traditional login', async () => {
      mockedApiClient.get.mockResolvedValueOnce({ data: [] }); // Mock SSO fetch
      mockedApiClient.post.mockRejectedValueOnce({
        response: { data: { error: 'Invalid credentials' } }
      });
      renderLoginPage();
      await waitFor(() => expect(screen.queryByText('common:loading_options')).not.toBeInTheDocument());

      fireEvent.change(screen.getByLabelText('login.email_label'), { target: { value: 'test@example.com' } });
      fireEvent.change(screen.getByLabelText('login.password_label'), { target: { value: 'wrongpassword' } });
      fireEvent.click(screen.getByRole('button', { name: 'login.submit_button' }));

      expect(await screen.findByText('login.error_login_failed', { exact: false })).toBeInTheDocument();
      expect(screen.getByText(text => text.includes('Invalid credentials'))).toBeInTheDocument();
      expect(mockLogin).not.toHaveBeenCalled();
    });
  });
});
