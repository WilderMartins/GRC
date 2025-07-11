import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import { useAuth } from '@/contexts/AuthContext';
import apiClient from '@/lib/axios';
import Head from 'next/head';
import { User } from '@/types';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';


export const getStaticProps: GetStaticProps = async ({ locale }) => ({
    props: {
      ...(await serverSideTranslations(locale ?? 'pt', ['common', 'authCallback'])),
    },
});

const AuthCallbackPage = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const router = useRouter();
  const auth = useAuth();
  const { t } = useTranslation(['authCallback', 'common']);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string>(t('processing_auth'));

  useEffect(() => {
    let timeoutId: NodeJS.Timeout;

    if (router.isReady) {
      const { token, error: ssoError, error_description: ssoErrorDescription, message: ssoMessage } = router.query;

      if (ssoError) {
        const displayError = ssoMessage || ssoErrorDescription || ssoError;
        console.error("SSO/OAuth2 callback error:", displayError);
        setError(t('error_external_auth', { error: displayError }));
        setMessage('');
        timeoutId = setTimeout(() => router.push('/auth/login'), 7000); // Redirect after 7s
        return;
      }

      if (typeof token === 'string' && token) {
        setMessage(t('token_received_verifying_user'));
        apiClient.defaults.headers.common['Authorization'] = `Bearer ${token}`;

        apiClient.get('/me')
          .then(response => {
            // Assumindo que /me retorna todos os campos necessários, incluindo is_totp_enabled
            const userData: User = response.data;
            // Validação mínima do objeto userData
            if (userData && userData.id && userData.email && userData.role) {
              auth.login(userData, token);
              setMessage(t('auth_successful_redirecting'));
              // auth.login já redireciona
            } else {
              throw new Error(t('error_invalid_user_data_from_me'));
            }
          })
          .catch(err => {
            console.error('Error fetching user data with new token:', err);
            const apiError = err.response?.data?.error || err.message || t('error_verifying_user_sso');
            setError(apiError);
            setMessage('');
            delete apiClient.defaults.headers.common['Authorization'];
            localStorage.removeItem('authToken');
            localStorage.removeItem('authUser');
            localStorage.removeItem('authBranding'); // Limpar tudo do AuthContext
            timeoutId = setTimeout(() => router.push('/auth/login'), 7000);
          });
      } else if (!auth.isLoading && !auth.isAuthenticated && Object.keys(router.query).length > 0 && !ssoError) {
        // Se há query params mas não é um erro e não tem token, é um estado inválido.
        // Se não houver query params, pode ser um acesso direto, que também é inválido.
        setError(t('error_token_not_found'));
        setMessage('');
        timeoutId = setTimeout(() => router.push('/auth/login'), 7000);
      } else if (!auth.isLoading && !auth.isAuthenticated && Object.keys(router.query).length === 0) {
        // Acesso direto sem query params e sem estar logado
         router.push('/auth/login'); // Redirecionar imediatamente
      }
    }
    return () => {
        if(timeoutId) clearTimeout(timeoutId);
    }
  }, [router.isReady, router.query, auth, t]); // Adicionado t

  return (
    <>
    <Head><title>{t('title_authenticating')} - {t('common:app_name')}</title></Head>
    <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900">
      <div className="p-8 bg-white dark:bg-gray-800 shadow-xl rounded-lg text-center w-full max-w-md">
        {message && !error && (
          <>
            <svg className="animate-spin h-12 w-12 text-brand-primary mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <p className="text-lg font-medium text-gray-700 dark:text-gray-300">{message}</p>
          </>
        )}
        {error && (
          <>
            <svg className="h-12 w-12 text-red-500 mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth="1.5" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
            </svg>
            <p className="text-lg font-medium text-red-600 dark:text-red-400">{t('error_title_auth_failed')}</p>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-2">{error}</p>
            <button
              onClick={() => router.push('/auth/login')}
              className="mt-6 rounded-md bg-brand-primary px-3.5 py-2 text-sm font-semibold text-white shadow-sm hover:bg-brand-primary/90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary transition-colors"
            >
              {t('button_back_to_login')}
            </button>
          </>
        )}
      </div>
    </div>
    </>
  );
};

export default AuthCallbackPage;
