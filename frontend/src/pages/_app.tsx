import '@/styles/globals.css';
import 'react-toastify/dist/ReactToastify.css';
import type { AppProps } from 'next/app';
import { AuthProvider, useAuth } from '../contexts/AuthContext';
import { ThemeProvider } from '../contexts/ThemeContext';
import { FeatureToggleProvider } from '../contexts/FeatureToggleContext';
import { ToastContainer } from 'react-toastify';
import { appWithTranslation } from 'next-i18next';
import { useEffect } from 'react';
import { useRouter } from 'next/router'; // Importar useRouter

// Componente interno para aplicar as variáveis CSS, pois useAuth só funciona dentro do AuthProvider
const DynamicBrandingStyles = () => {
  const { branding, isLoading: authIsLoading } = useAuth(); // Renomear isLoading para evitar conflito

  useEffect(() => {
    if (!authIsLoading) {
      const root = document.documentElement;
      root.style.setProperty('--phoenix-primary-color', branding.primaryColor || '#4F46E5');
      root.style.setProperty('--phoenix-secondary-color', branding.secondaryColor || '#7C3AED');
    }
  }, [branding, authIsLoading]);

  return null;
};

// Componente para lidar com o redirecionamento para o setup
const SetupRedirector: React.FC<{ children: ReactNode }> = ({ children }) => {
  const router = useRouter();
  const { isAuthenticated, isLoading: authIsLoading } = useAuth(); // Pegar isAuthenticated e isLoading

  useEffect(() => {
    if (typeof window !== 'undefined' && !authIsLoading) {
      const setupCompleted = localStorage.getItem('phoenixSetupCompleted') === 'true';
      const publicPaths = ['/setup', '/auth/login', '/auth/register', '/auth/forgot-password', '/auth/reset-password', '/auth/confirm-email', '/auth/callback'];
      const isPublicPath = publicPaths.some(path => router.pathname.startsWith(path)) || router.pathname === '/_error';


      if (!isAuthenticated && !setupCompleted && !isPublicPath) {
        // Se não autenticado, setup não completo, e não é uma página pública/setup, redirecionar para setup.
        // A própria página /setup irá chamar /api/v1/setup/status para verificar o estado real.
        // Se /setup/status retornar 'completed', a página de setup redirecionará para /auth/login.
        router.push('/setup');
      }
    }
  }, [isAuthenticated, authIsLoading, router]);

  return <>{children}</>;
};


function MyApp({ Component, pageProps }: AppProps) {
  return (
    <AuthProvider>
      <ThemeProvider>
        <FeatureToggleProvider>
          <SetupRedirector> {/* Envolver componentes que precisam desta lógica */}
            <DynamicBrandingStyles />
            <Component {...pageProps} />
          </SetupRedirector>
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
        </FeatureToggleProvider>
      </ThemeProvider>
    </AuthProvider>
  );
}

export default appWithTranslation(MyApp);
