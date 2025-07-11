import AdminLayout from '@/components/layouts/AdminLayout';
import WithAuth from '@/components/auth/WithAuth';
import Head from 'next/head';
import Link from 'next/link';
import { useRouter } from 'next/router';
import { useEffect, useState, useCallback } from 'react';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';
// import AssessmentForm from '@/components/audit/AssessmentForm'; // Removido, pois o modal o encapsula
import AssessmentFormModal from '@/components/audit/AssessmentFormModal'; // Importar o novo modal
import PaginationControls from '@/components/common/PaginationControls';
import {
    AuditFramework,
    AuditControl,
    AuditAssessment,
    ControlWithAssessment,
    PaginatedResponse,
    AuditAssessmentStatusFilter,
    ComplianceScoreResponse,
    AuditAssessmentStatus, // Certifique-se que este tipo/enum está importado se usado nas cores
} from '@/types';
import { useTranslation } from 'next-i18next';
import { serverSideTranslations } from 'next-i18next/serverSideTranslations';
import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip as RechartsTooltip, BarChart, Bar, XAxis, YAxis, CartesianGrid } from 'recharts';
import type { GetServerSideProps, InferGetServerSidePropsType } from 'next';


interface ControlFamiliesResponse { // Mantido local por simplicidade
    families: string[];
}

type Props = {
  // Props de getServerSideProps
}

export const getServerSideProps: GetServerSideProps<Props> = async ({ locale, params }) => {
  return {
    props: {
      ...(await serverSideTranslations(locale ?? 'pt', ['common', 'audit'])),
    },
  };
};

// Cores para os gráficos (exemplo) - Ajustar cores conforme o tema da aplicação
const ASSESSMENT_STATUS_COLORS: { [key: string]: string } = {
  conforme: '#10B981', // green-500
  parcialmente_conforme: '#F59E0B', // amber-500
  nao_conforme: '#EF4444', // red-500
  nao_avaliado: '#A0AEC0', // slate-400
  // nao_aplicavel pode não ser contado no score, mas se for, adicionar cor
};

// Componente para Gauge (simplificado com PieChart)
const ScoreGaugeChart: React.FC<{ score: number }> = ({ score }) => {
    const { t } = useTranslation('audit');
    const data = [
        { name: t('framework_detail.score_chart_segment_achieved', 'Concluído'), value: score, fill: ASSESSMENT_STATUS_COLORS.conforme /* ou brand-primary */ },
        { name: t('framework_detail.score_chart_segment_remaining', 'Restante'), value: 100 - score, fill: '#E5E7EB' /* gray-200 */ },
    ];

    return (
        <ResponsiveContainer width="100%" height={200}>
            <PieChart>
                <Pie
                    data={data}
                    cx="50%"
                    cy="50%"
                    startAngle={180}
                    endAngle={0}
                    innerRadius={60}
                    outerRadius={80}
                    paddingAngle={0}
                    dataKey="value"
                    animationDuration={800}
                >
                    {data.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={entry.fill} />
                    ))}
                </Pie>
                <RechartsTooltip formatter={(value: number, name: string) => [`${value.toFixed(1)}%`, name]}/>
                <text x="50%" y="50%" textAnchor="middle" dominantBaseline="middle" className="text-2xl font-bold fill-gray-700 dark:fill-gray-200">
                    {`${score.toFixed(1)}%`}
                </text>
            </PieChart>
        </ResponsiveContainer>
    );
};


const FrameworkDetailPageContent = (props: InferGetServerSidePropsType<typeof getServerSideProps>) => {
  const { t } = useTranslation(['audit', 'common']);
  const router = useRouter();
  const { frameworkId } = router.query; // Este é o frameworkId da URL
  const { user, isLoading: authIsLoading } = useAuth();

  const [frameworkInfo, setFrameworkInfo] = useState<Partial<AuditFramework> | null>(null);
  const [controlsWithAssessments, setControlsWithAssessments] = useState<ControlWithAssessment[]>([]);
  const [complianceScoreData, setComplianceScoreData] = useState<ComplianceScoreResponse | null>(null);

  const [isLoadingData, setIsLoadingData] = useState(true);
  const [isLoadingScore, setIsLoadingScore] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [scoreError, setScoreError] = useState<string | null>(null);

  const [controlsCurrentPage, setControlsCurrentPage] = useState(1);
  const [controlsPageSize, setControlsPageSize] = useState(10);
  const [controlsTotalPages, setControlsTotalPages] = useState(0);
  const [controlsTotalItems, setControlsTotalItems] = useState(0);

  const [availableControlFamilies, setAvailableControlFamilies] = useState<string[]>([]);
  const [filterFamily, setFilterFamily] = useState<string>("");
  const [filterAssessmentStatus, setFilterAssessmentStatus] = useState<AuditAssessmentStatusFilter>("");

  const [showAssessmentModal, setShowAssessmentModal] = useState(false);
  const [selectedControlForAssessment, setSelectedControlForAssessment] = useState<ControlWithAssessment | null>(null);

  useEffect(() => {
    if (frameworkId && typeof frameworkId === 'string' && !authIsLoading && user) {
      // Fetch Framework Info (Name)
      setIsLoadingData(true); // Pode ser combinado com isLoadingScore ou mantido separado
      apiClient.get<AuditFramework[]>('/audit/frameworks')
        .then(response => {
          const currentFramework = response.data.find(f => f.id === frameworkId);
          setFrameworkInfo(currentFramework || {id: frameworkId, name: t('framework_detail.unknown_framework_name')});
        })
        .catch(err => {
          console.error(t('framework_detail.error_fetching_framework_name_console'), err);
          setError(prev => prev ? prev + `; ${t('framework_detail.error_loading_framework_info')}` : t('framework_detail.error_loading_framework_info'));
        });

      // Fetch Control Families
      apiClient.get<ControlFamiliesResponse>(`/audit/frameworks/${frameworkId}/control-families`)
        .then(response => {
          setAvailableControlFamilies(response.data?.families?.sort() || []);
        })
        .catch(err => {
          console.warn(t('framework_detail.error_control_families_endpoint_console'), err);
          apiClient.get<PaginatedResponse<AuditControl> | AuditControl[]>(`/audit/frameworks/${frameworkId}/controls?page_size=10000`)
            .then(response => {
                let allControls: AuditControl[] = [];
                if (Array.isArray(response.data)) {
                    allControls = response.data;
                } else if (response.data && Array.isArray((response.data as PaginatedResponse<AuditControl>).items)) {
                    allControls = (response.data as PaginatedResponse<AuditControl>).items;
                }
                const families = Array.from(new Set(allControls.map(c => c.family))).sort();
                setAvailableControlFamilies(families);
            })
            .catch(deepErr => {
                console.error(t('framework_detail.error_fetching_families_alternative_console'), deepErr);
                setError(prev => prev ? prev + `; ${t('framework_detail.error_loading_family_filters')}` : t('framework_detail.error_loading_family_filters'));
            });
        });

      // Fetch Compliance Score
      if (user.organization_id) {
        setIsLoadingScore(true);
        setScoreError(null);
        apiClient.get<ComplianceScoreResponse>(`/audit/organizations/${user.organization_id}/frameworks/${frameworkId}/compliance-score`)
          .then(response => {
            setComplianceScoreData(response.data);
          })
          .catch(err => {
            console.error(t('framework_detail.error_fetching_score_console'), err);
            setScoreError(err.response?.data?.error || t('framework_detail.error_loading_score'));
          })
          .finally(() => {
            setIsLoadingScore(false);
          });
      } else {
        setIsLoadingScore(false);
        setScoreError(t('common:error_org_id_missing_for_score'));
      }
    }
  }, [frameworkId, authIsLoading, user, t]);

  const fetchControlsAndCombineAssessments = useCallback(async () => {
    if (!frameworkId || typeof frameworkId !== 'string' || !user?.organization_id || authIsLoading) {
      setIsLoadingData(false);
      if (router.isReady && !frameworkId && !error) setError(t('common:error_invalid_id_url', { entity: t('common:framework_singular')}));
      if (!authIsLoading && !user?.organization_id && !error) setError(t('common:error_org_id_missing'));
      return;
    }
    setIsLoadingData(true);
    // setError(null); // Reset error for this specific fetch, or manage errors more granularly

    try {
      const controlsParams: { page: number; page_size: number; family?: string } = {
        page: controlsCurrentPage,
        page_size: controlsPageSize
      };
      if (filterFamily) {
        controlsParams.family = filterFamily;
      }
      const controlsResponse = await apiClient.get<PaginatedResponse<AuditControl>>(`/audit/frameworks/${frameworkId}/controls`, { params: controlsParams });
      const fetchedControls = controlsResponse.data.items || [];
      setControlsTotalItems(controlsResponse.data.total_items);
      setControlsTotalPages(controlsResponse.data.total_pages);

      const assessmentsResponse = await apiClient.get<PaginatedResponse<AuditAssessment> | AuditAssessment[]>(
        `/audit/organizations/${user.organization_id}/frameworks/${frameworkId}/assessments?page_size=100000`
      );
      let allAssessmentsForFramework: AuditAssessment[] = [];
      if (Array.isArray(assessmentsResponse.data)) {
        allAssessmentsForFramework = assessmentsResponse.data;
      } else if (assessmentsResponse.data && Array.isArray((assessmentsResponse.data as PaginatedResponse<AuditAssessment>).items)) {
        allAssessmentsForFramework = (assessmentsResponse.data as PaginatedResponse<AuditAssessment>).items;
      }

      let combined: ControlWithAssessment[] = fetchedControls.map(control => {
        const assessment = allAssessmentsForFramework.find(a => a.audit_control_id === control.id);
        return { ...control, assessment };
      });

      if (filterAssessmentStatus) {
        combined = combined.filter(item => {
          if (filterAssessmentStatus === "nao_avaliado") {
            return !item.assessment;
          }
          return item.assessment?.status === filterAssessmentStatus;
        });
      }
      setControlsWithAssessments(combined);
      if (fetchedControls.length === 0 && controlsResponse.data.total_items > 0 && controlsCurrentPage > 1) {
        setControlsCurrentPage(1);
      }
    } catch (err: any) {
      console.error(t('framework_detail.error_fetching_controls_assessments_console'), err);
      const fetchError = err.response?.data?.error || err.message || t('common:unknown_error');
      setError(prev => prev ? `${prev}; ${fetchError}` : fetchError);
      setControlsWithAssessments([]);
    } finally {
      setIsLoadingData(false);
    }
  }, [
    frameworkId,
    user?.organization_id,
    authIsLoading,
    controlsCurrentPage,
    controlsPageSize,
    filterFamily,
    filterAssessmentStatus,
    router.isReady,
    t
  ]);

  useEffect(() => {
    if (router.isReady && frameworkId && user && !authIsLoading) {
        fetchControlsAndCombineAssessments();
    }
  }, [fetchControlsAndCombineAssessments, router.isReady, frameworkId, user, authIsLoading]);

  const handleOpenAssessmentModal = (controlItem: ControlWithAssessment) => {
    setSelectedControlForAssessment(controlItem);
    setShowAssessmentModal(true);
  };

  const handleCloseAssessmentModal = () => {
    setSelectedControlForAssessment(null);
    setShowAssessmentModal(false);
  };

  const handleAssessmentSubmitSuccess = () => {
    fetchControlsAndCombineAssessments();
    handleCloseAssessmentModal();
  };

  const handleControlsPageChange = (newPage: number) => {
    if (newPage !== controlsCurrentPage) {
        setControlsCurrentPage(newPage);
    }
  };

  const handleFilterFamilyChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setFilterFamily(e.target.value);
    setControlsCurrentPage(1);
  };

  const handleFilterAssessmentStatusChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setFilterAssessmentStatus(e.target.value as AuditAssessmentStatusFilter);
  };

  const clearFilters = () => {
    const hadFilters = filterFamily !== "" || filterAssessmentStatus !== "";
    setFilterFamily("");
    setFilterAssessmentStatus("");
    if (controlsCurrentPage !== 1) {
        setControlsCurrentPage(1);
    } else if (hadFilters) {
        fetchControlsAndCombineAssessments(); // Forçar re-fetch se estava na pág 1 mas tinha filtros
    }
  };

  const pageTitle = t('framework_detail.page_title_prefix');
  const currentFrameworkName = frameworkInfo?.name || frameworkId as string;
  const appName = t('common:app_name');
  const dynamicPageTitle = `${pageTitle}: ${currentFrameworkName} - ${appName}`;

  if (authIsLoading || (!router.isReady && !frameworkInfo)) {
    return <AdminLayout title={t('common:loading_ellipsis')}><div className="p-6 text-center">{t('framework_detail.loading_framework_info')}</div></AdminLayout>;
  }

  if (error && (!frameworkInfo || frameworkInfo.name === t('framework_detail.unknown_framework_name') || controlsWithAssessments.length === 0) && !isLoadingData ) {
    return (
        <AdminLayout title={t('common:error_page_title_prefix', { entityName: currentFrameworkName })}>
            <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
                <h1 className="text-2xl font-bold text-red-600 dark:text-red-400 mb-4">{t('common:error_loading_data')}</h1>
                <p className="text-red-500 dark:text-red-300">{error}</p>
                <Link href="/admin/audit/frameworks" legacyBehavior>
                    <a className="mt-4 inline-flex items-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700">
                    {t('framework_detail.back_to_frameworks_link')}
                    </a>
                </Link>
            </div>
        </AdminLayout>
    );
  }

  return (
    <AdminLayout title={dynamicPageTitle}>
      <Head>
        <title>{dynamicPageTitle}</title>
      </Head>
      <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="sm:flex sm:items-center sm:justify-between mb-6">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-brand-primary dark:text-brand-primary">
              {frameworkInfo ? frameworkInfo.name : t('common:loading_ellipsis')}
            </h1>
            <p className="mt-2 text-sm text-gray-700 dark:text-gray-400">
              {t('framework_detail.header_description')}
            </p>
          </div>
          <div className="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
            <Link href="/admin/audit/frameworks" legacyBehavior>
                <a className="inline-flex items-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 transition-colors">
                &larr; {t('framework_detail.back_to_frameworks_link')}
                </a>
            </Link>
          </div>
        </div>

        <div className="my-6 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg shadow">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
            <div>
              <label htmlFor="filterFamily" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('framework_detail.filter_family_label')}</label>
              <select id="filterFamily" name="filterFamily" value={filterFamily} onChange={handleFilterFamilyChange}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm rounded-md disabled:opacity-50"
                      disabled={availableControlFamilies.length === 0 && !isLoadingData}
              >
                <option value="">{t('framework_detail.all_families_option')}</option>
                {availableControlFamilies.map(family => (
                  <option key={family} value={family}>{family}</option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor="filterAssessmentStatus" className="block text-sm font-medium text-gray-700 dark:text-gray-200">{t('framework_detail.filter_assessment_status_label')}</label>
              <select id="filterAssessmentStatus" name="filterAssessmentStatus" value={filterAssessmentStatus} onChange={handleFilterAssessmentStatusChange}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 focus:outline-none focus:ring-brand-primary focus:border-brand-primary sm:text-sm rounded-md">
                <option value="">{t('framework_detail.all_statuses_option')}</option>
                <option value="conforme">{t('framework_detail.status_option_compliant')}</option>
                <option value="nao_conforme">{t('framework_detail.status_option_non_compliant')}</option>
                <option value="parcialmente_conforme">{t('framework_detail.status_option_partially_compliant')}</option>
                <option value="nao_avaliado">{t('framework_detail.status_option_not_assessed')}</option>
              </select>
            </div>
            <div>
              <button onClick={clearFilters}
                      className="w-full inline-flex items-center justify-center rounded-md border border-gray-300 dark:border-gray-500 bg-white dark:bg-gray-600 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-100 shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2">
                {t('framework_detail.clear_filters_button')}
              </button>
            </div>
          </div>
        </div>

        {isLoadingData && <p className="text-center py-10">{t('framework_detail.loading_controls_assessments')}</p>}
        {error && !isLoadingData && controlsWithAssessments.length === 0 && <p className="text-center text-red-500 py-10">{t('framework_detail.error_loading_data', {message: error})}</p>}

        {!isLoadingData && !error && controlsWithAssessments.length === 0 && (
            <div className="text-center py-10">
                <p className="text-gray-500 dark:text-gray-400">{t('framework_detail.no_controls_found_filters')}</p>
            </div>
        )}

        {!isLoadingData && !error && controlsWithAssessments.length > 0 && (
          <div className="mt-8 flow-root">
            <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
              <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                  <table className="min-w-full divide-y divide-gray-300 dark:divide-gray-700">
                    <thead className="bg-gray-50 dark:bg-gray-700">
                      <tr>
                        <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 dark:text-white sm:pl-6">{t('framework_detail.table_header_control_id')}</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('framework_detail.table_header_description')}</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('framework_detail.table_header_family')}</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('framework_detail.table_header_assessment_status')}</th>
                        <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900 dark:text-white">{t('framework_detail.table_header_score')}</th>
                        <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6">
                          <span className="sr-only">{t('framework_detail.table_header_actions')}</span>
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-600 bg-white dark:bg-gray-800">
                      {controlsWithAssessments.map((item) => (
                        <tr key={item.id}>
                          <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 dark:text-white sm:pl-6">{item.control_id}</td>
                          <td className="px-3 py-4 text-sm text-gray-500 dark:text-gray-300 max-w-md truncate hover:whitespace-normal">{item.description}</td>
                          <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{item.family}</td>
                          <td className="whitespace-nowrap px-3 py-4 text-sm">
                            {item.assessment ? (
                               <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                  item.assessment.status === 'conforme' ? 'bg-green-100 text-green-800 dark:bg-green-700 dark:text-green-100' :
                                  item.assessment.status === 'parcialmente_conforme' ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-700 dark:text-yellow-100' :
                                  item.assessment.status === 'nao_conforme' ? 'bg-red-100 text-red-800 dark:bg-red-700 dark:text-red-100' :
                                  'bg-gray-100 text-gray-800 dark:bg-gray-600 dark:text-gray-200'
                              }`}>
                                  {t(`framework_detail.status_option_${item.assessment.status.replace('_', '')}`, {defaultValue: item.assessment.status})}
                              </span>
                            ) : (
                              <span className="text-xs text-gray-400 dark:text-gray-500">{t('framework_detail.status_not_assessed_display')}</span>
                            )}
                          </td>
                          <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500 dark:text-gray-300">{item.assessment?.score ?? '-'}</td>
                          <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6">
                            <button
                              onClick={() => handleOpenAssessmentModal(item)}
                              className="text-brand-primary hover:text-brand-primary/80 dark:text-brand-primary dark:hover:text-brand-primary/80 transition-colors"
                            >
                              {item.assessment ? t('framework_detail.action_edit_assessment') : t('framework_detail.action_assess')}
                            </button>
                             {item.assessment?.evidence_url && (
                              <a
                                  href={item.assessment.evidence_url}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="ml-3 text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 transition-colors"
                                  title={item.assessment.evidence_url}
                              >
                                  {t('framework_detail.action_view_evidence')}
                              </a>
                             )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {controlsTotalPages > 0 && (
                    <PaginationControls
                        currentPage={controlsCurrentPage}
                        totalPages={controlsTotalPages}
                        totalItems={controlsTotalItems}
                        pageSize={controlsPageSize}
                        onPageChange={handleControlsPageChange}
                        isLoading={isLoadingData}
                    />
                )}
              </div>
            </div>
          </div>
        )}

        {showAssessmentModal && selectedControlForAssessment && currentUser?.organization_id && (
          <AssessmentFormModal
            isOpen={showAssessmentModal}
            onClose={handleCloseAssessmentModal}
            organizationId={currentUser.organization_id}
            control={selectedControlForAssessment}
            onSubmitSuccess={handleAssessmentSubmitSuccess}
          />
        )}
      </div>
    </AdminLayout>
  );
};

export default WithAuth(FrameworkDetailPageContent);
