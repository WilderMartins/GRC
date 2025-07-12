# Phoenix GRC - Instalação Rápida

Bem-vindo ao Phoenix GRC! Este guia mostra como instalar a plataforma em poucos minutos, de forma simples e visual.

## O que é o Phoenix GRC?

O Phoenix GRC é uma plataforma de Gestão de Governança, Risco e Conformidade de TI. Ele ajuda equipes a gerenciar riscos, responder a vulnerabilidades e demonstrar conformidade de forma eficiente.

---

## Instalação Super Simples

O processo de instalação foi desenhado para ser o mais fácil possível.

### Passo 1: Prepare o Ambiente (5 minutos)

Antes de tudo, você precisa de duas ferramentas no seu computador: **Git** e **Docker**.

Se você não os tiver, não se preocupe! Nosso guia abaixo ensina a instalar ambos com passo a passo e **imagens ilustrativas**.

➡️ **[Guia de Preparação do Ambiente](./PREPARACAO_AMBIENTE.md)**

### Passo 2: Baixe e Inicie o Projeto (2 minutos)

Com o Git e o Docker prontos, abra seu terminal (ou Prompt de Comando/PowerShell) e siga os comandos.

1.  **Baixe (clone) o projeto:**
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

4.  **Inicie a Aplicação:**
    > **Importante:** Certifique-se de que o aplicativo **Docker Desktop** está aberto e em execução no seu computador antes de rodar o comando abaixo.
    ```bash
    docker-compose up -d
    ```
    Este comando irá construir e iniciar a plataforma. Pode demorar alguns minutos na primeira vez.

### Passo 3: Acesse o Wizard de Instalação Visual

Pronto! A plataforma já está rodando. Agora, abra seu navegador de internet e acesse:

➡️ **http://localhost**

Você será recebido pelo nosso **Wizard de Instalação**, que o guiará visualmente pelo resto do processo:

1.  **Bem-vindo:** Uma tela de introdução.
2.  **Verificação do Sistema:** O wizard confirmará que tudo está funcionando (como a conexão com o banco de dados que o Docker criou para você).
3.  **Criação da Conta de Administrador:** Você definirá o nome da sua organização e criará o primeiro usuário administrador.
4.  **Conclusão:** Tudo pronto! Você será direcionado para a tela de login.

Após concluir o wizard, faça login com as credenciais que você acabou de criar e comece a usar o Phoenix GRC!

---

## Para Desenvolvedores

Se você é um desenvolvedor, toda a documentação técnica (detalhes da API, estrutura do projeto, etc.) está no nosso **[Guia do Desenvolvedor](./DEVELOPER_GUIDE.md)**.
