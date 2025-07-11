import React from 'react';
import AssessmentForm from './AssessmentForm'; // Supondo que AssessmentForm está na mesma pasta ou ajustar path
import { AuditControl, AuditAssessment, ControlWithAssessment } from '@/types';
import { useTranslation } from 'next-i18next';

interface AssessmentFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  organizationId: string;
  control: ControlWithAssessment; // Passar o controle completo, que pode ter a avaliação existente
  // initialData (assessment) é derivado de control.assessment
  onSubmitSuccess: () => void;
}

const AssessmentFormModal: React.FC<AssessmentFormModalProps> = ({
  isOpen,
  onClose,
  organizationId,
  control,
  onSubmitSuccess,
}) => {
  const { t } = useTranslation(['audit', 'common']);

  if (!isOpen) {
    return null;
  }

  const initialAssessmentData = control.assessment ? {
    audit_control_id: control.assessment.audit_control_id,
    status: control.assessment.status as any, // Cast se o tipo do form for mais estrito
    score: control.assessment.score,
    assessment_date: control.assessment.assessment_date
                       ? control.assessment.assessment_date.split('T')[0]
                       : new Date().toISOString().split('T')[0],
    evidence_url: control.assessment.evidence_url || '',
    // evidence_file não é passado como initialData, é apenas para novo upload
  } : undefined;


  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-75 backdrop-blur-sm transition-opacity duration-300 ease-in-out">
      <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-xl max-h-[90vh] overflow-y-auto">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
            {control.assessment
              ? t('assessment_form_modal.edit_assessment_title')
              : t('assessment_form_modal.add_assessment_title')}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            aria-label={t('common:close_button')}
          >
            <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-1">
          {t('assessment_form_modal.control_id_label')}: <span className="font-semibold">{control.control_id}</span>
        </p>
        <p className="text-sm text-gray-500 dark:text-gray-400 mb-4 truncate" title={control.description}>
          {t('assessment_form_modal.control_description_label')}: {control.description}
        </p>

        <AssessmentForm
          organizationId={organizationId} // Passar organizationId
          controlId={control.id}
          controlDisplayId={control.control_id} // Passar controlDisplayId
          initialData={initialAssessmentData}
          onClose={onClose} // Passar onClose para o form poder fechar o modal
          onSubmitSuccess={() => {
            onSubmitSuccess(); // Chamar o callback da página
            // onClose(); // O form já pode chamar onClose, ou podemos chamar aqui.
          }}
        />
      </div>
    </div>
  );
};

export default AssessmentFormModal;
