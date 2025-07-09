import React, { useState } from 'react';
import apiClient from '@/lib/axios';
import { useNotifier } from '@/hooks/useNotifier'; // Importar para notificações
import { useTranslation } from 'next-i18next'; // Importar para i18n
import { ApprovalDecision } from '@/types'; // Importar o tipo de decisão

interface ApprovalDecisionModalProps {
  riskId: string;
  riskTitle: string;
  approvalId: string;
  currentApproverId: string;
  onClose: () => void;
  onSubmitSuccess: () => void;
}

const ApprovalDecisionModal: React.FC<ApprovalDecisionModalProps> = ({
  riskId,
  riskTitle,
  approvalId,
  onClose,
  onSubmitSuccess,
}) => {
  const { t } = useTranslation(['risks', 'common']);
  const notify = useNotifier();
  const [decision, setDecision] = useState<ApprovalDecision | ''>(''); // Usar o tipo importado
  const [comments, setComments] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null); // Renomeado de 'error'

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!decision) {
      setFormError(t('approval_modal.error_select_decision'));
      return;
    }
    setIsLoading(true);
    setFormError(null);

    try {
      await apiClient.post(`/risks/${riskId}/approval/${approvalId}/decide`, {
        decision,
        comments,
      });
      notify.success(t('approval_modal.success_message'));
      onSubmitSuccess();
      onClose();
    } catch (err: any) {
      console.error("Erro ao registrar decisão:", err);
      const apiError = err.response?.data?.error || t('common:unknown_error');
      // setFormError(apiError); // Opcional: mostrar erro no modal além do toast
      notify.error(t('approval_modal.error_message', { message: apiError }));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm">
      <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-xl w-full max-w-lg">
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
          {t('approval_modal.title')}
        </h2>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-1">
          {t('approval_modal.risk_label')} <span className="font-semibold">{riskTitle}</span>
        </p>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
          {t('approval_modal.workflow_id_label')} <span className="font-mono text-xs">{approvalId}</span>
        </p>

        {formError && <p className="text-sm text-red-600 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md mb-4">{formError}</p>}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">{t('approval_modal.decision_label')}</label>
            <div className="mt-1 flex space-x-4">
              <button type="button" onClick={() => setDecision('aprovado')}
                      className={`px-4 py-2 rounded-md text-sm font-medium w-full
                                  ${decision === 'aprovado' ? 'bg-green-600 text-white ring-2 ring-green-500 ring-offset-2 dark:ring-offset-gray-800'
                                                            : 'bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 hover:bg-gray-300 dark:hover:bg-gray-600'}`}>
                {t('approval_modal.approve_button')}
              </button>
              <button type="button" onClick={() => setDecision('rejeitado')}
                      className={`px-4 py-2 rounded-md text-sm font-medium w-full
                                  ${decision === 'rejeitado' ? 'bg-red-600 text-white ring-2 ring-red-500 ring-offset-2 dark:ring-offset-gray-800'
                                                             : 'bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 hover:bg-gray-300 dark:hover:bg-gray-600'}`}>
                {t('approval_modal.reject_button')}
              </button>
            </div>
          </div>

          <div>
            <label htmlFor="comments" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('approval_modal.comments_label')}
            </label>
            <textarea name="comments" id="comments" value={comments} onChange={(e) => setComments(e.target.value)} rows={3}
                      className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
          </div>

          <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700 mt-6">
            <button type="button" onClick={onClose} disabled={isLoading}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500 disabled:opacity-50">
              {t('common:cancel_button')}
            </button>
            <button type="submit" disabled={isLoading || !decision}
                    className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md shadow-sm disabled:opacity-50 flex items-center">
              {isLoading && (
                <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              )}
              {t('approval_modal.submit_button')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default ApprovalDecisionModal;
