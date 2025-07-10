"use client";

import React, { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react';
import apiClient from '@/lib/axios';
import { FeatureToggle } from '@/types'; // Assumindo que FeatureToggle está em @/types

interface FeatureToggleContextType {
  toggles: Record<string, FeatureToggle>; // Armazena todos os toggles com seus detalhes
  isLoadingToggles: boolean;
  isFeatureEnabled: (key: string) => boolean;
  getFeatureToggle: (key: string) => FeatureToggle | undefined;
  refreshToggles: () => Promise<void>;
}

const FeatureToggleContext = createContext<FeatureToggleContextType | undefined>(undefined);

const FEATURE_TOGGLES_STORAGE_KEY = 'phoenix-feature-toggles';

export const FeatureToggleProvider = ({ children }: { children: ReactNode }) => {
  const [toggles, setToggles] = useState<Record<string, FeatureToggle>>({});
  const [isLoadingToggles, setIsLoadingToggles] = useState(true);

  const fetchAndSetToggles = useCallback(async (isInitialLoad = false) => {
    if (!isInitialLoad) setIsLoadingToggles(true); // Mostrar loading apenas em refresh manual

    try {
      // Tentar carregar do localStorage primeiro na carga inicial
      if (isInitialLoad) {
        const storedTogglesRaw = localStorage.getItem(FEATURE_TOGGLES_STORAGE_KEY);
        if (storedTogglesRaw) {
          const storedToggles = JSON.parse(storedTogglesRaw) as Record<string, FeatureToggle>;
          setToggles(storedToggles);
          // Mesmo que carregue do localStorage, buscar da API para atualizar em segundo plano
          // mas não setar isLoadingToggles para true para evitar piscar, a menos que seja um refresh.
          if (Object.keys(storedToggles).length > 0) {
             setIsLoadingToggles(false); // Considera carregado se tinha algo no storage
          }
        }
      }

      const response = await apiClient.get<FeatureToggle[]>('/feature-toggles'); // API Hipotética
      const fetchedTogglesArray = response.data || [];
      const togglesMap: Record<string, FeatureToggle> = {};
      for (const toggle of fetchedTogglesArray) {
        togglesMap[toggle.key] = toggle;
      }
      setToggles(togglesMap);
      localStorage.setItem(FEATURE_TOGGLES_STORAGE_KEY, JSON.stringify(togglesMap));
    } catch (error) {
      console.error("Failed to fetch feature toggles:", error);
      // Em caso de erro, pode-se manter os toggles do localStorage (se houver) ou limpar.
      // Por simplicidade, vamos manter o que estava no localStorage ou vazio.
      // Se for a carga inicial e o localStorage estiver vazio, o estado 'toggles' permanecerá vazio.
      if (isInitialLoad && Object.keys(toggles).length === 0) { // Apenas setar erro se não conseguiu carregar nada
          // Poderia ter um estado de erro específico aqui.
      }
    } finally {
      // Garantir que o loading seja false após a tentativa da API,
      // especialmente se o localStorage estava vazio.
      setIsLoadingToggles(false);
    }
  }, [toggles]); // Adicionado 'toggles' para que, se ele estiver vazio e a chamada falhar, não entre em loop de re-fetch

  useEffect(() => {
    fetchAndSetToggles(true); // Carga inicial
  }, [fetchAndSetToggles]); // fetchAndSetToggles é useCallback

  const isFeatureEnabled = (key: string): boolean => {
    return !!toggles[key]?.is_active;
  };

  const getFeatureToggle = (key: string): FeatureToggle | undefined => {
    return toggles[key];
  };

  const refreshToggles = async () => {
    await fetchAndSetToggles(false); // Chamar com false para indicar que é um refresh manual
  };

  return (
    <FeatureToggleContext.Provider value={{ toggles, isLoadingToggles, isFeatureEnabled, getFeatureToggle, refreshToggles }}>
      {children}
    </FeatureToggleContext.Provider>
  );
};

export const useFeatureToggles = (): FeatureToggleContextType => {
  const context = useContext(FeatureToggleContext);
  if (context === undefined) {
    throw new Error('useFeatureToggles must be used within a FeatureToggleProvider');
  }
  return context;
};
