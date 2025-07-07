import Head from 'next/head';
import Link from 'next/link';

export default function ForgotPasswordPage() {
  return (
    <>
      <Head>
        <title>Esqueci Minha Senha - Phoenix GRC</title>
      </Head>
      <div className="flex min-h-screen flex-col items-center justify-center bg-gray-100 dark:bg-gray-900">
        <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-xl dark:bg-gray-800">
          <div className="mb-8 text-center">
            <h1 className="text-3xl font-bold text-gray-800 dark:text-white">Recuperar Senha</h1>
            <p className="text-gray-600 dark:text-gray-300">Insira seu email para receber instruções.</p>
          </div>
          {/* Formulário de Esqueci Senha */}
          <form className="space-y-6" onSubmit={(e) => { e.preventDefault(); alert('Funcionalidade a ser implementada!'); }}>
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-gray-700 dark:text-gray-200">
                Endereço de Email
              </label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white sm:text-sm p-2"
                placeholder="voce@example.com"
              />
            </div>
            <div>
              <button
                type="submit"
                className="flex w-full justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800"
              >
                Enviar Link de Recuperação
              </button>
            </div>
          </form>
          <div className="mt-6 text-center text-sm">
            <Link href="/auth/login">
              <span className="font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300">
                Voltar para Login
              </span>
            </Link>
          </div>
        </div>
      </div>
    </>
  );
}
