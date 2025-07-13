import { useState, useEffect } from 'react';
import apiClient from '@/lib/axios';
import { useTranslation } from 'next-i18next';
import { ResponsiveContainer, ScatterChart, XAxis, YAxis, ZAxis, CartesianGrid, Tooltip, Legend, Scatter } from 'recharts';

type RiskMatrixData = {
  probability: string;
  impact: string;
  count: number;
};

const RiskMatrixChart = () => {
  const { t } = useTranslation('dashboard');
  const [data, setData] = useState<RiskMatrixData[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    apiClient.get('/api/v1/dashboard/risk-matrix')
      .then(response => {
        setData(response.data);
        setIsLoading(false);
      })
      .catch(error => {
        console.error("Failed to fetch risk matrix data:", error);
        setIsLoading(false);
      });
  }, []);

  const probabilityMap: { [key: string]: number } = { 'Baixo': 1, 'Médio': 2, 'Alto': 3, 'Crítico': 4 };
  const impactMap: { [key: string]: number } = { 'Baixo': 1, 'Médio': 2, 'Alto': 3, 'Crítico': 4 };

  const chartData = data.map(item => ({
    x: impactMap[item.impact],
    y: probabilityMap[item.probability],
    z: item.count * 100, // Scale factor for bubble size
    count: item.count,
  }));

  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className="bg-white dark:bg-gray-700 p-2 border rounded shadow">
          <p>{`${t('risk_matrix.impact')}: ${Object.keys(impactMap).find(key => impactMap[key] === data.x)}`}</p>
          <p>{`${t('risk_matrix.probability')}: ${Object.keys(probabilityMap).find(key => probabilityMap[key] === data.y)}`}</p>
          <p>{`${t('risk_matrix.count')}: ${data.count}`}</p>
        </div>
      );
    }
    return null;
  };

  if (isLoading) {
    return <div className="text-center p-4">{t('risk_matrix.loading')}</div>;
  }

  return (
    <div className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow h-96">
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{t('risk_matrix.title')}</h3>
      <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">{t('risk_matrix.description')}</p>
      <ResponsiveContainer width="100%" height="100%">
        <ScatterChart margin={{ top: 20, right: 20, bottom: 20, left: 20 }}>
          <CartesianGrid />
          <XAxis type="number" dataKey="x" name={t('risk_matrix.impact')} ticks={[1, 2, 3, 4]} tickFormatter={val => Object.keys(impactMap).find(key => impactMap[key] === val) || ''} />
          <YAxis type="number" dataKey="y" name={t('risk_matrix.probability')} ticks={[1, 2, 3, 4]} tickFormatter={val => Object.keys(probabilityMap).find(key => probabilityMap[key] === val) || ''} />
          <ZAxis type="number" dataKey="z" range={[100, 1000]} name={t('risk_matrix.count')} />
          <Tooltip content={<CustomTooltip />} cursor={{ strokeDasharray: '3 3' }} />
          <Legend />
          <Scatter name={t('risk_matrix.risks')} data={chartData} fill="#8884d8" />
        </ScatterChart>
      </ResponsiveContainer>
    </div>
  );
};

export default RiskMatrixChart;
