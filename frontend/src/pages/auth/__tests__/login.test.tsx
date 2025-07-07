import { render, screen, fireEvent } from '@testing-library/react';
import LoginPage from '../login'; // Ajuste o path conforme sua estrutura
import '@testing-library/jest-dom';

// Mock Next.js Link component, as it might cause issues outside Next.js router context in tests
jest.mock('next/link', () => {
  return ({children, href}: {children: React.ReactNode, href: string}) => {
    return <a href={href}>{children}</a>;
  };
});

// Mock window.alert
global.alert = jest.fn();


describe('LoginPage', () => {
  beforeEach(() => {
    // Limpar mocks antes de cada teste, se necessário
    (global.alert as jest.Mock).mockClear();
  });

  it('renders the login page with essential elements', () => {
    render(<LoginPage />);

    // Verifica o título da página (Head)
    // Testar o conteúdo de Head pode ser complicado e muitas vezes é omitido.
    // Se precisar, pode usar react-helmet-async ou similar e testar o document.title.

    // Verifica o cabeçalho principal
    expect(screen.getByRole('heading', { name: /Phoenix GRC/i })).toBeInTheDocument();
    expect(screen.getByText(/Bem-vindo de volta!/i)).toBeInTheDocument();

    // Verifica campos do formulário de login tradicional
    expect(screen.getByLabelText(/Endereço de Email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Senha/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Entrar/i })).toBeInTheDocument();

    // Verifica links
    expect(screen.getByText(/Esqueceu sua senha?/i)).toBeInTheDocument();
    expect(screen.getByText(/Registre-se aqui/i)).toBeInTheDocument();

    // Verifica a seção "OU"
    expect(screen.getByText(/OU/i)).toBeInTheDocument();

    // Verifica se os botões de SSO/Social (mockados) são renderizados
    // Os nomes vêm do mock `identityProviders` dentro de `login.tsx`
    expect(screen.getByRole('button', { name: /Login com Google/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Login com SAML \(SSO Corporativo\)/i })).toBeInTheDocument();
  });

  it('calls traditional login handler on form submit', () => {
    render(<LoginPage />);
    const emailInput = screen.getByLabelText(/Endereço de Email/i);
    const passwordInput = screen.getByLabelText(/Senha/i);
    const submitButton = screen.getByRole('button', { name: /Entrar/i });

    fireEvent.change(emailInput, { target: { value: 'test@example.com' } });
    fireEvent.change(passwordInput, { target: { value: 'password123' } });
    fireEvent.click(submitButton);

    // Verifica se o alert (placeholder da função de login) foi chamado
    expect(global.alert).toHaveBeenCalledWith('Login tradicional a ser implementado!');
  });

  it('calls SSO login handler when an SSO button is clicked', () => {
    render(<LoginPage />);
    const googleLoginButton = screen.getByRole('button', { name: /Login com Google/i });
    fireEvent.click(googleLoginButton);

    // Verifica se o alert (placeholder da função de SSO) foi chamado
    // A URL exata depende do mock no componente LoginPage
    expect(global.alert).toHaveBeenCalledWith(expect.stringContaining('Redirecionar para: Login com Google'));
  });
});
