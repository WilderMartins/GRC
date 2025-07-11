import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import ThemeSwitcher from '../ThemeSwitcher';
import { ThemeContext, ThemeProvider, useTheme } from '@/contexts/ThemeContext'; // Importar o contexto real e o hook

// Mock useTranslation
jest.mock('next-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key, // Simplesmente retorna a chave para facilitar a verificação
  }),
}));

// Mock Heroicons
const MockSunIcon = () => <svg data-testid="sun-icon"></svg>;
const MockMoonIcon = () => <svg data-testid="moon-icon"></svg>;
const MockComputerDesktopIcon = () => <svg data-testid="computer-icon"></svg>;

jest.mock('@heroicons/react/24/outline', () => ({
  SunIcon: MockSunIcon,
  MoonIcon: MockMoonIcon,
  ComputerDesktopIcon: MockComputerDesktopIcon,
}));

// Helper para renderizar com o provider
const renderWithThemeProvider = (
    ui: React.ReactElement,
    providerProps?: Partial<ReturnType<typeof useTheme>>
) => {
  const defaultProviderValue = {
    theme: 'system' as 'light' | 'dark' | 'system',
    setTheme: jest.fn(),
    resolvedTheme: 'light' as 'light' | 'dark', // Default system to light
    ...providerProps,
  };
  return render(
    <ThemeContext.Provider value={defaultProviderValue}>
      {ui}
    </ThemeContext.Provider>
  );
};


describe('ThemeSwitcher', () => {
  it('renders three theme buttons (light, dark, system)', () => {
    renderWithThemeProvider(<ThemeSwitcher />);
    expect(screen.getByTitle('theme_switcher.light')).toBeInTheDocument();
    expect(screen.getByTitle('theme_switcher.dark')).toBeInTheDocument();
    expect(screen.getByTitle('theme_switcher.system')).toBeInTheDocument();
  });

  it('highlights the active theme button (system, resolved to light)', () => {
    renderWithThemeProvider(<ThemeSwitcher />, { theme: 'system', resolvedTheme: 'light' });
    const systemButton = screen.getByTitle('theme_switcher.system');
    // Verificar a classe que indica atividade (ex: bg-white ou text-brand-primary)
    // A classe exata depende da implementação, mas deve ser diferente dos outros.
    expect(systemButton).toHaveClass('shadow'); // Exemplo de classe de ativo
    expect(screen.getByTitle('theme_switcher.light')).not.toHaveClass('shadow');
  });

  it('highlights the active theme button (dark)', () => {
    renderWithThemeProvider(<ThemeSwitcher />, { theme: 'dark', resolvedTheme: 'dark' });
    const darkButton = screen.getByTitle('theme_switcher.dark');
    expect(darkButton).toHaveClass('shadow');
    expect(screen.getByTitle('theme_switcher.light')).not.toHaveClass('shadow');
  });

  it('calls setTheme with "light" when light button is clicked', () => {
    const mockSetTheme = jest.fn();
    renderWithThemeProvider(<ThemeSwitcher />, { setTheme: mockSetTheme });
    fireEvent.click(screen.getByTitle('theme_switcher.light'));
    expect(mockSetTheme).toHaveBeenCalledWith('light');
  });

  it('calls setTheme with "dark" when dark button is clicked', () => {
    const mockSetTheme = jest.fn();
    renderWithThemeProvider(<ThemeSwitcher />, { setTheme: mockSetTheme });
    fireEvent.click(screen.getByTitle('theme_switcher.dark'));
    expect(mockSetTheme).toHaveBeenCalledWith('dark');
  });

  it('calls setTheme with "system" when system button is clicked', () => {
    const mockSetTheme = jest.fn();
    renderWithThemeProvider(<ThemeSwitcher />, { setTheme: mockSetTheme });
    fireEvent.click(screen.getByTitle('theme_switcher.system'));
    expect(mockSetTheme).toHaveBeenCalledWith('system');
  });

  it('renders correct icons for each button', () => {
    renderWithThemeProvider(<ThemeSwitcher />);
    expect(screen.getByTitle('theme_switcher.light').querySelector('[data-testid="sun-icon"]')).toBeInTheDocument();
    expect(screen.getByTitle('theme_switcher.dark').querySelector('[data-testid="moon-icon"]')).toBeInTheDocument();
    expect(screen.getByTitle('theme_switcher.system').querySelector('[data-testid="computer-icon"]')).toBeInTheDocument();
  });
});
