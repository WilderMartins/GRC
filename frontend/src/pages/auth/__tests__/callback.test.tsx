import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { useRouter } from 'next/router';
import { AuthContext, AuthProvider } from '@/contexts/AuthContext'; // Ajuste o path
import AuthCallbackPage from '../callback'; // Ajuste o path
import apiClient from '@/lib/axios';
import '@testing-library/jest-dom';

// Mocks
jest.mock('next/router', () => ({
  useRouter: jest.fn(),
}));
jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

describe('AuthCallbackPage', () => {
  let mockRouter: any;
  let mockLogin: jest.Mock;

  const mockUser = { id: 'user1', name: 'Test User', email: 'test@example.com', role: 'admin', organization_id: 'org1' };

  beforeEach(() => {
    mockLogin = jest.fn();
    mockRouter = {
      isReady: true,
      query: {},
      push: jest.fn(),
    };
    (useRouter as jest.Mock).mockReturnValue(mockRouter);
    mockedApiClient.get.mockReset();
    mockedApiClient.defaults.headers.common['Authorization'] = undefined; // Resetar header
  });

  const renderPageWithAuthProvider = () => {
    // Precisamos de um AuthProvider real para que o login seja chamado e o estado atualizado
    // mas o mockLogin nos permite espionar a chamada.
    return render(
      <AuthProvider> {/* Usar o AuthProvider real para que o contexto seja atualizado */}
        <AuthContext.Consumer>
          {(value) => {
            if (value) { // value pode ser undefined inicialmente
              value.login = mockLogin; // Sobrescrever a função login com nosso mock
            }
            return <AuthCallbackPage />;
          }}
        </AuthContext.Consumer>
      </AuthProvider>
    );
  };


  it('shows processing message initially', () => {
    mockRouter.query = {}; // Sem token
    renderPageWithAuthProvider();
    expect(screen.getByText('Processando autenticação...')).toBeInTheDocument();
  });

  it('handles successful token callback, fetches user, and calls auth.login', async () => {
    const fakeToken = 'fake-jwt-token';
    mockRouter.query = { token: fakeToken };
    mockedApiClient.get.mockResolvedValue({ data: mockUser }); // Mock para GET /me

    renderPageWithAuthProvider();

    expect(screen.getByText('Token recebido. Verificando usuário...')).toBeInTheDocument();

    await waitFor(() => {
      expect(mockedApiClient.defaults.headers.common['Authorization']).toBe(`Bearer ${fakeToken}`);
      expect(mockedApiClient.get).toHaveBeenCalledWith('/me');
    });

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith(
        expect.objectContaining({ id: mockUser.id, email: mockUser.email }),
        fakeToken
      );
    });
    // O redirecionamento para o dashboard é feito dentro do auth.login, que usa router.push mockado.
    // Não precisamos verificar router.push aqui diretamente se confiamos no AuthContext.
  });

  it('displays error if token is missing or invalid in URL', async () => {
    mockRouter.query = { token: '' }; // Token vazio
    renderPageWithAuthProvider();
    await waitFor(() => {
        expect(screen.getByText('Token de autenticação não encontrado ou inválido na URL.')).toBeInTheDocument();
    });
    expect(mockLogin).not.toHaveBeenCalled();
  });

  it('displays error if SSO/OAuth2 callback returns an error', async () => {
    const ssoError = "access_denied";
    const ssoErrorDescription = "User did not grant permission.";
    mockRouter.query = { error: ssoError, error_description: ssoErrorDescription };
    renderPageWithAuthProvider();

    await waitFor(() => {
      expect(screen.getByText(`Erro na autenticação externa: ${ssoErrorDescription}`)).toBeInTheDocument();
    });
    expect(mockLogin).not.toHaveBeenCalled();
  });

  it('displays error if /me call fails after receiving token', async () => {
    const fakeToken = 'fake-jwt-token';
    mockRouter.query = { token: fakeToken };
    const apiErrorMsg = "Falha ao buscar dados do usuário";
    mockedApiClient.get.mockRejectedValue({ response: { data: { error: apiErrorMsg } } });

    renderPageWithAuthProvider();

    await waitFor(() => {
      expect(mockedApiClient.get).toHaveBeenCalledWith('/me');
    });
    await waitFor(() => {
      expect(screen.getByText(`Falha ao verificar usuário após SSO/OAuth2.: ${apiErrorMsg}`)).toBeInTheDocument();
    });
    expect(mockLogin).not.toHaveBeenCalled();
    // Verificar se o token foi limpo do apiClient (indireto, pois não temos acesso direto ao default)
    // e do localStorage (o que AuthContext.logout faria, mas aqui o erro é antes do login)
  });
});
