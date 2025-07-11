import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { useState, useEffect } from 'react';
import apiClient from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import PasswordStrengthIndicator from '@/components/auth/PasswordStrengthIndicator';

// Interface para o estado de força da senha (copiada de register.tsx, idealmente seria um tipo compartilhado)
interface PasswordStrengthCriteria {
  minLength: boolean;
  uppercase: boolean;
  lowercase: boolean;
  number: boolean;
  specialChar: boolean;
}

type Props = {
  // Props from getStaticProps
}

// Usar getStaticProps pois o token é lido no lado do cliente.
// Se o token precisasse ser validado no servidor antes de renderizar, seria getServerSideProps.
export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'auth'])),
  },
});

export default function ResetPasswordPage(props: InferGetStaticPropsType<typeof getStaticProps>) {
  const { t } = useTranslation(['auth', 'common']);
  const router = useRouter();
  const notify = useNotifier();
  const [token, setToken] = useState<string | null>(null);
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [isSuccess, setIsSuccess] = useState(false);

  // Estado para os critérios de força da senha
  const [passwordStrength, setPasswordStrength] = useState<PasswordStrengthCriteria>({
    minLength: false,
    uppercase: false,
    lowercase: false,
    number: false,
    specialChar: false,
  });
  const [showPasswordCriteria, setShowPasswordCriteria] = useState(false);

  // Função para validar a força da senha e atualizar o estado
  const validatePasswordStrength = (password: string): boolean => {
    const newStrength = {
      minLength: password.length >= 8,
      uppercase: /[A-Z]/.test(password),
      lowercase: /[a-z]/.test(password),
      number: /[0-9]/.test(password),
      specialChar: /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~`]/.test(password),
    };
    setPasswordStrength(newStrength);
    return Object.values(newStrength).every(Boolean);
  };

  const handleNewPasswordChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const currentNewPassword = e.target.value;
    setNewPassword(currentNewPassword);
    if (currentNewPassword || showPasswordCriteria) {
      setShowPasswordCriteria(true);
      validatePasswordStrength(currentNewPassword);
    } else {
      setShowPasswordCriteria(false);
    }
  };

  useEffect(() => {
    if (router.isReady) {
      const { token: queryToken } = router.query;
      if (typeof queryToken === 'string' && queryToken) {
        setToken(queryToken);
      } else {
        const errMsg = t('reset_password.error_token_invalid_or_missing');
        setFormError(errMsg);
        // notify.error(errMsg); // Pode ser muito intrusivo se a página carregar sem token intencionalmente primeiro
      }
    }
  }, [router.isReady, router.query, t]); // Removido notify da dependência aqui para evitar múltiplos toasts

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsLoading(true);
    setFormError(null);

    if (!newPassword || !confirmPassword) {
      setFormError(t('reset_password.error_passwords_required'));
      setIsLoading(false);
      return;
    }
    if (newPassword !== confirmPassword) {
      setFormError(t('reset_password.error_passwords_do_not_match'));
      setIsLoading(false);
      return;
    }
    if (!token) {
      setFormError(t('reset_password.error_token_invalid_or_missing'));
      notify.error(t('reset_password.error_token_invalid_or_missing')); // Notificar aqui se tentar submeter sem token
      setIsLoading(false);
      return;
    }

    if (!validatePasswordStrength(newPassword)) {
      setFormError(t('register.error_password_not_strong', 'A senha não atende a todos os critérios de força.'));
      setShowPasswordCriteria(true); // Garantir que os critérios sejam mostrados
      setIsLoading(false);
      return;
    }

    try {
      const response = await apiClient.post('/auth/reset-password', {
        token,
        new_password: newPassword,
        confirm_password: confirmPassword,
      });
      notify.success(response.data?.message || t('reset_password.success_message'));
      setIsSuccess(true);
      setNewPassword('');
      setConfirmPassword('');
      setTimeout(() => {
        router.push('/auth/login');
      }, 3000);
    } catch (err: any) {
      console.error('Erro ao redefinir senha:', err);
      notify.error(err.response?.data?.error || t('common:unknown_error'));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <Head>
        <title>{t('reset_password.title')} - {t('common:app_name')}</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-8 text-center">
            <div className="mb-4 inline-block rounded-full bg-brand-primary p-3 text-white">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="h-8 w-8">
                    {/* Ícone de chave ou refresh para resetar senha */}
                    <path fillRule="evenodd" d="M15.75 7.5a3 3 0 00-3-3h-1.5a3 3 0 00-3 3V9a.75.75 0 001.5 0V7.5a1.5 1.5 0 011.5-1.5h1.5a1.5 1.5 0 011.5 1.5V9a.75.75 0 001.5 0V7.5z" clipRule="evenodd" />
                    <path fillRule="evenodd" d="M5.055 8.478A3.752 3.752 0 018.25 4.5h7.5a3.752 3.752 0 013.195 3.978C18.061 9.579 17.25 10.5 17.25 10.5V15a3 3 0 01-3 3h-1.5V16.5a.75.75 0 00-1.5 0V18h-3V16.5a.75.75 0 00-1.5 0V18h-1.5a3 3 0 01-3-3v-4.5c0-.93.537-1.436.994-2.142A2.246 2.246 0 005.054 8.478zM15 15.75a.75.75 0 00.75-.75v-3a.75.75 0 00-.75-.75h-6a.75.75 0 00-.75.75v3a.75.75 0 00.75.75h6z" clipRule="evenodd" />
                </svg>
            </div>
            <h1 className="text-3xl font-bold text-gray-800 dark:text-white">{t('reset_password.title')}</h1>
          </div>

          {formError && !isSuccess && (
            <div className="mb-4 rounded-md bg-red-50 p-3">
              <p className="text-sm font-medium text-red-700">{formError}</p>
              {formError.includes(t('reset_password.error_token_invalid_or_missing_check', 'Token de redefinição inválido ou não fornecido')) && (
                <p className="mt-2 text-sm">
                  <Link href="/auth/forgot-password">
                    <span className="font-medium text-brand-primary hover:text-brand-primary/80 dark:text-brand-primary dark:hover:text-brand-primary/70 transition-colors">
                      {t('common:request_new_link', 'Solicitar novo link de redefinição')}
                    </span>
                  </Link>
                </p>
              )}
            </div>
          )}
          {isSuccess && (
            <div className="mb-4 rounded-md bg-green-50 p-3">
              <p className="text-sm font-medium text-green-700">{t('reset_password.success_message')}</p>
            </div>
          )}

          {!token && !formError && !isSuccess && (
             <div className="text-center py-4">
                <p className="text-sm text-gray-500 dark:text-gray-400">{t('reset_password.token_verifying')}</p>
             </div>
          )}

          {token && !isSuccess && (
            <form className="space-y-6" onSubmit={handleSubmit}>
              <div>
                <label htmlFor="newPassword" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                  {t('reset_password.new_password_label')}
                </label>
                <input
                  id="newPassword"
                  name="newPassword"
                  type="password"
                  required
                  value={newPassword}
                  onChange={handleNewPasswordChange}
                  onFocus={() => setShowPasswordCriteria(true)}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                  placeholder={t('reset_password.new_password_placeholder')}
                  disabled={isLoading}
                />
                {showPasswordCriteria && (
                  <div className="mt-2 space-y-1 text-sm">
                    <PasswordStrengthIndicator isValid={passwordStrength.minLength} textKey="register.strength_min_length" />
                    <PasswordStrengthIndicator isValid={passwordStrength.uppercase} textKey="register.strength_uppercase" />
                    <PasswordStrengthIndicator isValid={passwordStrength.lowercase} textKey="register.strength_lowercase" />
                    <PasswordStrengthIndicator isValid={passwordStrength.number} textKey="register.strength_number" />
                    <PasswordStrengthIndicator isValid={passwordStrength.specialChar} textKey="register.strength_special_char" />
                  </div>
                )}
              </div>
              <div>
                <label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                  {t('reset_password.confirm_password_label')}
                </label>
                <input
                  id="confirmPassword"
                  name="confirmPassword"
                  type="password"
                  required
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                  placeholder={t('reset_password.confirm_password_placeholder')}
                  disabled={isLoading}
                />
              </div>
              <div>
                <button
                  type="submit"
                  disabled={isLoading || !Object.values(passwordStrength).every(Boolean)}
                  className="flex w-full justify-center rounded-md border border-transparent bg-brand-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 disabled:opacity-50 dark:focus:ring-offset-gray-800"
                >
                  {isLoading ? (
                    <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                  ) : (
                    t('reset_password.submit_button')
                  )}
                </button>
              </div>
            </form>
          )}
          <div className="mt-6 text-center text-sm">
            <Link href="/auth/login">
              <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                {t('reset_password.back_to_login_link')}
              </span>
            </Link>
          </div>
        </div>
      </div>
    </>
  );
}
