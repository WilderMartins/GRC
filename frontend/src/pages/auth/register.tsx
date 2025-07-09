import Head from 'next/head';
import Link from 'next/link';
import { useState } from 'react';
import apiClient from '@/lib/axios'; // Ajuste o path se necessário
import { useNotifier } from '@/hooks/useNotifier'; // Importar o hook

export default function RegisterPage() {
  const notify = useNotifier();
  const [userName, setUserName] = useState('');
  const [userEmail, setUserEmail] = useState('');
  const [userPassword, setUserPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [organizationName, setOrganizationName] = useState('');

  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null); // Para erros de validação em tela
  const [isSuccess, setIsSuccess] = useState(false); // Para controlar estado do form após sucesso

  // Basic password strength check (example)
  const isPasswordStrong = (password: string): boolean => {
    // TODO: Melhorar a validação de força da senha (ex: regex para maiúscula, minúscula, número, símbolo)
    if (password.length < 8) {
      setError('A senha deve ter pelo menos 8 caracteres.');
      return false;
    }
    // Exemplo de verificação adicional (pode ser expandido)
    // if (!/[A-Z]/.test(password)) {
    //   setError('A senha deve conter pelo menos uma letra maiúscula.');
    //   return false;
    // }
    // if (!/[0-9]/.test(password)) {
    //   setError('A senha deve conter pelo menos um número.');
    //   return false;
    // }
    return true;
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null); // Limpar erros de validação anteriores
    setIsLoading(true);
    // setIsSuccess(false); // Não precisa resetar success aqui, o form já estará desabilitado

    if (!userName || !userEmail || !userPassword || !confirmPassword || !organizationName) {
      setError('Todos os campos obrigatórios devem ser preenchidos.');
      setIsLoading(false);
      return;
    }
    if (userPassword !== confirmPassword) {
      setError('As senhas não conferem.');
      setIsLoading(false);
      return;
    }
    if (!isPasswordStrong(userPassword)) {
      // A mensagem de erro já é setada dentro de isPasswordStrong
      setIsLoading(false);
      return;
    }

    const payload = {
      user: {
        name: userName,
        email: userEmail,
        password: userPassword,
      },
      organization: {
        name: organizationName,
      },
    };

    try {
      const response = await apiClient.post('/auth/register', payload);
      notify.success(response.data?.message || 'Registro bem-sucedido! Verifique seu email para confirmação.');
      setIsSuccess(true); // Desabilita o formulário e mostra mensagem em tela
      // Não limpar formulário imediatamente para o usuário ver os dados enviados se desejar,
      // mas os campos estarão desabilitados.
    } catch (err: any) {
      console.error('Erro no registro:', err);
      notify.error(err.response?.data?.error || 'Ocorreu um erro durante o registro. Tente novamente.');
      // setError(err.response?.data?.error || 'Ocorreu um erro...'); // Opcional, para erro em tela
    } finally {
      setIsLoading(false);
    }
  };


  return (
    <>
      <Head>
        <title>Registrar - Phoenix GRC</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900 py-12">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-8 text-center">
            {/* Placeholder para Logo */}
            <div className="mb-4 inline-block rounded-full bg-indigo-500 p-3 text-white">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="h-8 w-8">
                    <path d="M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Zm4.28 13.43a.75.75 0 0 1-.976.02l-3.573-2.68a.75.75 0 0 0-.976 0l-3.573 2.68a.75.75 0 0 1-.976-.02l-1.141-.856a.75.75 0 0 1 .02-1.263l2.68-2.01a.75.75 0 0 0 0-1.264l-2.68-2.01a.75.75 0 0 1-.02-1.263l1.141-.856a.75.75 0 0 1 .976.02l3.573 2.68a.75.75 0 0 0 .976 0l3.573 2.68a.75.75 0 0 1 .976.02l1.141.856a.75.75 0 0 1-.02 1.263l-2.68 2.01a.75.75 0 0 0 0 1.264l2.68 2.01a.75.75 0 0 1 .02 1.263l-1.141-.856Z" />
                </svg>
            </div>
            <h1 className="text-3xl font-bold text-gray-800 dark:text-white">Criar Nova Conta</h1>
            <p className="text-gray-600 dark:text-gray-300">Junte-se ao Phoenix GRC.</p>
          </div>

          {error && ( // Erro de validação de campo
            <div className="mb-4 rounded-md bg-red-50 p-3">
              <p className="text-sm font-medium text-red-700">{error}</p>
            </div>
          )}
          {isSuccess && ( // Mensagem em tela após sucesso, complementando o toast
            <div className="mb-4 rounded-md bg-green-50 p-3">
              <p className="text-sm font-medium text-green-700">Registro enviado! Verifique seu email para confirmação.</p>
            </div>
          )}

          <form className="space-y-4" onSubmit={handleSubmit}>
            <div>
              <label htmlFor="userName" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Seu Nome Completo
              </label>
              <input id="userName" name="userName" type="text" autoComplete="name" required
                     value={userName} onChange={(e) => setUserName(e.target.value)}
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder="Seu Nome" disabled={isLoading || !!successMessage} />
            </div>
            <div>
              <label htmlFor="userEmail" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Seu Email Principal
              </label>
              <input id="userEmail" name="userEmail" type="email" autoComplete="email" required
                     value={userEmail} onChange={(e) => setUserEmail(e.target.value)}
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder="voce@example.com" disabled={isLoading || !!successMessage} />
            </div>
            <div>
              <label htmlFor="organizationName" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Nome da Organização/Empresa
              </label>
              <input id="organizationName" name="organizationName" type="text" required
                     value={organizationName} onChange={(e) => setOrganizationName(e.target.value)}
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder="Nome da Sua Empresa" disabled={isLoading || !!successMessage} />
            </div>
            <div>
              <label htmlFor="userPassword" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Crie uma Senha
              </label>
              <input id="userPassword" name="userPassword" type="password" autoComplete="new-password" required
                     value={userPassword} onChange={(e) => setUserPassword(e.target.value)}
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder="Mínimo 8 caracteres" disabled={isLoading || !!successMessage} />
            </div>
            <div>
              <label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Confirme Sua Senha
              </label>
              <input id="confirmPassword" name="confirmPassword" type="password" autoComplete="new-password" required
                     value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)}
                     className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                     placeholder="Repita a senha" disabled={isLoading || !!successMessage} />
            </div>

            <div>
              <button type="submit" disabled={isLoading || !!successMessage}
                      className="flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50 dark:focus:ring-offset-gray-800">
                {isLoading ? (
                  <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                ) : ( 'Criar Conta' )}
              </button>
            </div>
          </form>
          <div className="mt-6 text-center text-sm">
            <p className="text-gray-600 dark:text-gray-400">
              Já tem uma conta?{' '}
              <Link href="/auth/login">
                <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                  Faça login aqui
                </span>
              </Link>
            </p>
          </div>
        </div>
      </div>
    </>
  );
}
