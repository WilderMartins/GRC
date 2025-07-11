import Head from 'next/head';
import Link from 'next/link';
import { useState } from 'react';
import apiClient from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import PasswordStrengthIndicator from '@/components/auth/PasswordStrengthIndicator';
import { useRouter } from 'next/router'; // Importar useRouter

// Interface para o estado de força da senha
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

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'auth'])),
  },
});

export default function RegisterPage(props: InferGetStaticPropsType<typeof getStaticProps>) {
  const { t } = useTranslation(['auth', 'common']);
  const notify = useNotifier();
  const router = useRouter(); // Instanciar useRouter
  const [userName, setUserName] = useState('');
  const [userEmail, setUserEmail] = useState('');
  const [userPassword, setUserPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [organizationName, setOrganizationName] = useState('');

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

  const handlePasswordChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newPassword = e.target.value;
    setUserPassword(newPassword);
    if (newPassword || showPasswordCriteria) { // Validar se há texto ou se já estamos mostrando os critérios
        setShowPasswordCriteria(true); // Manter visível uma vez que o usuário começa a digitar
        validatePasswordStrength(newPassword);
    } else {
        setShowPasswordCriteria(false); // Esconder se o campo estiver vazio e não foi focado ainda
    }
  };


  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);
    setIsLoading(true);

    if (!userName || !userEmail || !userPassword || !confirmPassword || !organizationName) {
      setFormError(t('register.error_all_fields_required'));
      setIsLoading(false);
      return;
    }
    if (userPassword !== confirmPassword) {
      setFormError(t('register.error_passwords_do_not_match'));
      setIsLoading(false);
      return;
    }
    // Validar a força da senha no momento do submit também
    if (!validatePasswordStrength(userPassword)) {
      setFormError(t('register.error_password_not_strong', 'A senha não atende a todos os critérios de força.'));
      // Garantir que os critérios sejam mostrados se ainda não estiverem
      setShowPasswordCriteria(true);
      setIsLoading(false);
      return;
    }

    const payload = {
      user: {
        name: userName,
        email: userEmail,
        password: userPassword,
      },
      organization: {
        name: organizationName,
      },
    };

    try {
      const response = await apiClient.post('/auth/register', payload);
      notify.success(response.data?.message || t('register.success_message_detailed', 'Registro concluído! Você será redirecionado para o login.'));
      setIsSuccess(true);
      // Adicionar um pequeno delay para o usuário ver a mensagem de sucesso antes de redirecionar
      setTimeout(() => {
        router.push('/auth/login');
      }, 2000); // Redirecionar após 2 segundos
    } catch (err: any) {
      console.error('Erro no registro:', err);
      notify.error(err.response?.data?.error || t('common:unknown_error'));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <Head>
        <title>{t('register.title')} - {t('common:app_name')}</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900 py-12">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-8 text-center">
            {/* Ícone pode usar brand-primary para o bg se desejado, ou manter indigo como cor de destaque do Auth */}
            <div className="mb-4 inline-block rounded-full bg-brand-primary p-3 text-white">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="h-8 w-8">
                    <path d="M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Zm4.28 13.43a.75.75 0 0 1-.976.02l-3.573-2.68a.75.75 0 0 0-.976 0l-3.573 2.68a.75.75 0 0 1-.976-.02l-1.141-.856a.75.75 0 0 1 .02-1.263l2.68-2.01a.75.75 0 0 0 0-1.264l-2.68-2.01a.75.75 0 0 1-.02-1.263l1.141-.856a.75.75 0 0 1 .976.02l3.573 2.68a.75.75 0 0 0 .976 0l3.573 2.68a.75.75 0 0 1 .976.02l1.141.856a.75.75 0 0 1-.02 1.263l-2.68 2.01a.75.75 0 0 0 0 1.264l2.68 2.01a.75.75 0 0 1 .02 1.263l-1.141-.856Z" />
                </svg>
            </div>
            <h1 className="text-3xl font-bold text-gray-800 dark:text-white">{t('register.title')}</h1>
            <p className="text-gray-600 dark:text-gray-300">{t('register.join_message')}</p>
          </div>

          {formError && (
            <div className="mb-4 rounded-md bg-red-50 p-3">
              <p className="text-sm font-medium text-red-700">{formError}</p>
            </div>
          )}
          {isSuccess && (
            <div className="mb-4 rounded-md bg-green-50 p-3">
              <p className="text-sm font-medium text-green-700">{t('register.success_message_detailed', 'Registro concluído! Você será redirecionado para o login.')}</p>
            </div>
          )}

          <form className="space-y-4" onSubmit={handleSubmit}>
            <div>
              <label htmlFor="userName" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                {t('register.full_name_label')}
              </label>
              <input id="userName" name="userName" type="text" autoComplete="name" required
                     value={userName} onChange={(e) => setUserName(e.target.value)}
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder={t('register.full_name_placeholder')} disabled={isLoading || isSuccess} />
            </div>
            <div>
              <label htmlFor="userEmail" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                {t('register.email_label')}
              </label>
              <input id="userEmail" name="userEmail" type="email" autoComplete="email" required
                     value={userEmail} onChange={(e) => setUserEmail(e.target.value)}
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder={t('register.email_placeholder')} disabled={isLoading || isSuccess} />
            </div>
            <div>
              <label htmlFor="organizationName" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                {t('register.org_name_label')}
              </label>
              <input id="organizationName" name="organizationName" type="text" required
                     value={organizationName} onChange={(e) => setOrganizationName(e.target.value)}
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder={t('register.org_name_placeholder')} disabled={isLoading || isSuccess} />
            </div>
            <div>
              <label htmlFor="userPassword" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                {t('register.password_label')}
              </label>
              <input id="userPassword" name="userPassword" type="password" autoComplete="new-password" required
                     value={userPassword}
                     onChange={handlePasswordChange}
                     onFocus={() => setShowPasswordCriteria(true)}
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder={t('register.password_placeholder')} disabled={isLoading || isSuccess} />
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
                {t('register.confirm_password_label')}
              </label>
              <input id="confirmPassword" name="confirmPassword" type="password" autoComplete="new-password" required
                     value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)}
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder={t('register.confirm_password_placeholder')} disabled={isLoading || isSuccess} />
            </div>

            <div>
              <button type="submit" disabled={isLoading || isSuccess}
                      className="flex w-full justify-center rounded-md border border-transparent bg-brand-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 disabled:opacity-50 dark:focus:ring-offset-gray-800">
                {isLoading ? (
                  <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                ) : ( t('register.submit_button') )}
              </button>
            </div>
          </form>
          <div className="mt-6 text-center text-sm">
            <p className="text-gray-600 dark:text-gray-400">
              {t('register.already_have_account_prompt')}{' '}
              <Link href="/auth/login">
                <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                  {t('register.login_link')}
                </span>
              </Link>
            </p>
          </div>
        </div>
      </div>
    </>
  );
}
