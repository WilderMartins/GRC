import React from 'react';
import Link from 'next/link';

interface StatCardProps {
  title: string;
  value: number | string;
  isLoading: boolean;
  linkTo?: string;
  error?: string | null;
  className?: string; // Permite passar classes customizadas para o wrapper do card
}

const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  isLoading,
  linkTo,
  error,
  className = ''
}) => {
  const cardContent = (
    <>
      <h2 className="text-xl font-semibold text-gray-700 dark:text-white mb-2 truncate" title={title}>{title}</h2>
      {isLoading && <p className="text-2xl font-bold text-gray-500 dark:text-gray-400 animate-pulse">Carregando...</p>}
      {error && !isLoading && <p className="text-sm font-bold text-red-500 dark:text-red-400 h-8 flex items-center justify-center">{error}</p>}
      {!isLoading && !error && <p className="text-3xl font-bold text-indigo-600 dark:text-indigo-400 h-8">{value}</p>}
    </>
  );

  const baseClasses = "bg-white dark:bg-gray-800 p-6 rounded-lg shadow-md";
  const hoverClasses = linkTo ? "hover:shadow-lg transition-shadow duration-150" : "";

  if (linkTo) {
    return (
      <Link href={linkTo} legacyBehavior>
        <a className={`${baseClasses} ${hoverClasses} ${className} block`}>
          {cardContent}
        </a>
      </Link>
    );
  }
  return (
    <div className={`${baseClasses} ${className}`}>
      {cardContent}
    </div>
  );
};

export default StatCard;
