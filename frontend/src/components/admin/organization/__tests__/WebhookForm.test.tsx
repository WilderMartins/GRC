import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import WebhookForm from '../WebhookForm';
import { I18nextProvider } from 'react-i18next';
import i18n from '@/lib/i18n/i18n-test.config';
import { AuthContext, AuthContextType } from '@/contexts/AuthContext'; // WebhookForm não usa AuthContext diretamente
import { WebhookConfiguration } from '@/types';

// Mocks
jest.mock('next/router', () => ({
  useRouter: () => ({
    push: jest.fn(),
    query: {},
  }),
}));

const mockApiClient = {
  post: jest.fn(),
  put: jest.fn(),
};
jest.mock('@/lib/axios', () => ({
  __esModule: true,
  default: mockApiClient,
}));

const mockNotifySuccess = jest.fn();
const mockNotifyError = jest.fn();
jest.mock('@/hooks/useNotifier', () => ({
  useNotifier: () => ({
    success: mockNotifySuccess,
    error: mockNotifyError,
    warn: jest.fn(),
    info: jest.fn(),
  }),
}));

// WebhookForm não usa AuthContext, então um mock simples ou nenhum provider seria necessário
// se não fosse pelo AdminLayout que ele pode estar dentro em uma página.
// Para teste unitário do formulário isolado, não é estritamente necessário.
// Mas se o componente tiver dependências indiretas, pode ser preciso.

// Helper para renderizar com providers
const renderWebhookFormWithProviders = (ui: React.ReactElement) => {
  if (!i18n.isInitialized) { i18n.init(); }
  const namespaces = ['webhooks', 'common'];
  namespaces.forEach(ns => {
    if (!i18n.hasResourceBundle('pt-BR', ns)) {
      i18n.addResourceBundle('pt-BR', ns, {
        'form.name_label': 'Nome do Webhook',
        'form.url_label': 'URL do Endpoint',
        'form.secret_label': 'Segredo',
        'form.event_types_label': 'Tipos de Evento a Serem Enviados',
        'form.is_active_label': 'Ativo',
        'form.event_type_risk_created': 'Risco Criado',
        'form.event_type_risk_updated': 'Risco Atualizado',
        'form.error_event_types_required': 'Pelo menos um tipo de evento deve ser selecionado.',
        'form.create_success_message': 'Webhook criado com sucesso.',
        'form.update_success_message': 'Webhook atualizado com sucesso.',
        'form.save_error_message': 'Falha ao salvar webhook: {{message}}',
        ...(ns === 'common' ? {
            'create_button': 'Criar Webhook',
            'save_changes_button': 'Salvar Alterações',
            'cancel_button': 'Cancelar',
            'optional': '(Opcional)',
        } : {})
      });
    }
  });

  return render(
    <I18nextProvider i18n={i18n}>
      {ui}
    </I18nextProvider>
  );
};

describe('WebhookForm', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders correctly in creation mode', () => {
    renderWebhookFormWithProviders(
      <WebhookForm
        organizationId="org-123"
        onSubmitSuccess={jest.fn()}
        isEditing={false}
      />
    );

    expect(screen.getByLabelText(i18n.t('webhooks:form.name_label'))).toBeInTheDocument();
    expect(screen.getByLabelText(i18n.t('webhooks:form.url_label'))).toBeInTheDocument();
    expect(screen.getByLabelText(new RegExp(i18n.t('webhooks:form.secret_label')))).toBeInTheDocument(); // Inclui (Opcional)
    expect(screen.getByText(i18n.t('webhooks:form.event_types_label'))).toBeInTheDocument();
    expect(screen.getByLabelText(i18n.t('webhooks:form.event_type_risk_created'))).toBeInTheDocument(); // Verifica um tipo de evento
    expect(screen.getByLabelText(i18n.t('webhooks:form.is_active_label'))).toBeInTheDocument();
    expect(screen.getByRole('button', { name: i18n.t('common:create_button')})).toBeInTheDocument();
  });

  test('renders in editing mode with initialData', async () => {
    const initialData: WebhookConfiguration = {
      id: 'wh-123',
      organization_id: 'org-123',
      name: 'Webhook de Teste',
      url: 'https://example.com/webhook',
      event_types: JSON.stringify(['risk_created', 'risk_updated']),
      event_types_list: ['risk_created', 'risk_updated'],
      is_active: false,
      secret: 'supersecret',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    renderWebhookFormWithProviders(
      <WebhookForm
        organizationId="org-123"
        initialData={initialData}
        isEditing={true}
        onSubmitSuccess={jest.fn()}
      />
    );

    expect(screen.getByLabelText(i18n.t('webhooks:form.name_label'))).toHaveValue(initialData.name);
    expect(screen.getByLabelText(i18n.t('webhooks:form.url_label'))).toHaveValue(initialData.url);
    expect(screen.getByLabelText(new RegExp(i18n.t('webhooks:form.secret_label')))).toHaveValue(initialData.secret);
    expect(screen.getByLabelText(i18n.t('webhooks:form.is_active_label'))).not.toBeChecked();

    // Verificar checkboxes de evento
    expect(screen.getByLabelText(i18n.t('webhooks:form.event_type_risk_created'))).toBeChecked();
    expect(screen.getByLabelText(i18n.t('webhooks:form.event_type_risk_updated'))).toBeChecked();
    // Supondo que este não estava nos dados iniciais
    const riskDeletedCheckbox = screen.queryByLabelText(i18n.t('webhooks:form.event_type_risk_deleted'));
    if (riskDeletedCheckbox) { // Apenas checar se o evento existe na lista AVAILABLE_EVENT_TYPES
        expect(riskDeletedCheckbox).not.toBeChecked();
    }


    expect(screen.getByRole('button', { name: i18n.t('common:save_changes_button')})).toBeInTheDocument();
  });

  test('handles event type selection', () => {
    renderWebhookFormWithProviders(
      <WebhookForm organizationId="org-123" onSubmitSuccess={jest.fn()} />
    );
    const riskCreatedCheckbox = screen.getByLabelText(i18n.t('webhooks:form.event_type_risk_created'));
    const riskUpdatedCheckbox = screen.getByLabelText(i18n.t('webhooks:form.event_type_risk_updated'));

    expect(riskCreatedCheckbox).not.toBeChecked();
    expect(riskUpdatedCheckbox).not.toBeChecked();

    fireEvent.click(riskCreatedCheckbox);
    expect(riskCreatedCheckbox).toBeChecked();
    expect(riskUpdatedCheckbox).not.toBeChecked();

    fireEvent.click(riskUpdatedCheckbox);
    expect(riskCreatedCheckbox).toBeChecked();
    expect(riskUpdatedCheckbox).toBeChecked();

    fireEvent.click(riskCreatedCheckbox);
    expect(riskCreatedCheckbox).not.toBeChecked();
    expect(riskUpdatedCheckbox).toBeChecked();
  });

  test('handles form submission for creating a new webhook', async () => {
    const mockOnSubmitSuccess = jest.fn();
    const organizationId = 'org-123';
    mockApiClient.post.mockResolvedValueOnce({ data: { message: 'Webhook criado' } });

    renderWebhookFormWithProviders(
      <WebhookForm organizationId={organizationId} onSubmitSuccess={mockOnSubmitSuccess} isEditing={false} />
    );

    fireEvent.change(screen.getByLabelText(i18n.t('webhooks:form.name_label')), { target: { value: 'Meu Webhook' } });
    fireEvent.change(screen.getByLabelText(i18n.t('webhooks:form.url_label')), { target: { value: 'https://new.hook/target' } });
    fireEvent.click(screen.getByLabelText(i18n.t('webhooks:form.event_type_risk_created')));
    fireEvent.click(screen.getByLabelText(i18n.t('webhooks:form.event_type_vulnerability_created'))); // Assumindo que esta chave existe
    fireEvent.change(screen.getByLabelText(new RegExp(i18n.t('webhooks:form.secret_label'))), { target: { value: 'mysecret' } });


    fireEvent.click(screen.getByRole('button', { name: i18n.t('common:create_button') }));

    await waitFor(() => {
      expect(mockApiClient.post).toHaveBeenCalledWith(
        `/organizations/${organizationId}/webhooks`,
        {
          name: 'Meu Webhook',
          url: 'https://new.hook/target',
          event_types: JSON.stringify(['risk_created', 'vulnerability_created']),
          is_active: true, // Default
          secret: 'mysecret',
        }
      );
      expect(mockNotifySuccess).toHaveBeenCalledWith(i18n.t('webhooks:form.create_success_message'));
      expect(mockOnSubmitSuccess).toHaveBeenCalled();
    });
  });

  test('shows validation error if no event types are selected', async () => {
    renderWebhookFormWithProviders(
      <WebhookForm organizationId="org-123" onSubmitSuccess={jest.fn()} isEditing={false} />
    );
    fireEvent.change(screen.getByLabelText(i18n.t('webhooks:form.name_label')), { target: { value: 'Webhook Sem Eventos' } });
    fireEvent.change(screen.getByLabelText(i18n.t('webhooks:form.url_label')), { target: { value: 'https://noevents.hook' } });
    // Não selecionar nenhum tipo de evento

    fireEvent.click(screen.getByRole('button', { name: i18n.t('common:create_button') }));

    await waitFor(() => {
      expect(screen.getByText(i18n.t('webhooks:form.error_event_types_required'))).toBeInTheDocument();
    });
    expect(mockApiClient.post).not.toHaveBeenCalled();
  });

  // TODO:
  // - Testar submissão de edição
  // - Testar tratamento de erro da API
  // - Testar o campo 'secret' na edição (se deve ser apenas para inserir novo ou se mostra algo se já existir)
});
