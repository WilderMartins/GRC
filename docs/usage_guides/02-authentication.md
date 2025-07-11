# Autenticação e Segurança da Conta no Phoenix GRC

Este guia detalha os processos de autenticação no Phoenix GRC e como você pode aumentar a segurança da sua conta, especialmente através da Autenticação de Múltiplos Fatores (MFA).

## 1. Processo de Login

Para acessar o Phoenix GRC, você precisará fornecer suas credenciais na página de login.

*   **Acessando a Página de Login:** Normalmente, ao acessar a URL principal da sua instância Phoenix GRC, você será direcionado para a página de login se não estiver autenticado.
*   **Campos Necessários:**
    *   **Email:** Seu endereço de email registrado.
    *   **Senha:** Sua senha pessoal.
*   **Link "Esqueci minha senha":** Se você esqueceu sua senha, utilize este link para iniciar o processo de recuperação (esta funcionalidade depende da configuração do servidor de email).

[SCREENSHOT: Página de Login do Phoenix GRC com campos de Email, Senha e link "Esqueci minha senha"]

### Login com Provedores Externos (SSO/Social Login)

Se sua organização configurou Single Sign-On (SSO) com SAML ou login social (Google, GitHub), você verá botões adicionais na página de login.

*   Clique no botão correspondente ao provedor desejado (ex: "Login com Google", "Login com SAML Corporativo").
*   Você será redirecionado para a página de autenticação do provedor externo.
*   Após autenticar-se com sucesso no provedor externo, você será redirecionado de volta ao Phoenix GRC e logado automaticamente.

[SCREENSHOT: Página de Login do Phoenix GRC com exemplos de botões de SSO (ex: "Login com Google", "Login com SAML Corporativo")]

## 2. Autenticação de Múltiplos Fatores (MFA)

Recomendamos fortemente a ativação do MFA para adicionar uma camada extra de segurança à sua conta. O Phoenix GRC suporta TOTP (Time-based One-Time Password) usando aplicativos autenticadores como Google Authenticator, Authy, Microsoft Authenticator, etc.

### Acessando as Configurações de Segurança

1.  Após o login, procure por um link ou menu com seu nome de usuário ou ícone de perfil (geralmente no canto superior direito do `AdminLayout`).
2.  Navegue até a seção "Segurança" ou "Autenticação de Dois Fatores". (Ex: `http://sua-instancia/user/security`)

[SCREENSHOT: Exemplo de menu de usuário no AdminLayout, com item "Segurança" ou "Perfil" destacado, levando à página /user/security]

### Habilitando TOTP (Time-based One-Time Password)

Se o TOTP ainda não estiver ativo para sua conta:

1.  Na página de Configurações de Segurança, você verá uma opção para "Habilitar Autenticação TOTP". Clique neste botão.
    [SCREENSHOT: Seção de 2FA na página /user/security, mostrando status "TOTP Inativo" e o botão "Habilitar Autenticação TOTP"]
2.  **Configurar Aplicativo Autenticador:**
    *   Aparecerá um QR Code e um segredo em formato de texto.
    *   Abra seu aplicativo autenticador preferido (Google Authenticator, Authy, etc.).
    *   Escaneie o QR Code. Se não puder escanear, adicione a conta manualmente usando o segredo de texto fornecido.
    [SCREENSHOT: Interface de configuração do TOTP na página /user/security, exibindo o QR Code, o segredo em texto e o campo para o código de verificação]
3.  **Verificar e Ativar:**
    *   Seu aplicativo autenticador agora gerará códigos de 6 dígitos que mudam a cada 30-60 segundos.
    *   Insira o código atual gerado pelo seu aplicativo no campo "Código de Verificação" na página do Phoenix GRC.
    *   Clique em "Verificar e Ativar".
    [SCREENSHOT: Detalhe do campo "Código de Verificação" preenchido e o botão "Verificar e Ativar"]
4.  **Sucesso:** Se o código estiver correto, o TOTP será ativado para sua conta. Você receberá uma notificação de sucesso. (A página deve atualizar mostrando TOTP como ativo).

### Gerenciando Códigos de Backup

Imediatamente após ativar o TOTP com sucesso, ou a qualquer momento enquanto o TOTP estiver ativo, você pode (e deve) gerar e salvar seus códigos de backup. Esses códigos permitem que você acesse sua conta caso perca o acesso ao seu aplicativo autenticador.

1.  Na página de Configurações de Segurança (com TOTP ativo), clique no botão "Gerenciar Códigos de Backup" (ou "Gerar Novos Códigos de Backup").
    [SCREENSHOT: Página /user/security com TOTP ativo, destacando o botão "Gerenciar Códigos de Backup"]
2.  Você pode ser solicitado a confirmar esta ação, pois gerar novos códigos invalida quaisquer códigos antigos.
3.  **Salve seus Códigos:**
    *   Uma lista de códigos de backup será exibida em um modal. **Estes códigos são mostrados apenas uma vez.**
    *   **Copie** os códigos para um local seguro (ex: gerenciador de senhas, arquivo impresso em local seguro).
    *   Você também pode **baixar** os códigos como um arquivo `.txt`.
    *   **Trate esses códigos como senhas. Qualquer um com acesso a um código de backup pode contornar seu TOTP.**
    [SCREENSHOT: Modal exibindo a lista de códigos de backup gerados, com os botões "Copiar Códigos", "Baixar Códigos (.txt)" e "Entendi, guardei meus códigos"]
4.  Clique em "Entendi, guardei meus códigos" após salvá-los para fechar o modal.

### Fazendo Login com TOTP

Com o TOTP ativo, o processo de login terá uma etapa adicional:

1.  Insira seu email e senha normalmente na página de login.
2.  Se as credenciais estiverem corretas, você será solicitado a fornecer seu "Código de Autenticação" (ou similar).
    [SCREENSHOT: Página de Login do Phoenix GRC, segunda etapa, solicitando o "Código de Autenticação" após senha correta]
3.  Abra seu aplicativo autenticador, obtenha o código atual para sua conta Phoenix GRC e insira-o.
4.  Clique em "Verificar Código" (ou similar).

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
[SCREENSHOT: Página /user/security com TOTP ativo, destacando o botão "Desabilitar TOTP"]
3.  Você será solicitado a confirmar sua identidade inserindo sua **senha atual** da conta Phoenix GRC em um modal.
    [SCREENSHOT: Modal de confirmação para desabilitar TOTP, com campo para senha atual e botão "Confirmar Desativação"]
4.  Insira sua senha e clique em "Confirmar Desativação".
5.  Se a senha estiver correta, o TOTP será desabilitado, e a página de segurança refletirá este novo estado.

## 3. (Opcional) Alteração de Senha

*Se a funcionalidade de alteração de senha pelo usuário estiver implementada no perfil do usuário:*

1.  Navegue até a seção de perfil ou configurações de segurança da sua conta.
2.  Procure por uma opção como "Alterar Senha".
3.  Você geralmente precisará fornecer sua senha atual e, em seguida, a nova senha (com confirmação).
4.  Siga as instruções na tela.

[SCREENSHOT: Exemplo de formulário de alteração de senha dentro de uma seção de perfil ou segurança do usuário, se esta funcionalidade for implementada]

---

Lembre-se de manter suas credenciais e métodos de MFA seguros. Se você suspeitar que sua conta foi comprometida, entre em contato com o administrador do sistema.
