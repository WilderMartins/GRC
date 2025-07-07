import '@/styles/globals.css'; // Garanta que este arquivo exista em src/styles/globals.css
import type { AppProps } from 'next/app';
import { AuthProvider } from '../contexts/AuthContext'; // Ajuste o path se necessário

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <AuthProvider>
      <Component {...pageProps} />
    </AuthProvider>
  );
}

export default MyApp;
