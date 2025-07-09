import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import RegisterPage from '../register'; // A página que estamos testando
import { AuthProvider } from '../../../contexts/AuthContext'; // Necessário se useAuth for usado indiretamente
import apiClient from '../../../lib/axios';
import { useNotifier } from '../../../hooks/useNotifier';

// Mocking next/router
jest.mock('next/router', () => ({
  useRouter: () => ({
    route: '/auth/register',
    pathname: '/auth/register',
    query: '',
    asPath: '/auth/register',
    push: jest.fn(),
  }),
}));

// Mocking next-i18next
const mockT = jest.fn((key) => key); // Simples mock para t, retorna a chave
jest.mock('next-i18next', () => ({
  useTranslation: () => ({
    t: mockT,
    i18n: { language: 'pt', changeLanguage: jest.fn() },
  }),
  // serverSideTranslations mock não é necessário para testes unitários de componentes/páginas
  // se estivermos mockando useTranslation. As props de getStaticProps serão vazias ou mockadas.
}));

// Mock apiClient (axios instance)
jest.mock('../../../lib/axios');
const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

// Mock useNotifier
const mockNotifySuccess = jest.fn();
const mockNotifyError = jest.fn();
jest.mock('../../../hooks/useNotifier', () => ({
  useNotifier: () => ({
    success: mockNotifySuccess,
    error: mockNotifyError,
    info: jest.fn(),
    warn: jest.fn(),
  }),
}));

// Mock AuthContext (embora RegisterPage não use useAuth diretamente, é bom ter o Provider se algum subcomponente usar)
jest.mock('../../../contexts/AuthContext', () => ({
  useAuth: () => ({ /* ...valores mockados do useAuth se necessário... */ }),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));


const renderRegisterPage = () => {
  // Passar um objeto de props vazio, simulando o que getStaticProps retornaria (apenas as props de i18n)
  return render(
    <AuthProvider> {/* Envolver com AuthProvider se necessário para algum contexto */}
      <RegisterPage />
    </AuthProvider>
  );
};

describe('RegisterPage', () => {
  beforeEach(() => {
    mockT.mockClear();
    mockedApiClient.post.mockClear();
    mockNotifySuccess.mockClear();
    mockNotifyError.mockClear();
  });

  test('renders the registration form with all fields and translated texts', async () => {
    renderRegisterPage();

    // Verificar se as chaves de tradução são chamadas para os textos principais
    expect(mockT).toHaveBeenCalledWith('register.title');
    expect(mockT).toHaveBeenCalledWith('register.join_message');
    expect(mockT).toHaveBeenCalledWith('register.full_name_label');
    expect(mockT).toHaveBeenCalledWith('register.email_label');
    expect(mockT).toHaveBeenCalledWith('register.org_name_label');
    expect(mockT).toHaveBeenCalledWith('register.password_label');
    expect(mockT).toHaveBeenCalledWith('register.confirm_password_label');
    expect(mockT).toHaveBeenCalledWith('register.submit_button');
    expect(mockT).toHaveBeenCalledWith('register.already_have_account_prompt');
    expect(mockT).toHaveBeenCalledWith('register.login_link');

    // Verificar se os elementos estão em tela (usando as chaves como se fossem o texto, devido ao mock simples de `t`)
    expect(screen.getByRole('heading', { name: 'register.title' })).toBeInTheDocument();
    expect(screen.getByLabelText('register.full_name_label')).toBeInTheDocument();
    expect(screen.getByLabelText('register.email_label')).toBeInTheDocument();
    expect(screen.getByLabelText('register.org_name_label')).toBeInTheDocument();
    expect(screen.getByLabelText('register.password_label')).toBeInTheDocument();
    expect(screen.getByLabelText('register.confirm_password_label')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'register.submit_button' })).toBeInTheDocument();
  });

  test('shows validation error if required fields are empty on submit', async () => {
    renderRegisterPage();
    fireEvent.click(screen.getByRole('button', { name: 'register.submit_button' }));

    expect(await screen.findByText('register.error_all_fields_required')).toBeInTheDocument();
    expect(mockedApiClient.post).not.toHaveBeenCalled();
  });

  test('shows validation error if passwords do not match', async () => {
    renderRegisterPage();
    fireEvent.change(screen.getByLabelText('register.full_name_label'), { target: { value: 'Test User' } });
    fireEvent.change(screen.getByLabelText('register.email_label'), { target: { value: 'test@example.com' } });
    fireEvent.change(screen.getByLabelText('register.org_name_label'), { target: { value: 'Test Org' } });
    fireEvent.change(screen.getByLabelText('register.password_label'), { target: { value: 'password123' } });
    fireEvent.change(screen.getByLabelText('register.confirm_password_label'), { target: { value: 'password456' } });
    fireEvent.click(screen.getByRole('button', { name: 'register.submit_button' }));

    expect(await screen.findByText('register.error_passwords_do_not_match')).toBeInTheDocument();
    expect(mockedApiClient.post).not.toHaveBeenCalled();
  });

  test('shows validation error if password is too short', async () => {
    renderRegisterPage();
    fireEvent.change(screen.getByLabelText('register.full_name_label'), { target: { value: 'Test User' } });
    fireEvent.change(screen.getByLabelText('register.email_label'), { target: { value: 'test@example.com' } });
    fireEvent.change(screen.getByLabelText('register.org_name_label'), { target: { value: 'Test Org' } });
    fireEvent.change(screen.getByLabelText('register.password_label'), { target: { value: 'short' } });
    fireEvent.change(screen.getByLabelText('register.confirm_password_label'), { target: { value: 'short' } });
    fireEvent.click(screen.getByRole('button', { name: 'register.submit_button' }));

    expect(await screen.findByText('register.error_password_too_short')).toBeInTheDocument();
    expect(mockedApiClient.post).not.toHaveBeenCalled();
  });

  test('calls API and notifier on successful registration, then disables form', async () => {
    mockedApiClient.post.mockResolvedValueOnce({ data: { message: 'Success from API' } });
    renderRegisterPage();

    fireEvent.change(screen.getByLabelText('register.full_name_label'), { target: { value: 'Test User' } });
    fireEvent.change(screen.getByLabelText('register.email_label'), { target: { value: 'test@example.com' } });
    fireEvent.change(screen.getByLabelText('register.org_name_label'), { target: { value: 'Test Org' } });
    fireEvent.change(screen.getByLabelText('register.password_label'), { target: { value: 'password123Strong' } });
    fireEvent.change(screen.getByLabelText('register.confirm_password_label'), { target: { value: 'password123Strong' } });

    const submitButton = screen.getByRole('button', { name: 'register.submit_button' });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledWith('/auth/register', {
        user: { name: 'Test User', email: 'test@example.com', password: 'password123Strong' },
        organization: { name: 'Test Org' },
      });
    });
    await waitFor(() => {
      expect(mockNotifySuccess).toHaveBeenCalledWith('Success from API');
    });

    // Verificar se o formulário está desabilitado (isSuccess = true)
    expect(submitButton).toBeDisabled();
    expect(screen.getByLabelText('register.full_name_label')).toBeDisabled();
    // ... verificar outros campos desabilitados
    expect(await screen.findByText('register.success_message')).toBeInTheDocument();

  });

  test('calls API and notifier on failed registration (API error)', async () => {
    mockedApiClient.post.mockRejectedValueOnce({ response: { data: { error: 'Email already exists' } } });
    renderRegisterPage();

    fireEvent.change(screen.getByLabelText('register.full_name_label'), { target: { value: 'Test User' } });
    fireEvent.change(screen.getByLabelText('register.email_label'), { target: { value: 'test@example.com' } });
    fireEvent.change(screen.getByLabelText('register.org_name_label'), { target: { value: 'Test Org' } });
    fireEvent.change(screen.getByLabelText('register.password_label'), { target: { value: 'password123Strong' } });
    fireEvent.change(screen.getByLabelText('register.confirm_password_label'), { target: { value: 'password123Strong' } });

    fireEvent.click(screen.getByRole('button', { name: 'register.submit_button' }));

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(mockNotifyError).toHaveBeenCalledWith('Email already exists');
    });
  });
});
