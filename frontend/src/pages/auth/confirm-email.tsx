import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { useState, useEffect } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path se necessário
import { useAuth } from '@/contexts/AuthContext'; // Para possível login automático
import { useNotifier } from '@/hooks/useNotifier'; // Importar o hook

export default function ConfirmEmailPage() {
  const router = useRouter();
  const auth = useAuth();
  const notify = useNotifier();
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [message, setMessage] = useState<string>('Verificando seu email...');
  const [showLoginButton, setShowLoginButton] = useState(false);

  useEffect(() => {
    if (router.isReady) {
      const { token } = router.query;

      if (typeof token === 'string' && token) {
        apiClient.post('/auth/confirm-email', { token })
          .then(response => {
            setStatus('success');
            const successMsg = response.data?.message || 'Email confirmado com sucesso!';
            setMessage(successMsg);
            notify.success(successMsg);

            if (response.data?.token && response.data?.user) {
              const userData = {
                id: response.data.user.id,
                name: response.data.user.name,
                email: response.data.user.email,
                role: response.data.user.role,
                organization_id: response.data.user.organization_id,
              };
              auth.login(userData, response.data.token);
              setMessage('Email confirmado e login realizado com sucesso! Redirecionando...');
              // O redirecionamento é feito pelo auth.login()
              setShowLoginButton(false);
            } else {
              // Não houve login automático, mostrar botão de login
              setShowLoginButton(true);
            }
          })
          .catch(err => {
            setStatus('error');
            const errorMsg = err.response?.data?.error || 'Falha ao confirmar o email. O link pode ser inválido ou ter expirado.';
            setMessage(errorMsg);
            notify.error(errorMsg);
            setShowLoginButton(true); // Mostrar botão de login para o usuário tentar manualmente
            console.error("Erro ao confirmar email:", err);
          });
      } else {
        setStatus('error');
        const errorMsg = 'Token de confirmação não encontrado ou inválido.';
        setMessage(errorMsg);
        notify.error(errorMsg + " Por favor, verifique o link em seu email ou tente se registrar novamente.");
        setShowLoginButton(true);
      }
    }
  }, [router.isReady, router.query, auth, notify]);

  return (
    <>
      <Head>
        <title>Confirmação de Email - Phoenix GRC</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900 text-center">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-6">
            {/* Placeholder para Logo */}
            <div className="mx-auto mb-4 inline-block rounded-full bg-indigo-500 p-3 text-white">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="h-10 w-10">
                    <path d="M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Zm4.28 13.43a.75.75 0 0 1-.976.02l-3.573-2.68a.75.75 0 0 0-.976 0l-3.573 2.68a.75.75 0 0 1-.976-.02l-1.141-.856a.75.75 0 0 1 .02-1.263l2.68-2.01a.75.75 0 0 0 0-1.264l-2.68-2.01a.75.75 0 0 1-.02-1.263l1.141-.856a.75.75 0 0 1 .976.02l3.573 2.68a.75.75 0 0 0 .976 0l3.573 2.68a.75.75 0 0 1 .976.02l1.141.856a.75.75 0 0 1-.02 1.263l-2.68 2.01a.75.75 0 0 0 0 1.264l2.68 2.01a.75.75 0 0 1 .02 1.263l-1.141-.856Z" />
                </svg>
            </div>
            <h1 className="text-2xl font-bold text-gray-800 dark:text-white mb-4">Confirmação de Email</h1>
          </div>

          {status === 'loading' && (
            <>
              <svg className="animate-spin h-10 w-10 text-indigo-600 mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <p className="text-lg text-gray-700 dark:text-gray-300">{message}</p>
            </>
          )}

          {status === 'success' && (
            <>
              <svg className="h-12 w-12 text-green-500 mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth="1.5" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p className="text-lg text-green-700 dark:text-green-400 mb-6">{message}</p>
              {showLoginButton && (
                <Link href="/auth/login">
                    <span className="inline-block rounded-md border border-transparent bg-indigo-600 px-6 py-3 text-base font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2">
                    Ir para Login
                    </span>
                </Link>
              )}
            </>
          )}

          {status === 'error' && (
            <>
             <svg className="h-12 w-12 text-red-500 mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth="1.5" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
             </svg>
              <p className="text-lg text-red-600 dark:text-red-400 mb-6">{message}</p>
              <div className="space-y-3">
                {showLoginButton && (
                    <Link href="/auth/login">
                        <span className="inline-block rounded-md border border-transparent bg-indigo-600 px-6 py-3 text-base font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2">
                        Tentar Login
                        </span>
                    </Link>
                )}
                <p className="text-sm text-gray-500 dark:text-gray-400">
                    Se você ainda não se registrou, <Link href="/auth/register"><span className="text-indigo-600 hover:underline dark:text-indigo-400">registre-se aqui</span></Link>.
                </p>
                {/* TODO: Adicionar opção de "Reenviar email de confirmação" se o backend suportar */}
              </div>
            </>
          )}
        </div>
      </div>
    </>
  );
}
