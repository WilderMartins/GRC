"use client"; // Se estiver usando App Router e o HOC precisar de hooks do lado do cliente

import { useAuth } from '../../contexts/AuthContext'; // Ajuste o path
import { useRouter } from 'next/router'; // Pages Router
// import { useRouter, usePathname } from 'next/navigation'; // App Router
import React, { ComponentType, useEffect } from 'react';

interface WithAuthProps {
  // Você pode adicionar props específicas que o HOC pode passar para o WrappedComponent
}

const WithAuth = <P extends object>(WrappedComponent: ComponentType<P>) => {
  const ComponentWithAuth = (props: P & WithAuthProps) => {
    const auth = useAuth();
    const router = useRouter(); // Pages Router
    // const pathname = usePathname(); // App Router, se necessário para lógica de redirecionamento

    useEffect(() => {
      // Se o carregamento inicial do AuthContext ainda não terminou, não faz nada ainda.
      // Isso evita redirecionamentos prematuros antes do estado ser restaurado do localStorage.
      if (auth.isLoading) {
        return;
      }

      if (!auth.isAuthenticated) {
        // Salvar a rota atual para redirecionar de volta após o login (opcional)
        // router.replace(`/auth/login?redirect=${router.asPath}`);
        router.replace('/auth/login'); // Redirecionamento simples por enquanto
      }
    }, [auth.isAuthenticated, auth.isLoading, router]);

    // Se ainda estiver carregando o estado de auth, pode mostrar um loader global
    // ou simplesmente não renderizar o componente ainda.
    if (auth.isLoading || !auth.isAuthenticated) {
      // Pode retornar um componente de loading aqui, ou null para evitar flash de conteúdo.
      // Exemplo: return <GlobalPageLoader />;
      return null; // Ou um spinner/skeleton
    }

    // Se autenticado, renderiza o componente embrulhado
    return <WrappedComponent {...props} />;
  };

  // Adicionar um displayName para melhor debugging no React DevTools
  const displayName = WrappedComponent.displayName || WrappedComponent.name || 'Component';
  ComponentWithAuth.displayName = `WithAuth(${displayName})`;

  return ComponentWithAuth;
};

export default WithAuth;
