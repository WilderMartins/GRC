import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import { useState, useEffect, FormEvent } from 'react';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';
import { useNotifier } from '@/hooks/useNotifier';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';

type Props = {
  // Props from getStaticProps
}

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'organizationSettings'])),
  },
});

interface BrandingFormData {
  primary_color: string;
  secondary_color: string;
  logo_url?: string | null; // Para exibir o logo atual
}

const OrganizationBrandingPageContent = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation(['organizationSettings', 'common']);
  const { user, branding: currentBranding, refreshBranding, isLoading: authLoading } = useAuth();
  const notify = useNotifier();

  const [formData, setFormData] = useState<BrandingFormData>({
    primary_color: '',
    secondary_color: '',
    logo_url: null,
  });
  const [logoFile, setLogoFile] = useState<File | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [previewLogoUrl, setPreviewLogoUrl] = useState<string | null>(null);

  useEffect(() => {
    if (!authLoading && currentBranding) {
      setFormData({
        primary_color: currentBranding.primaryColor || '#4F46E5', // Default Tailwind Indigo
        secondary_color: currentBranding.secondaryColor || '#7C3AED', // Default Tailwind Purple
        logo_url: currentBranding.logoUrl,
      });
      setPreviewLogoUrl(currentBranding.logoUrl || null);
    }
  }, [currentBranding, authLoading]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0];
      if (file.size > 2 * 1024 * 1024) { // 2MB limit
        notify.error(t('branding.error_logo_too_large'));
        setLogoFile(null);
        setPreviewLogoUrl(formData.logo_url || null); // Reverter para o logo atual ou anterior
        e.target.value = ''; // Limpar o input
        return;
      }
      const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/svg+xml'];
      if (!allowedTypes.includes(file.type)) {
        notify.error(t('branding.error_invalid_logo_format'));
        setLogoFile(null);
        setPreviewLogoUrl(formData.logo_url || null);
        e.target.value = '';
        return;
      }
      setLogoFile(file);
      setPreviewLogoUrl(URL.createObjectURL(file)); // Preview do novo logo
    } else {
      setLogoFile(null);
      setPreviewLogoUrl(formData.logo_url || null); // Reverter se nenhum arquivo for selecionado
    }
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!user?.organization_id) {
      notify.error(t('common:error_missing_organization_id'));
      return;
    }
    setIsLoading(true);

    const submissionData = new FormData();
    submissionData.append('data', JSON.stringify({
      primary_color: formData.primary_color,
      secondary_color: formData.secondary_color,
    }));

    if (logoFile) {
      submissionData.append('logo_file', logoFile);
    }

    try {
      await apiClient.put(`/organizations/${user.organization_id}/branding`, submissionData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      notify.success(t('branding.success_updated'));
      await refreshBranding(); // Atualizar o branding no AuthContext
      // O useEffect já deve atualizar o preview se refreshBranding buscar o novo logo_url
      // Se o logo_file foi enviado, idealmente a API retornaria o novo logo_url para atualizar o preview
      // Por agora, refreshBranding deve ser suficiente se ele buscar o novo logo_url.
      setLogoFile(null); // Limpar o arquivo selecionado após o upload
      const fileInput = document.getElementById('logo_file_input') as HTMLInputElement;
      if (fileInput) fileInput.value = '';


    } catch (err: any) {
      console.error("Erro ao atualizar branding:", err);
      notify.error(err.response?.data?.error || t('common:unknown_error'));
    } finally {
      setIsLoading(false);
    }
  };

  const pageTitle = t('branding.page_title');
  const appName = t('common:app_name');

  return (
    <AdminLayout title={`${pageTitle} - ${appName}`}>
      <Head>
        <title>{`${pageTitle} - ${appName}`}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white mb-8">
          {pageTitle}
        </h1>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Logo Preview */}
            <div className="mb-6">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('branding.logo_preview_label')}</label>
                <div className="mt-1 flex items-center justify-center w-full h-32 bg-gray-100 dark:bg-gray-700 rounded-md border border-gray-300 dark:border-gray-600 overflow-hidden">
                    {previewLogoUrl ? (
                        <img src={previewLogoUrl} alt={t('branding.logo_preview_alt')} className="max-h-full max-w-full object-contain p-2" />
                    ) : (
                        <span className="text-gray-400 dark:text-gray-500 text-sm">{t('branding.no_logo_preview')}</span>
                    )}
                </div>
            </div>

            <div>
              <label htmlFor="logo_file_input" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                {t('branding.logo_label')} <span className="text-xs text-gray-500 dark:text-gray-400">({t('branding.logo_specs')})</span>
              </label>
              <input
                type="file"
                name="logo_file"
                id="logo_file_input"
                accept="image/jpeg,image/png,image/gif,image/svg+xml"
                onChange={handleFileChange}
                className="mt-1 block w-full text-sm text-gray-900 border border-gray-300 rounded-lg cursor-pointer bg-gray-50 dark:text-gray-400 focus:outline-none dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400 file:mr-4 file:py-2 file:px-4 file:rounded-l-lg file:border-0 file:text-sm file:font-semibold file:bg-brand-primary/10 file:text-brand-primary hover:file:bg-brand-primary/20 dark:file:bg-brand-primary/20 dark:file:text-brand-primary dark:hover:file:bg-brand-primary/30"
              />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label htmlFor="primary_color" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('branding.primary_color_label')}</label>
                <div className="mt-1 flex rounded-md shadow-sm">
                    <span className="inline-flex items-center px-3 rounded-l-md border border-r-0 border-gray-300 bg-gray-50 text-gray-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-400 text-sm"
                          style={{ backgroundColor: formData.primary_color, color: '#fff', textShadow: '0 0 2px black' }}>
                        Aa
                    </span>
                    <input type="text" name="primary_color" id="primary_color" value={formData.primary_color} onChange={handleInputChange} required pattern="^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$"
                          title={t('branding.hex_color_title')}
                          className="block w-full rounded-none rounded-r-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
                </div>
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{t('branding.hex_color_instruction')}</p>
              </div>
              <div>
                <label htmlFor="secondary_color" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('branding.secondary_color_label')}</label>
                 <div className="mt-1 flex rounded-md shadow-sm">
                    <span className="inline-flex items-center px-3 rounded-l-md border border-r-0 border-gray-300 bg-gray-50 text-gray-500 dark:bg-gray-700 dark:border-gray-600 dark:text-gray-400 text-sm"
                          style={{ backgroundColor: formData.secondary_color, color: '#fff', textShadow: '0 0 2px black' }}>
                        Aa
                    </span>
                    <input type="text" name="secondary_color" id="secondary_color" value={formData.secondary_color} onChange={handleInputChange} required pattern="^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$"
                          title={t('branding.hex_color_title')}
                          className="block w-full rounded-none rounded-r-md border-gray-300 shadow-sm focus:border-brand-primary focus:ring-brand-primary dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
                </div>
                <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{t('branding.hex_color_instruction')}</p>
              </div>
            </div>

            <div className="flex justify-end pt-4">
              <button type="submit" disabled={isLoading || authLoading}
                      className="px-6 py-2 text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 rounded-md shadow-sm disabled:opacity-50 flex items-center">
                {isLoading && (
                  <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                )}
                {t('common:save_changes_button')}
              </button>
            </div>
          </form>
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(OrganizationBrandingPageContent);
