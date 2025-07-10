import '@/styles/globals.css';
import 'react-toastify/dist/ReactToastify.css';
import type { AppProps } from 'next/app';
import { AuthProvider, useAuth } from '../contexts/AuthContext'; // Importar useAuth também
import { ToastContainer } from 'react-toastify';
import { appWithTranslation } from 'next-i18next';
import { useEffect } from 'react'; // Importar useEffect

// Componente interno para aplicar as variáveis CSS, pois useAuth só funciona dentro do AuthProvider
const DynamicBrandingStyles = () => {
  const { branding, isLoading } = useAuth();

  useEffect(() => {
    if (!isLoading) { // Aplicar apenas quando o carregamento inicial do branding estiver concluído
      const root = document.documentElement;
      root.style.setProperty('--phoenix-primary-color', branding.primaryColor || '#4F46E5'); // Indigo-600 como fallback
      root.style.setProperty('--phoenix-secondary-color', branding.secondaryColor || '#7C3AED'); // Purple-600 como fallback
      // Adicionar mais variáveis se necessário (ex: para texto sobre cor primária)
      // root.style.setProperty('--phoenix-primary-text-color', calculateContrastColor(branding.primaryColor || '#4F46E5'));
    }
  }, [branding, isLoading]);

  return null; // Este componente não renderiza nada visualmente
};

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <AuthProvider>
      <DynamicBrandingStyles /> {/* Adicionar o componente aqui */}
      <Component {...pageProps} />
      <ToastContainer
        position="top-right"
        autoClose={5000}
        hideProgressBar={false}
        newestOnTop={false}
        closeOnClick
        rtl={false}
        pauseOnFocusLoss
        draggable
        pauseOnHover
        theme="light"
      />
    </AuthProvider>
  );
}

export default appWithTranslation(MyApp);
