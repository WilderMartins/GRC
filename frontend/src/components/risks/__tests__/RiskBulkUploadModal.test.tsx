import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import RiskBulkUploadModal from '../RiskBulkUploadModal'; // Ajuste o path
import apiClient from '@/lib/axios';
import '@testing-library/jest-dom';

// Mock apiClient (axios)
jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

describe('RiskBulkUploadModal', () => {
  const mockOnClose = jest.fn();
  const mockOnUploadSuccess = jest.fn();

  const baseProps = {
    isOpen: true,
    onClose: mockOnClose,
    onUploadSuccess: mockOnUploadSuccess,
  };

  beforeEach(() => {
    mockedApiClient.post.mockReset();
    mockOnClose.mockClear();
    mockOnUploadSuccess.mockClear();
    // window.alert = jest.fn(); // Se o componente usar alert para feedback direto
  });

  it('renders correctly when open', () => {
    render(<RiskBulkUploadModal {...baseProps} />);
    expect(screen.getByText('Importar Riscos via CSV')).toBeInTheDocument();
    expect(screen.getByLabelText(/Selecione o arquivo CSV/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Enviar Arquivo/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Cancelar/i })).toBeInTheDocument();
  });

  it('does not render when isOpen is false', () => {
    render(<RiskBulkUploadModal {...baseProps} isOpen={false} />);
    expect(screen.queryByText('Importar Riscos via CSV')).not.toBeInTheDocument();
  });

  it('calls onClose when Cancel button is clicked', () => {
    render(<RiskBulkUploadModal {...baseProps} />);
    fireEvent.click(screen.getByRole('button', { name: /Cancelar/i }));
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('enables submit button only when a CSV file is selected', () => {
    render(<RiskBulkUploadModal {...baseProps} />);
    const submitButton = screen.getByRole('button', { name: /Enviar Arquivo/i });
    const fileInput = screen.getByLabelText(/Selecione o arquivo CSV/i);

    expect(submitButton).toBeDisabled();

    const testFile = new File(['col1,col2\nval1,val2'], 'test.csv', { type: 'text/csv' });
    fireEvent.change(fileInput, { target: { files: [testFile] } });

    expect(submitButton).not.toBeDisabled();
    expect(screen.getByText(`Arquivo selecionado: ${testFile.name}`)).toBeInTheDocument();
  });

  it('shows error if non-CSV file is selected', () => {
    render(<RiskBulkUploadModal {...baseProps} />);
    const fileInput = screen.getByLabelText(/Selecione o arquivo CSV/i);
    const testFile = new File(['content'], 'test.txt', { type: 'text/plain' });
    fireEvent.change(fileInput, { target: { files: [testFile] } });

    expect(screen.getByText(/Formato de arquivo inválido/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Enviar Arquivo/i })).toBeDisabled();
  });


  it('submits the file and calls onUploadSuccess on successful API response', async () => {
    const mockResponseData = { successfully_imported: 5, failed_rows: [] };
    mockedApiClient.post.mockResolvedValue({ data: mockResponseData });

    render(<RiskBulkUploadModal {...baseProps} />);

    const fileInput = screen.getByLabelText(/Selecione o arquivo CSV/i);
    const testFile = new File(['col1,col2\nval1,val2'], 'test.csv', { type: 'text/csv' });
    fireEvent.change(fileInput, { target: { files: [testFile] } });

    fireEvent.click(screen.getByRole('button', { name: /Enviar Arquivo/i }));

    expect(screen.getByText(/Enviando.../i)).toBeInTheDocument();

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledTimes(1);
      expect(mockedApiClient.post).toHaveBeenCalledWith(
        '/risks/bulk-upload-csv',
        expect.any(FormData), // Verifica se é um FormData
        expect.objectContaining({ headers: { 'Content-Type': 'multipart/form-data' } })
      );
    });

    expect(await screen.findByText(`Riscos importados com sucesso: ${mockResponseData.successfully_imported}`)).toBeInTheDocument();
    expect(mockOnUploadSuccess).toHaveBeenCalledTimes(1);

    // Input deve ser resetado
    // expect((fileInput as HTMLInputElement).value).toBe(''); // Isso pode ser difícil de testar diretamente com RTL
  });

  it('displays API error message and failed rows on API error response', async () => {
    const mockErrorResponse = {
      successfully_imported: 0,
      failed_rows: [{ line_number: 2, errors: ['title is required'] }],
      general_error: 'Some general error during processing'
    };
    mockedApiClient.post.mockRejectedValue({ response: { data: mockErrorResponse } });

    render(<RiskBulkUploadModal {...baseProps} />);
    const fileInput = screen.getByLabelText(/Selecione o arquivo CSV/i);
    const testFile = new File(['col1,col2\nval1,val2'], 'test.csv', { type: 'text/csv' });
    fireEvent.change(fileInput, { target: { files: [testFile] } });
    fireEvent.click(screen.getByRole('button', { name: /Enviar Arquivo/i }));

    expect(await screen.findByText(`Erro Geral: ${mockErrorResponse.general_error}`)).toBeInTheDocument();
    expect(screen.getByText(/Linha 2: title is required/i)).toBeInTheDocument();
    expect(mockOnUploadSuccess).not.toHaveBeenCalled(); // Não deve chamar onUploadSuccess se houve erro geral ou só falhas
  });

  it('displays generic error if API response is not structured', async () => {
    const genericErrorMessage = "Network Error";
    mockedApiClient.post.mockRejectedValue(new Error(genericErrorMessage));

    render(<RiskBulkUploadModal {...baseProps} />);
    const fileInput = screen.getByLabelText(/Selecione o arquivo CSV/i);
    const testFile = new File(['col1,col2\nval1,val2'], 'test.csv', { type: 'text/csv' });
    fireEvent.change(fileInput, { target: { files: [testFile] } });
    fireEvent.click(screen.getByRole('button', { name: /Enviar Arquivo/i }));

    expect(await screen.findByText(`Falha ao enviar arquivo CSV.`)).toBeInTheDocument();
  });

});
