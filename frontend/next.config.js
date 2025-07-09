/** @type {import('next').NextConfig} */

// Importar a configuração i18n do next-i18next.config.js
const { i18n } = require('./next-i18next.config.js');

const nextConfig = {
  reactStrictMode: true,
  // Adicionar outras configurações do Next.js aqui conforme necessário
  i18n, // Adicionar a configuração i18n aqui
};

module.exports = nextConfig;
