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
  user: User | null; // Usar o User importado
  token: string | null;
  branding: BrandingSettings; // Adicionar configurações de branding
  isLoading: boolean; // Para verificar se o estado inicial já foi carregado do localStorage
  login: (userData: User, token: string) => Promise<void>; // Modificado para async
  logout: () => void;
  refreshBranding: () => Promise<void>; // Função para recarregar branding
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

  const login = async (userData: User, newToken: string) => {
    setIsLoading(true); // Indicar carregamento durante o login e busca de branding
    localStorage.setItem('authToken', newToken);
    localStorage.setItem('authUser', JSON.stringify(userData));
    setUser(userData);
    setToken(newToken);

    if (userData.organization_id) {
      const fetchedBranding = await fetchBrandingSettings(userData.organization_id);
      setBranding(fetchedBranding);
      localStorage.setItem('authBranding', JSON.stringify(fetchedBranding));
    } else {
      // Caso raro: usuário sem organization_id? Usar default.
      setBranding(defaultBranding);
      localStorage.removeItem('authBranding');
    }

    setIsLoading(false);

    if (userData.role === 'admin' || userData.role === 'manager') {
      router.push('/admin/dashboard');
    } else {
      router.push('/dashboard');
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
    <AuthContext.Provider value={{ isAuthenticated: !!token, user, token, branding, isLoading, login, logout, refreshBranding }}>
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
