import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import IdentityProviderForm from '../IdentityProviderForm'; // Ajuste o path
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

describe('IdentityProviderForm', () => {
  const mockOnClose = jest.fn();
  const mockOnSubmitSuccess = jest.fn();

  const baseProps = {
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
        <IdentityProviderForm {...baseProps} {...props} />
      </AuthContext.Provider>
    );
  };

  it('renders correctly for a new provider', () => {
    renderForm();
    expect(screen.getByRole('heading', { name: /Adicionar Novo Provedor/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/Nome do Provedor/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Tipo de Provedor/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Configuração JSON/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Ativo/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Adicionar Provedor/i })).toBeInTheDocument();
  });

  it('submits new provider data correctly', async () => {
    mockedApiClient.post.mockResolvedValue({ data: { id: 'new-idp-id' } });
    renderForm();

    fireEvent.change(screen.getByLabelText(/Nome do Provedor/i), { target: { value: 'Test SAML' } });
    fireEvent.change(screen.getByLabelText(/Tipo de Provedor/i), { target: { value: 'saml' } });
    fireEvent.change(screen.getByLabelText(/Configuração JSON/i), { target: { value: '{"entity_id":"test"}' } });

    fireEvent.click(screen.getByRole('button', { name: /Adicionar Provedor/i }));

    await waitFor(() => {
      expect(mockedApiClient.post).toHaveBeenCalledWith(
        `/organizations/${mockUser.organization_id}/identity-providers`,
        expect.objectContaining({
          name: 'Test SAML',
          provider_type: 'saml',
          config_json: { entity_id: 'test' },
          is_active: true, // Default
        })
      );
    });
    expect(mockOnSubmitSuccess).toHaveBeenCalledWith({ id: 'new-idp-id' });
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('pre-fills form for editing and submits updates', async () => {
    const initialIdPData = {
      id: 'idp-edit-123',
      name: 'Existing Google IdP',
      provider_type: 'oauth2_google' as 'oauth2_google',
      is_active: true,
      config_json_parsed: { client_id: 'google-client', client_secret: 'secret' },
      attribute_mapping_json_parsed: { email: 'email_address' },
    };
    mockedApiClient.put.mockResolvedValue({ data: { ...initialIdPData, name: 'Updated Google IdP' } });

    renderForm({ initialData: initialIdPData, isEditing: true });

    expect((screen.getByLabelText(/Nome do Provedor/i) as HTMLInputElement).value).toBe('Existing Google IdP');
    expect((screen.getByLabelText(/Tipo de Provedor/i) as HTMLSelectElement).value).toBe('oauth2_google');
    expect((screen.getByLabelText(/Configuração JSON/i) as HTMLTextAreaElement).value).toContain('google-client');

    fireEvent.change(screen.getByLabelText(/Nome do Provedor/i), { target: { value: 'Updated Google IdP' } });
    fireEvent.click(screen.getByRole('button', { name: /Salvar Alterações/i }));

    await waitFor(() => {
      expect(mockedApiClient.put).toHaveBeenCalledWith(
        `/organizations/${mockUser.organization_id}/identity-providers/${initialIdPData.id}`,
        expect.objectContaining({ name: 'Updated Google IdP' })
      );
    });
    expect(mockOnSubmitSuccess).toHaveBeenCalledWith({ ...initialIdPData, name: 'Updated Google IdP' });
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('shows error if config_json is invalid', async () => {
    renderForm();
    fireEvent.change(screen.getByLabelText(/Nome do Provedor/i), { target: { value: 'Test SAML' } });
    fireEvent.change(screen.getByLabelText(/Tipo de Provedor/i), { target: { value: 'saml' } });
    fireEvent.change(screen.getByLabelText(/Configuração JSON/i), { target: { value: '{"entity_id":"test"' } }); // JSON Inválido

    fireEvent.click(screen.getByRole('button', { name: /Adicionar Provedor/i }));

    expect(await screen.findByText(/Configuração JSON inválida/i)).toBeInTheDocument();
    expect(mockedApiClient.post).not.toHaveBeenCalled();
  });
});
