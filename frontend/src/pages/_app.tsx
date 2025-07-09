import '@/styles/globals.css'; // Garanta que este arquivo exista em src/styles/globals.css
import 'react-toastify/dist/ReactToastify.css'; // Importar o CSS do react-toastify
import type { AppProps } from 'next/app';
import { AuthProvider } from '../contexts/AuthContext'; // Ajuste o path se necessário
import { ToastContainer } from 'react-toastify'; // Importar o ToastContainer

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <AuthProvider>
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
        theme="light" // Pode ser "light", "dark", ou "colored". Preferência por 'light' ou 'colored'.
                     // Para dark theme, precisaria de lógica para trocar o tema do toast.
                     // Vamos começar com 'light' ou 'colored' para simplicidade.
      />
    </AuthProvider>
  );
}

export default MyApp;
