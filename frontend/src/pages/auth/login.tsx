import Head from 'next/head';
import Link from 'next/link';
import { useAuth } from '../../contexts/AuthContext';
import apiClient from '../../lib/axios';
import { useState, useEffect } from 'react';
import { LoginIdentityProvider, User } from '@/types';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import { useRouter } from 'next/router'; // Importar useRouter

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
  const authContext = useAuth();
  const router = useRouter(); // Inicializar useRouter

  // Estados para login tradicional
  const [email, setEmail] = useState(''); // Adicionar estado para email
  const [password, setPassword] = useState(''); // Adicionar estado para password
  const [formError, setFormError] = useState<string | null>(null);
  const [isLoadingTraditionalLogin, setIsLoadingTraditionalLogin] = useState(false);

  // Estados para SSO
  const [ssoProviders, setSsoProviders] = useState<LoginIdentityProvider[]>([]);
  const [samlProviders, setSamlProviders] = useState<LoginIdentityProvider[]>([]);
  const [isLoadingSso, setIsLoadingSso] = useState(true);
  const [ssoError, setSsoError] = useState<string | null>(null);
  const [ssoLoadingProvider, setSsoLoadingProvider] = useState<string | null>(null); // Novo estado para loading de botão SSO

  // Estados para 2FA
  const [isTwoFactorStep, setIsTwoFactorStep] = useState(false);
  const [twoFactorUserId, setTwoFactorUserId] = useState<string | null>(null);
  const [twoFactorCode, setTwoFactorCode] = useState('');
  const [twoFactorError, setTwoFactorError] = useState<string | null>(null);
  const [isVerifyingTwoFactor, setIsVerifyingTwoFactor] = useState(false);
  const [intendingToUseBackupCode, setIntendingToUseBackupCode] = useState(false); // Novo estado


  useEffect(() => {
    // Se já autenticado via AuthContext (ex: token no localStorage), redirecionar
    // Isso precisa ser verificado ANTES de tentar buscar IdPs, para evitar chamadas desnecessárias
    if (authContext.isAuthenticated && !authContext.isLoading) {
      router.push('/admin/dashboard');
      return; // Importante para não continuar a execução do useEffect
    }

    // Apenas buscar IdPs se não estiver autenticado
    if (!authContext.isAuthenticated && !authContext.isLoading) {
        const fetchIdentityProviders = async () => {
          setIsLoadingSso(true);
          setSsoError(null);
          try {
            // Fetch Social Providers
            const socialResponse = await apiClient.get<LoginIdentityProvider[]>('/api/public/social-identity-providers');
            const transformedSocial = socialResponse.data.map(provider => ({
              ...provider,
              login_url: `${process.env.NEXT_PUBLIC_API_BASE_URL || ''}/auth/oauth2/${provider.provider_slug}/${provider.id}/login`
            }));
            setSsoProviders(transformedSocial || []);

            // Fetch SAML Providers (assumindo um endpoint similar)
            // NOTA: Este endpoint pode precisar ser ajustado dependendo da lógica de negócios.
            // Por exemplo, pode ser necessário passar um 'organization_slug' ou similar se a página de login for específica.
            // Por agora, vamos assumir um endpoint público que lista todos os IdPs SAML ativos.
            const samlResponse = await apiClient.get<LoginIdentityProvider[]>('/api/public/saml-identity-providers'); // Endpoint hipotético
            const transformedSaml = samlResponse.data.map(provider => ({
                ...provider,
                login_url: `${process.env.NEXT_PUBLIC_API_BASE_URL || ''}/auth/saml/${provider.id}/login`
            }));
            setSamlProviders(transformedSaml || []);

          } catch (err: any) {
            console.error('Error fetching identity providers:', err);
            setSsoError(t('login.error_loading_sso'));
            setSsoProviders([]);
            setSamlProviders([]);
          } finally {
            setIsLoadingSso(false);
          }
        };
        fetchIdentityProviders();
    } else {
        setIsLoadingSso(false); // Garantir que não fique em loading se já autenticado
    }
  }, [authContext.isAuthenticated, authContext.isLoading, router, t]);


  const handleTraditionalLogin = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);
    setTwoFactorError(null); // Limpar erro de 2FA também
    setIsLoadingTraditionalLogin(true);
    // const formData = new FormData(event.currentTarget); // Usar estados email e password
    // const email = formData.get('email') as string;
    // const password = formData.get('password') as string;

    if (!email || !password) {
      setFormError(t('login.error_email_password_required'));
      setIsLoadingTraditionalLogin(false);
      return;
    }

    try {
      const response = await apiClient.post('/auth/login', { email, password });

      if (response.data.two_fa_required && response.data.user_id) {
        setTwoFactorUserId(response.data.user_id);
        setIsTwoFactorStep(true);
        setPassword(''); // Limpar senha do estado por segurança
      } else if (response.data.token && response.data) { // User data também deve estar presente
        // A função login do AuthContext espera o objeto User completo e o token
        await authContext.login(response.data, response.data.token);
        // O redirecionamento é tratado pelo AuthContext.login
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

  const handleTwoFactorSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsVerifyingTwoFactor(true);
    setTwoFactorError(null);

    if (!twoFactorUserId || !twoFactorCode.trim()) {
      setTwoFactorError(t('login.error_2fa_code_required', 'Código 2FA é obrigatório.'));
      setIsVerifyingTwoFactor(false);
      return;
    }

    let endpoint = '';
    let payload = {};
    const currentCode = twoFactorCode.trim();

    if (intendingToUseBackupCode) {
      endpoint = '/auth/login/2fa/backup-code/verify';
      payload = { user_id: twoFactorUserId, backup_code: currentCode };
    } else {
      const isLikelyTotp = /^\d{6}$/.test(currentCode);
      if (isLikelyTotp) {
        endpoint = '/auth/login/2fa/verify';
        payload = { user_id: twoFactorUserId, token: currentCode };
      } else {
        // Se não parece TOTP, e o usuário não clicou em "usar backup", tenta como backup.
        endpoint = '/auth/login/2fa/backup-code/verify';
        payload = { user_id: twoFactorUserId, backup_code: currentCode };
      }
    }

    try {
      const response = await apiClient.post(endpoint, payload);
      await authContext.login(response.data, response.data.token);
      setIsTwoFactorStep(false);
      setTwoFactorUserId(null);
      setTwoFactorCode('');
      setIntendingToUseBackupCode(false); // Resetar intenção
    } catch (err: any) {
      console.error("2FA verification failed:", err);
      let apiError = err.response?.data?.error || t('login.error_2fa_verification_failed', 'Falha na verificação 2FA.');

      if (endpoint.includes('/2fa/verify') && !intendingToUseBackupCode) {
        apiError += ` ${t('login.try_backup_code_suggestion', 'Se o problema persistir, tente usar um código de backup.')}`;
      }
      setTwoFactorError(apiError);
    } finally {
      setIsVerifyingTwoFactor(false);
    }
  };

  // Adicionar um useEffect para redirecionar se o usuário já estiver autenticado
  // Isso é importante para o caso de o usuário navegar para /auth/login manualmente
  useEffect(() => {
    if (authContext.isAuthenticated && !authContext.isLoading) {
      router.push('/admin/dashboard');
    }
  }, [authContext.isAuthenticated, authContext.isLoading, router]);


  // Mostrar loading global apenas se o AuthContext estiver carregando E não estivermos na etapa 2FA
  if (authContext.isLoading && !isTwoFactorStep) {
    return <div className="flex justify-center items-center h-screen"><p>{t('common:loading_ellipsis')}</p></div>;
  }
  // Se autenticado (após login ou 2FA), AuthContext cuidará do redirecionamento, mas podemos mostrar msg
  if (authContext.isAuthenticated) {
     return <div className="flex justify-center items-center h-screen"><p>{t('common:redirecting_message')}</p></div>;
  }


  return (
    <>
      <Head>
        <title>{t('common:app_title_login', 'Login - Phoenix GRC')}</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-8 text-center">
             {/* Usar logo do branding se disponível, senão o default */}
            <img
                className="mx-auto h-12 w-auto"
                src={authContext.branding?.logoUrl || "/logos/phoenix-grc-logo-default.svg"}
                alt={t('common:app_name', 'Phoenix GRC')}
            />
            <h1 className="mt-4 text-3xl font-bold text-gray-800 dark:text-white">
                {!isTwoFactorStep ? t('login.welcome_message') : t('login.header_2fa')}
            </h1>
            {!isTwoFactorStep && <p className="text-gray-600 dark:text-gray-300">{t('login.prompt_sso_or_traditional')}</p>}
          </div>

          {/* Erro geral do formulário de login tradicional */}
          {formError && !isTwoFactorStep && (
            <div className="mb-4 rounded-md bg-red-50 p-4 dark:bg-red-900/30">
              <div className="flex">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-red-400 dark:text-red-300" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" /></svg>
                </div><div className="ml-3"><p className="text-sm font-medium text-red-800 dark:text-red-200">{formError}</p></div>
              </div>
            </div>
          )}
          {/* Erro do formulário 2FA */}
          {twoFactorError && isTwoFactorStep && (
             <div className="mb-4 rounded-md bg-red-50 p-4 dark:bg-red-900/30">
              <div className="flex">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-red-400 dark:text-red-300" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clipRule="evenodd" /></svg>
                </div><div className="ml-3"><p className="text-sm font-medium text-red-800 dark:text-red-200">{twoFactorError}</p></div>
              </div>
            </div>
          )}

          {/* Seção de Login SSO */}
          {!isTwoFactorStep && (
            <>
              {isLoadingSso && (
                <div className="text-center py-4"><p className="text-sm text-gray-500 dark:text-gray-400">{t('common:loading_options')}</p></div>
              )}
              {ssoError && !isLoadingSso && (
                <div className="mb-4 rounded-md bg-yellow-50 p-4 dark:bg-yellow-900/30"><div className="flex"><div className="flex-shrink-0"><svg className="h-5 w-5 text-yellow-400 dark:text-yellow-300" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true"><path fillRule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clipRule="evenodd" /></svg></div><div className="ml-3"><p className="text-sm font-medium text-yellow-800 dark:text-yellow-200">{ssoError}</p></div></div></div>
              )}
              {!isLoadingSso && (ssoProviders.length > 0 || samlProviders.length > 0) && (
                <div className="space-y-3">
                  {samlProviders.map((provider) => (
                    <button
                      key={provider.id}
                      onClick={() => {
                        setSsoLoadingProvider(provider.id);
                        window.location.href = provider.login_url;
                      }}
                      disabled={isLoadingTraditionalLogin || isVerifyingTwoFactor || !!ssoLoadingProvider}
                      className="flex w-full items-center justify-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600 transition-colors disabled:opacity-60"
                    >
                      {ssoLoadingProvider === provider.id ? (
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-gray-700 dark:text-gray-200" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                      ) : (
                        <span className="mr-2 text-xl">🔑</span>
                      )}
                      {ssoLoadingProvider === provider.id ? t('login.sso_redirecting_button') : t('login.sso_button_prefix', {providerName: provider.name})}
                    </button>
                  ))}
                  {ssoProviders.map((provider) => (
                    <button
                      key={provider.id}
                      onClick={() => {
                        setSsoLoadingProvider(provider.id); // ou provider.login_url se for mais único
                        window.location.href = provider.login_url;
                      }}
                      disabled={isLoadingTraditionalLogin || isVerifyingTwoFactor || !!ssoLoadingProvider}
                      className="flex w-full items-center justify-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600 transition-colors disabled:opacity-60"
                    >
                      {ssoLoadingProvider === provider.id ? (
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-gray-700 dark:text-gray-200" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                      ) : (
                        <span className="mr-2 text-xl">
                          {provider.type === 'oauth2_google' && '🇬'}
                          {provider.type === 'oauth2_github' && <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.5.49.09.66-.213.66-.473 0-.234-.01-1.028-.015-1.86-2.782.602-3.369-1.206-3.369-1.206-.445-1.13-.91-1.43-.91-1.43-.889-.608.067-.596.067-.596 1.003.07 1.531 1.03 1.531 1.03.892 1.527 2.341 1.087 2.91.831.091-.645.35-1.087.638-1.337-2.22-.252-4.555-1.11-4.555-4.937 0-1.09.39-1.984 1.029-2.682-.103-.254-.446-1.27.098-2.647 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.82c.85.004 1.705.115 2.504.336 1.909-1.296 2.747-1.026 2.747-1.026.546 1.377.203 2.393.1 2.647.64.698 1.028 1.592 1.028 2.682 0 3.837-2.339 4.683-4.567 4.93.359.307.678.915.678 1.846 0 1.337-.012 2.416-.012 2.74 0 .26.169.566.668.473A10.01 10.01 0 0022 12.017C22 6.484 17.522 2 12 2Z" clipRule="evenodd" /></svg>}
                        </span>
                      )}
                      {ssoLoadingProvider === provider.id ? t('login.sso_redirecting_button') : t('login.sso_button_prefix', {providerName: provider.name})}
                    </button>
                  ))}
                </div>
              )}
              {!isLoadingSso && (ssoProviders.length > 0 || samlProviders.length > 0) && (
                <div className="my-6 flex items-center"><div className="flex-grow border-t border-gray-300 dark:border-gray-600"></div><span className="mx-4 flex-shrink text-sm text-gray-500 dark:text-gray-400">{t('login.sso_divider_text')}</span><div className="flex-grow border-t border-gray-300 dark:border-gray-600"></div></div>
              )}
            </>
          )}

          {/* Formulário de Login Tradicional ou Formulário 2FA */}
          {!isTwoFactorStep ? (
            <form onSubmit={handleTraditionalLogin} className="space-y-6">
              <div>
                <label htmlFor="email-trad" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('login.email_label')}</label>
                <input id="email-trad" name="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" required
                       className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                       placeholder={t('login.email_placeholder')} />
              </div>
              <div>
                <label htmlFor="password-trad" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('login.password_label')}</label>
                <input id="password-trad" name="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" required
                       className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                       placeholder={t('login.password_placeholder')} />
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center">
                  <input id="remember-me" name="remember-me" type="checkbox" className="h-4 w-4 rounded border-gray-300 text-brand-primary focus:ring-brand-primary dark:border-gray-600 dark:bg-gray-700 dark:ring-offset-gray-800" />
                  <label htmlFor="remember-me" className="ml-2 block text-sm text-gray-900 dark:text-gray-300">{t('login.remember_me_label')}</label>
                </div>
                <div className="text-sm">
                  <Link href="/auth/forgot-password"><span className="font-medium text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70 transition-colors">{t('login.forgot_password_link')}</span></Link>
                </div>
              </div>
              <div>
                <button type="submit" disabled={isLoadingTraditionalLogin}
                        className="flex w-full justify-center rounded-md border border-transparent bg-brand-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 disabled:opacity-50 transition-colors">
                  {isLoadingTraditionalLogin ? (<svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>)
                                          : (t('login.submit_button'))}
                </button>
              </div>
            </form>
          ) : ( // Etapa de Verificação 2FA
            <form className="space-y-6" onSubmit={handleTwoFactorSubmit}>
              <p className="text-sm text-center text-gray-700 dark:text-gray-300">
                {t('login.info_2fa_required', { email: email })} {/* Manter email aqui pode ser útil para o usuário */}
              </p>
              <div>
                <label htmlFor="twoFactorCode" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {t('login.label_2fa_code')}
                </label>
                <div className="mt-1">
                  <input
                    id="twoFactorCode"
                    name="twoFactorCode"
                    type="text"
                    autoComplete="one-time-code"
                    required
                    value={twoFactorCode}
                    onChange={(e) => setTwoFactorCode(e.target.value.replace(/\s/g, ''))}
                    placeholder={intendingToUseBackupCode ? t('login.placeholder_backup_code', 'Código de Backup') : t('login.placeholder_2fa_code')}
                    className="appearance-none block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm dark:bg-gray-700 dark:text-white"
                  />
                </div>
                {!intendingToUseBackupCode && (
                  <button
                    type="button"
                    onClick={() => {
                      setIntendingToUseBackupCode(true);
                      setTwoFactorError(null); // Limpar erro anterior
                      // Opcional: focar no input
                      document.getElementById('twoFactorCode')?.focus();
                    }}
                    className="mt-2 text-sm text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70 transition-colors"
                  >
                    {t('login.use_backup_code_link', 'Usar um código de backup')}
                  </button>
                )}
                 {intendingToUseBackupCode && (
                  <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                    {t('login.info_insert_backup_code', 'Insira um dos seus códigos de backup não utilizados.')}
                  </p>
                )}
              </div>
              <div>
                <button
                  type="submit"
                  disabled={isVerifyingTwoFactor || !twoFactorCode.trim() || (twoFactorCode.trim().length !== 6 && !isNaN(Number(twoFactorCode.trim())) ) } // Desabilitar se não for 6 digitos E for numérico (tentativa de TOTP)
                  className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary disabled:opacity-50 transition-colors"
                >
                  {isVerifyingTwoFactor ? t('login.verifying_button') : t('login.verify_code_button')}
                </button>
              </div>
              <div className="text-center">
                <button
                    type="button"
                    onClick={() => {
                      setIsTwoFactorStep(false);
                      setTwoFactorUserId(null);
                      setPassword(''); // Limpar senha se ainda estiver no estado
                      setFormError(null);
                      setTwoFactorError(null);
                      setIntendingToUseBackupCode(false); // Resetar intenção
                    }}
                    className="text-sm font-medium text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70 transition-colors"
                    disabled={isVerifyingTwoFactor}
                >
                    {t('login.cancel_2fa_button')}
                </button>
              </div>
            </form>
          )}

          {!isTwoFactorStep && (
            <div className="mt-6 text-center text-sm">
              <p className="text-gray-600 dark:text-gray-400">
                {t('login.no_account_prompt')}{' '}
                <Link href="/auth/register">
                  <span className="font-medium text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70 transition-colors">
                    {t('login.register_link')}
                  </span>
                </Link>
              </p>
            </div>
          )}

        </div>
      </div>
    </>
  );
}
