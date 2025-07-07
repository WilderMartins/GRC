import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { AuthContext } from '../../contexts/AuthContext'; // Ajuste o path
import DashboardPage from '../dashboard'; // Ajuste o path
import '@testing-library/jest-dom';

// Mock do Next.js Router e Link, pois DashboardPage os utiliza
jest.mock('next/router', () => ({
  useRouter: jest.fn(() => ({
    push: jest.fn(),
  })),
}));

jest.mock('next/link', () => {
  return ({children, href}: {children: React.ReactNode, href: string}) => {
    return <a href={href}>{children}</a>;
  };
});


describe('DashboardPage', () => {
  const mockUser = {
    id: 'user123',
    name: 'João Silva',
    email: 'joao.silva@example.com',
    role: 'admin',
    organization_id: 'org123',
  };

  const mockLogout = jest.fn();

  const renderDashboardWithAuth = (user: any, isAuthenticated: boolean, isLoading: boolean = false) => {
    return render(
      <AuthContext.Provider value={{
        isAuthenticated: isAuthenticated,
        user: user,
        token: isAuthenticated ? 'fake-token' : null,
        isLoading: isLoading,
        login: jest.fn(),
        logout: mockLogout,
      }}>
        <DashboardPage />
      </AuthContext.Provider>
    );
  };

  beforeEach(() => {
    mockLogout.mockClear();
  });

  it('renders user information when authenticated', () => {
    renderDashboardWithAuth(mockUser, true);

    expect(screen.getByText(`Olá, ${mockUser.name}!`)).toBeInTheDocument();
    expect(screen.getByText('Seu Dashboard')).toBeInTheDocument();
    expect(screen.getByText(mockUser.email)).toBeInTheDocument();
    expect(screen.getByText(mockUser.role)).toBeInTheDocument();
    // Verifica se o link para o painel admin é renderizado para admin/manager
    expect(screen.getByText('Painel Administrativo')).toBeInTheDocument();
  });

  it('calls logout when logout button is clicked', () => {
    renderDashboardWithAuth(mockUser, true);

    const logoutButton = screen.getByRole('button', { name: /Logout/i });
    fireEvent.click(logoutButton);
    expect(mockLogout).toHaveBeenCalledTimes(1);
  });

  it('renders different message for non-admin/manager users', () => {
    const regularUser = { ...mockUser, role: 'user' };
    renderDashboardWithAuth(regularUser, true);
    expect(screen.queryByText('Painel Administrativo')).not.toBeInTheDocument();
  });

  // O HOC WithAuth já lida com o caso não autenticado ou carregando,
  // então DashboardPage em si não precisa ser testada nesses estados,
  // pois não seria renderizada. Os testes do WithAuth cobrem isso.
});
