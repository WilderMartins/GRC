import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import VulnerabilityForm from '@/components/vulnerabilities/VulnerabilityForm'; // Importar o formulário
import { useRouter } from 'next/router';

const NewVulnerabilityPageContent = () => {
  const router = useRouter();

  const handleSuccess = () => {
    // Poderia exibir uma notificação de sucesso aqui antes de redirecionar
    alert('Vulnerabilidade criada com sucesso!'); // Placeholder
    router.push('/admin/vulnerabilities');
  };

  return (
    <AdminLayout title="Adicionar Nova Vulnerabilidade - Phoenix GRC">
      <Head>
        <title>Adicionar Nova Vulnerabilidade - Phoenix GRC</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
            Adicionar Nova Vulnerabilidade
          </h1>
          <Link href="/admin/vulnerabilities" legacyBehavior>
            <a className="text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 dark:hover:text-indigo-200">
              &larr; Voltar para Lista de Vulnerabilidades
            </a>
          </Link>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow sm:rounded-lg p-6 md:p-8">
          <VulnerabilityForm onSubmitSuccess={handleSuccess} />
        </div>
      </div>
    </AdminLayout>
  );
};

export default WithAuth(NewVulnerabilityPageContent);
