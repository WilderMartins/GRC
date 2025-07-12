# Phoenix GRC - Instalação Rápida

Bem-vindo ao Phoenix GRC! Este guia irá ajudá-lo a instalar e configurar a plataforma de forma simples e rápida, sem a necessidade de conhecimento técnico avançado.

## O que é o Phoenix GRC?

O Phoenix GRC é uma plataforma de Gestão de Governança, Risco e Conformidade de TI. Ele ajuda equipes de todos os tamanhos a gerenciar riscos, responder a vulnerabilidades e demonstrar conformidade de forma eficiente.

## Instalação em 3 Passos

Nosso objetivo é que a instalação seja tão simples quanto possível. Siga os passos abaixo.

### Passo 1: Preparando o Ambiente

Antes de começar, você precisa de apenas duas ferramentas no seu computador: **Git** e **Docker**.

Se você não tem certeza se os possui, ou precisa de ajuda para instalá-los, criamos um guia passo a passo para você. Acesse o link abaixo:

➡️ **[Guia de Preparação do Ambiente](./PREPARACAO_AMBIENTE.md)**

### Passo 2: Baixando e Configurando o Projeto

Com o Git e o Docker prontos, abra o seu terminal (ou Prompt de Comando/PowerShell no Windows) e siga os comandos abaixo.

1.  **Baixe o projeto para a sua máquina:**
    ```bash
    git clone https://github.com/SEU_USUARIO/phoenix-grc.git
    ```
    *Substitua `https://github.com/SEU_USUARIO/phoenix-grc.git` pela URL correta do seu repositório.*

2.  **Entre na pasta do projeto:**
    ```bash
    cd phoenix-grc
    ```

3.  **Crie o arquivo de configuração:**
    Este comando simplesmente copia o nosso arquivo de exemplo. Não é necessário editar nada por enquanto.
    ```bash
    cp .env.example .env
    ```

### Passo 3: Iniciando a Aplicação

Este é o último passo! Este comando irá construir e iniciar todos os serviços necessários. Pode demorar alguns minutos na primeira vez.

```bash
docker-compose up -d
```

Aguarde alguns instantes para os serviços iniciarem.

## Acessando o Wizard de Instalação

Agora que a aplicação está rodando, abra o seu navegador de internet e acesse o seguinte endereço:

➡️ **http://localhost**

Você será recebido pelo nosso Wizard de Instalação, que irá guiá-lo através dos passos finais:

1.  **Bem-vindo:** Uma tela de introdução.
2.  **Verificação do Banco de Dados:** O sistema verificará se a conexão com o banco de dados (que o Docker criou para você) está funcionando.
3.  **Criação da Conta de Administrador:** Você irá definir o nome da sua organização e criar o primeiro usuário administrador.
4.  **Conclusão:** Tudo pronto! Você será direcionado para a tela de login.

Após concluir o wizard, você pode fazer login com as credenciais que acabou de criar e começar a usar o Phoenix GRC!

## Para Desenvolvedores

Se você é um desenvolvedor e quer contribuir para o projeto, toda a documentação técnica, incluindo detalhes da API, estrutura do projeto e guias de contribuição, foi movida para o nosso **[Guia do Desenvolvedor](./DEVELOPER_GUIDE.md)**.
