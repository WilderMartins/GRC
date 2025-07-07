import React, { useState, useEffect } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path
import { useAuth } from '@/contexts/AuthContext'; // Para organization_id

type AuditControlStatus = "conforme" | "nao_conforme" | "parcialmente_conforme" | "";

interface AssessmentFormData {
  audit_control_id: string; // UUID do AuditControl - virá das props
  status: AuditControlStatus;
  score?: number | string; // string para input, number para API
  assessment_date: string; // Formato YYYY-MM-DD
  evidence_url: string; // Para links externos
}

interface AssessmentFormProps {
  controlId: string; // UUID do AuditControl que está sendo avaliado
  controlDisplayId?: string; // ID textual do controle (ex: AC-1) para exibição
  initialData?: Partial<AssessmentFormData> & { id?: string }; // Para edição
  onClose: () => void; // Para fechar o modal/formulário
  onSubmitSuccess: (assessmentData: any) => void; // Callback com os dados da avaliação salva
}

const AssessmentForm: React.FC<AssessmentFormProps> = ({
  controlId,
  controlDisplayId,
  initialData,
  onClose,
  onSubmitSuccess,
}) => {
  const { user } = useAuth();
  const [formData, setFormData] = useState<AssessmentFormData>({
    audit_control_id: controlId,
    status: initialData?.status || "",
    score: initialData?.score ?? '', // Default para string vazia para o input
    assessment_date: initialData?.assessment_date || new Date().toISOString().split('T')[0], // YYYY-MM-DD
    evidence_url: initialData?.evidence_url || '',
  });
  const [evidenceFile, setEvidenceFile] = useState<File | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Se initialData mudar (ex: usuário seleciona outro controle para avaliar no mesmo modal)
    setFormData({
        audit_control_id: controlId, // Sempre usa o controlId atual passado como prop
        status: initialData?.status || "",
        score: initialData?.score ?? '',
        assessment_date: initialData?.assessment_date || new Date().toISOString().split('T')[0],
        evidence_url: initialData?.evidence_url || '',
    });
    setEvidenceFile(null); // Resetar arquivo se os dados iniciais mudarem
  }, [initialData, controlId]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
     if (type === 'number' && name === 'score') {
      setFormData(prev => ({ ...prev, [name]: value === '' ? '' : parseInt(value, 10) }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
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
    setError(null);

    if (!formData.status) {
      setError("O campo Status é obrigatório.");
      setIsLoading(false);
      return;
    }
    if (formData.score !== '' && (Number(formData.score) < 0 || Number(formData.score) > 100)) {
        setError("Score deve ser entre 0 e 100, ou deixado em branco.");
        setIsLoading(false);
        return;
    }


    const submissionData = new FormData();
    const assessmentPayload: any = {
        audit_control_id: formData.audit_control_id,
        status: formData.status,
        assessment_date: formData.assessment_date,
    };
    if (formData.score !== '') { // Enviar score apenas se preenchido
        assessmentPayload.score = Number(formData.score);
    }
    if (formData.evidence_url) { // Enviar URL textual apenas se preenchida E nenhum arquivo for enviado
        if (!evidenceFile) {
            assessmentPayload.evidence_url = formData.evidence_url;
        }
    }

    submissionData.append('data', JSON.stringify(assessmentPayload));

    if (evidenceFile) {
      submissionData.append('evidence_file', evidenceFile);
    }

    try {
      // O endpoint /api/v1/audit/assessments faz upsert
      const response = await apiClient.post('/audit/assessments', submissionData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      onSubmitSuccess(response.data); // Passa os dados da avaliação salva para o callback
      onClose(); // Fecha o modal/formulário
    } catch (err: any) {
      console.error("Erro ao salvar avaliação:", err);
      setError(err.response?.data?.error || err.message || "Falha ao salvar avaliação.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <h3 className="text-lg font-medium leading-6 text-gray-900 dark:text-white">
        Avaliar Controle: {controlDisplayId || controlId.substring(0,8) + "..."}
      </h3>
      {error && <p className="text-sm text-red-600 bg-red-100 dark:bg-red-900 dark:text-red-200 p-3 rounded-md">{error}</p>}

      <div>
        <label htmlFor="status" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Status da Avaliação</label>
        <select name="status" id="status" value={formData.status} onChange={handleChange} required
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2">
          <option value="" disabled>Selecione um Status</option>
          <option value="conforme">Conforme</option>
          <option value="nao_conforme">Não Conforme</option>
          <option value="parcialmente_conforme">Parcialmente Conforme</option>
        </select>
      </div>

      <div>
        <label htmlFor="score" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Score (0-100, Opcional)</label>
        <input type="number" name="score" id="score" value={formData.score} onChange={handleChange} min="0" max="100"
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Se não informado, um score padrão pode ser aplicado com base no status.
        </p>
      </div>

      <div>
        <label htmlFor="assessment_date" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Data da Avaliação</label>
        <input type="date" name="assessment_date" id="assessment_date" value={formData.assessment_date} onChange={handleChange} required
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
      </div>

      <div>
        <label htmlFor="evidence_file" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Arquivo de Evidência (Opcional)</label>
        <input type="file" name="evidence_file" id="evidence_file" onChange={handleFileChange}
               className="mt-1 block w-full text-sm text-gray-900 border border-gray-300 rounded-lg cursor-pointer bg-gray-50 dark:text-gray-400 focus:outline-none dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400"/>
        {evidenceFile && <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">Selecionado: {evidenceFile.name}</p>}
      </div>

      <div>
        <label htmlFor="evidence_url" className="block text-sm font-medium text-gray-700 dark:text-gray-300">OU Link Externo para Evidência (Opcional)</label>
        <input type="url" name="evidence_url" id="evidence_url" value={formData.evidence_url} onChange={handleChange} placeholder="https://example.com/evidence"
               className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white p-2"/>
         <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Se um arquivo for selecionado acima, este link será ignorado.
        </p>
      </div>


      <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700 mt-6">
        <button type="button" onClick={onClose}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white dark:bg-gray-600 dark:text-gray-200 border border-gray-300 dark:border-gray-500 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-500">
          Cancelar
        </button>
        <button type="submit" disabled={isLoading}
                className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md shadow-sm disabled:opacity-50 flex items-center">
          {isLoading && (
            <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          )}
          Salvar Avaliação
        </button>
      </div>
    </form>
  );
};

export default AssessmentForm;
