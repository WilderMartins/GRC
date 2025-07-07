import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import VulnerabilityForm from '@/components/vulnerabilities/VulnerabilityForm'; // Importar
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path

// Tipos (copiados de index.tsx para simplificar, idealmente seriam compartilhados)
type VulnerabilitySeverity = "Baixo" | "Médio" | "Alto" | "Crítico";
type VulnerabilityStatus = "descoberta" | "em_correcao" | "corrigida";
interface Vulnerability {
  id: string;
  organization_id: string;
  title: string;
  description: string;
  cve_id?: string;
  severity: VulnerabilitySeverity;
  status: VulnerabilityStatus;
  asset_affected: string;
  created_at: string;
  updated_at: string;
}


const EditVulnerabilityPageContent = () => {
  const router = useRouter();
  const { vulnId } = router.query;
  const [initialData, setInitialData] = useState<Vulnerability | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (vulnId && typeof vulnId === 'string') {
      setIsLoading(true);
      setError(null);
      apiClient.get(`/vulnerabilities/${vulnId}`)
        .then(response => {
          setInitialData(response.data);
        })
        .catch(err => {
          console.error("Erro ao buscar dados da vulnerabilidade:", err);
          setError(err.response?.data?.error || err.message || "Falha ao buscar dados da vulnerabilidade.");
        })
        .finally(() => setIsLoading(false));
    } else if (vulnId) {
        setError("ID da Vulnerabilidade inválido.");
        setIsLoading(false);
    }
  }, [vulnId]);

  const handleSuccess = () => {
    alert('Vulnerabilidade atualizada com sucesso!'); // Placeholder
    router.push('/admin/vulnerabilities');
  };

  if (isLoading && !initialData) {
    return <AdminLayout title="Carregando..."><div className="text-center p-10">Carregando dados da vulnerabilidade...</div></AdminLayout>;
  }

  if (error) {
    return <AdminLayout title="Erro"><div className="text-center p-10 text-red-500">Erro: {error}</div></AdminLayout>;
  }

  if (!initialData && !isLoading) {
     return <AdminLayout title="Vulnerabilidade não encontrada"><div className="text-center p-10">Vulnerabilidade não encontrada.</div></AdminLayout>;
  }

  return (
    <AdminLayout title={`Editar Vulnerabilidade - Phoenix GRC`}>
      <Head>
        <title>Editar Vulnerabilidade {initialData?.title || ''} - Phoenix GRC</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            Editar Vulnerabilidade: <span className="text-indigo-600 dark:text-indigo-400">{initialData?.title}</span>
          </h1>
          <Link href="/admin/vulnerabilities" legacyBehavior>
            <a className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-200">
              &larr; Voltar para Lista de Vulnerabilidades
            </a>
          </Link>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          {initialData && (
            <VulnerabilityForm
              initialData={{
                id: initialData.id,
                title: initialData.title,
                description: initialData.description,
                cve_id: initialData.cve_id || '',
                severity: initialData.severity,
                status: initialData.status,
                asset_affected: initialData.asset_affected,
              }}
              isEditing={true}
              onSubmitSuccess={handleSuccess}
            />
          )}
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(EditVulnerabilityPageContent);
