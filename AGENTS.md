## Instruções para Agentes de IA

Olá! Sou o Jules, um engenheiro de software de IA. Este arquivo contém algumas diretrizes e informações para me ajudar (e outros agentes de IA) a trabalhar neste projeto de forma eficaz.

### Stack Tecnológica Principal

*   **Backend:** Go (Golang)
    *   ORM: GORM
    *   Framework Web (a ser definido, possivelmente Gin ou net/http padrão)
*   **Frontend:** Next.js com TypeScript
    *   Estilização: Tailwind CSS
*   **Banco de Dados:** PostgreSQL 16
*   **Containerização:** Docker, Docker Compose

### Convenções Gerais

1.  **Commits:** Siga o padrão Conventional Commits ([https://www.conventionalcommits.org/](https://www.conventionalcommits.org/)).
    *   Exemplo: `feat: adiciona módulo de autenticação de usuários`
    *   Exemplo: `fix: corrige cálculo de nível de risco`
    *   Exemplo: `docs: atualiza README com instruções de setup`
2.  **Branches:** Use nomes de branch descritivos, prefixados com o tipo de trabalho.
    *   Exemplo: `feat/user-authentication`
    *   Exemplo: `fix/risk-calculation-bug`
    *   Exemplo: `docs/update-readme`
3.  **Código Go:**
    *   Formate o código com `go fmt` antes de commitar.
    *   Siga as diretrizes de Go Efetivo ([https://go.dev/doc/effective_go](https://go.dev/doc/effective_go)).
    *   Organize o código em pacotes lógicos (ex: `internal/models`, `internal/handlers`, `internal/services`).
4.  **Código Frontend (Next.js/TypeScript):**
    *   Use Prettier e ESLint para formatação e linting (configurações a serem adicionadas).
    *   Siga as melhores práticas para React e Next.js.
    *   Componentize a UI de forma clara.
5.  **Testes:** Escreva testes unitários e de integração sempre que possível.
    *   Testes Go no mesmo pacote com sufixo `_test.go`.
    *   Testes frontend usando Jest/React Testing Library (a ser configurado).
6.  **Documentação:** Mantenha o `README.md` atualizado com instruções de setup e informações relevantes do projeto. Comente o código onde necessário.

### Fluxo de Trabalho (Proposto)

1.  **Planejamento:** Antes de iniciar uma tarefa complexa, detalhe o plano usando a ferramenta `set_plan`.
2.  **Desenvolvimento Iterativo:** Implemente funcionalidades em pequenos passos.
3.  **Testes:** Escreva testes para as funcionalidades implementadas.
4.  **Revisão (Simulada):** Revise o código e os testes.
5.  **Commit e Push:** Faça o commit das alterações com mensagens claras.

### Considerações Específicas do Projeto

*   **Microsserviços:** A arquitetura final visa microsserviços. Inicialmente, podemos começar com um backend monolítico e refatorar posteriormente. Tenha isso em mente ao estruturar o código.
*   **Segurança:** A segurança é crucial.
    *   Valide todas as entradas do usuário.
    *   Use senhas hasheadas (bcrypt já está em uso para o setup inicial).
    *   Prepare para JWT, SSO SAML, OAuth2.
    *   Evite vulnerabilidades comuns (SQL Injection, XSS, etc.).
*   **Whitelabeling:** O design deve permitir fácil customização de logo e cores.

### Se Precisar de Ajuda

*   Se uma solicitação for ambígua, peça esclarecimentos.
*   Se encontrar um bloqueio técnico significativo, descreva o problema e as tentativas de solução.

Obrigado por colaborar com o Phoenix GRC!
