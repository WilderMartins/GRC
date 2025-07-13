# Estrutura do Projeto Frontend - Phoenix GRC

Este documento descreve a organização das pastas e arquivos principais dentro do diretório `frontend/` do projeto Phoenix GRC.

## Visão Geral

O frontend é construído usando Next.js com TypeScript, estilizado com Tailwind CSS e utiliza `next-i18next` para internacionalização. A estrutura de pastas segue as convenções do Next.js e padrões comuns para organização de projetos React.

## Diretórios Principais

```
frontend/
├── docs/                        # Documentação específica do Frontend (este arquivo)
├── public/                      # Arquivos estáticos servidos publicamente (imagens, fontes, favicons)
├── src/                         # Código fonte principal da aplicação
│   ├── components/              # Componentes React reutilizáveis
│   │   ├── admin/               # Componentes específicos para seções de administração
│   │   │   └── organization/    # Componentes para configuração da organização (ex: IdentityProviderForm, WebhookForm)
│   │   ├── audit/               # Componentes para o módulo de Auditoria (ex: AssessmentFormModal)
│   │   ├── auth/                # Componentes relacionados à autenticação (ex: WithAuth HOC, PasswordStrengthIndicator)
│   │   ├── common/              # Componentes genéricos e utilitários (ex: PaginationControls, StatCard, ThemeSwitcher)
│   │   ├── layouts/             # Componentes de layout de página (ex: AdminLayout, SetupLayout)
│   │   ├── risks/               # Componentes para o módulo de Riscos (ex: RiskForm, ApprovalDecisionModal)
│   │   └── vulnerabilities/     # Componentes para o módulo de Vulnerabilidades (ex: VulnerabilityForm)
│   ├── contexts/                # Contextos React para gerenciamento de estado global (ex: AuthContext, ThemeContext)
│   ├── hooks/                   # Hooks React customizados (ex: useNotifier, useDebounce)
│   ├── lib/                     # Bibliotecas, helpers e configurações
│   │   ├── axios.ts             # Configuração da instância do Axios para chamadas API
│   │   └── i18n/                # Configuração do i18n (ex: i18n-test.config.ts) - pode variar
│   ├── pages/                   # Rotas da aplicação (convenção do Next.js)
│   │   ├── admin/               # Páginas da área administrativa
│   │   │   ├── audit/
│   │   │   ├── organization/
│   │   │   ├── risks/
│   │   │   └── vulnerabilities/
│   │   ├── api/                 # Rotas de API do Next.js (se houver backend no frontend)
│   │   ├── auth/                # Páginas de autenticação (login, registro, etc.)
│   │   ├── user/                # Páginas de perfil do usuário (ex: security.tsx)
│   │   ├── _app.tsx             # Componente App principal do Next.js
│   │   ├── _document.tsx        # Documento HTML customizado do Next.js (opcional)
│   │   └── index.tsx            # Página inicial da aplicação
│   ├── styles/                  # Arquivos de estilo globais (ex: globals.css)
│   └── types/                   # Definições de tipos TypeScript (enums.ts, models.ts, api.ts)
├── .env.example                 # Exemplo de variáveis de ambiente para o frontend
├── .eslintrc.json               # Configuração do ESLint
├── .gitignore                   # Arquivos ignorados pelo Git
├── .prettierrc.json             # Configuração do Prettier
├── jest.config.js               # Configuração do Jest para testes
├── jest.setup.ts                # Arquivo de setup para Jest (ex: mocks globais, polyfills)
├── next-i18next.config.js       # Configuração do next-i18next
├── next.config.js               # Configuração do Next.js
├── package.json                 # Dependências e scripts do projeto
├── postcss.config.js            # Configuração do PostCSS (usado pelo Tailwind CSS)
├── tailwind.config.js           # Configuração do Tailwind CSS
└── tsconfig.json                # Configuração do TypeScript
```

## Descrição dos Diretórios em `src/`

*   **`src/components/`**: Contém todos os componentes React. A intenção é que sejam o mais reutilizáveis possível.
    *   **`admin/`**: Componentes que são usados primariamente dentro das seções de administração, muitas vezes específicos para um módulo (como `organization/`).
    *   **`audit/`**, **`auth/`**, **`risks/`**, **`vulnerabilities/`**: Componentes específicos para seus respectivos módulos.
    *   **`common/`**: Componentes de UI genéricos que podem ser usados em qualquer parte da aplicação (ex: botões customizados, modais genéricos, controles de paginação).
    *   **`layouts/`**: Componentes que definem a estrutura visual principal das páginas (ex: sidebar, header, footer para a área administrativa ou para o fluxo de setup).
*   **`src/contexts/`**: Provedores de Contexto React para gerenciar o estado global que precisa ser compartilhado entre diferentes partes da aplicação.
    *   `AuthContext.tsx`: Gerencia o estado de autenticação do usuário, informações do usuário, token JWT e configurações de branding da organização.
    *   `ThemeContext.tsx`: Gerencia o tema da aplicação (ex: light/dark).
    *   `FeatureToggleContext.tsx`: Gerencia feature flags/toggles.
*   **`src/hooks/`**: Hooks React customizados para encapsular lógica reutilizável e efeitos colaterais.
    *   `useNotifier.ts`: Hook para exibir notificações (toasts) de forma padronizada.
    *   `useDebounce.ts`: Hook para adicionar debounce a inputs (ex: em campos de busca).
*   **`src/lib/`**: Utilitários, configurações de bibliotecas e lógica não-React.
    *   `axios.ts`: Configuração centralizada da instância do Axios, incluindo interceptors para adicionar o token JWT e para tratamento global de erros (como 401).
    *   `i18n/`: Arquivos de configuração para internacionalização.
*   **`src/pages/`**: Cada arquivo `.tsx` (ou pasta com `index.tsx`) neste diretório se torna uma rota na aplicação, conforme a convenção do Next.js.
    *   A estrutura de subpastas (ex: `admin/risks`) reflete a estrutura de URLs.
    *   `_app.tsx` é o componente raiz que inicializa todas as páginas. É onde os providers de contexto globais são aplicados.
*   **`src/styles/`**: Arquivos CSS globais. `globals.css` é usado para estilos base e para as camadas do Tailwind CSS.
*   **`src/types/`**: Contém todas as definições de interface e tipo do TypeScript usadas no projeto, ajudando a manter a consistência e a robustez do código.
    *   `enums.ts`: Definições de enums.
    *   `models.ts`: Interfaces para os modelos de dados principais da aplicação (User, Risk, Vulnerability, etc.).
    *   `api.ts`: Tipos relacionados a payloads e respostas de API.

## Próximos Passos na Documentação

1.  Documentar os principais componentes reutilizáveis (props, uso).
2.  Detalhar as decisões de arquitetura (gerenciamento de estado, chamadas API, autenticação).
3.  Criar guias para fluxos de usuário importantes.
4.  Adicionar instruções sobre como rodar e testar o frontend.
5.  Definir um guia de estilo de código para contribuições.

Este documento serve como ponto de partida para entender a organização do código frontend.
