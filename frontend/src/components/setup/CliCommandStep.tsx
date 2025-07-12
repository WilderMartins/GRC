import React from 'react';
import { useTranslation } from 'next-i18next';

interface CliCommandStepProps {
  onNext: () => void;
}

const CliCommandStep: React.FC<CliCommandStepProps> = ({ onNext }) => {
  const { t } = useTranslation('setupWizard');

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-2xl font-bold text-gray-900 dark:text-white text-center">
          {t('steps.cli_command.title', 'Executar o Setup via Terminal')}
        </h3>
        <p className="mt-3 text-sm text-gray-600 dark:text-gray-300">
          {t('steps.cli_command.intro_paragraph', 'Agora que o ambiente está configurado, o próximo passo é executar o script de setup. Este script irá criar as tabelas no banco de dados e o primeiro usuário administrador.')}
        </p>
      </div>

      <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg space-y-3">
        <p className="text-sm text-gray-700 dark:text-gray-200">
          {t('steps.cli_command.instructions_1', 'Abra um novo terminal na pasta raiz do projeto (onde o arquivo `docker-compose.yml` está localizado).')}
        </p>
        <p className="text-sm font-semibold text-gray-800 dark:text-white">
          {t('steps.cli_command.instructions_2', 'Execute o seguinte comando:')}
        </p>
        <div className="p-3 bg-gray-900 text-white font-mono rounded-md text-sm overflow-x-auto">
          <code>
            docker-compose run --rm backend setup
          </code>
        </div>
         <p className="text-sm text-gray-700 dark:text-gray-200">
          {t('steps.cli_command.instructions_3', 'O terminal solicitará que você insira o nome da sua organização e as credenciais (nome, email, senha) para a conta de administrador. Siga as instruções que aparecerem.')}
        </p>
      </div>

      <div className="bg-blue-50 dark:bg-blue-700/30 border-l-4 border-blue-400 dark:border-blue-500 p-4 rounded-md">
        <div className="flex">
          <div className="flex-shrink-0">
            <svg className="h-5 w-5 text-blue-400 dark:text-blue-300" viewBox="0 0 20 20" fill="currentColor">
               <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z" clipRule="evenodd" />
            </svg>
          </div>
          <div className="ml-3">
            <p className="text-sm text-blue-700 dark:text-blue-200">
              {t('steps.cli_command.tip', 'Este comando executa o processo de setup em um container temporário. Após a conclusão, o container será removido automaticamente.')}
            </p>
          </div>
        </div>
      </div>

      <div>
        <button
          type="button"
          onClick={onNext}
          className="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-brand-primary hover:bg-brand-primary/90 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-primary transition-colors"
        >
          {t('steps.cli_command.next_button', 'Eu executei o comando e o setup foi concluído. Próximo Passo')}
        </button>
      </div>
    </div>
  );
};

export default CliCommandStep;
