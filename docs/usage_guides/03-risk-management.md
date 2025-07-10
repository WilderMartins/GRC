# Guia de Gestão de Riscos no Phoenix GRC

O módulo de Gestão de Riscos é uma parte central do Phoenix GRC, permitindo identificar, analisar, avaliar e tratar os riscos de TI da sua organização.

## 1. Acessando o Módulo de Gestão de Riscos

Após o login, você pode acessar o módulo de Gestão de Riscos através do menu de navegação principal (geralmente na barra lateral esquerda).

[SCREENSHOT: Menu de navegação destacando "Gestão de Riscos"]

## 2. Tela de Listagem de Riscos

Ao entrar no módulo, você verá a tela de listagem de riscos, que apresenta uma tabela com os riscos registrados.

[SCREENSHOT: Tela de Listagem de Riscos com filtros e tabela]

### Funcionalidades da Listagem:

*   **Visualização em Tabela:** Os riscos são exibidos com colunas como Título, Categoria, Impacto, Probabilidade, Nível de Risco (calculado), Status e Proprietário.
*   **Filtros:** Você pode refinar a lista de riscos usando os filtros disponíveis acima da tabela (ex: por Categoria, Impacto, Probabilidade, Status, Proprietário, busca por Título).
    [SCREENSHOT: Detalhe dos controles de filtro]
*   **Ordenação:** Clique nos cabeçalhos das colunas para ordenar os riscos de forma ascendente ou descendente.
*   **Paginação:** Se houver muitos riscos, a paginação permitirá navegar entre as diferentes páginas da lista.
*   **Ações por Risco:**
    *   **Editar:** Permite modificar os detalhes de um risco existente.
    *   **Deletar:** Remove um risco (geralmente com uma confirmação).
    *   **Submeter para Aceite / Decidir Aceite:** Dependendo da sua role e do status do risco, botões para o workflow de aceite podem estar disponíveis.

### Botões de Ação Principais:

*   **Adicionar Novo Risco:** Leva ao formulário para registrar um novo risco.
*   **Importar Riscos (CSV):** Abre um modal para realizar o upload em massa de riscos a partir de um arquivo CSV.

## 3. Criando um Novo Risco

1.  Na tela de listagem de riscos, clique no botão "Adicionar Novo Risco".
2.  Você será direcionado para o formulário de criação de risco.
    [SCREENSHOT: Formulário de Criação de Risco]
3.  **Preencha os Campos:**
    *   **Título:** Um nome claro e conciso para o risco (obrigatório).
    *   **Descrição:** Detalhes sobre o risco, suas causas, potenciais consequências, etc.
    *   **Categoria:** Selecione a categoria do risco (ex: Tecnológico, Operacional, Legal).
    *   **Impacto:** Avalie o impacto potencial do risco (ex: Baixo, Médio, Alto, Crítico).
    *   **Probabilidade:** Avalie a probabilidade de ocorrência do risco (ex: Baixa, Média, Alta, Crítica).
    *   **Status:** O status inicial geralmente é "Aberto".
    *   **Proprietário (Owner):** Designe um usuário responsável pelo gerenciamento deste risco. Por padrão, pode ser você mesmo.
4.  Clique em "Criar Risco" (ou similar) para salvar.

O **Nível de Risco** (ex: Baixo, Moderado, Alto, Extremo) é geralmente calculado automaticamente com base nos valores de Impacto e Probabilidade.

## 4. Editando um Risco Existente

1.  Na lista de riscos, encontre o risco que deseja modificar.
2.  Clique na ação "Editar" correspondente a esse risco.
3.  O formulário será pré-preenchido com os dados do risco.
4.  Faça as alterações necessárias e clique em "Salvar Alterações".

## 5. Deletando um Risco

1.  Na lista de riscos, encontre o risco que deseja remover.
2.  Clique na ação "Deletar" correspondente.
3.  Uma mensagem de confirmação aparecerá. Confirme para excluir o risco permanentemente.

## 6. Gerenciando Stakeholders de um Risco

Para cada risco, você pode associar múltiplos stakeholders (partes interessadas).

1.  Acesse a tela de edição ou visualização detalhada de um risco.
2.  Procure por uma seção ou aba "Stakeholders".
    [SCREENSHOT: Seção de Stakeholders na página de um Risco]
3.  **Adicionar Stakeholder:**
    *   Clique em "Adicionar Stakeholder".
    *   Selecione o usuário desejado a partir de uma lista de usuários da organização.
    *   Confirme a adição.
4.  **Remover Stakeholder:**
    *   Na lista de stakeholders do risco, clique na ação para remover o usuário.

## 7. Workflow de Aceite de Risco

Riscos que não podem ser mitigados a um nível aceitável podem precisar passar por um processo formal de aceite.

1.  **Submeter um Risco para Aceite:**
    *   Usuários com a role `manager` ou `admin` podem submeter um risco para aceite.
    *   Na lista de riscos (ou na página de detalhes do risco), se o risco tiver um proprietário definido e estiver em um status apropriado (ex: "Aberto", "Em Andamento"), um botão como "Submeter para Aceite" estará disponível.
    *   Ao clicar, um workflow de aprovação é iniciado, e o proprietário do risco (Approver) é notificado.
    [SCREENSHOT: Botão "Submeter para Aceite" em um risco]
2.  **Aprovar ou Rejeitar um Risco (Proprietário do Risco):**
    *   O proprietário do risco (designado como aprovador) verá uma indicação de que há um risco pendente de sua decisão (ex: no dashboard, ou um badge na lista de riscos).
    *   Ao acessar o risco, haverá opções para "Aprovar Aceite" ou "Rejeitar Aceite".
    *   Um modal aparecerá para confirmar a decisão e adicionar comentários.
    [SCREENSHOT: Modal de Decisão de Aceite de Risco para o proprietário]
    *   Se aprovado, o status do risco muda para "Aceito". Se rejeitado, ele pode voltar para um status anterior ou requerer mais análise.
3.  **Histórico de Aprovações:**
    *   A página de detalhes de um risco mostrará o histórico de todas as tentativas de aceite, incluindo quem solicitou, quem decidiu, quando, e os comentários.

## 8. Upload em Massa de Riscos (Importar CSV)

Para adicionar múltiplos riscos de uma vez, você pode usar a funcionalidade de importação de CSV.

1.  Na tela de listagem de riscos, clique no botão "Importar Riscos (CSV)".
2.  Um modal aparecerá.
    [SCREENSHOT: Modal de Upload de CSV para Riscos]
3.  **Preparar o Arquivo CSV:**
    *   O arquivo CSV deve conter as seguintes colunas (cabeçalhos): `title`, `description`, `category`, `impact`, `probability`.
        *   `title` (obrigatório): Título do risco.
        *   `description` (opcional): Descrição.
        *   `category` (opcional): "tecnologico", "operacional", "legal". Se inválido ou ausente, assume "tecnologico".
        *   `impact` (obrigatório): "Baixo", "Médio", "Alto", "Crítico".
        *   `probability` (obrigatório): "Baixo", "Médio", "Alto", "Crítico".
    *   Você pode baixar um modelo de CSV na própria interface do modal.
4.  **Realizar o Upload:**
    *   Selecione seu arquivo CSV no modal.
    *   Clique em "Importar".
5.  **Resultados:** O sistema processará o arquivo. Você verá um resumo de quantos riscos foram importados com sucesso e quais linhas falharam (com os motivos do erro).

---

Este guia cobre as funcionalidades centrais do módulo de Gestão de Riscos. Explore a interface para descobrir mais detalhes e familiarize-se com o fluxo de trabalho da sua organização.
