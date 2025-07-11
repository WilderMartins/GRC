import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import RiskForm from '../RiskForm';
import { I18nextProvider } from 'react-i18next';
import i18n from '@/lib/i18n/i18n-test.config'; // Ajuste o caminho se o seu config de teste i18n estiver em outro lugar
import { AuthContext, AuthContextType } from '@/contexts/AuthContext';

// Mocks
jest.mock('next/router', () => ({
  useRouter: () => ({
    push: jest.fn(),
    query: {},
  }),
}));

const mockApiClient = {
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
};
jest.mock('@/lib/axios', () => ({
  __esModule: true,
  default: mockApiClient,
}));

const mockNotifySuccess = jest.fn();
const mockNotifyError = jest.fn();
const mockNotifyWarn = jest.fn(); // Adicionado para cobrir todas as funções do notifier
const mockNotifyInfo = jest.fn(); // Adicionado para cobrir todas as funções do notifier

jest.mock('@/hooks/useNotifier', () => ({
  useNotifier: () => ({
    success: mockNotifySuccess,
    error: mockNotifyError,
    warn: mockNotifyWarn,
    info: mockNotifyInfo,
  }),
}));

const mockAuthContextValue: AuthContextType = {
  isAuthenticated: true,
  user: { id: 'user-123', name: 'Test User', email: 'test@example.com', role: 'admin', organization_id: 'org-123', is_totp_enabled: false },
  token: 'fake-token',
  branding: { primaryColor: '#4F46E5', secondaryColor: '#7C3AED', logoUrl: null },
  isLoading: false,
  login: jest.fn().mockResolvedValue(undefined),
  logout: jest.fn(),
  refreshBranding: jest.fn().mockResolvedValue(undefined),
  refreshUser: jest.fn().mockResolvedValue(undefined),
};

// Helper para renderizar com providers (sem NotifierContext.Provider, pois o hook é mockado)
const renderRiskFormWithProviders = (ui: React.ReactElement, authContextValue = mockAuthContextValue) => {
  // Carregar recursos de tradução para os namespaces usados no componente
  // Isso garante que t() funcione nos testes.
  // Idealmente, isso estaria em um jest.setup.ts ou i18n-test.config.ts global
  if (!i18n.isInitialized) { i18n.init(); } // Garantir inicialização

  if (!i18n.hasResourceBundle('pt-BR', 'risks')) {
    i18n.addResourceBundle('pt-BR', 'risks', {
        'form.field_title_label': 'Título do Risco',
        'form.field_description_label': 'Descrição',
        'form.field_category_label': 'Categoria',
        'form.field_status_label': 'Status',
        'form.field_impact_label': 'Impacto',
        'form.field_probability_label': 'Probabilidade',
        'form.field_owner_label': 'Proprietário do Risco',
        'form.error_impact_probability_required': 'Impacto e Probabilidade são obrigatórios.',
        'form.error_owner_required': 'Proprietário do Risco é obrigatório.',
        'form.create_success_message': 'Risco criado com sucesso!',
        'form.update_success_message': 'Risco atualizado com sucesso!',
        'form.save_error_message': 'Falha ao salvar risco: {{message}}',
    });
  }
  if (!i18n.hasResourceBundle('pt-BR', 'common')) {
    i18n.addResourceBundle('pt-BR', 'common', {
        'create_button': 'Criar Risco',
        'save_changes_button': 'Salvar Alterações',
        'unknown_error': 'Ocorreu um erro desconhecido.',
    });
  }

  return render(
    <I18nextProvider i18n={i18n}>
      <AuthContext.Provider value={authContextValue}>
        {ui}
      </AuthContext.Provider>
    </I18nextProvider>
  );
};


describe('RiskForm', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockApiClient.get.mockResolvedValue({ data: [{ id: 'user-123', name: 'Test User', email: 'test@example.com' }] });
  });

  test('renders correctly in creation mode and pre-selects owner', async () => {
    renderRiskFormWithProviders(
      <RiskForm organizationId="org-123" onSubmitSuccess={jest.fn()} isEditing={false} />
    );

    expect(screen.getByLabelText(i18n.t('risks:form.field_title_label'))).toBeInTheDocument();
    expect(screen.getByRole('button', { name: i18n.t('common:create_button') })).toBeInTheDocument();

    await waitFor(() => {
      expect(mockApiClient.get).toHaveBeenCalledWith('/users/organization-lookup');
      const ownerSelect = screen.getByLabelText(i18n.t('risks:form.field_owner_label')) as HTMLSelectElement;
      expect(ownerSelect.value).toBe('user-123');
    });
  });

  test('renders correctly in editing mode with initialData', async () => {
    const initialData = {
      id: 'risk-001',
      title: 'Risco de Teste Editável',
      description: 'Descrição editável',
      category: 'operacional' as const,
      impact: 'Alto' as const,
      probability: 'Médio' as const,
      status: 'em_andamento' as const,
      owner_id: 'owner-test-id',
    };
    mockApiClient.get.mockResolvedValueOnce({ data: [{id: 'owner-test-id', name: 'Owner Test'}, {id: 'user-123', name: 'Test User'}] });

    renderRiskFormWithProviders(
      <RiskForm
        organizationId="org-123"
        initialData={initialData}
        isEditing={true}
        onSubmitSuccess={jest.fn()}
      />
    );

    expect(screen.getByLabelText(i18n.t('risks:form.field_title_label'))).toHaveValue(initialData.title);
    await waitFor(() => {
      expect(screen.getByLabelText(i18n.t('risks:form.field_owner_label'))).toHaveValue(initialData.owner_id);
    });
    expect(screen.getByRole('button', { name: i18n.t('common:save_changes_button') })).toBeInTheDocument();
  });

  test('handles form submission for creating a new risk', async () => {
    const mockSubmitSuccess = jest.fn();
    mockApiClient.post.mockResolvedValueOnce({ data: { message: 'Risco criado com sucesso' } });
    // mockApiClient.get já está mockado no beforeEach para retornar 'Test User'

    renderRiskFormWithProviders(
      <RiskForm organizationId="org-123" onSubmitSuccess={mockSubmitSuccess} isEditing={false} />
    );

    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_title_label')), { target: { value: 'Novo Risco Submetido' } });
    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_impact_label')), { target: { value: 'Baixo' } });
    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_probability_label')), { target: { value: 'Baixo' } });

    fireEvent.click(screen.getByRole('button', { name: i18n.t('common:create_button') }));

    await waitFor(() => {
      expect(mockApiClient.post).toHaveBeenCalledWith('/risks', expect.objectContaining({
        title: 'Novo Risco Submetido',
        impact: 'Baixo',
        probability: 'Baixo',
        owner_id: 'user-123',
      }));
      expect(mockNotifySuccess).toHaveBeenCalledWith(i18n.t('risks:form.create_success_message'));
      expect(mockSubmitSuccess).toHaveBeenCalled();
    });
  });

  test('handles form submission for editing an existing risk', async () => {
    const mockSubmitSuccess = jest.fn();
    const initialData = {
      id: 'risk-001',
      title: 'Risco Antigo',
      description: 'Descrição antiga',
      category: 'legal' as const,
      impact: 'Crítico' as const,
      probability: 'Crítico' as const,
      status: 'aceito' as const,
      owner_id: 'owner-old-id',
    };
    mockApiClient.get.mockResolvedValueOnce({ data: [{id: 'owner-old-id', name: 'Old Owner'}, {id: 'user-123', name: 'Test User'}] });
    mockApiClient.put.mockResolvedValueOnce({ data: { message: 'Risco atualizado com sucesso' } });

    renderRiskFormWithProviders(
      <RiskForm
        organizationId="org-123"
        initialData={initialData}
        isEditing={true}
        onSubmitSuccess={mockSubmitSuccess}
      />
    );

    await waitFor(() => {
        expect(screen.getByLabelText(i18n.t('risks:form.field_owner_label'))).toHaveValue(initialData.owner_id);
    });

    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_title_label')), { target: { value: 'Risco Editado' } });
    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_impact_label')), { target: { value: 'Médio' } });
    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_owner_label')), { target: { value: 'user-123' } });

    fireEvent.click(screen.getByRole('button', { name: i18n.t('common:save_changes_button') }));

    await waitFor(() => {
      expect(mockApiClient.put).toHaveBeenCalledWith(`/risks/${initialData.id}`, expect.objectContaining({
        title: 'Risco Editado',
        impact: 'Médio',
        owner_id: 'user-123',
      }));
      expect(mockNotifySuccess).toHaveBeenCalledWith(i18n.t('risks:form.update_success_message'));
      expect(mockSubmitSuccess).toHaveBeenCalled();
    });
  });

  test('shows validation error if required fields are missing on submit', async () => {
    renderRiskFormWithProviders(
      <RiskForm organizationId="org-123" onSubmitSuccess={jest.fn()} isEditing={false} />
    );

    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_title_label')), { target: { value: 'Risco Incompleto' } });

    fireEvent.click(screen.getByRole('button', { name: i18n.t('common:create_button') }));

    await waitFor(() => {
      expect(screen.getByText(i18n.t('risks:form.error_impact_probability_required'))).toBeInTheDocument();
    });
    expect(mockApiClient.post).not.toHaveBeenCalled();
  });

  test('shows API error on submission failure during creation', async () => {
    const mockSubmitSuccess = jest.fn();
    const apiErrorMessage = 'Erro da API ao Criar';
    // mockApiClient.get já mockado no beforeEach
    mockApiClient.post.mockRejectedValueOnce({
      response: { data: { error: apiErrorMessage } },
    });

    renderRiskFormWithProviders(
      <RiskForm organizationId="org-123" onSubmitSuccess={mockSubmitSuccess} isEditing={false} />
    );

    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_title_label')), { target: { value: 'Risco com Erro API' } });
    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_impact_label')), { target: { value: 'Alto' } });
    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_probability_label')), { target: { value: 'Médio' } });
    // Owner deve ser auto-selecionado pelo mock do beforeEach

    fireEvent.click(screen.getByRole('button', { name: i18n.t('common:create_button') }));

    await waitFor(() => {
      expect(mockApiClient.post).toHaveBeenCalledTimes(1);
      expect(screen.getByText(apiErrorMessage)).toBeInTheDocument();
      expect(mockNotifyError).toHaveBeenCalledWith(i18n.t('risks:form.save_error_message', { message: apiErrorMessage }));
    });
    expect(mockSubmitSuccess).not.toHaveBeenCalled();
  });

  test('shows API error on submission failure during editing', async () => {
    const mockSubmitSuccess = jest.fn();
    const apiErrorMessage = 'Erro da API ao Atualizar';
    const initialData = {
      id: 'risk-edit-fail',
      title: 'Risco para Falhar Edição',
      description: 'Descrição original',
      category: 'tecnologico' as const,
      impact: 'Médio' as const,
      probability: 'Baixo' as const,
      status: 'aberto' as const,
      owner_id: 'user-123', // Usar o mesmo owner do mock geral para simplificar
    };

    // mockApiClient.get já mockado no beforeEach
    mockApiClient.put.mockRejectedValueOnce({
      response: { data: { error: apiErrorMessage } },
    });

    renderRiskFormWithProviders(
      <RiskForm
        organizationId="org-123"
        initialData={initialData}
        isEditing={true}
        onSubmitSuccess={mockSubmitSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.getByLabelText(i18n.t('risks:form.field_title_label'))).toHaveValue(initialData.title);
    });

    fireEvent.change(screen.getByLabelText(i18n.t('risks:form.field_description_label')), { target: { value: 'Nova Descrição com Erro' } });
    fireEvent.click(screen.getByRole('button', { name: i18n.t('common:save_changes_button') }));

    await waitFor(() => {
      expect(mockApiClient.put).toHaveBeenCalledWith(`/risks/${initialData.id}`, expect.objectContaining({
        description: 'Nova Descrição com Erro',
      }));
      expect(screen.getByText(apiErrorMessage)).toBeInTheDocument();
      expect(mockNotifyError).toHaveBeenCalledWith(i18n.t('risks:form.save_error_message', { message: apiErrorMessage }));
    });
    expect(mockSubmitSuccess).not.toHaveBeenCalled();
  });
});
