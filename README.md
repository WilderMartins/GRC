# Phoenix GRC - Sistema de Gestão de Governança, Risco e Conformidade de TI

Phoenix GRC é uma plataforma SaaS (Software as a Service), whitelabel, projetada para ser intuitiva, escalável e segura. O objetivo é permitir que equipes de TI e segurança de todos os tamanhos gerenciem riscos, respondam a vulnerabilidades e demonstrem conformidade com os principais frameworks do setor de forma eficiente.

## Stack Tecnológica

- **Backend:** Go (Golang)
- **Frontend:** Next.js com TypeScript
- **Banco de Dados:** PostgreSQL 16
- **ORM (Go):** GORM
- **UI (Frontend):** Tailwind CSS
- **Containerização:** Docker

## Ambiente de Desenvolvimento com Docker

Este projeto utiliza Docker e Docker Compose para facilitar a configuração do ambiente de desenvolvimento.

### Pré-requisitos

- Docker: [Instruções de Instalação](https://docs.docker.com/get-docker/)
- Docker Compose: (Normalmente incluído com Docker Desktop) [Instruções de Instalação](https://docs.docker.com/compose/install/)

### Configuração Inicial

1.  **Clone o Repositório:**
    ```bash
    git clone <url_do_repositorio>
    cd phoenix-grc # Ou o nome do diretório do projeto
    ```

2.  **Configure as Variáveis de Ambiente:**
    Copie o arquivo de exemplo `.env.example` para `.env` e ajuste as configurações do banco de dados conforme necessário.
    ```bash
    cp .env.example .env
    ```
    Edite o arquivo `.env` com suas preferências (usuário, senha, nome do banco de dados, porta do host para o PostgreSQL).

3.  **Construa e Inicie os Contêineres:**
    Este comando irá construir a imagem Docker para o backend (se ainda não construída) e iniciar os serviços do backend e do banco de dados.
    ```bash
    docker-compose up --build
    ```
    O `--build` força a reconstrução da imagem do backend, útil se você fez alterações no `Dockerfile.backend` ou no código Go antes da primeira instalação.

### Executando o Script de Instalação Interativo

Na primeira vez que você executar `docker-compose up`, o serviço `backend` irá automaticamente iniciar o script de instalação interativo no terminal. Siga as instruções para:

1.  **Configurar a Conexão com o Banco de Dados:**
    *   **Database Host:** `db` (este é o nome do serviço do PostgreSQL no `docker-compose.yml`)
    *   **Database Port:** `5432` (porta padrão do PostgreSQL dentro da rede Docker)
    *   **Database User:** O valor de `POSTGRES_USER` que você definiu no seu arquivo `.env` (padrão: `admin`).
    *   **Database Password:** O valor de `POSTGRES_PASSWORD` que você definiu no seu arquivo `.env` (padrão: `password123`).
    *   **Database Name:** O valor de `POSTGRES_DB` que você definiu no seu arquivo `.env` (padrão: `phoenix_grc_dev`).
    *   **Database SSL Mode:** `disable` (para desenvolvimento local na rede Docker).

2.  **Criar a Primeira Organização:**
    *   Forneça um nome para a organização inicial.

3.  **Criar o Usuário Administrador:**
    *   Forneça nome, email e senha para o primeiro usuário com perfil de administrador.

Após o script de instalação ser concluído com sucesso, a aplicação base estará configurada. O contêiner do `backend` irá finalizar, pois sua tarefa (`CMD ["./server"]` no `Dockerfile.backend`) é executar o script de setup.

**Próximos Passos (Desenvolvimento Futuro):**

*   Para desenvolvimento contínuo do servidor backend, o `CMD` no `Dockerfile.backend` precisará ser alterado para iniciar o servidor HTTP Go (após sua implementação).
*   O frontend Next.js precisará ser configurado e iniciado separadamente ou adicionado ao `docker-compose.yml`.

## Estrutura do Projeto (Inicial)

```
.
├── .env.example            # Exemplo de variáveis de ambiente
├── Dockerfile.backend      # Dockerfile para a aplicação Go (backend)
├── README.md               # Este arquivo
├── AGENTS.md               # Instruções para agentes de IA (como você!)
├── backend/                # Código fonte do backend (Go)
│   ├── cmd/server/main.go  # Ponto de entrada (atualmente o script de setup)
│   ├── internal/           # Código interno da aplicação
│   │   ├── database/       # Lógica de conexão e migração com DB
│   │   ├── models/         # Structs GORM (schema do DB)
│   │   └── ...             # Outros pacotes (handlers, services, etc.)
│   ├── pkg/                # Pacotes reutilizáveis
│   ├── go.mod              # Módulos Go
│   └── go.sum
├── frontend/               # Código fonte do frontend (Next.js)
│   ├── src/                # Código fonte Next.js
│   ├── package.json
│   ├── next.config.js
│   ├── tsconfig.json
│   └── ...
└── docker-compose.yml      # Orquestração dos contêineres Docker
```

## Contribuindo

Detalhes sobre como contribuir serão adicionados futuramente.
