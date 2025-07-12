import { useRouter } from 'next/router';
import Head from 'next/head';
import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import type { GetServerSideProps, InferGetServerSidePropsType } from 'next';
import Link from 'next/link';
import { useEffect, useState, useMemo } from 'react';
import apiClient from '@/lib/axios';
import { ControlWithAssessment, AuditAssessmentStatus, AuditFramework, ComplianceScoreResponse, C2M2MaturityFrameworkSummaryResponse } from '@/types';
import AssessmentFormModal from '@/components/audit/AssessmentFormModal';
import StatCard from '@/components/common/StatCard';

type Props = {}

export const getServerSideProps: GetServerSideProps<Props> = async ({ locale }) => {
  return {
    props: {
      ...(await serverSideTranslations(locale ?? 'pt', ['common', 'audit'])),
    },
  };
};

const FrameworkDetailsPageContent = (props: InferGetServerSidePropsType<typeof getServerSideProps>) => {
  const { t } = useTranslation(['audit', 'common']);
  const router = useRouter();
  const { frameworkId } = router.query;
  const { user } = useAuth(); // Para obter o orgId

  const [framework, setFramework] = useState<AuditFramework | null>(null);
  const [controls, setControls] = useState<ControlWithAssessment[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Estados para os sumários
  const [complianceScore, setComplianceScore] = useState<ComplianceScoreResponse | null>(null);
  const [c2m2Summary, setC2m2Summary] = useState<C2M2MaturityFrameworkSummaryResponse | null>(null);
  const [isLoadingSummaries, setIsLoadingSummaries] = useState(true);

  // Estados para o modal
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedControl, setSelectedControl] = useState<ControlWithAssessment | null>(null);

  // Filtros
  const [statusFilter, setStatusFilter] = useState<AuditAssessmentStatus | 'nao_avaliado' | ''>('');
  const [familyFilter, setFamilyFilter] = useState<string>('');

  const controlFamilies = useMemo(() => {
    if (!controls) return [];
    const families = controls.map(c => c.Family);
    return [...new Set(families)].sort();
  }, [controls]);

  const filteredControls = useMemo(() => {
    return controls.filter(control => {
      const statusMatch = !statusFilter || (statusFilter === 'nao_avaliado' ? !control.assessment : control.assessment?.Status === statusFilter);
      const familyMatch = !familyFilter || control.Family === familyFilter;
      return statusMatch && familyMatch;
    });
  }, [controls, statusFilter, familyFilter]);

  const handleOpenModal = (control: ControlWithAssessment) => {
    setSelectedControl(control);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setSelectedControl(null);
  };

  const handleAssessmentSuccess = () => {
    // Re-fetch dos controles para obter a avaliação atualizada
    if (frameworkId && typeof frameworkId === 'string') {
        apiClient.get<ControlWithAssessment[]>(`/api/v1/audit/frameworks/${frameworkId}/controls`)
            .then(response => {
                setControls(response.data);
            })
            .catch(err => {
                console.error("Failed to re-fetch controls after assessment:", err);
                // Opcional: notificar o usuário sobre o erro de re-fetch
            });
    }
    handleCloseModal();
  };


  useEffect(() => {
    if (frameworkId && user?.organization_id) {
      setIsLoading(true);
      setIsLoadingSummaries(true);
      setError(null);

      const orgId = user.organization_id;

      const fetchFrameworkDetails = apiClient.get<AuditFramework[]>(`/api/v1/audit/frameworks`).then(res => {
          const currentFramework = res.data.find(fw => fw.ID === frameworkId);
          if (currentFramework) {
              setFramework(currentFramework);
          } else {
              throw new Error(t('controls_list.error_framework_not_found'));
          }
      });

      const fetchControls = apiClient.get<ControlWithAssessment[]>(`/api/v1/audit/frameworks/${frameworkId}/controls`);

      const fetchSummaries = () => {
          const scorePromise = apiClient.get<ComplianceScoreResponse>(`/api/v1/audit/organizations/${orgId}/frameworks/${frameworkId}/compliance-score`);
          const c2m2Promise = apiClient.get<C2M2MaturityFrameworkSummaryResponse>(`/api/v1/audit/organizations/${orgId}/frameworks/${frameworkId}/c2m2-maturity-summary`);

          Promise.all([scorePromise, c2m2Promise])
            .then(([scoreRes, c2m2Res]) => {
                setComplianceScore(scoreRes.data);
                setC2m2Summary(c2m2Res.data);
            }).catch(summaryErr => {
                console.error("Failed to load summary data:", summaryErr);
                // Não definir erro principal para não bloquear a lista de controles
            }).finally(() => {
                setIsLoadingSummaries(false);
            });
      };

      Promise.all([fetchFrameworkDetails, fetchControls])
        .then(([, controlsResponse]) => { // O resultado de fetchFrameworkDetails já foi tratado no .then
          setControls(controlsResponse.data);
          // Buscar sumários após o carregamento principal ter sucesso
          fetchSummaries();
        })
        .catch(err => {
          console.error(t('controls_list.error_loading_console'), err);
          setError(err.message || err.response?.data?.error || t('controls_list.error_loading'));
        })
        .finally(() => {
          setIsLoading(false);
        });
    }
  }, [frameworkId, user?.organization_id, t]);

  const getStatusColor = (status: AuditAssessmentStatus | null) => {
    if (!status) return 'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200';
    switch (status) {
      case 'conforme': return 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100';
      case 'nao_conforme': return 'bg-red-100 text-red-800 dark:bg-red-700 dark:text-red-100';
      case 'parcialmente_conforme': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-700 dark:text-yellow-100';
      default: return 'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200';
    }
  };

  const pageTitle = framework?.Name || t('controls_list.page_title_fallback');
  const appName = t('common:app_name');

  return (
    <AdminLayout title={`${pageTitle} - ${appName}`}>
      <Head>
        <title>{`${pageTitle} - ${appName}`}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-6">
            <Link href="/admin/audit" legacyBehavior>
                <a className="text-sm text-brand-primary hover:text-brand-primary/80 dark:hover:text-brand-primary/70">
                    &larr; {t('common:back_to_list_link_generic', { list_name: t('framework_list.page_title') })}
                </a>
            </Link>
        </div>
        <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white mb-2">
          {pageTitle}
        </h1>
        <p className="text-gray-600 dark:text-gray-400 mb-8">{framework?.Description}</p>

        {/* Seção de Sumários */}
        <div className="mb-8">
            <h2 className="text-xl font-semibold text-gray-800 dark:text-white mb-4">{t('summaries.compliance_score_title')}</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                <StatCard title={t('summaries.score')} value={`${complianceScore?.compliance_score.toFixed(1) ?? '-'}%`} isLoading={isLoadingSummaries} />
                <StatCard title={t('summaries.evaluated_controls')} value={`${complianceScore?.evaluated_controls ?? '-'}/${complianceScore?.total_controls ?? '-'}`} isLoading={isLoadingSummaries} />
                <StatCard title={t('summaries.conformant')} value={complianceScore?.conformant_controls ?? '-'} isLoading={isLoadingSummaries} />
                <StatCard title={t('summaries.non_conformant')} value={complianceScore?.non_conformant_controls ?? '-'} isLoading={isLoadingSummaries} />
            </div>
        </div>

        <div className="mb-8">
             <h2 className="text-xl font-semibold text-gray-800 dark:text-white mb-4">{t('summaries.c2m2_maturity_title')}</h2>
             {isLoadingSummaries && <p>{t('common:loading_ellipsis')}</p>}
             {c2m2Summary && (
                <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                        <thead className="bg-gray-50 dark:bg-gray-700">
                            <tr>
                                <th className="py-3.5 px-3 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('summaries.c2m2_function_header')}</th>
                                <th className="py-3.5 px-3 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('summaries.c2m2_achieved_mil_header')}</th>
                                <th className="py-3.5 px-3 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('summaries.c2m2_evaluated_header')}</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                            {c2m2Summary.summary_by_function.map(item => (
                                <tr key={item.nist_component_name}>
                                    <td className="whitespace-nowrap py-4 px-3 text-sm font-medium text-gray-900 dark:text-white">{item.nist_component_name}</td>
                                    <td className="whitespace-nowrap py-4 px-3 text-sm text-gray-500 dark:text-gray-300">MIL {item.achieved_mil}</td>
                                    <td className="whitespace-nowrap py-4 px-3 text-sm text-gray-500 dark:text-gray-300">{item.evaluated_controls} / {item.total_controls}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
             )}
        </div>

        {isLoading && <p className="text-center">{t('common:loading_ellipsis')}</p>}
        {error && <p className="text-center text-red-500">{error}</p>}

        {!isLoading && !error && (
            <>
            {/* Filtros */}
            <div className="my-6 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg shadow">
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                    <div>
                        <label htmlFor="familyFilter" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('controls_list.filter_family_label')}</label>
                        <select id="familyFilter" value={familyFilter} onChange={(e) => setFamilyFilter(e.target.value)}
                                className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm rounded-md">
                            <option value="">{t('common:all_option')}</option>
                            {controlFamilies.map(family => <option key={family} value={family}>{family}</option>)}
                        </select>
                    </div>
                    <div>
                        <label htmlFor="statusFilter" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('controls_list.filter_status_label')}</label>
                        <select id="statusFilter" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value as any)}
                                className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm rounded-md">
                            <option value="">{t('common:all_option')}</option>
                            <option value="nao_avaliado">{t('controls_list.status_not_assessed')}</option>
                            <option value="conforme">{t('assessment_status.conforme')}</option>
                            <option value="nao_conforme">{t('assessment_status.nao_conforme')}</option>
                            <option value="parcialmente_conforme">{t('assessment_status.parcialmente_conforme')}</option>
                        </select>
                    </div>
                </div>
            </div>

            {/* Tabela de Controles */}
            <div className="mt-8 flow-root">
              <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                  <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                    <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                      <thead className="bg-gray-50 dark:bg-gray-700">
                        <tr>
                          <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6 w-1/6">{t('controls_list.header_control_id')}</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white w-3/6">{t('controls_list.header_description')}</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white w-1/6">{t('controls_list.header_family')}</th>
                          <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white w-1/6">{t('controls_list.header_status')}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                        {filteredControls.map((control) => (
                          <tr key={control.ID} onClick={() => handleOpenModal(control)} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer">
                            <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{control.ControlID}</td>
                            <td className="px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{control.Description}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{control.Family}</td>
                            <td className="whitespace-nowrap px-3 py-4 text-sm">
                                <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(control.assessment?.Status || null)}`}>
                                    {control.assessment ? t(`assessment_status.${control.assessment.Status}`) : t('controls_list.status_not_assessed')}
                                </span>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                     {filteredControls.length === 0 && (
                        <p className="text-center text-gray-500 py-4">{t('controls_list.no_controls_found')}</p>
                    )}
                  </div>
                </div>
              </div>
            </div>
            </>
        )}

        {isModalOpen && selectedControl && (
            <AssessmentFormModal
                isOpen={isModalOpen}
                onClose={handleCloseModal}
                control={selectedControl}
                onSubmitSuccess={handleAssessmentSuccess}
            />
        )}
      </div>
    </AdminLayout>
  );
};

export default WithAuth(FrameworkDetailsPageContent);
