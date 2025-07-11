# Autenticação e Segurança da Conta no Phoenix GRC

Este guia detalha os processos de autenticação no Phoenix GRC e como você pode aumentar a segurança da sua conta, especialmente através da Autenticação de Múltiplos Fatores (MFA).

## 1. Processo de Login

Para acessar o Phoenix GRC, você precisará fornecer suas credenciais na página de login.

*   **Acessando a Página de Login:** Normalmente, ao acessar a URL principal da sua instância Phoenix GRC, você será direcionado para a página de login se não estiver autenticado.
*   **Campos Necessários:**
    *   **Email:** Seu endereço de email registrado.
    *   **Senha:** Sua senha pessoal.
*   **Link "Esqueci minha senha":** Se você esqueceu sua senha, utilize este link para iniciar o processo de recuperação (esta funcionalidade depende da configuração do servidor de email).

[SCREENSHOT: Página de Login com campos de email e senha]

### Login com Provedores Externos (SSO/Social Login)

Se sua organização configurou Single Sign-On (SSO) com SAML ou login social (Google, GitHub), você verá botões adicionais na página de login.

*   Clique no botão correspondente ao provedor desejado (ex: "Login com Google", "Login com SAML Corporativo").
*   Você será redirecionado para a página de autenticação do provedor externo.
*   Após autenticar-se com sucesso no provedor externo, você será redirecionado de volta ao Phoenix GRC e logado automaticamente.

[SCREENSHOT: Página de Login exibindo botões de SSO/Social Login, se houver]

## 2. Autenticação de Múltiplos Fatores (MFA)

Recomendamos fortemente a ativação do MFA para adicionar uma camada extra de segurança à sua conta. O Phoenix GRC suporta TOTP (Time-based One-Time Password) usando aplicativos autenticadores como Google Authenticator, Authy, Microsoft Authenticator, etc.

### Acessando as Configurações de Segurança

1.  Após o login, procure por um link ou menu com seu nome de usuário ou ícone de perfil (geralmente no canto superior direito do `AdminLayout`).
2.  Navegue até a seção "Segurança" ou "Autenticação de Dois Fatores". (Ex: `http://sua-instancia/user/security`)

[SCREENSHOT: Menu do usuário apontando para a página de Segurança/2FA]

### Habilitando TOTP (Time-based One-Time Password)

Se o TOTP ainda não estiver ativo para sua conta:

1.  Na página de Configurações de Segurança, você verá uma opção para "Habilitar Autenticação TOTP". Clique neste botão.
    [SCREENSHOT: Página de Segurança mostrando TOTP Inativo e botão "Habilitar"]
2.  **Configurar Aplicativo Autenticador:**
    *   Aparecerá um QR Code e um segredo em formato de texto.
    *   Abra seu aplicativo autenticador preferido (Google Authenticator, Authy, etc.).
    *   Escaneie o QR Code. Se não puder escanear, adicione a conta manualmente usando o segredo de texto fornecido.
    [SCREENSHOT: Tela de setup do TOTP com QR Code e segredo]
3.  **Verificar e Ativar:**
    *   Seu aplicativo autenticador agora gerará códigos de 6 dígitos que mudam a cada 30-60 segundos.
    *   Insira o código atual gerado pelo seu aplicativo no campo "Código de Verificação" na página do Phoenix GRC.
    *   Clique em "Verificar e Ativar".
    [SCREENSHOT: Campo para inserir o código de verificação TOTP]
4.  **Sucesso:** Se o código estiver correto, o TOTP será ativado para sua conta. Você receberá uma notificação de sucesso.

### Gerenciando Códigos de Backup

Imediatamente após ativar o TOTP com sucesso, ou a qualquer momento enquanto o TOTP estiver ativo, você pode (e deve) gerar e salvar seus códigos de backup. Esses códigos permitem que você acesse sua conta caso perca o acesso ao seu aplicativo autenticador.

1.  Na página de Configurações de Segurança (com TOTP ativo), clique no botão "Gerenciar Códigos de Backup" (ou "Gerar Novos Códigos de Backup").
    [SCREENSHOT: Botão "Gerenciar Códigos de Backup" na página de segurança]
2.  Você pode ser solicitado a confirmar esta ação, pois gerar novos códigos invalida quaisquer códigos antigos.
3.  **Salve seus Códigos:**
    *   Uma lista de códigos de backup será exibida. **Estes códigos são mostrados apenas uma vez.**
    *   **Copie** os códigos para um local seguro (ex: gerenciador de senhas, arquivo impresso em local seguro).
    *   Você também pode **baixar** os códigos como um arquivo `.txt`.
    *   **Trate esses códigos como senhas. Qualquer um com acesso a um código de backup pode contornar seu TOTP.**
    [SCREENSHOT: Modal exibindo a lista de códigos de backup, com botões Copiar e Baixar]
4.  Clique em "Fechar" ou "Entendi, guardei meus códigos" após salvá-los.

### Fazendo Login com TOTP

Com o TOTP ativo, o processo de login terá uma etapa adicional:

1.  Insira seu email e senha normalmente na página de login.
2.  Se as credenciais estiverem corretas, você será solicitado a fornecer seu "Código de Autenticação" (ou similar).
    [SCREENSHOT: Página de Login na etapa 2FA, pedindo código TOTP]
3.  Abra seu aplicativo autenticador, obtenha o código atual para sua conta Phoenix GRC e insira-o.
4.  Clique em "Verificar" (ou similar).

### Fazendo Login com um Código de Backup

Se você não tiver acesso ao seu aplicativo autenticador, mas tiver seus códigos de backup:

1.  Na etapa de login onde o código 2FA é solicitado, insira um dos seus códigos de backup não utilizados no campo de código.
2.  Clique em "Verificar".
3.  **Cada código de backup só pode ser usado uma vez.** Após usar um código, risque-o da sua lista salva.
4.  Se você usar todos os seus códigos de backup, certifique-se de gerar um novo conjunto na página de Configurações de Segurança.

### Desabilitando TOTP

Se desejar desabilitar o TOTP (o que reduzirá a segurança da sua conta):

1.  Vá para a página de Configurações de Segurança.
2.  Clique no botão "Desabilitar Autenticação TOTP".
    [SCREENSHOT: Botão "Desabilitar TOTP" na página de segurança]
3.  Você será solicitado a confirmar sua identidade inserindo sua **senha atual** da conta Phoenix GRC.
    [SCREENSHOT: Modal pedindo senha para confirmar desativação do TOTP]
4.  Insira sua senha e clique em "Confirmar Desativação".
5.  Se a senha estiver correta, o TOTP será desabilitado.

## 3. (Opcional) Alteração de Senha

*Se a funcionalidade de alteração de senha pelo usuário estiver implementada no perfil do usuário:*

1.  Navegue até a seção de perfil ou configurações de segurança da sua conta.
2.  Procure por uma opção como "Alterar Senha".
3.  Você geralmente precisará fornecer sua senha atual e, em seguida, a nova senha (com confirmação).
4.  Siga as instruções na tela.

[SCREENSHOT: Formulário de alteração de senha, se existir]

---

Lembre-se de manter suas credenciais e métodos de MFA seguros. Se você suspeitar que sua conta foi comprometida, entre em contato com o administrador do sistema.
