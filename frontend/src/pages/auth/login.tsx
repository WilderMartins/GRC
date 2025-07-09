import Head from 'next/head';
import Link from 'next/link';
import { useAuth } from '../../contexts/AuthContext'; // Ajuste o path se necessário
import apiClient from '../../lib/axios'; // Ajuste o path se necessário
import { useState, useEffect } from 'react'; // Adicionado useEffect

// Tipos para provedores de identidade (vindo da API)
interface LoadedIdentityProvider {
  id: string;
  name: string;
  type: 'saml' | 'oauth2_google' | 'oauth2_github'; // Tipos esperados da API
  login_url: string; // URL completa fornecida pelo backend
}

export default function LoginPage() {
  const auth = useAuth();
  const [error, setError] = useState<string | null>(null); // Erro para login tradicional
  const [isLoadingState, setIsLoadingState] = useState(false); // Loading para login tradicional

  const [loadedIdentityProviders, setLoadedIdentityProviders] = useState<LoadedIdentityProvider[]>([]);
  const [isLoadingIdps, setIsLoadingIdps] = useState(true);
  const [idpError, setIdpError] = useState<string | null>(null);

  useEffect(() => {
    const fetchIdentityProviders = async () => {
      setIsLoadingIdps(true);
      setIdpError(null);
      try {
        // O endpoint público definido foi GET /api/public/auth/identity-providers
        // Assumindo que apiClient.defaults.baseURL está configurado para a raiz do backend (ex: http://localhost:8080)
        const response = await apiClient.get<LoadedIdentityProvider[]>('/api/public/auth/identity-providers');
        setLoadedIdentityProviders(response.data || []);
      } catch (err: any) {
        console.error('Erro ao buscar provedores de identidade:', err);
        const defaultMessage = 'Falha ao carregar opções de login SSO. O login tradicional ainda está disponível.';
        // Não sobrescrever o erro de login tradicional se este erro for apenas sobre os IdPs
        setIdpError(err.response?.data?.error || err.message || defaultMessage);
        setLoadedIdentityProviders([]); // Limpa em caso de erro para não mostrar botões quebrados
      } finally {
        setIsLoadingIdps(false);
      }
    };

    fetchIdentityProviders();
  }, []);

  const handleTraditionalLogin = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null); // Limpa erro de login anterior
    setIsLoadingState(true);
    const formData = new FormData(event.currentTarget);
    const email = formData.get('email') as string;
    const password = formData.get('password') as string;

    if (!email || !password) {
      setError('Email e senha são obrigatórios.');
      setIsLoadingState(false);
      return;
    }

    try {
      const response = await apiClient.post('/auth/login', { email, password });
      if (response.data && response.data.token && response.data.user_id) {
        const userData = {
          id: response.data.user_id,
          name: response.data.name,
          email: response.data.email,
          role: response.data.role,
          organization_id: response.data.organization_id,
        };
        auth.login(userData, response.data.token);
      } else {
        setError('Falha no login: Resposta inesperada do servidor.');
      }
    } catch (err: any) {
      console.error('Erro no login tradicional:', err);
      const errorMessage = err.response?.data?.error || err.message || 'Erro desconhecido ao tentar fazer login.';
      setError(`Falha no login: ${errorMessage}`);
    } finally {
      setIsLoadingState(false);
    }
  };

  // handleSSOLogin não é mais necessário, pois a URL vem completa do backend.
  // A ação onClick do botão de SSO fará o redirecionamento direto.

  return (
    <>
      <Head>
        <title>Login - Phoenix GRC</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-8 text-center">
            {/* Placeholder para Logo */}
            <div className="mb-4 inline-block rounded-full bg-indigo-500 p-3 text-white">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="h-8 w-8">
                <path d="M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Zm4.28 13.43a.75.75 0 0 1-.976.02l-3.573-2.68a.75.75 0 0 0-.976 0l-3.573 2.68a.75.75 0 0 1-.976-.02l-1.141-.856a.75.75 0 0 1 .02-1.263l2.68-2.01a.75.75 0 0 0 0-1.264l-2.68-2.01a.75.75 0 0 1-.02-1.263l1.141-.856a.75.75 0 0 1 .976.02l3.573 2.68a.75.75 0 0 0 .976 0l3.573-2.68a.75.75 0 0 1 .976.02l1.141.856a.75.75 0 0 1-.02 1.263l-2.68 2.01a.75.75 0 0 0 0 1.264l2.68 2.01a.75.75 0 0 1 .02 1.263l-1.141.856Z" />
              </svg>
            </div>
            <h1 className="text-3xl font-bold text-gray-800 dark:text-white">Phoenix GRC</h1>
            <p className="text-gray-600 dark:text-gray-300">Bem-vindo de volta!</p>
          </div>

          {/* Erro do Login Tradicional */}
          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <div className="flex">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-red-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
                  </svg>
                </div>
                <div className="ml-3">
                  <p className="text-sm font-medium text-red-800">{error}</p>
                </div>
              </div>
            </div>
          )}

          {/* Seção de Login SSO/Social */}
          {isLoadingIdps && (
            <div className="text-center py-4">
              <p className="text-sm text-gray-500 dark:text-gray-400">Carregando opções de login...</p>
              {/* Pode adicionar um spinner aqui */}
            </div>
          )}
          {idpError && !isLoadingIdps && ( // Mostrar erro dos IdPs apenas se não estiver carregando e houver erro
            <div className="mb-4 rounded-md bg-yellow-50 p-4">
              <div className="flex">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-yellow-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                    <path fillRule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clipRule="evenodd" />
                  </svg>
                </div>
                <div className="ml-3">
                  <p className="text-sm font-medium text-yellow-800">{idpError}</p>
                </div>
              </div>
            </div>
          )}
          {!isLoadingIdps && loadedIdentityProviders.length > 0 && (
            <div className="space-y-4">
              {loadedIdentityProviders.map((provider) => (
                <button
                  key={provider.id}
                  onClick={() => window.location.href = provider.login_url}
                  className="flex w-full items-center justify-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600"
                >
                  <span className="mr-2">
                    {provider.type === 'oauth2_google' && '🇬'}
                    {provider.type === 'saml' && '🔑'}
                    {provider.type === 'oauth2_github' && '🐙'}
                    {/* Adicionar mais ícones conforme necessário */}
                  </span>
                  {provider.name}
                </button>
              ))}
            </div>
          )}

          {/* Divisor "OU" - só mostrar se houver opções SSO E o login tradicional estiver presente */}
          {!isLoadingIdps && loadedIdentityProviders.length > 0 && (
            <div className="my-6 flex items-center">
              <div className="flex-grow border-t border-gray-300 dark:border-gray-600"></div>
              <span className="mx-4 flex-shrink text-sm text-gray-500 dark:text-gray-400">OU</span>
              <div className="flex-grow border-t border-gray-300 dark:border-gray-600"></div>
            </div>
          )}

          {/* Formulário de Login Tradicional */}
          <form onSubmit={handleTraditionalLogin} className="space-y-6">
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Endereço de Email
              </label>
              <input id="email" name="email" type="email" autoComplete="email" required
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder="voce@example.com" />
            </div>
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Senha
              </label>
              <input id="password" name="password" type="password" autoComplete="current-password" required
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder="Sua senha" />
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <input id="remember-me" name="remember-me" type="checkbox"
                       className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:ring-offset-gray-800" />
                <label htmlFor="remember-me" className="ml-2 block text-sm text-gray-900 dark:text-gray-300">
                  Lembrar de mim
                </label>
              </div>
              <div className="text-sm">
                <Link href="/auth/forgot-password">
                  <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                    Esqueceu sua senha?
                  </span>
                </Link>
              </div>
            </div>
            <div>
              <button type="submit" disabled={isLoadingState}
                      className="flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50 dark:focus:ring-offset-gray-800">
                {isLoadingState ? (
                  <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                ) : ( 'Entrar' )}
              </button>
            </div>
          </form>

          <div className="mt-6 text-center text-sm">
            <p className="text-gray-600 dark:text-gray-400">
              Não tem uma conta?{' '}
              <Link href="/auth/register">
                <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                  Registre-se aqui
                </span>
              </Link>
            </p>
          </div>

        </div>
      </div>
    </>
  );
}
