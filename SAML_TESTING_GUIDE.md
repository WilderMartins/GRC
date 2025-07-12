# Guia de Testes de Integração SAML

Este guia fornece um passo a passo para testar a integração SAML do Phoenix GRC com um provedor de identidade de teste, o [SAMLtest.id](https://samltest.id/).

## Passo 1: Configurar o Phoenix GRC

1.  **Inicie a Aplicação:**
    Certifique-se de que a sua instância do Phoenix GRC está rodando localmente.
    ```bash
    docker-compose up -d
    ```

2.  **Acesse como Administrador:**
    Faça login na sua instância com a conta de administrador.

3.  **Vá para a Configuração de Provedores de Identidade:**
    -   Navegue para a seção de administração da sua organização.
    -   Clique em "Provedores de Identidade" e em "Adicionar Novo".

4.  **Preencha o Formulário com os Dados do SP (Phoenix GRC):**
    -   **Tipo de Provedor:** Selecione `saml`.
    -   **Nome:** Dê um nome, por exemplo, `SAMLtest.id`.
    -   **Configuração JSON:** Aqui, vamos precisar dos metadados do nosso próprio serviço (Service Provider). Para obtê-los, abra em uma nova aba a URL de metadados do seu IdP recém-criado. O ID estará na URL da página de edição.
        -   URL de Metadados do SP: `https://localhost/auth/saml/ID_DO_IDP/metadata`
        -   **ACS URL:** `https://localhost/auth/saml/ID_DO_IDP/acs`
    -   **Mapeamento de Atributos JSON:**
        ```json
        {
            "email": "urn:oid:0.9.2342.19200300.100.1.3",
            "firstName": "urn:oid:2.5.4.42",
            "lastName": "urn:oid:2.5.4.4"
        }
        ```

5.  **Salve o Provedor de Identidade.**

## Passo 2: Configurar o Provedor de Identidade (SAMLtest.id)

1.  **Acesse o SAMLtest.id:**
    -   Vá para [https://samltest.id/](https://samltest.id/).
    -   Clique em **"Upload IdP Metadata"**.

2.  **Faça o Upload dos Metadados do Phoenix GRC:**
    -   Na página do SAMLtest.id, cole a URL de metadados do seu SP (ex: `https://localhost/auth/saml/ID_DO_IDP/metadata`) no campo "Or paste your metadata URL".
    -   Clique em **"Fetch Metadata"**.
    -   O SAMLtest.id irá carregar os detalhes do seu SP (Phoenix GRC).

3.  **Faça o Login de Teste:**
    -   Role para baixo na página do SAMLtest.id até a seção "Login Test".
    -   Clique em **"Test Login"**.

## Passo 3: Fluxo de Teste

1.  Você será redirecionado para a página de login do SAMLtest.id.
2.  Use as credenciais de exemplo fornecidas na página.
3.  Após o login bem-sucedido, o SAMLtest.id irá redirecioná-lo de volta para o Phoenix GRC (para a ACS URL).
4.  O Phoenix GRC irá processar a asserção SAML.
5.  Se tudo estiver correto, um novo usuário será criado no Phoenix GRC (se `ALLOW_SAML_USER_CREATION` for `true`) e você será redirecionado para o dashboard da aplicação, já logado como este novo usuário.

## Verificação

-   **No Phoenix GRC:** Verifique se um novo usuário foi criado na sua organização com o e-mail do usuário de teste do SAMLtest.id.
-   **No Log do Backend:** Verifique os logs do container do backend (`docker-compose logs -f backend`) para mensagens de sucesso ou erro relacionadas ao SAML.
