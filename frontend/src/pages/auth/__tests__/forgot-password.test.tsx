import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import ForgotPasswordPage from '../forgot-password'; // A página que estamos testando
import apiClient from '../../../lib/axios';
import { useNotifier } from '../../../hooks/useNotifier';
import { AuthProvider } from '../../../contexts/AuthContext'; // Para envolver, embora não usado diretamente

// Mocking next/router
jest.mock('next/router', () => ({
  useRouter: () => ({
    route: '/auth/forgot-password',
    pathname: '/auth/forgot-password',
    query: '',
    asPath: '/auth/forgot-password',
    push: jest.fn(),
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

// Mock AuthContext (apenas para o Provider, ForgotPasswordPage não usa useAuth diretamente)
jest.mock('../../../contexts/AuthContext', () => ({
  useAuth: () => ({}), // Mock vazio, pois não é usado
  AuthProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));


const renderForgotPasswordPage = () => {
  return render(
    <AuthProvider>
      <ForgotPasswordPage />
    </AuthProvider>
  );
};

describe('ForgotPasswordPage', () => {
  beforeEach(() => {
    mockT.mockClear();
    mockedApiClient.post.mockClear();
    mockNotifySuccess.mockClear();
    mockNotifyError.mockClear();
  });

  test('renders the forgot password form with translated texts', () => {
    renderForgotPasswordPage();

    expect(mockT).toHaveBeenCalledWith('forgot_password.title');
    expect(mockT).toHaveBeenCalledWith('forgot_password.instructions');
    expect(mockT).toHaveBeenCalledWith('forgot_password.email_label');
    expect(mockT).toHaveBeenCalledWith('forgot_password.submit_button');
    expect(mockT).toHaveBeenCalledWith('forgot_password.back_to_login_link');

    expect(screen.getByRole('heading', { name: 'forgot_password.title' })).toBeInTheDocument();
    expect(screen.getByLabelText('forgot_password.email_label')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'forgot_password.submit_button' })).toBeInTheDocument();
    expect(screen.getByText('forgot_password.back_to_login_link')).toBeInTheDocument();
  });

  test('shows validation error if email is empty on submit', async () => {
    renderForgotPasswordPage();
    fireEvent.click(screen.getByRole('button', { name: 'forgot_password.submit_button' }));

    expect(await screen.findByText('forgot_password.error_email_required')).toBeInTheDocument();
    expect(mockedApiClient.post).not.toHaveBeenCalled();
  });

  test('calls API and notifier on successful submission, then clears email and disables form', async () => {
    mockedApiClient.post.mockResolvedValueOnce({ data: { message: 'API Success Message' } });
    renderForgotPasswordPage();

    const emailInput = screen.getByLabelText('forgot_password.email_label');
    const submitButton = screen.getByRole('button', { name: 'forgot_password.submit_button' });

    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledWith('/auth/forgot-password', {
        email: 'test@example.com',
      });
    });
    await waitFor(() => {
      expect(mockNotifySuccess).toHaveBeenCalledWith('API Success Message');
    });

    expect(emailInput).toHaveValue(''); // Email field should be cleared
    expect(submitButton).toBeDisabled(); // Form should be disabled via isSuccess state
    expect(screen.getByText('forgot_password.success_message')).toBeInTheDocument(); // Check for success message in UI
  });

  test('calls API and notifier on failed submission (API error)', async () => {
    mockedApiClient.post.mockRejectedValueOnce({ response: { data: { error: 'API Error Message' } } });
    renderForgotPasswordPage();

    const emailInput = screen.getByLabelText('forgot_password.email_label');
    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
    fireEvent.click(screen.getByRole('button', { name: 'forgot_password.submit_button' }));

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(mockNotifyError).toHaveBeenCalledWith('API Error Message');
    });

    // Form should not be disabled, email should persist
    expect(emailInput).toHaveValue('test@example.com');
    expect(screen.getByRole('button', { name: 'forgot_password.submit_button' })).not.toBeDisabled();
  });
});
