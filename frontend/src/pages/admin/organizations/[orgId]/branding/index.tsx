import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import { useRouter } from 'next/router';
import { useAuth } from '@/contexts/AuthContext';
import { useEffect, useState, useCallback, FormEvent } from 'react';
import apiClient from '@/lib/axios';
import Link from 'next/link';

interface OrganizationBranding {
  logo_url?: string;
  primary_color?: string;
  secondary_color?: string;
}

const OrgBrandingPageContent = () => {
  const router = useRouter();
  const { orgId } = router.query;
  const { user, isLoading: authIsLoading, refreshBranding } = useAuth(); // Adicionar refreshBranding

  const [canAccess, setCanAccess] = useState(false);
  const [pageError, setPageError] = useState<string | null>(null);

  const [currentBranding, setCurrentBranding] = useState<OrganizationBranding>({});
  const [isLoadingData, setIsLoadingData] = useState(true);
  const [dataError, setDataError] = useState<string | null>(null);

  const [primaryColor, setPrimaryColor] = useState('');
  const [secondaryColor, setSecondaryColor] = useState('');
  const [logoFile, setLogoFile] = useState<File | null>(null);
  const [logoPreview, setLogoPreview] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitSuccess, setSubmitSuccess] = useState<string | null>(null);


  const fetchBranding = useCallback(async () => {
    if (!canAccess || !orgId || typeof orgId !== 'string') return;
    setIsLoadingData(true);
    setDataError(null);
    try {
      const response = await apiClient.get<OrganizationBranding>(`/organizations/${orgId}/branding`);
      setCurrentBranding(response.data || {});
      setPrimaryColor(response.data.primary_color || '#FFFFFF'); // Default to white
      setSecondaryColor(response.data.secondary_color || '#000000'); // Default to black
      if (response.data.logo_url) {
        setLogoPreview(response.data.logo_url);
      }
    } catch (err: any) {
      setDataError(err.response?.data?.error || "Falha ao buscar configurações de branding.");
    } finally {
      setIsLoadingData(false);
    }
  }, [orgId, canAccess]);

  useEffect(() => {
    if (authIsLoading) return;
    if (!user) { setPageError("Usuário não autenticado."); setCanAccess(false); setIsLoadingData(false); return; }
    if (user.organization_id !== orgId) {
      setPageError("Você não tem permissão para acessar as configurações desta organização.");
      setCanAccess(false); setIsLoadingData(false); return;
    }
    if (user.role !== 'admin' && user.role !== 'manager') {
      setPageError("Você não tem privilégios suficientes (requer Admin ou Manager).");
      setCanAccess(false); setIsLoadingData(false); return;
    }
    setCanAccess(true);
    setPageError(null);
  }, [orgId, user, authIsLoading]);

  useEffect(() => {
    if (canAccess) { // Fetch data only if access is confirmed
        fetchBranding();
    }
  }, [canAccess, fetchBranding]);


  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      setLogoFile(file);
      const reader = new FileReader();
      reader.onloadend = () => {
        setLogoPreview(reader.result as string);
      };
      reader.readAsDataURL(file);
    } else {
      setLogoFile(null);
      setLogoPreview(currentBranding.logo_url || null); // Reverter para o logo atual se o arquivo for desmarcado
    }
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!orgId || typeof orgId !== 'string' || !canAccess) return;

    setIsSubmitting(true);
    setSubmitSuccess(null);
    setDataError(null); // Limpar erro de fetch anterior

    const formData = new FormData();
    const brandingData: any = {};
    if (primaryColor !== (currentBranding.primary_color || '#FFFFFF')) {
        brandingData.primary_color = primaryColor;
    }
    if (secondaryColor !== (currentBranding.secondary_color || '#000000')) {
        brandingData.secondary_color = secondaryColor;
    }

    // Adicionar 'data' apenas se houver cores para atualizar,
    // ou se o backend esperar o campo 'data' mesmo que vazio.
    // Para PUT, geralmente enviamos apenas o que muda.
    // O handler do backend espera 'data' se cores forem enviadas.
    if (Object.keys(brandingData).length > 0) {
        formData.append('data', JSON.stringify(brandingData));
    }


    if (logoFile) {
      formData.append('logo_file', logoFile);
    } else if (!logoFile && logoPreview !== currentBranding.logo_url) {
        // Caso o usuário tenha removido o preview de um logo existente e não selecionou novo,
        // e queiramos que isso signifique "remover o logo".
        // Isso exigiria uma lógica adicional no backend para tratar um valor especial ou ausência de logo_url.
        // Por enquanto, não enviar nada significa "manter o logo_url existente".
        // Se quisermos permitir remover o logo, precisaríamos de um campo "remover_logo: true" no JSON 'data'.
        // Para este exemplo, se nenhum arquivo for enviado, o backend não altera o LogoURL a menos que um novo seja enviado.
    }


    try {
      const response = await apiClient.put<OrganizationBranding>(`/organizations/${orgId}/branding`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      setCurrentBranding(response.data || {}); // Atualiza o estado local com a resposta
      setPrimaryColor(response.data.primary_color || '#FFFFFF');
      setSecondaryColor(response.data.secondary_color || '#000000');
      if (response.data.logo_url) {
        setLogoPreview(response.data.logo_url);
      } else if (!logoFile) { // Se a resposta não tem logo_url e não enviamos um novo, significa que foi removido
        setLogoPreview(null);
      }
      setLogoFile(null); // Resetar o input do arquivo
      (document.getElementById('logo_file_input') as HTMLInputElement).value = ''; // Limpar o input file visualmente
      setSubmitSuccess('Configurações de branding atualizadas com sucesso!');
      await refreshBranding(); // Chamar refreshBranding para atualizar o contexto global
    } catch (err: any) {
      setDataError(err.response?.data?.error || "Falha ao atualizar branding.");
    } finally {
      setIsSubmitting(false);
    }
  };


  if (authIsLoading || (isLoadingData && canAccess)) {
    return <AdminLayout title="Carregando..."><div className="p-6 text-center">Carregando configurações...</div></AdminLayout>;
  }
  if (!canAccess && pageError) {
    return <AdminLayout title="Acesso Negado"><div className="p-6 text-center text-red-500">{pageError}</div></AdminLayout>;
  }
   if (!canAccess && !pageError && !authIsLoading) {
    return <AdminLayout title="Carregando..."><div className="p-6 text-center">Verificando organização...</div></AdminLayout>;
  }


  return (
    <AdminLayout title={`Branding - Organização`}>
      <Head><title>Identidade Visual (Branding) - Phoenix GRC</title></Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="sm:flex sm:items-center sm:justify-between mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">Identidade Visual da Organização</h1>
           <Link href={`/admin/dashboard`} legacyBehavior>
            <a className="text-sm text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-200">
              &larr; Voltar para o Dashboard Admin
            </a>
          </Link>
        </div>

        {dataError && <p className="mb-4 text-sm text-red-600 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md">{dataError}</p>}
        {submitSuccess && <p className="mb-4 text-sm text-green-600 bg-green-100 dark:bg-green-900 dark:text-green-200 p-3 rounded-md">{submitSuccess}</p>}

        <form onSubmit={handleSubmit} className="space-y-8 bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          {/* Logo Upload */}
          <div>
            <label htmlFor="logo_file_input" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Logo da Organização (PNG, JPG, GIF, SVG - Máx 2MB)
            </label>
            <div className="mt-2 flex items-center space-x-6">
              <div className="shrink-0">
                {logoPreview ? (
                  <img className="h-16 w-auto object-contain bg-gray-200 dark:bg-gray-700 p-1 rounded" src={logoPreview} alt="Preview do Logo" />
                ) : (
                  <div className="h-16 w-32 flex items-center justify-center rounded border-2 border-dashed border-gray-300 dark:border-gray-600 text-gray-400 dark:text-gray-500">
                    Sem Logo
                  </div>
                )}
              </div>
              <label htmlFor="logo_file_input" className="cursor-pointer rounded-md bg-white dark:bg-gray-700 px-3 py-2 text-sm font-semibold text-gray-900 dark:text-gray-100 shadow-sm ring-1 ring-inset ring-gray-300 dark:ring-gray-600 hover:bg-gray-50 dark:hover:bg-gray-600">
                <span>{logoFile ? logoFile.name : (currentBranding.logo_url ? 'Alterar logo' : 'Carregar logo')}</span>
                <input id="logo_file_input" name="logo_file" type="file" className="sr-only" onChange={handleFileChange} accept="image/png, image/jpeg, image/gif, image/svg+xml" />
              </label>
              {logoPreview && (
                <button type="button" onClick={() => { setLogoFile(null); setLogoPreview(null); (document.getElementById('logo_file_input') as HTMLInputElement).value = ''; }}
                        className="text-sm text-red-600 hover:text-red-500 dark:text-red-400 dark:hover:text-red-300">
                    Remover/Resetar Preview
                </button>
              )}
            </div>
          </div>

          {/* Color Pickers */}
          <div className="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-6">
            <div className="sm:col-span-3">
              <label htmlFor="primary_color" className="block text-sm font-medium leading-6 text-gray-900 dark:text-gray-300">
                Cor Primária
              </label>
              <div className="mt-2 flex items-center space-x-3">
                <input type="color" name="primary_color_picker" id="primary_color_picker" value={primaryColor || '#FFFFFF'}
                       onChange={(e) => setPrimaryColor(e.target.value)}
                       className="h-10 w-10 rounded-md border-gray-300 dark:border-gray-600 cursor-pointer" />
                <input type="text" name="primary_color" id="primary_color" value={primaryColor || ''}
                       onChange={(e) => setPrimaryColor(e.target.value)}
                       placeholder="#RRGGBB" maxLength={7}
                       className="block w-full rounded-md border-0 py-1.5 px-2 text-gray-900 dark:text-white shadow-sm ring-1 ring-inset ring-gray-300 dark:ring-gray-600 placeholder:text-gray-400 dark:placeholder-gray-500 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6 dark:bg-gray-700"/>
              </div>
            </div>

            <div className="sm:col-span-3">
              <label htmlFor="secondary_color" className="block text-sm font-medium leading-6 text-gray-900 dark:text-gray-300">
                Cor Secundária
              </label>
               <div className="mt-2 flex items-center space-x-3">
                <input type="color" name="secondary_color_picker" id="secondary_color_picker" value={secondaryColor || '#000000'}
                       onChange={(e) => setSecondaryColor(e.target.value)}
                       className="h-10 w-10 rounded-md border-gray-300 dark:border-gray-600 cursor-pointer"/>
                <input type="text" name="secondary_color" id="secondary_color" value={secondaryColor || ''}
                       onChange={(e) => setSecondaryColor(e.target.value)}
                       placeholder="#RRGGBB" maxLength={7}
                       className="block w-full rounded-md border-0 py-1.5 px-2 text-gray-900 dark:text-white shadow-sm ring-1 ring-inset ring-gray-300 dark:ring-gray-600 placeholder:text-gray-400 dark:placeholder-gray-500 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6 dark:bg-gray-700"/>
              </div>
            </div>
          </div>

          <div className="pt-6 flex items-center justify-end gap-x-6 border-t border-gray-200 dark:border-gray-700">
            <button type="button" onClick={() => fetchBranding()} disabled={isSubmitting || isLoadingData}
                    className="text-sm font-semibold leading-6 text-gray-900 dark:text-gray-100 disabled:opacity-50 transition-colors">
              Resetar (Recarregar do Servidor)
            </button>
            <button type="submit" disabled={isSubmitting || isLoadingData}
                    className="rounded-md bg-brand-primary px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-brand-primary/90 focus:ring-brand-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary disabled:opacity-50 transition-colors">
              {isSubmitting ? 'Salvando...' : 'Salvar Configurações'}
            </button>
          </div>
        </form>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(OrgBrandingPageContent);
