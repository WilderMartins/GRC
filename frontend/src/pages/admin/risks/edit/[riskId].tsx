import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import RiskForm from '@/components/risks/RiskForm'; // Importar o formulário
import { useEffect, useState } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path

// Supondo que a interface Risk já está definida em algum lugar acessível ou a definimos aqui
// Para simplificar, vou copiar a interface Risk de index.tsx, mas idealmente seria compartilhada.
type RiskStatus = "aberto" | "em_andamento" | "mitigado" | "aceito";
type RiskImpact = "Baixo" | "Médio" | "Alto" | "Crítico";
type RiskProbability = "Baixo" | "Médio" | "Alto" | "Crítico";
interface RiskOwner { id: string; name: string; email: string; }
interface Risk {
  id: string;
  organization_id: string;
  title: string;
  description: string;
  category: string;
  impact: RiskImpact;
  probability: RiskProbability;
  status: RiskStatus;
  owner_id: string;
  owner?: RiskOwner;
  created_at: string;
  updated_at: string;
}


const EditRiskPageContent = () => {
  const router = useRouter();
  const { riskId } = router.query;
  const [initialData, setInitialData] = useState<Risk | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (riskId && typeof riskId === 'string') {
      setIsLoading(true);
      setError(null);
      apiClient.get(`/risks/${riskId}`)
        .then(response => {
          // Ajustar os dados para o formato esperado por RiskFormData se necessário
          // Por exemplo, owner_id já é uma string.
          setInitialData(response.data);
        })
        .catch(err => {
          console.error("Erro ao buscar dados do risco:", err);
          setError(err.response?.data?.error || err.message || "Falha ao buscar dados do risco.");
        })
        .finally(() => setIsLoading(false));
    } else if (riskId) { // Se riskId for um array (não esperado, mas para segurança)
        setError("ID do Risco inválido.");
        setIsLoading(false);
    }
    // Não executar se riskId for undefined (na primeira renderização antes do router estar pronto)
  }, [riskId]);

  const handleSuccess = () => {
    alert('Risco atualizado com sucesso!'); // Placeholder
    router.push('/admin/risks');
  };

  if (isLoading && !initialData) { // Mostrar loading apenas se não houver dados ainda
    return <AdminLayout title="Carregando..."><div className="text-center p-10">Carregando dados do risco...</div></AdminLayout>;
  }

  if (error) {
    return <AdminLayout title="Erro"><div className="text-center p-10 text-red-500">Erro: {error}</div></AdminLayout>;
  }

  if (!initialData && !isLoading) { // Se terminou de carregar e não encontrou dados (ou ID inválido)
     return <AdminLayout title="Risco não encontrado"><div className="text-center p-10">Risco não encontrado.</div></AdminLayout>;
  }


  return (
    <AdminLayout title={`Editar Risco - Phoenix GRC`}>
      <Head>
        <title>Editar Risco {initialData?.title || ''} - Phoenix GRC</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            Editar Risco: <span className="text-indigo-600 dark:text-indigo-400">{initialData?.title}</span>
          </h1>
          <Link href="/admin/risks" legacyBehavior>
            <a className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-200">
              &larr; Voltar para Lista de Riscos
            </a>
          </Link>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          {initialData && (
            <RiskForm
              initialData={{
                // Mapear dados do Risco para RiskFormData
                id: initialData.id,
                title: initialData.title,
                description: initialData.description,
                category: initialData.category as any, // Cast se necessário e se RiskCategory do form for mais restrito
                impact: initialData.impact,
                probability: initialData.probability,
                status: initialData.status,
                owner_id: initialData.owner_id,
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

export default WithAuth(EditRiskPageContent);
