/** @type {import('next-i18next').UserConfig} */
module.exports = {
  i18n: {
    // Todos os idiomas suportados pela sua aplicação
    locales: ['en', 'pt', 'es'],
    // O idioma padrão usado quando o locale do usuário não está disponível
    defaultLocale: 'pt',
    // Opcional: Se você quiser que os caminhos de locale não sejam prefixados para o idioma padrão
    // defaultLocalePrefix: false,
    // Opcional: Caminho para seus arquivos de tradução (relativo à pasta public)
    // localePath: typeof window === 'undefined' ? require('path').resolve('./public/locales') : '/locales',
    // Simplesmente usar o default que é './public/locales' já deve funcionar bem com a estrutura do Next.js
  },
  // Opcional: recarregar em dev mode quando os arquivos de tradução mudam
  // reloadOnPrerender: process.env.NODE_ENV === 'development',

  // Opcional: Adicionar namespaces padrão ou outras configurações do i18next
  // defaultNS: 'common',
  // serializeConfig: false, // Para evitar "Text content did not match" em alguns casos de SSR
};
