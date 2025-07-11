import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import UserSecurityPageContent from '../security'; // Ajustar o path se a exportação default for UserSecurityPage
import { AuthProvider, AuthContextType } from '@/contexts/AuthContext'; // Usar o provider real e o tipo
import apiClient from '@/lib/axios';
import { I18nextProvider } from 'react-i18next';
import i18n from '@/../test/i18n-test-config'; // Supondo uma config de i18n para testes

jest.mock('@/lib/axios');
const mockedApiClientPost = apiClient.post as jest.Mock;

// Mock useRouter
const mockRouterPush = jest.fn();
jest.mock('next/router', () => ({
  useRouter: () => ({
    push: mockRouterPush,
    query: {},
    pathname: '/user/security',
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
    info: jest.fn(),
    warn: jest.fn(),
  }),
}));

const mockUserBase = {
  id: 'user-123',
  name: 'Test User',
  email: 'test@example.com',
  role: 'admin',
  organization_id: 'org-123',
};

const renderPage = (authContextValue: Partial<AuthContextType>) => {
  const fullAuthContextValue: AuthContextType = {
    isAuthenticated: true,
    user: mockUserBase,
    token: 'fake-token',
    branding: {},
    isLoading: false,
    login: jest.fn(() => Promise.resolve()),
    logout: jest.fn(),
    refreshBranding: jest.fn(() => Promise.resolve()),
    refreshUser: jest.fn(() => Promise.resolve()),
    ...authContextValue,
  };

  return render(
    <AuthProvider value={fullAuthContextValue}>
      <I18nextProvider i18n={i18n}>
        <UserSecurityPageContent />
      </I18nextProvider>
    </AuthProvider>
  );
};

describe('UserSecurityPageContent - MFA Management', () => {
  beforeEach(() => {
    mockedApiClientPost.mockClear();
    mockNotifySuccess.mockClear();
    mockNotifyError.mockClear();
    (mockUserBase as any).is_totp_enabled = false; // Reset
  });

  describe('TOTP Not Enabled State', () => {
    it('shows "Enable TOTP" button when TOTP is not enabled', () => {
      renderPage({ user: { ...mockUserBase, is_totp_enabled: false } });
      expect(screen.getByRole('button', { name: 'userSecurity:button_enable_totp' })).toBeInTheDocument();
      expect(screen.getByText('userSecurity:totp_status_inactive_description')).toBeInTheDocument();
    });

    it('starts TOTP setup process', async () => {
      renderPage({ user: { ...mockUserBase, is_totp_enabled: false } });
      mockedApiClientPost.mockResolvedValueOnce({ data: { qr_code: 'data:image/png;base64,testqr', secret: 'TESTSECRET' } });

      fireEvent.click(screen.getByRole('button', { name: 'userSecurity:button_enable_totp' }));

      await waitFor(() => expect(apiClient.post).toHaveBeenCalledWith('/users/me/2fa/totp/setup'));
      expect(await screen.findByText('userSecurity:setup_totp.title')).toBeInTheDocument();
      expect(screen.getByAltText('userSecurity:setup_totp.qr_code_alt')).toHaveAttribute('src', 'data:image/png;base64,testqr');
      expect(screen.getByText('TESTSECRET')).toBeInTheDocument();
    });

    // Adicionar testes para verificação de token TOTP, cancelamento, etc.
  });

  describe('TOTP Enabled State', () => {
    it('shows "Disable TOTP" and "Manage Backup Codes" buttons when TOTP is enabled', () => {
      renderPage({ user: { ...mockUserBase, is_totp_enabled: true } });
      expect(screen.getByRole('button', { name: 'userSecurity:button_disable_totp' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'userSecurity:button_manage_backup_codes' })).toBeInTheDocument();
      expect(screen.getByText('userSecurity:totp_status_active')).toBeInTheDocument();
    });

    // Adicionar testes para o fluxo de desabilitar TOTP
    // Adicionar testes para o fluxo de gerenciar códigos de backup
  });
});
