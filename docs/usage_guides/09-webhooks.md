# Guia de Uso: Configurando Webhooks

Este guia detalha como configurar webhooks para receber notificações em tempo real do Phoenix GRC em outras ferramentas, como Slack, Microsoft Teams, ou qualquer serviço que possa receber uma requisição HTTP POST.

## Visão Geral

Webhooks são uma maneira poderosa de automatizar fluxos de trabalho. Em vez de você verificar a plataforma para ver o que há de novo, o Phoenix GRC pode notificar ativamente seus sistemas quando eventos específicos ocorrem.

**Exemplos de uso:**
-   Receber uma notificação no Slack quando um novo risco crítico é criado.
-   Criar um card automaticamente no Trello ou Jira quando uma vulnerabilidade é atribuída a um engenheiro.
-   Registrar logs de auditoria em um sistema de SIEM.

## Passo 1: Criando um Webhook no Phoenix GRC

1.  **Navegue até a Configuração de Webhooks:**
    -   Faça login como administrador.
    -   Vá para `Admin` -> `Configurações da Organização` -> `Webhooks`.

2.  **Adicione um Novo Webhook:**
    -   Clique em "Adicionar Novo".
    -   Você verá o formulário de configuração de webhook.

3.  **Preencha os Campos:**
    -   **Nome:** Um nome descritivo para sua referência (ex: "Notificações de Risco no Slack").
    -   **URL:** A URL do serviço que receberá a notificação. Este é o campo mais importante.
    -   **Segredo (Opcional):** Um token secreto que será usado para assinar o payload. O serviço receptor pode usar este segredo para verificar se a requisição veio de fato do Phoenix GRC.
    -   **Tipos de Evento:** Selecione os eventos que você quer que disparem este webhook. Você pode selecionar múltiplos eventos.
        -   `risk_created`: Disparado quando um novo risco é criado.
        -   `vulnerability_assigned`: Disparado quando uma vulnerabilidade é atribuída a um usuário.
    -   **Ativo:** Marque esta caixa para ativar o webhook.

4.  **Salve o Webhook.**

## Passo 2: Exemplo de Integração com Slack

O Slack facilita o recebimento de notificações de fontes externas através dos "Incoming Webhooks".

1.  **Crie um Aplicativo no Slack:**
    -   Vá para [api.slack.com/apps](https://api.slack.com/apps) e clique em "Create New App".
    -   Escolha "From scratch", dê um nome ao seu aplicativo (ex: "Phoenix GRC Notifier") e selecione o seu Workspace do Slack.

2.  **Ative os Incoming Webhooks:**
    -   Na página do seu novo aplicativo, vá para a seção "Incoming Webhooks" e ative-a.
    -   Clique em "Add New Webhook to Workspace".
    -   Escolha um canal para o qual as notificações serão enviadas e clique em "Allow".

3.  **Copie a URL do Webhook:**
    -   O Slack irá gerar uma URL que começa com `https://hooks.slack.com/services/...`. **Esta é a URL que você deve colar no campo "URL" do formulário de webhook no Phoenix GRC.**

## Passo 3: Testando a Integração

Depois de salvar o seu webhook no Phoenix GRC com a URL do Slack:

1.  **Volte para a lista de Webhooks** e clique em "Editar" no webhook que você acabou de criar.
2.  Clique no botão **"Enviar Evento de Teste"**.
3.  Imediatamente, você deve receber uma notificação no canal do Slack que você configurou. A notificação conterá um payload de exemplo.

## Entendendo o Payload do Webhook

O Phoenix GRC envia os dados do webhook como um payload JSON via HTTP POST. A estrutura do payload é a seguinte:

```json
{
  "event": "event_type_name",
  "timestamp": "2023-10-27T10:00:00Z",
  "data": {
    // O objeto de dados varia dependendo do evento.
    // Para 'risk_created', será o objeto de Risco completo.
    // Para 'vulnerability_assigned', será o objeto de Vulnerabilidade completo.
  }
}
```

### Verificação de Assinatura (Segurança)

Se você configurou um **Segredo** no seu webhook, cada requisição incluirá um cabeçalho `X-Phoenix-Signature-256`. O valor deste cabeçalho é um código HMAC-SHA256 do corpo do payload, usando o seu segredo como chave.

Seu serviço receptor pode (e deve) recalcular esta assinatura para verificar a autenticidade da requisição.
