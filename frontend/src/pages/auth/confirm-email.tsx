import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { useState, useEffect } from 'react';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';
import { useNotifier } from '@/hooks/useNotifier';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import { User } from '@/types'; // Import User para o caso de login automático

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'auth'])),
  },
});

export default function ConfirmEmailPage(props: InferGetStaticPropsType<typeof getStaticProps>) {
  const { t } = useTranslation(['auth', 'common']);
  const router = useRouter();
  const authContext = useAuth();
  const notify = useNotifier();
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [message, setMessage] = useState<string>(t('confirm_email.verifying_email_message'));
  const [showLoginButton, setShowLoginButton] = useState(false);

  useEffect(() => {
    // Atualizar a mensagem inicial se o idioma mudar após a montagem inicial
    setMessage(t('confirm_email.verifying_email_message'));
  }, [t]);

  useEffect(() => {
    if (router.isReady) {
      const { token } = router.query;

      if (typeof token === 'string' && token) {
        apiClient.post('/auth/confirm-email', { token })
          .then(response => {
            setStatus('success');
            const successMsg = response.data?.message || t('confirm_email.success_message_login_now');
            setMessage(successMsg);
            notify.success(successMsg);

            if (response.data?.token && response.data?.user) {
              const userData: User = response.data.user; // Assumindo que a API retorna o objeto User completo
              authContext.login(userData, response.data.token);
              setMessage(t('confirm_email.success_message_logged_in'));
              setShowLoginButton(false);
            } else {
              setShowLoginButton(true);
            }
          })
          .catch(err => {
            setStatus('error');
            const errorMsg = err.response?.data?.error || t('confirm_email.error_generic_failure');
            setMessage(errorMsg);
            notify.error(errorMsg);
            setShowLoginButton(true);
            console.error("Erro ao confirmar email:", err);
          });
      } else {
        setStatus('error');
        const errorMsg = t('confirm_email.error_token_missing');
        setMessage(errorMsg);
        notify.error(errorMsg);
        setShowLoginButton(true);
      }
    }
  }, [router.isReady, router.query, authContext, notify, t]);

  return (
    <>
      <Head>
        <title>{t('confirm_email.title')} - {t('common:app_name')}</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900 text-center">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-6">
            <div className="mx-auto mb-4 inline-block rounded-full bg-indigo-500 p-3 text-white">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="h-10 w-10">
                    <path d="M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Zm4.28 13.43a.75.75 0 0 1-.976.02l-3.573-2.68a.75.75 0 0 0-.976 0l-3.573 2.68a.75.75 0 0 1-.976-.02l-1.141-.856a.75.75 0 0 1 .02-1.263l2.68-2.01a.75.75 0 0 0 0-1.264l-2.68-2.01a.75.75 0 0 1-.02-1.263l1.141-.856a.75.75 0 0 1 .976.02l3.573 2.68a.75.75 0 0 0 .976 0l3.573 2.68a.75.75 0 0 1 .976.02l1.141.856a.75.75 0 0 1-.02 1.263l-2.68 2.01a.75.75 0 0 0 0-1.264l2.68 2.01a.75.75 0 0 1 .02 1.263l-1.141-.856Z" />
                </svg>
            </div>
            <h1 className="text-2xl font-bold text-gray-800 dark:text-white mb-4">{t('confirm_email.title')}</h1>
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
                    {t('confirm_email.go_to_login_button')}
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
                        {t('confirm_email.try_login_button')}
                        </span>
                    </Link>
                )}
                <p className="text-sm text-gray-500 dark:text-gray-400">
                    {t('confirm_email.register_prompt')}
                    <Link href="/auth/register"><span className="text-indigo-600 hover:underline dark:text-indigo-400">{t('login.register_link')}</span></Link>.
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
