"use client"; // Indicar que este é um Client Component se usar App Router

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { useRouter } from 'next/router'; // Para Next.js Pages Router
// import { useRouter } from 'next/navigation'; // Para Next.js App Router
import { User } from '@/types'; // Importar User de @/types

// Definição local de User removida

import apiClient from '@/lib/axios'; // Importar apiClient

interface BrandingSettings {
  logoUrl?: string | null;
  primaryColor?: string | null;
  secondaryColor?: string | null;
}
interface AuthContextType {
  isAuthenticated: boolean;
  user: User | null; // Usar o User importado (que deve incluir is_totp_enabled)
  token: string | null;
  branding: BrandingSettings; // Adicionar configurações de branding
  isLoading: boolean; // Para verificar se o estado inicial já foi carregado do localStorage
  login: (userData: any, token: string) => Promise<void>; // Modificado para async, userData: any para flexibilidade da API
  logout: () => void;
  refreshBranding: () => Promise<void>; // Função para recarregar branding
  refreshUser: () => Promise<void>; // Função para recarregar dados do usuário (incluindo status MFA)
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const defaultBranding: BrandingSettings = {
  logoUrl: null, // Ou um logo padrão do Phoenix GRC
  primaryColor: '#4F46E5', // Exemplo: Indigo-600 Tailwind
  secondaryColor: '#7C3AED', // Exemplo: Purple-600 Tailwind
};

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [branding, setBranding] = useState<BrandingSettings>(defaultBranding);
  const [isLoading, setIsLoading] = useState(true); // Começa true até carregar do localStorage
  const router = useRouter(); // Pages Router

  const fetchBrandingSettings = async (organizationId: string): Promise<BrandingSettings> => {
    try {
      const response = await apiClient.get(`/organizations/${organizationId}/branding`);
      const { logo_url, primary_color, secondary_color } = response.data;
      return {
        logoUrl: logo_url || defaultBranding.logoUrl,
        primaryColor: primary_color || defaultBranding.primaryColor,
        secondaryColor: secondary_color || defaultBranding.secondaryColor,
      };
    } catch (error) {
      console.error("Failed to fetch branding settings, using defaults:", error);
      return defaultBranding; // Retorna padrão em caso de erro
    }
  };

  useEffect(() => {
    const loadStoredData = async () => {
      setIsLoading(true);
      try {
        const storedToken = localStorage.getItem('authToken');
        const storedUserString = localStorage.getItem('authUser');
        const storedBrandingString = localStorage.getItem('authBranding');

        if (storedToken && storedUserString) {
          const storedUser = JSON.parse(storedUserString) as User;
          setUser(storedUser);
          setToken(storedToken);

          if (storedBrandingString) {
            setBranding(JSON.parse(storedBrandingString));
          } else if (storedUser.organization_id) {
            // Se não houver branding no localStorage mas houver usuário, buscar da API
            const fetchedBranding = await fetchBrandingSettings(storedUser.organization_id);
            setBranding(fetchedBranding);
            localStorage.setItem('authBranding', JSON.stringify(fetchedBranding));
          }
        } else {
          // Se não há token/usuário, resetar para o branding padrão
          setBranding(defaultBranding);
          localStorage.removeItem('authBranding'); // Garantir que está limpo
        }
      } catch (error) {
        console.error("Failed to parse auth data from localStorage", error);
        localStorage.removeItem('authToken');
        localStorage.removeItem('authUser');
        localStorage.removeItem('authBranding');
        setUser(null);
        setToken(null);
        setBranding(defaultBranding);
      }
      setIsLoading(false);
    };
    loadStoredData();
  }, []);

  const login = async (apiUserData: any, newToken: string) => { // apiUserData da API pode não ter todos os campos de User type
    setIsLoading(true);

    // Mapear apiUserData para o tipo User, incluindo is_totp_enabled se vier da API de login
    // Se a API de login não retornar is_totp_enabled, será buscado por refreshUser ou /me
    const appUser: User = {
      id: apiUserData.user_id || apiUserData.id, // Ajustar conforme a resposta real da API de login
      name: apiUserData.name,
      email: apiUserData.email,
      role: apiUserData.role,
      organization_id: apiUserData.organization_id,
      is_totp_enabled: apiUserData.is_totp_enabled || false, // Assumir que pode vir da API de login
      // organization: apiUserData.organization, // Se vier da API
    };

    localStorage.setItem('authToken', newToken);
    localStorage.setItem('authUser', JSON.stringify(appUser));
    setUser(appUser);
    setToken(newToken);

    if (appUser.organization_id) {
      const fetchedBranding = await fetchBrandingSettings(appUser.organization_id);
      setBranding(fetchedBranding);
      localStorage.setItem('authBranding', JSON.stringify(fetchedBranding));
    } else {
      setBranding(defaultBranding);
      localStorage.removeItem('authBranding');
    }

    // Se a API de login não retornou is_totp_enabled, e precisamos dele imediatamente,
    // poderíamos chamar refreshUser aqui. Mas loadStoredData também tenta buscar.
    // Para simplificar, a primeira carga de is_totp_enabled virá do localStorage ou de uma chamada /me em refreshUser.

    setIsLoading(false);

    if (appUser.role === 'admin' || appUser.role === 'manager') {
      router.push('/admin/dashboard');
    } else {
      router.push('/dashboard');
    }
  };

  const refreshUser = async () => {
    if (!token) return; // Não pode recarregar usuário sem token
    setIsLoading(true);
    try {
      const response = await apiClient.get('/me'); // Assumindo que /me retorna o objeto User completo, incluindo is_totp_enabled
      const updatedUser: User = {
        id: response.data.user_id || response.data.id,
        name: response.data.name,
        email: response.data.email,
        role: response.data.role,
        organization_id: response.data.organization_id,
        is_totp_enabled: response.data.is_totp_enabled || false,
        // organization: response.data.organization, // Se /me retornar isso
      };
      setUser(updatedUser);
      localStorage.setItem('authUser', JSON.stringify(updatedUser));

      // Se organization_id mudou (improvável) ou se branding não estava carregado, recarregar branding
      if (updatedUser.organization_id && (!branding || branding === defaultBranding)) {
        const fetchedBranding = await fetchBrandingSettings(updatedUser.organization_id);
        setBranding(fetchedBranding);
        localStorage.setItem('authBranding', JSON.stringify(fetchedBranding));
      }
    } catch (error: any) {
      console.error("Failed to refresh user data:", error);
      if (error.response && error.response.status === 401) {
        console.log("AuthContext: Refresh user failed with 401, logging out.");
        logout(); // Chamar logout se /me retornar 401
      }
      // Outros erros não necessariamente invalidam a sessão, podem ser temporários.
    } finally {
      setIsLoading(false);
    }
  };

  const refreshBranding = async () => {
    if (user && user.organization_id) {
      setIsLoading(true); // Pode querer um isLoadingBranding separado
      const fetchedBranding = await fetchBrandingSettings(user.organization_id);
      setBranding(fetchedBranding);
      localStorage.setItem('authBranding', JSON.stringify(fetchedBranding));
      setIsLoading(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('authToken');
    localStorage.removeItem('authUser');
    localStorage.removeItem('authBranding');
    setUser(null);
    setToken(null);
    setBranding(defaultBranding); // Resetar para o branding padrão
    router.push('/auth/login');
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated: !!token, user, token, branding, isLoading, login, logout, refreshBranding, refreshUser }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
