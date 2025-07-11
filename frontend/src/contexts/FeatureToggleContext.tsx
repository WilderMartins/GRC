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
    if (!isInitialLoad) {
        setIsLoadingToggles(true); // Mostrar loading em refresh manual
    } else {
        // Na carga inicial, tentar carregar do localStorage primeiro
        const storedTogglesRaw = localStorage.getItem(FEATURE_TOGGLES_STORAGE_KEY);
        if (storedTogglesRaw) {
            try {
                const storedToggles = JSON.parse(storedTogglesRaw) as Record<string, FeatureToggle>;
                setToggles(storedToggles);
                if (Object.keys(storedToggles).length > 0) {
                    setIsLoadingToggles(false); // Considera carregado se tinha algo no storage
                }
            } catch (e) {
                console.error("Failed to parse stored feature toggles:", e);
                localStorage.removeItem(FEATURE_TOGGLES_STORAGE_KEY); // Limpar se estiver corrompido
            }
        }
    }

    try {
      const response = await apiClient.get<FeatureToggle[]>('/feature-toggles'); // API Hipotética
      const fetchedTogglesArray = response.data || [];
      const togglesMap: Record<string, FeatureToggle> = {};
      for (const toggle of fetchedTogglesArray) {
        togglesMap[toggle.key] = toggle;
      }
      setToggles(togglesMap);
      localStorage.setItem(FEATURE_TOGGLES_STORAGE_KEY, JSON.stringify(togglesMap));
    } catch (error) {
      console.error("Failed to fetch feature toggles from API:", error);
      // Mantém os toggles do localStorage em caso de erro na API,
      // ou vazio se o localStorage também estava vazio/corrompido.
      // Se isInitialLoad era true e o localStorage estava vazio, isLoadingToggles já é true e será setado para false no finally.
    } finally {
      setIsLoadingToggles(false); // Garantir que o loading termine
    }
  }, []); // Removido 'toggles' da lista de dependências

  useEffect(() => {
    fetchAndSetToggles(true); // Carga inicial
  }, [fetchAndSetToggles]);

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
