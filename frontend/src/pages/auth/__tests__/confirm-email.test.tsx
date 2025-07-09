import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import ConfirmEmailPage from '../confirm-email';
import apiClient from '../../../lib/axios';
import { useNotifier } from '../../../hooks/useNotifier';
import { AuthProvider, useAuth } from '../../../contexts/AuthContext'; // Importando useAuth para mock
import { User } from '@/types';

// Mocking next/router
const mockRouterPush = jest.fn();
let mockRouterQuery: { token?: string | string[] } = {};
let mockRouterIsReady: boolean = false;

jest.mock('next/router', () => ({
  useRouter: () => ({
    route: '/auth/confirm-email',
    pathname: '/auth/confirm-email',
    query: mockRouterQuery,
    asPath: `/auth/confirm-email${mockRouterQuery.token ? `?token=${mockRouterQuery.token}` : ''}`,
    isReady: mockRouterIsReady,
    push: mockRouterPush,
  }),
}));

// Mocking next-i18next
const mockT = jest.fn((key) => key);
jest.mock('next-i18next', () => ({
  useTranslation: () => ({
    t: mockT,
    i18n: { language: 'pt', changeLanguage: jest.fn() },
  }),
}));

// Mock apiClient (axios instance)
jest.mock('../../../lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

// Mock useNotifier
const mockNotifySuccess = jest.fn();
const mockNotifyError = jest.fn();
jest.mock('../../../hooks/useNotifier', () => ({
  useNotifier: () => ({
    success: mockNotifySuccess,
    error: mockNotifyError,
    info: jest.fn(),
    warn: jest.fn(),
  }),
}));

// Mock AuthContext
const mockAuthLogin = jest.fn();
jest.mock('../../../contexts/AuthContext', () => ({
  // Temos que mockar o useAuth retornado pelo AuthProvider também, não apenas o hook importado diretamente
  __esModule: true, // Necessário para mockar módulos com exports nomeados e default
  useAuth: () => ({ // O que o componente ConfirmEmailPage vai receber de useAuth()
    login: mockAuthLogin,
    // Incluir outros valores do contexto se o componente os usar diretamente
    isAuthenticated: false,
    user: null,
    isLoading: false,
    logout: jest.fn(),
  }),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));


const renderConfirmEmailPage = () => {
  return render(
    <AuthProvider> {/* Envolver com AuthProvider para que useAuth funcione */}
      <ConfirmEmailPage />
    </AuthProvider>
  );
};

describe('ConfirmEmailPage', () => {
  beforeEach(() => {
    mockT.mockClear();
    mockedApiClient.post.mockClear();
    mockNotifySuccess.mockClear();
    mockNotifyError.mockClear();
    mockAuthLogin.mockClear();
    mockRouterQuery = {};
    mockRouterIsReady = false; // Resetar para simular o estado inicial do router
  });

  test('renders initial "verifying email" state', () => {
    mockRouterIsReady = true; // Simular que o router está pronto, mas sem token ainda
    renderConfirmEmailPage();
    expect(screen.getByText('confirm_email.verifying_email_message')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'confirm_email.go_to_login_button' })).not.toBeInTheDocument();
  });

  test('shows error if token is missing from URL on mount', async () => {
    mockRouterIsReady = true;
    mockRouterQuery = { token: undefined };
    renderConfirmEmailPage();

    await waitFor(() => {
      expect(mockNotifyError).toHaveBeenCalledWith('confirm_email.error_token_missing');
    });
    expect(screen.getByText('confirm_email.error_token_missing')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'confirm_email.try_login_button' })).toBeInTheDocument();
  });

  test('calls API and shows success (no auto-login) if token is valid and API succeeds without user data', async () => {
    mockRouterIsReady = true;
    mockRouterQuery = { token: 'valid-token' };
    mockedApiClient.post.mockResolvedValueOnce({ data: { message: 'API Email confirmed' } });
    renderConfirmEmailPage();

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledWith('/auth/confirm-email', { token: 'valid-token' });
    });
    await waitFor(() => {
      expect(mockNotifySuccess).toHaveBeenCalledWith('API Email confirmed');
    });
    expect(screen.getByText('API Email confirmed')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'confirm_email.go_to_login_button' })).toBeInTheDocument();
    expect(mockAuthLogin).not.toHaveBeenCalled();
  });

  test('calls API, logs in, and shows success if token is valid and API returns user data and token', async () => {
    mockRouterIsReady = true;
    mockRouterQuery = { token: 'valid-token-for-login' };
    const mockUser: User = { id: 'user1', name: 'Test User', email: 'test@example.com', role: 'user', organization_id: 'org1' };
    const mockJwtToken = 'fake-jwt-after-confirmation';

    mockedApiClient.post.mockResolvedValueOnce({
      data: {
        message: 'API Email confirmed, logged in',
        token: mockJwtToken,
        user: mockUser
      }
    });
    renderConfirmEmailPage();

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledWith('/auth/confirm-email', { token: 'valid-token-for-login' });
    });
    await waitFor(() => {
      expect(mockNotifySuccess).toHaveBeenCalledWith('API Email confirmed, logged in');
    });
    await waitFor(() => {
        expect(mockAuthLogin).toHaveBeenCalledWith(mockUser, mockJwtToken);
    });
    expect(screen.getByText('confirm_email.success_message_logged_in')).toBeInTheDocument();
    // O botão de login não deve aparecer se o login foi automático
    expect(screen.queryByRole('link', { name: 'confirm_email.go_to_login_button' })).not.toBeInTheDocument();
  });

  test('shows error if API call fails during confirmation', async () => {
    mockRouterIsReady = true;
    mockRouterQuery = { token: 'valid-token-api-fail' };
    mockedApiClient.post.mockRejectedValueOnce({ response: { data: { error: 'API Token Expired' } } });
    renderConfirmEmailPage();

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledWith('/auth/confirm-email', { token: 'valid-token-api-fail' });
    });
    await waitFor(() => {
      expect(mockNotifyError).toHaveBeenCalledWith('API Token Expired');
    });
    expect(screen.getByText('API Token Expired')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'confirm_email.try_login_button' })).toBeInTheDocument();
  });
});
