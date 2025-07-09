import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import '@testing-library/jest-dom';
import ResetPasswordPage from '../reset-password';
import apiClient from '../../../lib/axios';
import { useNotifier } from '../../../hooks/useNotifier';
import { AuthProvider } from '../../../contexts/AuthContext';

// Mocking next/router
const mockRouterPush = jest.fn();
let mockRouterQuery: { token?: string | string[] } = {};

jest.mock('next/router', () => ({
  useRouter: () => ({
    route: '/auth/reset-password',
    pathname: '/auth/reset-password',
    query: mockRouterQuery,
    asPath: `/auth/reset-password${mockRouterQuery.token ? `?token=${mockRouterQuery.token}` : ''}`,
    isReady: true, // Simular que o router está pronto para ler a query
    push: mockRouterPush,
  }),
}));

// Mocking next-i18next
const mockT = jest.fn((key) => key); // Simples mock para t, retorna a chave
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

// Mock AuthContext (apenas para o Provider)
jest.mock('../../../contexts/AuthContext', () => ({
  useAuth: () => ({}),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

const renderResetPasswordPage = () => {
  return render(
    <AuthProvider>
      <ResetPasswordPage />
    </AuthProvider>
  );
};

describe('ResetPasswordPage', () => {
  beforeEach(() => {
    mockT.mockClear();
    mockedApiClient.post.mockClear();
    mockNotifySuccess.mockClear();
    mockNotifyError.mockClear();
    mockRouterPush.mockClear();
    mockRouterQuery = {}; // Resetar query do router
    jest.useRealTimers(); // Usar timers reais por padrão
  });

  test('renders initial state (verifying token) if no token in query initially', () => {
    mockRouterQuery = {}; // Sem token
    renderResetPasswordPage();
    expect(screen.getByText('reset_password.token_verifying')).toBeInTheDocument();
  });

  test('shows error and notification if token is missing or invalid on mount', async () => {
    mockRouterQuery = { token: undefined }; // Simula token ausente/inválido
    renderResetPasswordPage();

    await waitFor(() => {
      expect(mockNotifyError).toHaveBeenCalledWith('reset_password.error_token_invalid_or_missing');
    });
    expect(screen.getByText('reset_password.error_token_invalid_or_missing')).toBeInTheDocument();
  });

  test('renders password form if token is present in query', () => {
    mockRouterQuery = { token: 'valid-token' };
    renderResetPasswordPage();
    expect(screen.getByLabelText('reset_password.new_password_label')).toBeInTheDocument();
    expect(screen.getByLabelText('reset_password.confirm_password_label')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'reset_password.submit_button' })).toBeInTheDocument();
  });

  test('shows validation error if password fields are empty on submit', async () => {
    mockRouterQuery = { token: 'valid-token' };
    renderResetPasswordPage();
    fireEvent.click(screen.getByRole('button', { name: 'reset_password.submit_button' }));
    expect(await screen.findByText('reset_password.error_passwords_required')).toBeInTheDocument();
    expect(mockedApiClient.post).not.toHaveBeenCalled();
  });

  test('shows validation error if passwords do not match', async () => {
    mockRouterQuery = { token: 'valid-token' };
    renderResetPasswordPage();
    fireEvent.change(screen.getByLabelText('reset_password.new_password_label'), { target: { value: 'newPass123' } });
    fireEvent.change(screen.getByLabelText('reset_password.confirm_password_label'), { target: { value: 'differentPass' } });
    fireEvent.click(screen.getByRole('button', { name: 'reset_password.submit_button' }));
    expect(await screen.findByText('reset_password.error_passwords_do_not_match')).toBeInTheDocument();
    expect(mockedApiClient.post).not.toHaveBeenCalled();
  });

  test('calls API, notifier, and redirects on successful password reset', async () => {
    jest.useFakeTimers();
    mockRouterQuery = { token: 'valid-token' };
    mockedApiClient.post.mockResolvedValueOnce({ data: { message: 'API Password reset success' } });
    renderResetPasswordPage();

    fireEvent.change(screen.getByLabelText('reset_password.new_password_label'), { target: { value: 'newSecurePassword123!' } });
    fireEvent.change(screen.getByLabelText('reset_password.confirm_password_label'), { target: { value: 'newSecurePassword123!' } });

    const submitButton = screen.getByRole('button', { name: 'reset_password.submit_button' });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledWith('/auth/reset-password', {
        token: 'valid-token',
        new_password: 'newSecurePassword123!',
        confirm_password: 'newSecurePassword123!',
      });
    });
    await waitFor(() => {
      expect(mockNotifySuccess).toHaveBeenCalledWith('API Password reset success');
    });

    expect(await screen.findByText('reset_password.success_message')).toBeInTheDocument();
    expect(submitButton).toBeDisabled();

    act(() => {
      jest.runAllTimers();
    });

    await waitFor(() => {
      expect(mockRouterPush).toHaveBeenCalledWith('/auth/login');
    });
    jest.useRealTimers();
  });

  test('calls API and notifier on failed password reset (API error)', async () => {
    mockRouterQuery = { token: 'valid-token' };
    mockedApiClient.post.mockRejectedValueOnce({ response: { data: { error: 'Token expired' } } });
    renderResetPasswordPage();

    fireEvent.change(screen.getByLabelText('reset_password.new_password_label'), { target: { value: 'newSecurePassword123!' } });
    fireEvent.change(screen.getByLabelText('reset_password.confirm_password_label'), { target: { value: 'newSecurePassword123!' } });
    fireEvent.click(screen.getByRole('button', { name: 'reset_password.submit_button' }));

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(mockNotifyError).toHaveBeenCalledWith('Token expired');
    });
    expect(screen.getByRole('button', { name: 'reset_password.submit_button' })).not.toBeDisabled();
  });
});
