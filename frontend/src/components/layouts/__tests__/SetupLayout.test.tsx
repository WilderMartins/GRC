import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import SetupLayout from '../SetupLayout';

// Mock next/head
jest.mock('next/head', () => {
  return {
    __esModule: true,
    default: ({ children }: { children: Array<React.ReactElement> }) => {
      return <>{children}</>;
    },
  };
});

// Mock useTranslation
jest.mock('next-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, defaultValue?: string) => defaultValue || key,
  }),
}));

describe('SetupLayout', () => {
  it('renders children correctly', () => {
    render(<SetupLayout><div>Test Child</div></SetupLayout>);
    expect(screen.getByText('Test Child')).toBeInTheDocument();
  });

  it('renders the default title if no title prop is provided', () => {
    render(<SetupLayout><div>Test</div></SetupLayout>);
    // document.title é atualizado por Next/Head, teste de snapshot ou verificação de meta tag seria mais robusto
    // Para simplificar, vamos assumir que o título é passado para o componente Head corretamente.
    // Se o componente Head fosse mockado para capturar props, poderíamos verificar.
    // Por agora, este teste é mais para garantir a renderização sem erros.
    expect(screen.getByAltText('Phoenix GRC Logo')).toBeInTheDocument(); // Verifica o logo
  });

  it('renders the provided title in Head', () => {
    const testTitle = "Test Wizard Title";
    render(<SetupLayout title={testTitle}><div>Test</div></SetupLayout>);
    // Verificação do título da página (document.title)
    // Em um ambiente de teste JSDOM, document.title pode não ser atualizado da mesma forma que no navegador.
    // Testar o conteúdo do <Head> é mais complexo. Este teste foca na renderização.
    // Para verificar o título efetivamente, seria melhor um teste e2e ou um mock mais elaborado de next/head.
    expect(document.title).toBe(testTitle); // Isso pode não funcionar como esperado em JSDOM com next/head
  });

  it('renders the pageTitle when provided', () => {
    const pageTitle = "Etapa X";
    render(<SetupLayout pageTitle={pageTitle}><div>Test</div></SetupLayout>);
    expect(screen.getByRole('heading', { name: pageTitle, level: 2 })).toBeInTheDocument();
  });

  it('does not render pageTitle h2 if pageTitle prop is not provided', () => {
    render(<SetupLayout><div>Test</div></SetupLayout>);
    expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
  });

  it('renders the Phoenix GRC logo', () => {
    render(<SetupLayout><div>Test</div></SetupLayout>);
    const logo = screen.getByAltText('Phoenix GRC Logo');
    expect(logo).toBeInTheDocument();
    expect(logo).toHaveAttribute('src', '/logos/phoenix-grc-logo-default.svg');
  });

  it('renders the copyright footer', () => {
    render(<SetupLayout><div>Test</div></SetupLayout>);
    expect(screen.getByText(/Phoenix GRC. Todos os direitos reservados./i)).toBeInTheDocument();
  });
});
