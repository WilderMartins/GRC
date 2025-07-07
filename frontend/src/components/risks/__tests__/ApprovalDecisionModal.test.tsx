import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ApprovalDecisionModal from '../ApprovalDecisionModal'; // Ajuste o path
import apiClient from '@/lib/axios';
import '@testing-library/jest-dom';

jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

describe('ApprovalDecisionModal', () => {
  const mockOnClose = jest.fn();
  const mockOnSubmitSuccess = jest.fn();
  const baseProps = {
    riskId: 'risk-uuid-123',
    riskTitle: 'Risco de Teste para Decisão',
    approvalId: 'approval-wf-uuid-456',
    currentApproverId: 'user-approver-uuid', // Não usado diretamente no componente, mas útil para contexto
    onClose: mockOnClose,
    onSubmitSuccess: mockOnSubmitSuccess,
  };

  beforeEach(() => {
    mockedApiClient.post.mockReset();
    mockOnClose.mockClear();
    mockOnSubmitSuccess.mockClear();
    window.alert = jest.fn(); // Mock se o componente usar alert
  });

  it('renders correctly with risk title and approval ID', () => {
    render(<ApprovalDecisionModal {...baseProps} />);
    expect(screen.getByText('Decidir sobre Aceite do Risco')).toBeInTheDocument();
    expect(screen.getByText(`Risco: ${baseProps.riskTitle}`)).toBeInTheDocument();
    expect(screen.getByText(`Workflow ID: ${baseProps.approvalId}`)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Aprovar Aceite/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Rejeitar Aceite/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/Comentários/i)).toBeInTheDocument();
  });

  it('requires a decision to be selected before submitting', async () => {
    render(<ApprovalDecisionModal {...baseProps} />);
    fireEvent.click(screen.getByRole('button', { name: /Registrar Decisão/i }));

    expect(await screen.findByText('Por favor, selecione uma decisão (Aprovar ou Rejeitar).')).toBeInTheDocument();
    expect(mockedApiClient.post).not.toHaveBeenCalled();
  });

  it('submits "aprovado" decision correctly', async () => {
    mockedApiClient.post.mockResolvedValue({ data: { status: 'aprovado' } });
    render(<ApprovalDecisionModal {...baseProps} />);

    fireEvent.click(screen.getByRole('button', { name: /Aprovar Aceite/i }));
    fireEvent.change(screen.getByLabelText(/Comentários/i), { target: { value: 'Tudo certo.' } });
    fireEvent.click(screen.getByRole('button', { name: /Registrar Decisão/i }));

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledWith(
        `/risks/${baseProps.riskId}/approval/${baseProps.approvalId}/decide`,
        { decision: 'aprovado', comments: 'Tudo certo.' }
      );
    });
    expect(mockOnSubmitSuccess).toHaveBeenCalledTimes(1);
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('submits "rejeitado" decision correctly', async () => {
    mockedApiClient.post.mockResolvedValue({ data: { status: 'rejeitado' } });
    render(<ApprovalDecisionModal {...baseProps} />);

    fireEvent.click(screen.getByRole('button', { name: /Rejeitar Aceite/i }));
    fireEvent.change(screen.getByLabelText(/Comentários/i), { target: { value: 'Precisa de mais informações.' } });
    fireEvent.click(screen.getByRole('button', { name: /Registrar Decisão/i }));

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledWith(
        `/risks/${baseProps.riskId}/approval/${baseProps.approvalId}/decide`,
        { decision: 'rejeitado', comments: 'Precisa de mais informações.' }
      );
    });
    expect(mockOnSubmitSuccess).toHaveBeenCalledTimes(1);
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when Cancel button is clicked', () => {
    render(<ApprovalDecisionModal {...baseProps} />);
    fireEvent.click(screen.getByRole('button', { name: /Cancelar/i }));
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('displays API error message on submission failure', async () => {
    const errorMessage = "Erro da API ao decidir";
    mockedApiClient.post.mockRejectedValue({ response: { data: { error: errorMessage } } });
    render(<ApprovalDecisionModal {...baseProps} />);

    fireEvent.click(screen.getByRole('button', { name: /Aprovar Aceite/i }));
    fireEvent.click(screen.getByRole('button', { name: /Registrar Decisão/i }));

    expect(await screen.findByText(`Falha ao registrar decisão: ${errorMessage}`)).toBeInTheDocument();
    expect(mockOnSubmitSuccess).not.toHaveBeenCalled();
    expect(mockOnClose).not.toHaveBeenCalled();
  });
});
