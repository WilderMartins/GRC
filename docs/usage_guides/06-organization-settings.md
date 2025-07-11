# Guia de Configurações da Organização

Este guia detalha as configurações que um administrador ou gerente de organização (`admin` ou `manager` role) pode ajustar no Phoenix GRC para personalizar a instância e gerenciar o acesso.

## 1. Acessando as Configurações da Organização

As configurações específicas da sua organização são geralmente acessadas através de sub-itens no menu de navegação principal, contextualizadas para a organização à qual seu usuário pertence. Por exemplo:

*   **Branding (Identidade Visual):** Link no menu como "Organização" -> "Branding" ou diretamente se o layout permitir. (URL típica: `/admin/organizations/:orgId/branding`)
*   **Usuários:** Link no menu como "Organização" -> "Usuários". (URL típica: `/admin/organizations/:orgId/users`)
*   **Webhooks:** Link no menu como "Organização" -> "Webhooks". (URL típica: `/admin/organizations/:orgId/webhooks`)
*   **Provedores de Identidade (SSO):** Link no menu como "Provedores de Identidade" ou "Configurações de SSO". (URL típica: `/admin/identity-providers` - esta pode ser global para o admin da organização sem `:orgId` na URL, pois o `orgId` vem do usuário logado).

[SCREENSHOT: Menu de navegação principal (AdminLayout sidebar) mostrando os itens "Branding", "Usuários da Organização", "Webhooks" e "Provedores de Identidade" (ou agrupados sob um item "Organização").]

## 2. Branding (Identidade Visual)

Personalize a aparência da plataforma com a marca da sua organização. Acesse a seção de Branding (ex: `/admin/organizations/:orgId/branding`).

[SCREENSHOT: Página de Configurações de Branding (/admin/organizations/:orgId/branding) mostrando o preview do logo atual, campos de upload de novo logo, e os seletores de Cor Primária e Cor Secundária.]

*   **Logo da Organização:**
    *   Faça upload de um novo logo clicando em "Carregar logo" ou "Alterar logo".
    *   Formatos suportados: PNG, JPG, GIF, SVG. Observe o limite de tamanho (ex: Máx 2MB).
    *   Um preview do logo selecionado ou do logo atual é exibido.
    *   É possível remover o preview de um logo recém-selecionado (antes de salvar) ou resetar para o logo carregado anteriormente.
*   **Cor Primária:**
    *   Selecione a cor primária da sua marca. Esta cor será usada em botões principais, links de navegação ativos, e outros elementos de destaque na interface.
    *   Use o seletor de cores visual ou insira diretamente o código hexadecimal (ex: `#1A2B3C`).
*   **Cor Secundária:**
    *   Selecione a cor secundária. Pode ser usada para acentos visuais ou elementos secundários, dependendo do tema.
*   **Salvar:** Clique em "Salvar Configurações" para aplicar as mudanças. A interface da aplicação deve refletir as novas cores e o logo (pode ser necessário recarregar algumas partes ou o `AuthContext` pode atualizar dinamicamente).

## 3. Gerenciamento de Usuários

Acesse a seção de Usuários da Organização (ex: `/admin/organizations/:orgId/users`) para listar, gerenciar roles e status dos usuários pertencentes à sua organização.

[SCREENSHOT: Tabela de listagem de usuários da organização (/admin/organizations/:orgId/users) com colunas Nome, Email, Cargo (Role), Status (Ativo/Inativo) e Ações.]

*   **Listar Usuários:** A tabela exibe todos os usuários da organização com paginação, se houver muitos.
*   **Mudar Cargo (Role) de um Usuário:**
    *   Clique no botão "Mudar Cargo" (ou similar) na linha do usuário desejado.
    *   Um modal aparecerá, permitindo selecionar a nova role para o usuário (ex: Admin, Manager, User).
        [SCREENSHOT: Modal de Edição de Role de Usuário, com dropdown para selecionar nova role e botão de salvar.]
    *   Selecione a nova role e confirme. O backend pode ter regras para prevenir o rebaixamento do último admin.
*   **Ativar/Desativar um Usuário:**
    *   Use o botão "Ativar" ou "Desativar" na linha do usuário.
    *   Uma confirmação (`window.confirm`) será solicitada.
    *   Esta ação altera o status `is_active` do usuário. O backend pode ter regras para prevenir a desativação do próprio usuário ou do último admin ativo.
*   **Convidar Novo Usuário:** (Funcionalidade Futura) Se implementado, um botão "Convidar Usuário" permitiria adicionar novos membros à organização enviando um convite por email. (Atualmente, a criação de novos usuários pode ser via registro público, se habilitado, ou por um superadmin do sistema).

## 4. Configuração de Webhooks

Acesse a seção de Webhooks (ex: `/admin/organizations/:orgId/webhooks`) para configurar endpoints que receberão notificações em tempo real sobre eventos específicos no Phoenix GRC.

[SCREENSHOT: Tela de listagem de Webhooks configurados para a organização, mostrando Nome, URL (parcial), Eventos (como badges coloridos) e Status (Ativo/Inativo com indicador visual).]

*   **Adicionar Novo Webhook:**
    *   Clique em "Adicionar Novo Webhook". Um modal com o formulário será exibido.
        [SCREENSHOT: Modal/Formulário de criação/edição de Webhook com campos para Nome, URL, checkboxes para Tipos de Evento (ex: "Risco Criado", "Status do Risco Alterado"), e toggle para Ativo/Inativo.]
    *   **Nome:** Um nome descritivo para o webhook (ex: "Notificações de Risco para Slack").
    *   **URL do Webhook:** A URL do seu sistema externo que receberá o payload JSON da notificação.
    *   **Tipos de Evento:** Selecione (usando checkboxes) quais eventos do Phoenix GRC devem disparar este webhook. Pelo menos um deve ser selecionado.
    *   **Status:** Marque como "Ativo" para que o webhook comece a enviar notificações.
    *   Clique em "Adicionar Webhook" (ou "Salvar Alterações" se editando).
*   **Editar/Deletar Webhook:** Na lista de webhooks, use as ações "Editar" ou "Deletar" para modificar ou remover uma configuração existente.

## 5. Configuração de Provedores de Identidade (SSO/Social Login)

Acesse a seção de Provedores de Identidade (ex: `/admin/identity-providers`) para integrar o Phoenix GRC com seus sistemas de autenticação SAML 2.0 ou permitir login com contas sociais (OAuth2 - Google, GitHub).

[SCREENSHOT: Tela de listagem de Provedores de Identidade configurados, mostrando Nome, Tipo (SAML, Google, GitHub com ícones), Status (Ativo/Inativo com indicador visual) e Ações.]

*   **Adicionar Novo Provedor:**
    *   Clique em "Adicionar Novo Provedor".
    *   Selecione o **Tipo de Provedor** (SAML, OAuth2 Google, OAuth2 GitHub). O formulário se adaptará aos campos necessários.
    *   **Para SAML 2.0:**
        [SCREENSHOT: Formulário de configuração de um Provedor SAML, mostrando campos para Nome, Entity ID do IdP, URL de SSO do IdP, campo para colar o Certificado X.509 do IdP (PEM), e seções para opções como "Assinar Requisições" e "Mapeamento de Atributos".]
        *   **Nome do Provedor:** Um nome para esta configuração (ex: "Login Corporativo Okta").
        *   **Entity ID do IdP:** O identificador único do seu Identity Provider SAML.
        *   **URL de SSO do IdP:** O endpoint de login do seu IdP para onde os usuários serão redirecionados.
        *   **Certificado X.509 do IdP:** O certificado público (formato PEM) do seu IdP, usado para verificar as assinaturas das respostas SAML.
        *   **(Opcional) Mapeamento de Atributos:** Configure como os atributos da asserção SAML (ex: `email`, `displayName`, `firstName`, `lastName`) devem ser mapeados para os campos do usuário no Phoenix GRC.
    *   **Para OAuth2 (Google/GitHub):**
        [SCREENSHOT: Formulário de configuração de um Provedor OAuth2 (ex: Google), mostrando campos para Nome, Client ID, Client Secret e (opcionalmente) Escopos.]
        *   **Nome do Provedor:** (ex: "Login com Google Workspace").
        *   **Client ID:** O Client ID da sua aplicação OAuth2 registrada no Google Cloud Console ou GitHub Developer Settings.
        *   **Client Secret:** O Client Secret correspondente.
        *   **(Opcional) Escopos:** Escopos adicionais do OAuth2 a serem solicitados (o padrão geralmente inclui `email` e `profile`).
    *   **Status:** Marque como "Ativo" para que este provedor apareça como uma opção na página de login.
    *   Salve o provedor.
*   **Metadados do SP (Service Provider - para SAML):**
    *   Ao configurar um IdP SAML, você precisará fornecer a ele os metadados do Phoenix GRC (que atua como SP). A URL para os metadados do Phoenix GRC SP é geralmente `[SUA_APP_ROOT_URL]/auth/saml/metadata` ou, se específico por configuração de IdP no Phoenix, `/auth/saml/:idpIdNoPhoenixGRC/metadata`. Esta URL pode ser encontrada na documentação da API ou na interface de configuração do provedor SAML dentro do Phoenix GRC.
*   **Editar/Deletar/Ativar/Desativar Provedor:** Ações disponíveis na lista para gerenciar configurações existentes.

---

Manter as configurações da sua organização atualizadas garante que o Phoenix GRC se alinhe com as necessidades, políticas de segurança e identidade visual da sua empresa.
