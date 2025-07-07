import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import AssessmentForm from '../AssessmentForm'; // Ajuste o path
import apiClient from '@/lib/axios';
import { AuthContext } from '@/contexts/AuthContext';
import '@testing-library/jest-dom';

jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

jest.mock('next/router', () => ({
  useRouter: jest.fn(() => ({ push: jest.fn() })),
}));

const mockUser = { id: 'user1', name: 'Test User', email: 'test@example.com', role: 'admin', organization_id: 'org1' };
const mockAuthContextValue = {
  isAuthenticated: true, user: mockUser, token: 'fake-token', isLoading: false, login: jest.fn(), logout: jest.fn(),
};

const mockControlId = 'control-uuid-123';
const mockControlDisplayId = 'AC-1';

describe('AssessmentForm', () => {
  const mockOnClose = jest.fn();
  const mockOnSubmitSuccess = jest.fn();

  beforeEach(() => {
    mockedApiClient.post.mockReset();
    mockOnClose.mockClear();
    mockOnSubmitSuccess.mockClear();
    window.alert = jest.fn();
  });

  const renderAssessmentForm = (props?: any) => {
    const defaultProps = {
      controlId: mockControlId,
      controlDisplayId: mockControlDisplayId,
      onClose: mockOnClose,
      onSubmitSuccess: mockOnSubmitSuccess,
    };
    return render(
      <AuthContext.Provider value={mockAuthContextValue}>
        <AssessmentForm {...defaultProps} {...props} />
      </AuthContext.Provider>
    );
  };

  it('renders all form fields correctly', () => {
    renderAssessmentForm();
    expect(screen.getByLabelText(/Status da Avaliação/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Score/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Data da Avaliação/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Arquivo de Evidência/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Link Externo para Evidência/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Salvar Avaliação/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Cancelar/i })).toBeInTheDocument();
  });

  it('submits form data correctly without file upload', async () => {
    mockedApiClient.post.mockResolvedValue({ data: { id: 'new-assessment-id' } });
    renderAssessmentForm();

    fireEvent.change(screen.getByLabelText(/Status da Avaliação/i), { target: { value: 'conforme' } });
    fireEvent.change(screen.getByLabelText(/Score/i), { target: { value: '100' } });
    fireEvent.change(screen.getByLabelText(/Data da Avaliação/i), { target: { value: '2023-10-27' } });
    fireEvent.change(screen.getByLabelText(/Link Externo para Evidência/i), { target: { value: 'http://example.com/evidence.pdf' } });

    fireEvent.click(screen.getByRole('button', { name: /Salvar Avaliação/i }));

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledTimes(1);
      const formDataSent = mockedApiClient.post.mock.calls[0][1] as FormData;
      const jsonData = formDataSent.get('data') as string;
      const parsedData = JSON.parse(jsonData);

      expect(parsedData.audit_control_id).toBe(mockControlId);
      expect(parsedData.status).toBe('conforme');
      expect(parsedData.score).toBe(100);
      expect(parsedData.assessment_date).toBe('2023-10-27');
      expect(parsedData.evidence_url).toBe('http://example.com/evidence.pdf');
      expect(formDataSent.get('evidence_file')).toBeNull();
    });
    expect(mockOnSubmitSuccess).toHaveBeenCalledWith({ id: 'new-assessment-id' });
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('submits form data correctly with file upload', async () => {
    const mockFile = new File(['dummy content'], 'evidence.pdf', { type: 'application/pdf' });
    const mockApiResponse = { data: { id: 'assessment-with-file', evidence_url: 'http://gcs.com/evidence.pdf' }};
    mockedApiClient.post.mockResolvedValue(mockApiResponse);

    renderAssessmentForm();

    fireEvent.change(screen.getByLabelText(/Status da Avaliação/i), { target: { value: 'parcialmente_conforme' } });
    const fileInput = screen.getByLabelText(/Arquivo de Evidência/i);
    fireEvent.change(fileInput, { target: { files: [mockFile] } });

    fireEvent.click(screen.getByRole('button', { name: /Salvar Avaliação/i }));

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledTimes(1);
      const formDataSent = mockedApiClient.post.mock.calls[0][1] as FormData;
      const jsonData = formDataSent.get('data') as string;
      const parsedData = JSON.parse(jsonData);
      expect(parsedData.status).toBe('parcialmente_conforme');
      expect(formDataSent.get('evidence_file')).toEqual(mockFile);
      // evidence_url no JSON não deve ser enviada se um arquivo foi anexado
      expect(parsedData.evidence_url).toBeUndefined();
    });
    expect(mockOnSubmitSuccess).toHaveBeenCalledWith(mockApiResponse.data);
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('shows error if status is not selected', async () => {
    renderAssessmentForm();
    fireEvent.click(screen.getByRole('button', { name: /Salvar Avaliação/i }));
    expect(await screen.findByText(/O campo Status é obrigatório./i)).toBeInTheDocument();
    expect(mockedApiClient.post).not.toHaveBeenCalled();
  });

  it('pre-fills form with initialData for editing', () => {
    const initialTestData = {
      id: 'assess-edit-id',
      audit_control_id: mockControlId,
      status: 'nao_conforme' as any,
      score: 20,
      assessment_date: '2023-01-15',
      evidence_url: 'http://existing.com/evidence.doc',
    };
    renderAssessmentForm({ initialData: initialTestData });

    expect((screen.getByLabelText(/Status da Avaliação/i) as HTMLSelectElement).value).toBe('nao_conforme');
    expect((screen.getByLabelText(/Score/i) as HTMLInputElement).value).toBe('20');
    expect((screen.getByLabelText(/Data da Avaliação/i) as HTMLInputElement).value).toBe('2023-01-15');
    expect((screen.getByLabelText(/Link Externo para Evidência/i) as HTMLInputElement).value).toBe('http://existing.com/evidence.doc');
  });

});
