# Guia de Preparação do Ambiente

Bem-vindo! Este guia foi feito para ajudar você a instalar as ferramentas necessárias para rodar o Phoenix GRC. O processo é simples, e vamos guiá-lo passo a passo. Precisamos de apenas duas ferramentas: **Git** e **Docker**.

---

## O que são essas ferramentas?

*   **Git:** É como um "salvamento" para códigos. Usaremos ele uma única vez para baixar o código do Phoenix GRC para o seu computador.
*   **Docker:** É como uma "caixa mágica" que contém tudo o que o Phoenix GRC precisa para rodar, sem que você precise instalar coisas complicadas como banco de dados ou servidores web manualmente.

---

## Instalação no Windows

### 1. Instalando o Git

A forma mais fácil de instalar o Git no Windows é usando o **Git for Windows**.

1.  **Acesse o site oficial:**
    *   [Clique aqui para baixar o Git for Windows](https://git-scm.com/download/win)
    *   O download deve começar automaticamente.

2.  **Instale o programa:**
    *   Abra o arquivo que você baixou.
    *   Você pode clicar em "Next" em todas as telas, mantendo as opções padrão. Não é necessário alterar nada. A instalação é longa, mas basta aceitar os padrões.

    ![Instalador do Git no Windows](docs/images/windows_git_installer.png "Instalador do Git no Windows")

3.  **Verifique a instalação:**
    *   Após a instalação, abra o "Prompt de Comando" (você pode pesquisar por `cmd` no menu Iniciar).
    *   Digite o comando `git --version` e pressione Enter. Se aparecer uma versão (ex: `git version 2.39.1.windows.1`), está tudo certo!

    ![Verificação do Git no Prompt de Comando](docs/images/windows_git_check.png "Verificação do Git no Prompt de Comando")

### 2. Instalando o Docker Desktop

1.  **Acesse o site oficial:**
    *   [Clique aqui para baixar o Docker Desktop for Windows](https://www.docker.com/products/docker-desktop/)
    *   Clique no botão de download para Windows.

2.  **Instale o programa:**
    *   Abra o arquivo que você baixou.
    *   Mantenha a opção "Use WSL 2 instead of Hyper-V" (ou similar) marcada se ela aparecer.
    *   Siga as instruções na tela. Pode ser necessário reiniciar o computador durante o processo.

    ![Instalador do Docker no Windows](docs/images/windows_docker_installer.png "Instalador do Docker no Windows")

3.  **Inicie o Docker Desktop:**
    *   Após a instalação, procure por "Docker Desktop" no seu menu Iniciar e abra-o.
    *   Aguarde o ícone da baleia do Docker aparecer na sua barra de tarefas (perto do relógio). Ele precisa estar estável (sem animações).
    *   Pode aparecer um tutorial inicial. Você pode pulá-lo.

    ![Ícone do Docker na barra de tarefas](docs/images/windows_docker_icon.png "Ícone do Docker na barra de tarefas")

---

## Instalação no macOS

### 1. Instalando o Git

O Git geralmente já vem instalado no macOS.

1.  **Verifique a instalação:**
    *   Abra o aplicativo "Terminal" (você pode encontrá-lo em `Aplicativos/Utilitários` ou pesquisar no Spotlight).
    *   Digite `git --version` e pressione Enter.
    *   Se o Git não estiver instalado, o próprio sistema irá pedir para você instalar as "Command Line Developer Tools". Apenas confirme e siga os passos.

    ![Verificação do Git no Terminal do macOS](docs/images/macos_git_check.png "Verificação do Git no Terminal do macOS")

### 2. Instalando o Docker Desktop

1.  **Acesse o site oficial:**
    *   [Clique aqui para baixar o Docker Desktop for Mac](https://www.docker.com/products/docker-desktop/)
    *   Clique no botão de download para Mac (escolha "Mac with Intel chip" ou "Mac with Apple chip" de acordo com o seu modelo).

2.  **Instale o programa:**
    *   Abra o arquivo `.dmg` que você baixou.
    *   Arraste o ícone do Docker para a sua pasta de Aplicativos.

    ![Instalação do Docker no macOS](docs/images/macos_docker_installer.png "Instalação do Docker no macOS")

3.  **Inicie o Docker Desktop:**
    *   Vá até a sua pasta de Aplicativos e abra o "Docker".
    *   Você precisará dar permissão para ele rodar.
    *   Aguarde o ícone da baleia do Docker aparecer na sua barra de menu (no topo da tela).

    ![Ícone do Docker na barra de menu do macOS](docs/images/macos_docker_icon.png "Ícone do Docker na barra de menu do macOS")

---

## Instalação no Linux (Ubuntu/Debian)

### 1. Instalando o Git

1.  **Abra o Terminal.**
2.  **Execute os seguintes comandos:**
    ```bash
    sudo apt update
    sudo apt install git
    ```
3.  **Verifique a instalação:**
    *   Digite `git --version` e pressione Enter. Se aparecer uma versão, está tudo certo.

### 2. Instalando o Docker

As instruções oficiais do Docker são as melhores para garantir que tudo funcione bem.

1.  **Siga o guia oficial do Docker:**
    *   [Clique aqui para ver as instruções de instalação do Docker para Ubuntu](https://docs.docker.com/engine/install/ubuntu/)
    *   Siga a seção **"Install using the convenience script"** para uma instalação mais rápida e fácil.

2.  **Adicione seu usuário ao grupo do Docker (Passo importante!):**
    *   Para poder usar o Docker sem precisar digitar `sudo` toda vez, execute o comando abaixo:
        ```bash
        sudo usermod -aG docker $USER
        ```
    *   **Você precisa fechar o terminal e abrir um novo, ou reiniciar o computador, para que essa alteração tenha efeito.**

---

## Tudo Pronto!

Com o **Git** e o **Docker** instalados e rodando, seu ambiente está pronto!

Agora você pode voltar para o [guia de instalação principal](./README.md) e seguir para o **Passo 2**.
