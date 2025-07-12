# Guia de Uso: Configurando a Integração SAML

Este guia detalha como configurar a integração Single Sign-On (SSO) com um provedor de identidade (IdP) SAML, como Okta, Azure AD, ou ADFS.

## Visão Geral

A integração SAML permite que os usuários da sua organização façam login no Phoenix GRC usando as credenciais corporativas existentes, centralizando a autenticação e melhorando a segurança.

O fluxo geral é:
1.  Um administrador do Phoenix GRC configura um novo Provedor de Identidade SAML na plataforma.
2.  O administrador usa as informações do Phoenix GRC (Service Provider) para configurar uma nova aplicação no Provedor de Identidade (ex: Okta).
3.  Os usuários podem então fazer login via SSO.

## Passo 1: Configurando o Phoenix GRC

1.  **Navegue até a Administração:**
    -   Faça login como administrador.
    -   Vá para `Admin` -> `Configurações da Organização` -> `Provedores de Identidade`.

2.  **Crie um Novo Provedor de Identidade:**
    -   Clique em "Adicionar Novo".
    -   Selecione o tipo `SAML`.
    -   Dê um nome descritivo, por exemplo, "Login Corporativo (Okta)".

3.  **Obtenha as URLs do Service Provider (Phoenix GRC):**
    -   Após criar o provedor, a página de edição exibirá as URLs que você precisará para configurar o seu IdP. A mais importante é a **Assertion Consumer Service (ACS) URL**.
    -   **ACS URL:** `https://seu-dominio.com/auth/saml/ID_DO_IDP/acs`
    -   **Entity ID do SP:** Por padrão, será a URL de metadados: `https://seu-dominio.com/auth/saml/ID_DO_IDP/metadata`

## Passo 2: Configurando o seu Provedor de Identidade (Exemplo com Okta)

1.  **Crie uma Nova Aplicação no Okta:**
    -   No seu painel de administrador do Okta, vá para `Applications` -> `Applications` e clique em `Create App Integration`.
    -   Selecione `SAML 2.0` como método de login.

2.  **Configurações Gerais:**
    -   Dê um nome para a aplicação (ex: Phoenix GRC).

3.  **Configurações SAML:**
    -   **Single sign on URL:** Cole a **ACS URL** do Phoenix GRC.
    -   **Audience URI (SP Entity ID):** Cole o **Entity ID do SP** do Phoenix GRC.
    -   **Name ID format:** Deixe como `Unspecified` ou `EmailAddress`.
    -   **Application username:** Selecione `Email`.

4.  **Mapeamento de Atributos (Attribute Statements):**
    -   Esta é a parte mais importante. Você precisa mapear os atributos do Okta para os nomes que o Phoenix GRC espera.
    -   Adicione os seguintes mapeamentos:
        -   **Name:** `email` | **Value:** `user.email`
        -   **Name:** `firstName` | **Value:** `user.firstName`
        -   **Name:** `lastName` | **Value:** `user.lastName`
    -   *Nota: Os nomes (`email`, `firstName`, `lastName`) são os valores padrão que o Phoenix GRC espera. Se você precisar usar nomes diferentes, poderá customizá-los no JSON de Mapeamento de Atributos no Phoenix GRC.*

5.  **Finalize a Configuração no Okta:**
    -   Após salvar a aplicação no Okta, vá para a aba `Sign On` e clique em `View SAML setup instructions` ou `View Metadata`.
    -   Você precisará das seguintes informações para o próximo passo:
        -   **Identity Provider Single Sign-On URL** (ou `SSO URL`)
        -   **Identity Provider Issuer** (ou `Entity ID` do IdP)
        -   **X.509 Certificate**

## Passo 3: Finalizando a Configuração no Phoenix GRC

1.  **Volte para a Edição do Provedor de Identidade no Phoenix GRC.**

2.  **Preencha a Configuração JSON:**
    -   Use as informações que você obteve do Okta no passo anterior.
    ```json
    {
        "idp_entity_id": "COLE_O_ISSUER_DO_OKTA_AQUI",
        "idp_sso_url": "COLE_A_SSO_URL_DO_OKTA_AQUI",
        "idp_x509_cert": "COLE_O_CERTIFICADO_X509_COMPLETO_AQUI"
    }
    ```
    -   Certifique-se de copiar o certificado completo, incluindo as linhas `-----BEGIN CERTIFICATE-----` e `-----END CERTIFICATE-----`.

3.  **Mapeamento de Atributos (Opcional):**
    -   Se você usou nomes de atributos diferentes no Okta, pode mapeá-los aqui. Por exemplo, se você usou `user_email` em vez de `email` no Okta:
    ```json
    {
        "email": "user_email",
        "firstName": "firstName",
        "lastName": "lastName"
    }
    ```

4.  **Ative e Salve:**
    -   Marque a opção "Ativo".
    -   Salve as alterações.

## Passo 4: Testando o Login

1.  **Acesse a Página de Login:**
    -   Abra uma nova janela anônima ou deslogue da sua sessão atual.
    -   Acesse a página de login do Phoenix GRC.

2.  **Use o Botão de SSO:**
    -   Um novo botão "Login com [Nome do seu IdP]" deve aparecer.
    -   Clique no botão.

3.  **Autentique no IdP:**
    -   Você será redirecionado para a página de login do seu provedor de identidade (Okta).
    -   Faça login com uma conta de usuário que esteja associada à aplicação Phoenix GRC no Okta.

4.  **Redirecionamento e Acesso:**
    -   Após a autenticação bem-sucedida, você será redirecionado de volta para o Phoenix GRC e deverá estar logado.

Parabéns! Você configurou com sucesso a integração SAML.
