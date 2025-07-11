# Primeiros Passos com Phoenix GRC

Parabéns! Se você está lendo este guia, significa que a instalação e configuração inicial do Phoenix GRC foram concluídas com sucesso. Este guia o ajudará a dar os primeiros passos na plataforma.

## 1. Acessando a Plataforma

Após a conclusão do [Wizard de Instalação](./../../README.md#método-1-wizard-de-instalação-via-browser-recomendado) (ou do setup via CLI), sua instância do Phoenix GRC estará rodando.

*   **URL de Acesso:** Abra seu navegador e acesse a URL que você configurou para a aplicação (normalmente `http://localhost` ou `http://localhost:PORTA_NGINX` para ambientes de desenvolvimento local, ou a URL de produção da sua instância).
*   Você será direcionado para a **Página de Login**.

[SCREENSHOT: Página de Login do Phoenix GRC]

## 2. Realizando o Primeiro Login

Utilize as credenciais do usuário administrador que você criou durante a etapa final do wizard de instalação (ou do script CLI).

*   **Email:** O email do administrador que você definiu.
*   **Senha:** A senha que você definiu para este administrador.

Após inserir suas credenciais, clique em "Login" (ou o texto equivalente no botão).

## 3. Explorando o Dashboard Principal

Após o login bem-sucedido, você será direcionado para o Dashboard Principal do Phoenix GRC.

[SCREENSHOT: Dashboard Principal do Phoenix GRC - Visão Geral]

O dashboard é projetado para fornecer uma visão geral e acesso rápido às principais funcionalidades da plataforma. Aqui você poderá encontrar (dependendo das suas permissões e features ativadas):

*   **Resumos e Métricas:** Cards ou gráficos resumindo o estado dos riscos, vulnerabilidades, ou progresso de conformidade.
*   **Navegação Principal:** Uma barra lateral ou menu superior para acessar os diferentes módulos:
    *   Gestão de Riscos
    *   Gestão de Vulnerabilidades
    *   Auditoria e Conformidade
    *   Configurações da Organização (se você for admin/manager da organização)
    *   Outras seções administrativas (se você for admin do sistema)

## 4. Próximos Passos Recomendados

Com o acesso inicial estabelecido, sugerimos os seguintes passos:

*   **Configure a Autenticação de Múltiplos Fatores (MFA):** Para aumentar a segurança da sua conta de administrador, navegue até as configurações de segurança do seu perfil e habilite o MFA. Veja o guia [Autenticação e Segurança da Conta](./02-authentication.md) para mais detalhes.
*   **Explore as Configurações da Organização:** Se você é um administrador, familiarize-se com as [Configurações da Organização](./06-organization-settings.md), onde você pode:
    *   Personalizar o branding (logo e cores).
    *   Gerenciar usuários.
    *   Configurar Webhooks para notificações.
    *   Configurar Provedores de Identidade para SSO.
*   **Comece a Usar os Módulos Principais:**
    *   [Gestão de Riscos](./03-risk-management.md)
    *   [Gestão de Vulnerabilidades](./04-vulnerability-management.md)
    *   [Auditoria e Conformidade](./05-audit-compliance.md)

Seja bem-vindo ao Phoenix GRC! Esperamos que a plataforma o ajude a gerenciar efetivamente a governança, os riscos e a conformidade de TI da sua organização.
