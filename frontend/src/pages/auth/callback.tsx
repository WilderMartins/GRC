import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import { useAuth } from '@/contexts/AuthContext'; // Ajuste o path se necessário
import apiClient from '@/lib/axios'; // Ajuste o path se necessário
import Head from 'next/head';
import { User } from '@/types'; // Importar User de @/types

// Definição local de User removida

const AuthCallbackPage = () => {
  const router = useRouter();
  const auth = useAuth();
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string>('Processando autenticação...');

  useEffect(() => {
    // Este efeito só deve rodar uma vez quando o router estiver pronto e tiver o token.
    if (router.isReady) {
      const { token, error: ssoError, error_description: ssoErrorDescription } = router.query;

      if (ssoError) {
        console.error("Erro de SSO/OAuth2 no callback:", ssoError, ssoErrorDescription);
        setError(`Erro na autenticação externa: ${ssoErrorDescription || ssoError}`);
        setMessage('');
        // Opcionalmente, redirecionar para o login após um tempo
        // setTimeout(() => router.push('/auth/login'), 5000);
        return;
      }

      if (typeof token === 'string' && token) {
        setMessage('Token recebido. Verificando usuário...');
        // Configurar o apiClient para usar este token temporariamente para a chamada /me
        apiClient.defaults.headers.common['Authorization'] = `Bearer ${token}`;

        apiClient.get('/me') // O backend retorna dados do usuário para /me
          .then(response => {
            if (response.data && response.data.id) { // Supondo que /me retorna dados do usuário incluindo 'id'
              const userData: User = {
                id: response.data.id,
                name: response.data.name,
                email: response.data.email,
                role: response.data.role,
                organization_id: response.data.organization_id,
              };
              auth.login(userData, token); // Isso fará o redirecionamento para o dashboard
              setMessage('Autenticação bem-sucedida! Redirecionando...');
            } else {
              throw new Error('Dados do usuário inválidos recebidos da API /me.');
            }
          })
          .catch(err => {
            console.error('Erro ao buscar dados do usuário com novo token:', err);
            setError(err.response?.data?.error || err.message || 'Falha ao verificar usuário após SSO/OAuth2.');
            setMessage('');
            // Limpar o token possivelmente inválido se a chamada /me falhar
            delete apiClient.defaults.headers.common['Authorization'];
            localStorage.removeItem('authToken'); // Garantir limpeza
            localStorage.removeItem('authUser');
            // router.push('/auth/login'); // Redirecionar para login
          });
      } else if (!auth.isLoading && !auth.isAuthenticated) {
        // Se não há token na URL e o usuário não está autenticado (e o auth carregou)
        // pode ser um acesso direto à página de callback sem fluxo, redirecionar para login.
        // Ou se o token for undefined/empty string.
        setError('Token de autenticação não encontrado ou inválido na URL.');
        setMessage('');
        // router.push('/auth/login');
      }
      // Se já estiver autenticado (auth.isAuthenticated), o AuthContext já deve ter redirecionado.
      // Ou o HOC WithAuth em uma página anterior.
    }
  }, [router.isReady, router.query, auth]); // auth adicionado como dependência

  return (
    <>
    <Head><title>Autenticando... - Phoenix GRC</title></Head>
    <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900">
      <div className="p-8 bg-white dark:bg-gray-800 shadow-xl rounded-lg text-center">
        {message && !error && (
          <>
            <svg className="animate-spin h-12 w-12 text-indigo-600 mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <p className="text-lg font-medium text-gray-700 dark:text-gray-300">{message}</p>
          </>
        )}
        {error && (
          <>
            <svg className="h-12 w-12 text-red-500 mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth="1.5" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m0-10.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.75c0 5.592 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.57-.598-3.75h-.158c-1.519-.022-2.941-.086-4.286-.247M10.5 15a1.125 1.125 0 11-2.25 0 1.125 1.125 0 012.25 0zm4.244-3.392a.75.75 0 01.016 1.056l-.229.266c-.05.057-.089.126-.115.201l-.57 1.68a.75.75 0 01-1.408-.473l.46-1.363a.75.75 0 00-.308-.815l-.945-.502a.75.75 0 11.656-1.232l.945.502a.75.75 0 00.815-.308l.46-1.363a.75.75 0 011.408.473l-.57 1.68c-.026.075-.064.144-.115.201l-.229-.266a.75.75 0 011.056-.016z" />
            </svg>
            <p className="text-lg font-medium text-red-600">Falha na Autenticação</p>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">{error}</p>
            <button
              onClick={() => router.push('/auth/login')}
              className="mt-6 rounded-md bg-indigo-600 px-3.5 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
            >
              Voltar para Login
            </button>
          </>
        )}
      </div>
    </div>
    </>
  );
};

export default AuthCallbackPage;
