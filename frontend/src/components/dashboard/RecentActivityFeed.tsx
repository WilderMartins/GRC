import { useState, useEffect } from 'react';
import apiClient from '@/lib/axios';
import { useTranslation } from 'next-i18next';
import Link from 'next/link';
import { formatDistanceToNow } from 'date-fns';
import { ptBR } from 'date-fns/locale';

type Activity = {
  type: string;
  title: string;
  timestamp: string;
  link: string;
};

const RecentActivityFeed = () => {
  const { t } = useTranslation('dashboard');
  const [activities, setActivities] = useState<Activity[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    apiClient.get('/api/v1/dashboard/recent-activity')
      .then(response => {
        setActivities(response.data);
        setIsLoading(false);
      })
      .catch(error => {
        console.error("Failed to fetch recent activity:", error);
        setIsLoading(false);
      });
  }, []);

  if (isLoading) {
    return <div>{t('recent_activity.loading')}</div>;
  }

  return (
    <div className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow">
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{t('recent_activity.title')}</h3>
      <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">{t('recent_activity.description')}</p>
      <ul className="divide-y dark:divide-gray-700">
        {activities.map((activity, index) => (
          <li key={index} className="py-2">
            <Link href={activity.link}>
              <a className="hover:underline">
                <p className="font-semibold">{activity.type}: {activity.title}</p>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {formatDistanceToNow(new Date(activity.timestamp), { addSuffix: true, locale: ptBR })}
                </p>
              </a>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default RecentActivityFeed;
