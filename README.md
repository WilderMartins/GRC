# Phoenix GRC - Instalação Rápida

Bem-vindo ao Phoenix GRC! Este guia mostra como instalar a plataforma de forma simples e rápida, usando Docker.

## O que é o Phoenix GRC?

O Phoenix GRC é uma plataforma de Gestão de Governança, Risco e Conformidade de TI. Ele ajuda equipes a gerenciar riscos, responder a vulnerabilidades e demonstrar conformidade de forma eficiente.

---

## Processo de Instalação Automatizado

O processo de instalação foi totalmente automatizado com Docker. Você não precisa instalar `Node.js` ou `Go` em sua máquina.

### Pré-requisitos

Você só precisa de duas ferramentas instaladas e em execução:

1.  **Git:** Para baixar o código-fonte.
2.  **Docker:** Para construir e rodar a aplicação.

Se você não os tiver, siga nosso **[Guia de Preparação do Ambiente](./PREPARACAO_AMBIENTE.md)**.

### Passo a Passo (2 minutos)

Com o Git e o Docker prontos, abra seu terminal (ou Prompt de Comando/PowerShell) e siga os comandos abaixo.

1.  **Baixe (clone) o projeto do repositório:**
    ```bash
    git clone https://github.com/SEU_USUARIO/phoenix-grc.git
    ```
    *Lembre-se de usar a URL correta do seu repositório.*

2.  **Entre na pasta do projeto:**
    ```bash
    cd phoenix-grc
    ```

3.  **Crie o arquivo de configuração (apenas copie, não precisa editar):**
    ```bash
    cp .env.example .env
    ```

4.  **Inicie a Aplicação com Docker Compose:**
    > **Importante:** Certifique-se de que o aplicativo **Docker Desktop** está aberto e em execução antes de rodar o comando.
    ```bash
    docker-compose up -d --build
    ```
    Este único comando irá:
    - Construir a imagem do backend (Go).
    - Construir a imagem do frontend (Next.js), incluindo a instalação de dependências `npm` e o `build`.
    - Iniciar todos os serviços (backend, frontend, banco de dados e Nginx).
    - O processo pode demorar alguns minutos na primeira vez.

### Acesse o Wizard de Instalação

Após a conclusão do comando, a plataforma estará rodando. Abra seu navegador e acesse:

➡️ **http://localhost**

Você será recebido pelo nosso **Wizard de Instalação Visual**, que o guiará na configuração inicial da sua organização e da conta de administrador.

---

## Para Desenvolvedores

Se você é um desenvolvedor e deseja entender a arquitetura, as tecnologias utilizadas e como contribuir, consulte nosso **[Guia do Desenvolvedor](./DEVELOPER_GUIDE.md)**.
