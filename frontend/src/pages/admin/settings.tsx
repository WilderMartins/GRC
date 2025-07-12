import { useState, useEffect } from 'react';
import AdminLayout from '@/components/layouts/AdminLayout';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetStaticProps, InferGetStaticPropsType } from 'next';
import axios from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier';

type SystemSetting = {
  key: string;
  value: string;
  description: string;
  is_encrypted: boolean;
};

type Props = {};

export const getStaticProps: GetStaticProps<Props> = async ({ locale }) => ({
  props: {
    ...(await serverSideTranslations(locale ?? 'pt', ['common', 'adminSettings'])),
  },
});

const AdminSettingsPage = (props: InferGetStaticPropsType<typeof getStaticProps>) => {
  const { t } = useTranslation('adminSettings');
  const { showSuccess, showError } = useNotifier();
  const [settings, setSettings] = useState<SystemSetting[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    const fetchSettings = async () => {
      try {
        const response = await axios.get('/api/v1/admin/settings');
        setSettings(response.data);
      } catch (error) {
        showError(t('notifications.load_error'));
      } finally {
        setIsLoading(false);
      }
    };
    fetchSettings();
  }, [showError, t]);

  const handleInputChange = (key: string, value: string) => {
    setSettings((prevSettings) =>
      prevSettings.map((setting) =>
        setting.key === key ? { ...setting, value } : setting
      )
    );
  };

  const handleSave = async () => {
    setIsSaving(true);
    try {
      const payload = {
        settings: settings.map(({ key, value }) => ({ key, value })),
      };
      await axios.put('/api/v1/admin/settings', payload);
      showSuccess(t('notifications.save_success'));
    } catch (error) {
      showError(t('notifications.save_error'));
    } finally {
      setIsSaving(false);
    }
  };

  if (isLoading) {
    return (
      <AdminLayout title={t('page_title')}>
        <div className="text-center py-10">{t('loading')}</div>
      </AdminLayout>
    );
  }

  return (
    <AdminLayout title={t('page_title')}>
      <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">
            {t('form_title')}
          </h3>
          <p className="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-300">
            {t('form_description')}
          </p>
        </div>
        <div className="border-t border-gray-200 dark:border-gray-700 px-4 py-5 sm:p-0">
          <dl className="sm:divide-y sm:divide-gray-200 dark:sm:divide-gray-700">
            {settings.map((setting) => (
              <div key={setting.key} className="py-4 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
                <dt className="text-sm font-medium text-gray-500 dark:text-gray-300">
                  {t(`settings.${setting.key}.label`, setting.key)}
                  <p className="text-xs text-gray-400 dark:text-gray-500">
                    {t(`settings.${setting.key}.description`, setting.description)}
                  </p>
                </dt>
                <dd className="mt-1 text-sm text-gray-900 dark:text-white sm:mt-0 sm:col-span-2">
                  <input
                    type={setting.is_encrypted ? 'password' : 'text'}
                    id={setting.key}
                    value={setting.value}
                    onChange={(e) => handleInputChange(setting.key, e.target.value)}
                    className="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm dark:bg-gray-700 dark:text-white"
                  />
                </dd>
              </div>
            ))}
          </dl>
        </div>
        <div className="px-4 py-3 bg-gray-50 dark:bg-gray-700/50 text-right sm:px-6">
          <button
            type="button"
            onClick={handleSave}
            disabled={isSaving}
            className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary disabled:opacity-50"
          >
            {isSaving ? t('buttons.saving') : t('buttons.save')}
          </button>
        </div>
      </div>
    </AdminLayout>
  );
};

export default AdminSettingsPage;
