import Head from 'next/head';
import Link from 'next/link';
import { useState } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path se necessário
import { useNotifier } from '@/hooks/useNotifier'; // Importar o hook

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  // O estado 'error' em tela pode ser removido se todos os erros forem para toasts.
  // Mas pode ser útil manter para erros de validação de campo específicos.
  // Por agora, vamos manter e usar toasts para feedback da API.
  const [error, setError] = useState<string | null>(null);
  // const [successMessage, setSuccessMessage] = useState<string | null>(null); // Será substituído por toast
  const notify = useNotifier();

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsLoading(true);
    setError(null);
    // setSuccessMessage(null);

    if (!email) {
      // Para erros de validação de campo, podemos ainda usar o setError em tela ou um toast.
      // notify.warn('Por favor, insira seu endereço de email.');
      setError('Por favor, insira seu endereço de email.');
      setIsLoading(false);
      return;
    }

    try {
      const response = await apiClient.post('/auth/forgot-password', { email });
      notify.success(response.data?.message || 'Se o seu email estiver registrado, você receberá um link para redefinir sua senha em breve.');
      setEmail('');
    } catch (err: any)      const apiError = err.response?.data?.error;
      notify.error(apiError || 'Ocorreu um erro ao processar sua solicitação. Tente novamente mais tarde.');
      // Poderíamos setar o erro em tela também se desejado: setError(apiError || 'Ocorreu um erro...');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <Head>
        <title>Esqueci Minha Senha - Phoenix GRC</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-8 text-center">
            {/* Placeholder para Logo */}
            <div className="mb-4 inline-block rounded-full bg-indigo-500 p-3 text-white">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="h-8 w-8">
                    <path d="M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Zm4.28 13.43a.75.75 0 0 1-.976.02l-3.573-2.68a.75.75 0 0 0-.976 0l-3.573 2.68a.75.75 0 0 1-.976-.02l-1.141-.856a.75.75 0 0 1 .02-1.263l2.68-2.01a.75.75 0 0 0 0-1.264l-2.68-2.01a.75.75 0 0 1-.02-1.263l1.141-.856a.75.75 0 0 1 .976.02l3.573 2.68a.75.75 0 0 0 .976 0l3.573 2.68a.75.75 0 0 1 .976.02l1.141.856a.75.75 0 0 1-.02 1.263l-2.68 2.01a.75.75 0 0 0 0-1.264l2.68 2.01a.75.75 0 0 1 .02 1.263l-1.141-.856Z" />
                </svg>
            </div>
            <h1 className="text-3xl font-bold text-gray-800 dark:text-white">Recuperar Senha</h1>
            <p className="text-gray-600 dark:text-gray-300">Insira seu email para receber instruções.</p>
          </div>

          {error && ( // Erro de validação de campo ainda pode ser mostrado em tela
            <div className="mb-4 rounded-md bg-red-50 p-3">
              <p className="text-sm font-medium text-red-700">{error}</p>
            </div>
          )}
          {/* {successMessage && !error && ( // Removido, pois será tratado por toast
            <div className="mb-4 rounded-md bg-green-50 p-3">
              <p className="text-sm font-medium text-green-700">{successMessage}</p>
            </div>
          )} */}

          <form className="space-y-6" onSubmit={handleSubmit}>
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Endereço de Email
              </label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                placeholder="voce@example.com"
                disabled={isLoading || !!successMessage} // Desabilitar se carregando ou se já houve sucesso
              />
            </div>
            <div>
              <button
                type="submit"
                disabled={isLoading || !!successMessage}
                className="flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50 dark:focus:ring-offset-gray-800"
              >
                {isLoading ? (
                  <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                ) : (
                  'Enviar Link de Recuperação'
                )}
              </button>
            </div>
          </form>
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
