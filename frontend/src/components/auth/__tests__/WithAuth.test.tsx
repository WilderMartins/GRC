import React from 'react';
import { render, screen, act } from '@testing-library/react';
import { useRouter } from 'next/router';
import { AuthContext, AuthProvider } from '../../../contexts/AuthContext'; // Ajuste o path
import WithAuth from '../WithAuth'; // Ajuste o path
import '@testing-library/jest-dom';

// Mock do Next.js Router
jest.mock('next/router', () => ({
  useRouter: jest.fn(),
}));

// Componente de Teste Simples
const MockProtectedComponent = () => <div>Conteúdo Protegido</div>;
const ProtectedPage = WithAuth(MockProtectedComponent);

describe('WithAuth HOC', () => {
  let mockRouterPush: jest.Mock;

  beforeEach(() => {
    mockRouterPush = jest.fn();
    (useRouter as jest.Mock).mockReturnValue({
      replace: mockRouterPush,
      asPath: '/protected-route', // Exemplo de rota atual
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should return null and redirect to login if not authenticated and not loading', () => {
    const mockAuthContextValue = {
      isAuthenticated: false,
      user: null,
      token: null,
      isLoading: false, // Importante: o loading deve ter terminado
      login: jest.fn(),
      logout: jest.fn(),
    };

    render(
      <AuthContext.Provider value={mockAuthContextValue}>
        <ProtectedPage />
      </AuthContext.Provider>
    );

    expect(screen.queryByText('Conteúdo Protegido')).not.toBeInTheDocument();
    expect(mockRouterPush).toHaveBeenCalledWith('/auth/login');
  });

  it('should return null if auth state is loading', () => {
    const mockAuthContextValue = {
      isAuthenticated: false, // Ou true, não importa enquanto isLoading é true
      user: null,
      token: null,
      isLoading: true, // Estado de carregamento
      login: jest.fn(),
      logout: jest.fn(),
    };

    render(
      <AuthContext.Provider value={mockAuthContextValue}>
        <ProtectedPage />
      </AuthContext.Provider>
    );
    expect(screen.queryByText('Conteúdo Protegido')).not.toBeInTheDocument();
    expect(mockRouterPush).not.toHaveBeenCalled(); // Não deve redirecionar enquanto carrega
  });

  it('should render wrapped component if authenticated and not loading', () => {
    const mockUser = { id: '1', name: 'Test User', email: 'test@test.com', role: 'user', organization_id: 'org1' };
    const mockAuthContextValue = {
      isAuthenticated: true,
      user: mockUser,
      token: 'fake-token',
      isLoading: false,
      login: jest.fn(),
      logout: jest.fn(),
    };

    render(
      <AuthContext.Provider value={mockAuthContextValue}>
        <ProtectedPage />
      </AuthContext.Provider>
    );

    expect(screen.getByText('Conteúdo Protegido')).toBeInTheDocument();
    expect(mockRouterPush).not.toHaveBeenCalled();
  });
});
