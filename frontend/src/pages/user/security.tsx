import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import { useAuth } from '@/contexts/AuthContext';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import Link from 'next/link';
import { useState, useEffect } from 'react';
import apiClient from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'userSecurity'])),
  },
});

const UserSecurityPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['userSecurity', 'common']);
  const { user, isLoading: authIsLoading, refreshUser } = useAuth();
  const notify = useNotifier();

  const [isSettingUpTotp, setIsSettingUpTotp] = useState(false);
  const [setupQrCode, setSetupQrCode] = useState<string | null>(null);
  const [setupSecret, setSetupSecret] = useState<string | null>(null);
  const [verificationToken, setVerificationToken] = useState('');
  const [setupError, setSetupError] = useState<string | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);
  const [isSubmittingInitial, setIsSubmittingInitial] = useState(false);

  // Estados para o fluxo de desativação
  const [showDisableModal, setShowDisableModal] = useState(false);
  const [disablePassword, setDisablePassword] = useState('');
  const [disableError, setDisableError] = useState<string | null>(null);
  const [isDisabling, setIsDisabling] = useState(false);

  // Estados para códigos de backup
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [showBackupCodesModal, setShowBackupCodesModal] = useState(false);
  const [isLoadingBackupCodes, setIsLoadingBackupCodes] = useState(false);
  const [backupCodesError, setBackupCodesError] = useState<string | null>(null);


  // Sincronizar o estado local de isTotpEnabled com o do contexto, se necessário para UI complexa,
  // mas para esta página, podemos ler diretamente de user.is_totp_enabled.
  const isTotpEnabled = user?.is_totp_enabled || false;

  const handleStartTotpSetup = async () => {
    setIsSubmittingInitial(true);
    setSetupError(null);
    try {
      const response = await apiClient.post('/users/me/2fa/totp/setup');
      setSetupQrCode(response.data.qr_code);
      setSetupSecret(response.data.secret);
      setIsSettingUpTotp(true);
    } catch (err: any) {
      notify.error(t('setup_totp.error_starting_setup', { message: err.response?.data?.error || t('common:unknown_error') }));
      setSetupError(t('setup_totp.error_starting_setup', { message: err.response?.data?.error || t('common:unknown_error') }));
    } finally {
      setIsSubmittingInitial(false);
    }
  };

  const handleVerifyAndActivateTotp = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!verificationToken.trim()) {
      setSetupError(t('setup_totp.error_token_required'));
      return;
    }
    setIsVerifying(true);
    setSetupError(null);
    try {
      await apiClient.post('/users/me/2fa/totp/verify', { token: verificationToken });
      notify.success(t('setup_totp.success_totp_enabled'));
      await refreshUser(); // Recarregar dados do usuário para obter is_totp_enabled atualizado
      setIsSettingUpTotp(false);
      setVerificationToken('');
      // Chamar a geração de códigos de backup automaticamente após habilitar TOTP
      await handleGenerateAndShowBackupCodes(true); // Passar true para skipInitialConfirm
    } catch (err: any) {
      notify.error(t('setup_totp.error_verifying_token', { message: err.response?.data?.error || t('common:unknown_error') }));
      setSetupError(t('setup_totp.error_verifying_token', { message: err.response?.data?.error || t('common:unknown_error') }));
    } finally {
      setIsVerifying(false);
    }
  };

  const cancelTotpSetup = () => {
    setIsSettingUpTotp(false);
    setSetupQrCode(null);
    setSetupSecret(null);
    setVerificationToken('');
    setSetupError(null);
  };

  const handleOpenDisableModal = () => {
    setDisablePassword('');
    setDisableError(null);
    setShowDisableModal(true);
  };

  const handleConfirmDisableTotp = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!disablePassword) {
      setDisableError(t('disable_totp.error_password_required'));
      return;
    }
    setIsDisabling(true);
    setDisableError(null);
    try {
      await apiClient.post('/users/me/2fa/totp/disable', { password: disablePassword });
      notify.success(t('disable_totp.success_totp_disabled'));
      await refreshUser();
      setShowDisableModal(false);
    } catch (err: any) {
      notify.error(t('disable_totp.error_disabling_totp', { message: err.response?.data?.error || t('common:unknown_error') }));
      setDisableError(t('disable_totp.error_disabling_totp', { message: err.response?.data?.error || t('common:unknown_error') }));
    } finally {
      setIsDisabling(false);
    }
  };

  const pageTitle = t('page_title');
  const appName = t('common:app_name');

  const handleGenerateAndShowBackupCodes = async (skipInitialConfirm = false) => {
    if (!skipInitialConfirm) {
      if (!window.confirm(t('backup_codes.confirm_generate_new'))) {
        return;
      }
    }

    setIsLoadingBackupCodes(true);
    setBackupCodesError(null);
    try {
      const response = await apiClient.post('/users/me/2fa/backup-codes/generate');
      if (response.data && response.data.backup_codes && response.data.backup_codes.length > 0) {
        setBackupCodes(response.data.backup_codes);
        setShowBackupCodesModal(true);
      } else {
        // Isso pode indicar um problema com a resposta da API ou que não foram gerados códigos.
        throw new Error(t('backup_codes.error_no_codes_returned'));
      }
    } catch (err: any) {
      const errorMessage = err.response?.data?.error || err.message || t('common:unknown_error');
      notify.error(t('backup_codes.error_generating_codes', { message: errorMessage }));
      setBackupCodesError(t('backup_codes.error_generating_codes', { message: errorMessage }));
      setBackupCodes([]); // Limpar códigos antigos se houver erro
    } finally {
      setIsLoadingBackupCodes(false);
    }
  };

  const copyBackupCodesToClipboard = () => {
    if (backupCodes.length > 0) {
      navigator.clipboard.writeText(backupCodes.join('\n'))
        .then(() => notify.success(t('backup_codes.success_copied_to_clipboard')))
        .catch(err => notify.error(t('backup_codes.error_copying_to_clipboard')));
    }
  };

  const downloadBackupCodes = () => {
    if (backupCodes.length > 0) {
      const textContent = backupCodes.join('\n');
      const element = document.createElement('a');
      const file = new Blob([textContent], { type: 'text/plain' });
      element.href = URL.createObjectURL(file);
      element.download = 'phoenix-grc-backup-codes.txt';
      document.body.appendChild(element); // Necessário para o Firefox
      element.click();
      document.body.removeChild(element);
      URL.revokeObjectURL(element.href); // Limpar
      notify.success(t('backup_codes.success_downloaded'));
    }
  };

  if (authIsLoading || !user) {
    return <AdminLayout title={t('common:loading_ellipsis')}><div className="p-6 text-center">{t('common:loading_user_data')}</div></AdminLayout>;
  }

  return (
    <AdminLayout title={`${pageTitle} - ${appName}`}>
      <Head>
        <title>{`${pageTitle} - ${appName}`}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white mb-8">
          {t('header')}
        </h1>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          <h2 className="text-xl font-semibold text-gray-800 dark:text-white mb-6">
            {t('section_2fa_title')}
          </h2>

          {!isSettingUpTotp && (
            <>
              {isTotpEnabled ? (
                <div>
                  <p className="text-green-600 dark:text-green-400 mb-4">
                    {t('totp_status_active')}
                  </p>
                  <div className="space-y-3 sm:space-y-0 sm:flex sm:space-x-3">
                    <button
                      onClick={handleOpenDisableModal}
                      className="w-full sm:w-auto px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50 transition-colors"
                      disabled={isSubmittingInitial || authIsLoading || isLoadingBackupCodes}
                    >
                      {t('button_disable_totp')}
                    </button>
                    <button
                      onClick={handleGenerateAndShowBackupCodes}
                      className="w-full sm:w-auto mt-3 sm:mt-0 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary transition-colors flex items-center justify-center"
                      disabled={isSubmittingInitial || authIsLoading || isLoadingBackupCodes}
                    >
                      {isLoadingBackupCodes && (
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                      )}
                      {t('button_manage_backup_codes')}
                    </button>
                  </div>
                </div>
              ) : (
                <div>
                  <p className="text-gray-600 dark:text-gray-400 mb-4">
                    {t('totp_status_inactive_description')}
                  </p>
                  <button
                    onClick={handleStartTotpSetup}
                    className="w-full sm:w-auto px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary disabled:opacity-50 transition-colors flex items-center justify-center"
                    disabled={isSubmittingInitial}
                  >
                    {isSubmittingInitial && (
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                    )}
                    {t('button_enable_totp')}
                  </button>
                </div>
              )}
            </>
          )}

          {isSettingUpTotp && setupQrCode && setupSecret && (
            <div className="mt-6 border-t border-gray-200 dark:border-gray-700 pt-6">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white">{t('setup_totp.title')}</h3>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{t('setup_totp.scan_qr_instruction')}</p>
              <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-700 inline-block rounded-lg">
                <img src={setupQrCode} alt={t('setup_totp.qr_code_alt')} className="w-48 h-48 md:w-56 md:h-56" />
              </div>
              <p className="mt-4 text-sm text-gray-600 dark:text-gray-400">{t('setup_totp.manual_entry_instruction')}</p>
              <div className="mt-2 p-3 bg-gray-100 dark:bg-gray-900 rounded font-mono text-sm text-gray-700 dark:text-gray-300 break-all">
                {setupSecret}
                <button
                    onClick={() => navigator.clipboard.writeText(setupSecret)}
                    className="ml-2 text-xs text-brand-primary hover:underline"
                    title={t('setup_totp.copy_secret_button_title')}
                >
                    ({t('common:copy_button')})
                </button>
              </div>

              <form onSubmit={handleVerifyAndActivateTotp} className="mt-6 space-y-4">
                <div>
                  <label htmlFor="verificationToken" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {t('setup_totp.verification_code_label')}
                  </label>
                  <input
                    type="text"
                    name="verificationToken"
                    id="verificationToken"
                    value={verificationToken}
                    onChange={(e) => setVerificationToken(e.target.value.replace(/\s/g, ''))}
                    required
                    pattern="\d{6}"
                    maxLength={6}
                    placeholder="123456"
                    className="mt-1 block w-full max-w-xs rounded-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"
                  />
                   <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{t('setup_totp.verification_code_help_text')}</p>
                </div>
                {setupError && <p className="text-sm text-red-500">{setupError}</p>}
                <div className="flex items-center space-x-3">
                  <button
                    type="submit"
                    className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50 flex items-center justify-center"
                    disabled={isVerifying || verificationToken.length !== 6}
                  >
                    {isVerifying && (
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                    )}
                    {t('setup_totp.button_verify_activate')}
                  </button>
                  <button
                    type="button"
                    onClick={cancelTotpSetup}
                    className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary"
                    disabled={isVerifying}
                  >
                    {t('common:cancel_button')}
                  </button>
                </div>
              </form>
            </div>
          )}

          {/* Modal para Desabilitar TOTP */}
          {showDisableModal && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm">
              <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-md">
                <h3 className="text-lg font-medium mb-4 text-gray-900 dark:text-white">{t('disable_totp.modal_title')}</h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                  {t('disable_totp.confirmation_message')}
                </p>
                <form onSubmit={handleConfirmDisableTotp}>
                  <div>
                    <label htmlFor="disablePassword" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                      {t('disable_totp.password_label')}
                    </label>
                    <input
                      type="password"
                      name="disablePassword"
                      id="disablePassword"
                      value={disablePassword}
                      onChange={(e) => setDisablePassword(e.target.value)}
                      required
                      className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-red-500 focus:ring-red-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2" // Mantém foco vermelho para ação de perigo
                    />
                  </div>
                  {disableError && <p className="text-sm text-red-500 mt-2">{disableError}</p>}
                  <div className="mt-6 flex justify-end space-x-3">
                    <button
                      type="button"
                      onClick={() => setShowDisableModal(false)}
                      className="px-4 py-2 text-sm rounded-md text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-600 hover:bg-gray-200 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary transition-colors"
                      disabled={isDisabling}
                    >
                      {t('common:cancel_button')}
                    </button>
                    <button
                      type="submit"
                      className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50 flex items-center justify-center"
                      disabled={isDisabling || !disablePassword}
                    >
                       {isDisabling && (
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                       )}
                      {t('disable_totp.button_confirm_disable')}
                    </button>
                  </div>
                </form>
              </div>
            </div>
          )}

          {/* Modal para Exibir Códigos de Backup */}
          {showBackupCodesModal && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm">
              <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-md">
                <h3 className="text-lg font-medium mb-2 text-gray-900 dark:text-white">{t('backup_codes.modal_title')}</h3>
                <p className="text-sm text-red-600 dark:text-red-400 mb-1">{t('backup_codes.warning_save_securely')}</p>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">{t('backup_codes.info_one_time_display')}</p>

                {backupCodesError && <p className="text-sm text-red-500 my-2">{backupCodesError}</p>}

                {backupCodes.length > 0 && (
                  <div className="my-4 p-3 bg-gray-100 dark:bg-gray-900 rounded font-mono text-sm text-gray-700 dark:text-gray-300 space-y-1">
                    {backupCodes.map((code, index) => (
                      <div key={index} className="flex justify-between items-center">
                        <span>{code.substring(0,Math.ceil(code.length/2))} - {code.substring(Math.ceil(code.length/2))}</span>
                      </div>
                    ))}
                  </div>
                )}
                <div className="mt-4 flex flex-col sm:flex-row justify-end sm:space-x-3 space-y-2 sm:space-y-0">
                    <button
                        type="button"
                        onClick={copyBackupCodesToClipboard}
                        className="w-full sm:w-auto px-4 py-2 text-sm rounded-md text-gray-700 dark:text-gray-200 bg-gray-200 dark:bg-gray-600 hover:bg-gray-300 dark:hover:bg-gray-500 transition-colors"
                        disabled={isLoadingBackupCodes || backupCodes.length === 0}
                    >
                        {t('backup_codes.button_copy_codes')}
                    </button>
                    <button
                        type="button"
                        onClick={downloadBackupCodes}
                        className="w-full sm:w-auto px-4 py-2 text-sm rounded-md text-gray-700 dark:text-gray-200 bg-gray-200 dark:bg-gray-600 hover:bg-gray-300 dark:hover:bg-gray-500 transition-colors"
                        disabled={isLoadingBackupCodes || backupCodes.length === 0}
                    >
                        {t('backup_codes.button_download_codes')}
                    </button>
                    <button
                        type="button"
                        onClick={() => { setShowBackupCodesModal(false); setBackupCodes([]); }} // Limpar códigos ao fechar
                        className="w-full sm:w-auto px-4 py-2 text-sm rounded-md text-white bg-brand-primary hover:bg-brand-primary/90 transition-colors"
                        disabled={isLoadingBackupCodes}
                    >
                        {t('backup_codes.button_close_understood')}
                    </button>
                </div>
              </div>
            </div>
          )}
          {/* Futuramente: Adicionar outras configurações de segurança, como gerenciamento de senha, sessões ativas, etc. */}
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(UserSecurityPageContent);
