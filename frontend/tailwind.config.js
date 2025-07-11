/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  darkMode: 'class', // Habilitar dark mode baseado em classe
  theme: {
    extend: {
      colors: {
        'brand-primary': 'var(--phoenix-primary-color)',
        'brand-secondary': 'var(--phoenix-secondary-color)',
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'), // Adicionar plugin de formulários, útil para estilização
  ],
};
