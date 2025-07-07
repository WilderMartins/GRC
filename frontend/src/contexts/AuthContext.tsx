"use client"; // Indicar que este é um Client Component se usar App Router

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { useRouter } from 'next/router'; // Para Next.js Pages Router
// import { useRouter } from 'next/navigation'; // Para Next.js App Router

interface User {
  id: string;
  name: string;
  email: string;
  role: string;
  organization_id: string;
}

interface AuthContextType {
  isAuthenticated: boolean;
  user: User | null;
  token: string | null;
  isLoading: boolean; // Para verificar se o estado inicial já foi carregado do localStorage
  login: (userData: User, token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true); // Começa true até carregar do localStorage
  const router = useRouter(); // Pages Router

  useEffect(() => {
    // Tentar carregar dados de autenticação do localStorage ao inicializar
    try {
      const storedToken = localStorage.getItem('authToken');
      const storedUserString = localStorage.getItem('authUser');
      if (storedToken && storedUserString) {
        const storedUser = JSON.parse(storedUserString) as User;
        setToken(storedToken);
        setUser(storedUser);
      }
    } catch (error) {
      console.error("Failed to parse auth data from localStorage", error);
      localStorage.removeItem('authToken');
      localStorage.removeItem('authUser');
    }
    setIsLoading(false); // Finaliza o carregamento inicial
  }, []);

  const login = (userData: User, newToken: string) => {
    localStorage.setItem('authToken', newToken);
    localStorage.setItem('authUser', JSON.stringify(userData));
    setUser(userData);
    setToken(newToken);
    // Redirecionar para o dashboard ou página principal após o login
    // A role pode ser usada para redirecionar para paineis diferentes
    if (userData.role === 'admin' || userData.role === 'manager') {
        router.push('/admin/dashboard');
    } else {
        router.push('/dashboard'); // Um dashboard geral para 'user'
    }
  };

  const logout = () => {
    localStorage.removeItem('authToken');
    localStorage.removeItem('authUser');
    setUser(null);
    setToken(null);
    router.push('/auth/login');
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated: !!token, user, token, isLoading, login, logout }}>
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
