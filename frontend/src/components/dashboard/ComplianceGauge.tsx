import { useState, useEffect } from 'react';
import apiClient from '@/lib/axios';
import { useTranslation } from 'next-i18next';
import { RadialBarChart, RadialBar, Legend, ResponsiveContainer, Tooltip } from 'recharts';

type ComplianceData = {
  framework_name: string;
  score: number;
};

const ComplianceGauge = () => {
  const { t } = useTranslation('dashboard');
  const [data, setData] = useState<ComplianceData[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    apiClient.get('/api/v1/dashboard/compliance-overview')
      .then(response => {
        setData(response.data);
        setIsLoading(false);
      })
      .catch(error => {
        console.error("Failed to fetch compliance overview:", error);
        setIsLoading(false);
      });
  }, []);

  const chartData = data.map(item => ({
    name: item.framework_name,
    uv: item.score,
    fill: `hsl(${item.score}, 100%, 40%)`, // Color based on score
  }));

  if (isLoading) {
    return <div className="text-center p-4">{t('compliance_overview.loading')}</div>;
  }

  return (
    <div className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow h-96">
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{t('compliance_overview.title')}</h3>
      <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">{t('compliance_overview.description')}</p>
      <ResponsiveContainer width="100%" height="100%">
        <RadialBarChart
          cx="50%"
          cy="50%"
          innerRadius="10%"
          outerRadius="80%"
          barSize={10}
          data={chartData}
        >
          <RadialBar
            minAngle={15}
            label={{ position: 'insideStart', fill: '#fff' }}
            background
            dataKey='uv'
          />
          <Legend iconSize={10} width={120} height={140} layout='vertical' verticalAlign='middle' wrapperStyle={{ right: 0 }} />
          <Tooltip />
        </RadialBarChart>
      </ResponsiveContainer>
    </div>
  );
};

export default ComplianceGauge;
