import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { useState, useEffect } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path se necessário

export default function ResetPasswordPage() {
  const router = useRouter();
  const [token, setToken] = useState<string | null>(null);
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  useEffect(() => {
    if (router.isReady) {
      const { token: queryToken } = router.query;
      if (typeof queryToken === 'string' && queryToken) {
        setToken(queryToken);
      } else {
        setError('Token de redefinição inválido ou não fornecido. Por favor, solicite um novo link de redefinição.');
        // Opcionalmente, redirecionar para forgot-password após um tempo
        // setTimeout(() => router.push('/auth/forgot-password'), 5000);
      }
    }
  }, [router.isReady, router.query]);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsLoading(true);
    setError(null);
    setSuccessMessage(null);

    if (!newPassword || !confirmPassword) {
      setError('Por favor, preencha ambos os campos de senha.');
      setIsLoading(false);
      return;
    }
    if (newPassword !== confirmPassword) {
      setError('As senhas não conferem.');
      setIsLoading(false);
      return;
    }
    if (!token) {
      setError('Token de redefinição não encontrado. Solicite um novo link.');
      setIsLoading(false);
      return;
    }

    // TODO: Adicionar validação de força da senha no frontend (opcional, mas bom UX)

    try {
      const response = await apiClient.post('/auth/reset-password', {
        token,
        new_password: newPassword,
        confirm_password: confirmPassword, // Enviando confirmação para o backend também
      });
      setSuccessMessage(response.data?.message || 'Sua senha foi redefinida com sucesso! Você será redirecionado para o login.');
      // Limpar campos e desabilitar formulário
      setNewPassword('');
      setConfirmPassword('');
      setTimeout(() => {
        router.push('/auth/login');
      }, 3000); // Redirecionar após 3 segundos
    } catch (err: any) {
      console.error('Erro ao redefinir senha:', err);
      setError(err.response?.data?.error || 'Ocorreu um erro ao redefinir sua senha. O link pode ter expirado ou ser inválido.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <Head>
        <title>Redefinir Senha - Phoenix GRC</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-8 text-center">
            {/* Placeholder para Logo */}
            <div className="mb-4 inline-block rounded-full bg-indigo-500 p-3 text-white">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="h-8 w-8">
                    <path d="M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Zm4.28 13.43a.75.75 0 0 1-.976.02l-3.573-2.68a.75.75 0 0 0-.976 0l-3.573 2.68a.75.75 0 0 1-.976-.02l-1.141-.856a.75.75 0 0 1 .02-1.263l2.68-2.01a.75.75 0 0 0 0-1.264l-2.68-2.01a.75.75 0 0 1-.02-1.263l1.141-.856a.75.75 0 0 1 .976.02l3.573 2.68a.75.75 0 0 0 .976 0l3.573 2.68a.75.75 0 0 1 .976.02l1.141.856a.75.75 0 0 1-.02 1.263l-2.68 2.01a.75.75 0 0 0 0 1.264l2.68 2.01a.75.75 0 0 1 .02 1.263l-1.141-.856Z" />
                </svg>
            </div>
            <h1 className="text-3xl font-bold text-gray-800 dark:text-white">Redefinir Sua Senha</h1>
          </div>

          {error && !successMessage && ( // Mostrar erro apenas se não houver mensagem de sucesso
            <div className="mb-4 rounded-md bg-red-50 p-3">
              <p className="text-sm font-medium text-red-700">{error}</p>
              {error.includes("inválido ou não fornecido") && (
                <p className="mt-2 text-sm">
                  <Link href="/auth/forgot-password">
                    <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                      Solicitar novo link de redefinição
                    </span>
                  </Link>
                </p>
              )}
            </div>
          )}
          {successMessage && (
            <div className="mb-4 rounded-md bg-green-50 p-3">
              <p className="text-sm font-medium text-green-700">{successMessage}</p>
            </div>
          )}

          {!token && !error && ( // Mostra carregando token ou se o token não foi encontrado inicialmente e não há erro ainda
             <div className="text-center py-4">
                <p className="text-sm text-gray-500 dark:text-gray-400">Verificando token...</p>
             </div>
          )}

          {token && !successMessage && ( // Mostrar formulário apenas se token existir e não houver mensagem de sucesso
            <form className="space-y-6" onSubmit={handleSubmit}>
              <div>
                <label htmlFor="newPassword" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                  Nova Senha
                </label>
                <input
                  id="newPassword"
                  name="newPassword"
                  type="password"
                  required
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                  placeholder="Digite sua nova senha"
                  disabled={isLoading}
                />
              </div>
              <div>
                <label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                  Confirmar Nova Senha
                </label>
                <input
                  id="confirmPassword"
                  name="confirmPassword"
                  type="password"
                  required
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                  placeholder="Confirme sua nova senha"
                  disabled={isLoading}
                />
              </div>
              <div>
                <button
                  type="submit"
                  disabled={isLoading}
                  className="flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50 dark:focus:ring-offset-gray-800"
                >
                  {isLoading ? (
                    <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                  ) : (
                    'Redefinir Senha'
                  )}
                </button>
              </div>
            </form>
          )}
          <div className="mt-6 text-center text-sm">
            <Link href="/auth/login">
              <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                Voltar para Login
              </span>
            </Link>
          </div>
        </div>
      </div>
    </>
  );
}
