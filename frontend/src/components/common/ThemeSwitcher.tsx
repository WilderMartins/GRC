"use client"; // Para futura compatibilidade com App Router

import React from 'react';
import { useTheme } from '@/contexts/ThemeContext';
import { SunIcon, MoonIcon, ComputerDesktopIcon } from '@heroicons/react/24/outline'; // Usar outline para consistência
import { useTranslation } from 'next-i18next';

const ThemeSwitcher: React.FC = () => {
  const { theme, setTheme, resolvedTheme } = useTheme();
  const { t } = useTranslation('common'); // Assumindo que as traduções estarão em 'common'

  const options = [
    { nameKey: 'theme_switcher.light', value: 'light', icon: SunIcon },
    { nameKey: 'theme_switcher.dark', value: 'dark', icon: MoonIcon },
    { nameKey: 'theme_switcher.system', value: 'system', icon: ComputerDesktopIcon },
  ];

  // Determinar qual ícone mostrar se o botão for um toggle único,
  // ou para destacar o botão ativo.
  // Se o tema atual é 'system', o ícone a ser exibido deve ser o do tema resolvido.
  const currentDisplayIcon = resolvedTheme === 'dark' ? MoonIcon : SunIcon;
  const CurrentIcon = theme === 'system' ? ComputerDesktopIcon : currentDisplayIcon;


  return (
    <div className="flex items-center space-x-1 p-1 bg-gray-200 dark:bg-gray-700 rounded-lg">
      {options.map((option) => {
        const Icon = option.icon;
        const isActive = theme === option.value;
        return (
          <button
            key={option.value}
            onClick={() => setTheme(option.value as 'light' | 'dark' | 'system')}
            title={t(option.nameKey)}
            className={`p-2 rounded-md focus:outline-none focus:ring-2 focus:ring-brand-primary transition-colors
              ${isActive
                ? 'bg-white dark:bg-gray-900 text-brand-primary dark:text-brand-primary shadow'
                : 'text-gray-500 dark:text-gray-400 hover:bg-gray-300 dark:hover:bg-gray-600 hover:text-gray-700 dark:hover:text-gray-200'
              }
            `}
          >
            <Icon className="w-5 h-5" />
            <span className="sr-only">{t(option.nameKey)}</span>
          </button>
        );
      })}
    </div>
    // Alternativa: Um único botão que cicla ou abre um dropdown
    // <button
    //   onClick={() => {
    //     const newTheme = resolvedTheme === 'dark' ? 'light' : 'dark';
    //     setTheme(newTheme); // Simplificado: apenas alterna entre light/dark, não suporta 'system' diretamente assim
    //   }}
    //   title={t(resolvedTheme === 'dark' ? 'theme_switcher.activate_light' : 'theme_switcher.activate_dark')}
    //   className="p-2 rounded-md text-gray-500 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-brand-primary"
    // >
    //   <CurrentIcon className="w-5 h-5" />
    //   <span className="sr-only">{t(resolvedTheme === 'dark' ? 'theme_switcher.activate_light' : 'theme_switcher.activate_dark')}</span>
    // </button>
  );
};

export default ThemeSwitcher;
