import Head from 'next/head';
import Link from 'next/link';
import { useAuth } from '../../contexts/AuthContext';
import apiClient from '../../lib/axios';
import { useState, useEffect } from 'react';
import { LoginIdentityProvider, User } from '@/types'; // Importar User também para userData
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';

type Props = {
  // Adicionar quaisquer outras props que getStaticProps possa retornar
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'auth'])),
  },
});

export default function LoginPage(props: InferGetStaticPropsType<typeof getStaticProps>) {
  const { t } = useTranslation(['auth', 'common']);
  const authContext = useAuth(); // Renomeado para evitar conflito com a importação 'auth' se houver

  const [formError, setFormError] = useState<string | null>(null); // Erro para login tradicional
  const [isLoadingTraditionalLogin, setIsLoadingTraditionalLogin] = useState(false);

  const [ssoProviders, setSsoProviders] = useState<LoginIdentityProvider[]>([]);
  const [isLoadingSso, setIsLoadingSso] = useState(true);
  const [ssoError, setSsoError] = useState<string | null>(null);

  useEffect(() => {
    const fetchIdentityProviders = async () => {
      setIsLoadingSso(true);
      setSsoError(null);
      try {
        const response = await apiClient.get<LoginIdentityProvider[]>('/api/public/auth/identity-providers');
        setSsoProviders(response.data || []);
      } catch (err: any) {
        console.error('Error fetching identity providers:', err);
        setSsoError(t('login.error_loading_sso'));
        setSsoProviders([]);
      } finally {
        setIsLoadingSso(false);
      }
    };

    fetchIdentityProviders();
  }, [t]); // Adicionado t como dependência se as mensagens de erro usarem t()

  const handleTraditionalLogin = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);
    setIsLoadingTraditionalLogin(true);
    const formData = new FormData(event.currentTarget);
    const email = formData.get('email') as string;
    const password = formData.get('password') as string;

    if (!email || !password) {
      setFormError(t('login.error_email_password_required'));
      setIsLoadingTraditionalLogin(false);
      return;
    }

    try {
      const response = await apiClient.post('/auth/login', { email, password });
      if (response.data && response.data.token && response.data.user_id) {
        const userData: User = { // Usando o tipo User importado
          id: response.data.user_id,
          name: response.data.name,
          email: response.data.email,
          role: response.data.role,
          organization_id: response.data.organization_id,
          // is_active, created_at, updated_at podem não vir do /auth/login, User tem eles como opcionais
        };
        authContext.login(userData, response.data.token);
      } else {
        setFormError(t('login.error_unexpected_response'));
      }
    } catch (err: any) {
      console.error('Error in traditional login:', err);
      const errorMessage = err.response?.data?.error || t('common:unknown_error');
      setFormError(t('login.error_login_failed', { message: errorMessage }));
    } finally {
      setIsLoadingTraditionalLogin(false);
    }
  };

  return (
    <>
      <Head>
        <title>{t('common:app_title_login', 'Login - Phoenix GRC')}</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-8 text-center">
            <div className="mb-4 inline-block rounded-full bg-indigo-500 p-3 text-white">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="h-8 w-8">
                <path d="M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Zm4.28 13.43a.75.75 0 0 1-.976.02l-3.573-2.68a.75.75 0 0 0-.976 0l-3.573 2.68a.75.75 0 0 1-.976-.02l-1.141-.856a.75.75 0 0 1 .02-1.263l2.68-2.01a.75.75 0 0 0 0-1.264l-2.68-2.01a.75.75 0 0 1-.02-1.263l1.141-.856a.75.75 0 0 1 .976.02l3.573 2.68a.75.75 0 0 0 .976 0l3.573 2.68a.75.75 0 0 1 .976.02l1.141.856a.75.75 0 0 1-.02 1.263l-2.68 2.01a.75.75 0 0 0 0 1.264l2.68 2.01a.75.75 0 0 1 .02 1.263l-1.141-.856Z" />
              </svg>
            </div>
            <h1 className="text-3xl font-bold text-gray-800 dark:text-white">{t('common:app_name', 'Phoenix GRC')}</h1>
            <p className="text-gray-600 dark:text-gray-300">{t('login.welcome_message')}</p>
          </div>

          {formError && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <div className="flex">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-red-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" />
                  </svg>
                </div>
                <div className="ml-3">
                  <p className="text-sm font-medium text-red-800">{formError}</p>
                </div>
              </div>
            </div>
          )}

          {isLoadingSso && (
            <div className="text-center py-4">
              <p className="text-sm text-gray-500 dark:text-gray-400">{t('common:loading_options', 'Carregando opções...')}</p>
            </div>
          )}
          {ssoError && !isLoadingSso && (
            <div className="mb-4 rounded-md bg-yellow-50 p-4">
              <div className="flex">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-yellow-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                    <path fillRule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clipRule="evenodd" />
                  </svg>
                </div>
                <div className="ml-3">
                  <p className="text-sm font-medium text-yellow-800">{ssoError}</p>
                </div>
              </div>
            </div>
          )}
          {!isLoadingSso && ssoProviders.length > 0 && (
            <div className="space-y-4">
              {ssoProviders.map((provider) => (
                <button
                  key={provider.id}
                  onClick={() => window.location.href = provider.login_url}
                  className="flex w-full items-center justify-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600"
                >
                  <span className="mr-2">
                    {provider.type === 'oauth2_google' && '🇬'}
                    {provider.type === 'saml' && '🔑'}
                    {provider.type === 'oauth2_github' && '🐙'}
                  </span>
                  {provider.name}
                </button>
              ))}
            </div>
          )}

          {!isLoadingSso && ssoProviders.length > 0 && (
            <div className="my-6 flex items-center">
              <div className="flex-grow border-t border-gray-300 dark:border-gray-600"></div>
              <span className="mx-4 flex-shrink text-sm text-gray-500 dark:text-gray-400">{t('login.sso_divider_text')}</span>
              <div className="flex-grow border-t border-gray-300 dark:border-gray-600"></div>
            </div>
          )}

          <form onSubmit={handleTraditionalLogin} className="space-y-6">
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                {t('login.email_label')}
              </label>
              <input id="email" name="email" type="email" autoComplete="email" required
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder={t('login.email_placeholder')} />
            </div>
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                {t('login.password_label')}
              </label>
              <input id="password" name="password" type="password" autoComplete="current-password" required
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder={t('login.password_placeholder')} />
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <input id="remember-me" name="remember-me" type="checkbox"
                       className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:ring-offset-gray-800" />
                <label htmlFor="remember-me" className="ml-2 block text-sm text-gray-900 dark:text-gray-300">
                  {t('login.remember_me_label')}
                </label>
              </div>
              <div className="text-sm">
                <Link href="/auth/forgot-password">
                  <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                    {t('login.forgot_password_link')}
                  </span>
                </Link>
              </div>
            </div>
            <div>
              <button type="submit" disabled={isLoadingTraditionalLogin}
                      className="flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50 dark:focus:ring-offset-gray-800">
                {isLoadingTraditionalLogin ? (
                  <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                ) : ( t('login.submit_button') )}
              </button>
            </div>
          </form>

          <div className="mt-6 text-center text-sm">
            <p className="text-gray-600 dark:text-gray-400">
              {t('login.no_account_prompt')}{' '}
              <Link href="/auth/register">
                <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                  {t('login.register_link')}
                </span>
              </Link>
            </p>
          </div>

        </div>
      </div>
    </>
  );
}
