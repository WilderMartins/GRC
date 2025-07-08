import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import WebhookForm from '../WebhookForm'; // Ajuste o path
import apiClient from '@/lib/axios';
import { AuthContext } from '@/contexts/AuthContext';
import '@testing-library/jest-dom';

jest.mock('@/lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

jest.mock('next/router', () => ({
  useRouter: jest.fn(() => ({ push: jest.fn() })),
}));

const mockUser = { id: 'user1', name: 'Admin User', email: 'admin@example.com', role: 'admin', organization_id: 'org123' };
const mockAuthContextValue = {
  isAuthenticated: true, user: mockUser, token: 'fake-token', isLoading: false, login: jest.fn(), logout: jest.fn(),
};

describe('WebhookForm', () => {
  const mockOnClose = jest.fn();
  const mockOnSubmitSuccess = jest.fn();
  const organizationId = mockUser.organization_id;

  const baseProps = {
    organizationId,
    onClose: mockOnClose,
    onSubmitSuccess: mockOnSubmitSuccess,
  };

  beforeEach(() => {
    mockedApiClient.post.mockReset();
    mockedApiClient.put.mockReset();
    mockOnClose.mockClear();
    mockOnSubmitSuccess.mockClear();
  });

  const renderForm = (props?: any) => {
    return render(
      <AuthContext.Provider value={mockAuthContextValue}>
        <WebhookForm {...baseProps} {...props} />
      </AuthContext.Provider>
    );
  };

  it('renders correctly for a new webhook', () => {
    renderForm();
    expect(screen.getByRole('heading', { name: /Adicionar Novo Webhook/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/Nome do Webhook/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/URL do Webhook/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Risco Criado/i)).toBeInTheDocument(); // Checkbox de tipo de evento
    expect(screen.getByLabelText(/Status do Risco Alterado/i)).toBeInTheDocument(); // Checkbox
    expect(screen.getByLabelText(/Ativo/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Adicionar Webhook/i })).toBeInTheDocument();
  });

  it('submits new webhook data correctly', async () => {
    mockedApiClient.post.mockResolvedValue({ data: { id: 'new-webhook-id' } });
    renderForm();

    fireEvent.change(screen.getByLabelText(/Nome do Webhook/i), { target: { value: 'Test Webhook' } });
    fireEvent.change(screen.getByLabelText(/URL do Webhook/i), { target: { value: 'https://example.com/webhook' } });
    fireEvent.click(screen.getByLabelText(/Risco Criado/i)); // Seleciona um evento

    fireEvent.click(screen.getByRole('button', { name: /Adicionar Webhook/i }));

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledWith(
        `/organizations/${organizationId}/webhooks`,
        expect.objectContaining({
          name: 'Test Webhook',
          url: 'https://example.com/webhook',
          event_types: ['risk_created'],
          is_active: true,
        })
      );
    });
    expect(mockOnSubmitSuccess).toHaveBeenCalledWith({ id: 'new-webhook-id' });
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('pre-fills form for editing and submits updates', async () => {
    const initialWebhookData = {
      id: 'wh-edit-123',
      name: 'Existing Webhook',
      url: 'https://existing.com/hook',
      event_types: 'risk_created,risk_status_changed', // String como vem da lista
      is_active: true,
    };
    mockedApiClient.put.mockResolvedValue({ data: { ...initialWebhookData, name: 'Updated Webhook' } });

    renderForm({ initialData: initialWebhookData, isEditing: true });

    expect((screen.getByLabelText(/Nome do Webhook/i) as HTMLInputElement).value).toBe('Existing Webhook');
    expect((screen.getByLabelText(/URL do Webhook/i) as HTMLInputElement).value).toBe('https://existing.com/hook');
    expect(screen.getByLabelText(/Risco Criado/i)).toBeChecked();
    expect(screen.getByLabelText(/Status do Risco Alterado/i)).toBeChecked();

    fireEvent.change(screen.getByLabelText(/Nome do Webhook/i), { target: { value: 'Updated Webhook' } });
    fireEvent.click(screen.getByLabelText(/Status do Risco Alterado/i)); // Desseleciona um evento

    fireEvent.click(screen.getByRole('button', { name: /Salvar Alterações/i }));

    await waitFor(() => {
      expect(mockedApiClient.put).toHaveBeenCalledWith(
        `/organizations/${organizationId}/webhooks/${initialWebhookData.id}`,
        expect.objectContaining({
            name: 'Updated Webhook',
            event_types: ['risk_created']
        })
      );
    });
    expect(mockOnSubmitSuccess).toHaveBeenCalledWith({ ...initialWebhookData, name: 'Updated Webhook' });
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('shows error if no event types are selected', async () => {
    renderForm();
    fireEvent.change(screen.getByLabelText(/Nome do Webhook/i), { target: { value: 'Test Webhook' } });
    fireEvent.change(screen.getByLabelText(/URL do Webhook/i), { target: { value: 'https://example.com/webhook' } });
    // Nenhum tipo de evento selecionado

    fireEvent.click(screen.getByRole('button', { name: /Adicionar Webhook/i }));

    expect(await screen.findByText(/Selecione pelo menos um tipo de evento./i)).toBeInTheDocument();
    expect(mockedApiClient.post).not.toHaveBeenCalled();
  });
});
