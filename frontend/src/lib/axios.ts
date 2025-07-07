import axios from 'axios';

const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Interceptor para adicionar o token JWT às requisições
apiClient.interceptors.request.use(
  (config) => {
    if (typeof window !== 'undefined') {
      const token = localStorage.getItem('authToken');
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Interceptor para lidar com erros de resposta (ex: 401 Unauthorized)
apiClient.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    if (typeof window !== 'undefined' && error.response && error.response.status === 401) {
      // Se receber 401, limpar o token e redirecionar para login
      // Idealmente, isso seria tratado pelo AuthContext/hook de autenticação
      localStorage.removeItem('authToken');
      localStorage.removeItem('authUser'); // Limpar dados do usuário também
      // Evitar redirecionamento direto aqui para não acoplar o client http com a navegação.
      // O AuthContext ou um hook de autenticação deve observar esse erro e redirecionar.
      // Disparar um evento customizado ou retornar um erro específico pode ser uma opção.
      // window.location.href = '/auth/login';
      console.error('Interceptor: Unauthorized (401). Token may be invalid or expired.');
    }
    return Promise.reject(error);
  }
);

export default apiClient;
