"use client"; // Se estiver usando App Router no futuro, para Client Components

import React, { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react';

type Theme = 'light' | 'dark' | 'system';

interface ThemeContextType {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  resolvedTheme?: 'light' | 'dark'; // O tema efetivamente aplicado (light ou dark)
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

const applyThemePreference = (theme: Theme): 'light' | 'dark' => {
  let currentTheme: 'light' | 'dark';
  if (theme === 'system') {
    currentTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  } else {
    currentTheme = theme;
  }

  if (currentTheme === 'dark') {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
  return currentTheme;
};

export const ThemeProvider = ({ children }: { children: ReactNode }) => {
  const [theme, setThemeState] = useState<Theme>('system'); // Default inicial pode ser 'system'
  const [resolvedTheme, setResolvedTheme] = useState<'light' | 'dark' | undefined>(undefined);

  // Efeito para carregar o tema do localStorage ou preferência do sistema na montagem inicial
  useEffect(() => {
    const storedTheme = localStorage.getItem('phoenix-theme') as Theme | null;
    const initialTheme = storedTheme || 'system';
    setThemeState(initialTheme);
    const currentAppliedTheme = applyThemePreference(initialTheme);
    setResolvedTheme(currentAppliedTheme);
  }, []);

  // Efeito para lidar com mudanças na preferência do sistema se theme === 'system'
  useEffect(() => {
    if (theme !== 'system') {
      return;
    }
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => {
      const currentAppliedTheme = applyThemePreference('system'); // Re-aplica baseado no sistema
      setResolvedTheme(currentAppliedTheme);
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [theme]);


  const setTheme = useCallback((newTheme: Theme) => {
    localStorage.setItem('phoenix-theme', newTheme);
    setThemeState(newTheme);
    const currentAppliedTheme = applyThemePreference(newTheme);
    setResolvedTheme(currentAppliedTheme);
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, setTheme, resolvedTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = (): ThemeContextType => {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};
