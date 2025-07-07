import Head from 'next/head';
import Link from 'next/link'; // Para links futuros, como "Esqueci minha senha"

// Tipos para provedores de identidade (simulados por enquanto)
interface IdentityProvider {
  id: string;
  name: string;
  type: 'saml' | 'oauth2_google' | 'oauth2_github' | 'email'; // 'email' para login tradicional
  loginUrl?: string; // URL para iniciar o fluxo de SSO/OAuth2 no backend
}

// Mock de dados - no futuro, isso viria de uma API ou contexto
const identityProviders: IdentityProvider[] = [
  { id: 'email-password', name: 'Login com Email e Senha', type: 'email' },
  {
    id: 'google-uuid-123', // Exemplo de ID de um IdentityProvider configurado
    name: 'Login com Google',
    type: 'oauth2_google',
    // A URL de login seria algo como: /auth/oauth2/google/google-uuid-123/login (no backend)
    // Esta URL seria construída dinamicamente no futuro.
    loginUrl: '/api/backend/auth/oauth2/google/google-uuid-123/login', // Placeholder para URL do backend
  },
  {
    id: 'saml-uuid-456', // Exemplo
    name: 'Login com SAML (SSO Corporativo)',
    type: 'saml',
    loginUrl: '/api/backend/auth/saml/saml-uuid-456/login', // Placeholder para URL do backend
  },
];


export default function LoginPage() {
  const handleTraditionalLogin = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    // TODO: Implementar lógica de login com email/senha
    // Pegar email e senha do form
    // Chamar o endpoint /auth/login do backend
    alert('Login tradicional a ser implementado!');
  };

  const handleSSOLogin = (provider: IdentityProvider) => {
    // TODO: Implementar redirecionamento para provider.loginUrl
    // window.location.href = provider.loginUrl; // Exemplo simples
    if (provider.loginUrl) {
      alert(`Redirecionar para: ${provider.name} (${provider.loginUrl})`);
    } else {
      alert(`Configuração de login para ${provider.name} pendente.`);
    }
  };

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

          {/* Formulário de Login Tradicional */}
          <form onSubmit={handleTraditionalLogin} className="space-y-6">
            <div>
              <label
                htmlFor="email"
                className="block text-sm font-medium text-gray-700 dark:text-gray-200"
              >
                Endereço de Email
              </label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                placeholder="voce@example.com"
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="block text-sm font-medium text-gray-700 dark:text-gray-200"
              >
                Senha
              </label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                placeholder="Sua senha"
              />
            </div>

            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <input
                  id="remember-me"
                  name="remember-me"
                  type="checkbox"
                  className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:ring-offset-gray-800"
                />
                <label
                  htmlFor="remember-me"
                  className="ml-2 block text-sm text-gray-900 dark:text-gray-300"
                >
                  Lembrar de mim
                </label>
              </div>
              <div className="text-sm">
                <Link href="/auth/forgot-password"> {/* Placeholder link */}
                  <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                    Esqueceu sua senha?
                  </span>
                </Link>
              </div>
            </div>

            <div>
              <button
                type="submit"
                className="flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800"
              >
                Entrar
              </button>
            </div>
          </form>

          {/* Divisor "OU" */}
          <div className="my-6 flex items-center">
            <div className="flex-grow border-t border-gray-300 dark:border-gray-600"></div>
            <span className="mx-4 flex-shrink text-sm text-gray-500 dark:text-gray-400">OU</span>
            <div className="flex-grow border-t border-gray-300 dark:border-gray-600"></div>
          </div>

          {/* Botões de Login SSO/Social */}
          <div className="space-y-4">
            {identityProviders
              .filter(p => p.type !== 'email') // Filtra o login tradicional que já tem form
              .map((provider) => (
                <button
                  key={provider.id}
                  onClick={() => handleSSOLogin(provider)}
                  className="flex w-full items-center justify-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600"
                >
                  {/* Placeholder para Ícone do Provedor */}
                  <span className="mr-2"> {/* Ex: <GoogleIcon />, <SamlIcon /> */}
                    {provider.type === 'oauth2_google' && '🇬'}
                    {provider.type === 'saml' && '🔑'}
                  </span>
                  {provider.name}
                </button>
              ))}
          </div>

          <div className="mt-6 text-center text-sm">
            <p className="text-gray-600 dark:text-gray-400">
              Não tem uma conta?{' '}
              <Link href="/auth/register"> {/* Placeholder link */}
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
