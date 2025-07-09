import React, { useState, useEffect } from 'react';
import apiClient from '@/lib/axios';
import { useAuth } from '@/contexts/AuthContext';
import { AuditAssessmentStatus } from '@/types';
import { useTranslation } from 'next-i18next'; // Importar useTranslation
import { useNotifier } from '@/hooks/useNotifier'; // Para notificações de erro da API

interface AssessmentFormData {
  audit_control_id: string;
  status: AuditAssessmentStatus;
  score?: number | string;
  assessment_date: string;
  evidence_url: string;
}

interface AssessmentFormProps {
  controlId: string;
  controlDisplayId?: string;
  initialData?: Partial<AssessmentFormData> & { id?: string };
  onClose: () => void;
  onSubmitSuccess: (assessmentData: any) => void;
}

const AssessmentForm: React.FC<AssessmentFormProps> = ({
  controlId,
  controlDisplayId,
  initialData,
  onClose,
  onSubmitSuccess,
}) => {
  const { t } = useTranslation(['audit', 'common']); // Adicionar hook
  const { user } = useAuth(); // user pode ser usado para alguma lógica futura, mas não diretamente aqui
  const notify = useNotifier(); // Para erros da API

  const [formData, setFormData] = useState<AssessmentFormData>({
    audit_control_id: controlId,
    status: (initialData?.status as AuditAssessmentStatus) || "", // Cast e fallback
    score: initialData?.score ?? '',
    assessment_date: initialData?.assessment_date || new Date().toISOString().split('T')[0],
    evidence_url: initialData?.evidence_url || '',
  });
  const [evidenceFile, setEvidenceFile] = useState<File | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null); // Renomeado de 'error'

  useEffect(() => {
    setFormData({
        audit_control_id: controlId,
        status: (initialData?.status as AuditAssessmentStatus) || "",
        score: initialData?.score ?? '',
        assessment_date: initialData?.assessment_date || new Date().toISOString().split('T')[0],
        evidence_url: initialData?.evidence_url || '',
    });
    setEvidenceFile(null);
  }, [initialData, controlId]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
     if (type === 'number' && name === 'score') {
      setFormData(prev => ({ ...prev, [name]: value === '' ? '' : parseInt(value, 10) }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value as any })); // as any para status
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setEvidenceFile(e.target.files[0]);
    } else {
      setEvidenceFile(null);
    }
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsLoading(true);
    setFormError(null);

    if (!formData.status) {
      setFormError(t('assessment_form.error_status_required'));
      setIsLoading(false);
      return;
    }
    if (formData.score !== '' && (Number(formData.score) < 0 || Number(formData.score) > 100)) {
        setFormError(t('assessment_form.error_score_invalid'));
        setIsLoading(false);
        return;
    }

    const submissionData = new FormData();
    const assessmentPayload: any = {
        audit_control_id: formData.audit_control_id,
        status: formData.status,
        assessment_date: formData.assessment_date,
    };
    if (formData.score !== '') {
        assessmentPayload.score = Number(formData.score);
    }
    if (formData.evidence_url) {
        if (!evidenceFile) {
            assessmentPayload.evidence_url = formData.evidence_url;
        }
    }

    submissionData.append('data', JSON.stringify(assessmentPayload));

    if (evidenceFile) {
      submissionData.append('evidence_file', evidenceFile);
    }

    try {
      const response = await apiClient.post('/audit/assessments', submissionData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      // Sucesso é tratado pela página pai via onSubmitSuccess, que pode chamar notify.success
      onSubmitSuccess(response.data);
      onClose();
    } catch (err: any) {
      console.error("Erro ao salvar avaliação:", err);
      const apiError = err.response?.data?.error || t('common:unknown_error');
      // setFormError(apiError); // Opcional: mostrar erro no formulário
      notify.error(t('common:error_saving_data', { entity: t('common:assessment_singular'), message: apiError }));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-white">
        {t('assessment_form.modal_title_prefix')} {controlDisplayId || controlId.substring(0,8) + "..."}
      </h3>
      {formError && <p className="text-sm text-red-600 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md">{formError}</p>}

      <div>
        <label htmlFor="status" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('assessment_form.field_status_label')}</label>
        <select name="status" id="status" value={formData.status} onChange={handleChange} required
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
          <option value="" disabled>{t('assessment_form.option_select_status')}</option>
          <option value="conforme">{t('assessment_form.option_status_compliant')}</option>
          <option value="nao_conforme">{t('assessment_form.option_status_non_compliant')}</option>
          <option value="parcialmente_conforme">{t('assessment_form.option_status_partially_compliant')}</option>
        </select>
      </div>

      <div>
        <label htmlFor="score" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('assessment_form.field_score_label')}</label>
        <input type="number" name="score" id="score" value={formData.score} onChange={handleChange} min="0" max="100"
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {t('assessment_form.score_help_text')}
        </p>
      </div>

      <div>
        <label htmlFor="assessment_date" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('assessment_form.field_assessment_date_label')}</label>
        <input type="date" name="assessment_date" id="assessment_date" value={formData.assessment_date} onChange={handleChange} required
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>

      <div>
        <label htmlFor="evidence_file" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('assessment_form.field_evidence_file_label')}</label>
        <input type="file" name="evidence_file" id="evidence_file" onChange={handleFileChange}
               className="mt-1 block w-full text-sm text-gray-900 border border-gray-300 rounded-lg cursor-pointer bg-gray-50 dark:text-gray-400 focus:outline-none dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400"/>
        {evidenceFile && <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{t('assessment_form.selected_file_label', {fileName: evidenceFile.name})}</p>}
      </div>

      <div>
        <label htmlFor="evidence_url" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('assessment_form.field_evidence_url_label')}</label>
        <input type="url" name="evidence_url" id="evidence_url" value={formData.evidence_url} onChange={handleChange}
               placeholder={t('assessment_form.evidence_url_placeholder')}
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
         <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {t('assessment_form.evidence_url_help_text')}
        </p>
      </div>

      <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700 mt-6">
        <button type="button" onClick={onClose}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500">
          {t('common:cancel_button')}
        </button>
        <button type="submit" disabled={isLoading}
                className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md shadow-sm disabled:opacity-50 flex items-center">
          {isLoading && (
            <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          )}
          {t('assessment_form.save_button')}
        </button>
      </div>
    </form>
  );
};

export default AssessmentForm;
