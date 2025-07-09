import { toast, ToastOptions, Id } from 'react-toastify';

// Opções padrão para os toasts, podem ser customizadas aqui
const defaultOptions: ToastOptions = {
  position: "top-right",
  autoClose: 5000,
  hideProgressBar: false,
  closeOnClick: true,
  pauseOnHover: true,
  draggable: true,
  progress: undefined,
  // theme: "light", // O tema é globalmente definido no ToastContainer, mas pode ser sobrescrito por toast
};

interface Notifier {
  success: (message: string, options?: ToastOptions) => Id;
  error: (message: string, options?: ToastOptions) => Id;
  info: (message: string, options?: ToastOptions) => Id;
  warn: (message: string, options?: ToastOptions) => Id;
  loading: (message: string, options?: ToastOptions) => Id;
  dismiss: (toastId?: Id) => void;
  update: (toastId: Id, message: string, options?: ToastOptions & { type?: 'success' | 'error' | 'info' | 'warning' | 'default' | 'loading' }) => void;
}

/**
 * Hook customizado para exibir notificações (toasts) de forma padronizada.
 * Utiliza react-toastify por baixo dos panos.
 *
 * @example
 * const notify = useNotifier();
 * notify.success("Operação realizada com sucesso!");
 * notify.error("Ocorreu um erro.");
 */
export function useNotifier(): Notifier {
  const success = (message: string, options: ToastOptions = {}): Id => {
    return toast.success(message, { ...defaultOptions, ...options });
  };

  const error = (message: string, options: ToastOptions = {}): Id => {
    return toast.error(message, { ...defaultOptions, ...options });
  };

  const info = (message: string, options: ToastOptions = {}): Id => {
    return toast.info(message, { ...defaultOptions, ...options });
  };

  const warn = (message: string, options: ToastOptions = {}): Id => {
    return toast.warn(message, { ...defaultOptions, ...options });
  };

  const loading = (message: string, options: ToastOptions = {}): Id => {
    return toast.loading(message, { ...defaultOptions, ...options });
  };

  const dismiss = (toastId?: Id): void => {
    if (toastId) {
      toast.dismiss(toastId);
    } else {
      toast.dismiss(); // Fecha todos os toasts se nenhum ID for fornecido
    }
  };

  const update = (
    toastId: Id,
    message: string,
    options?: ToastOptions & { type?: 'success' | 'error' | 'info' | 'warning' | 'default' | 'loading' }
  ): void => {
    toast.update(toastId, {
      render: message,
      ...defaultOptions,
      ...options,
      isLoading: options?.type === 'loading', // Controla o spinner para o tipo 'loading'
    });
  };


  return {
    success,
    error,
    info,
    warn,
    loading,
    dismiss,
    update,
  };
}
