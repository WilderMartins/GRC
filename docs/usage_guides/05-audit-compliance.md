# Guia de Auditoria e Conformidade

Este guia explica como usar o módulo de Auditoria e Conformidade do Phoenix GRC para avaliar sua postura em relação a frameworks de segurança reconhecidos.

## 1. Acessando o Módulo

No menu de navegação principal (AdminLayout sidebar), selecione "Auditoria e Conformidade".

[SCREENSHOT: Menu de navegação principal (AdminLayout sidebar) com o item "Auditoria e Conformidade" destacado.]

## 2. Listagem de Frameworks de Auditoria

Você verá uma página (`/admin/audit/frameworks`) com cards ou uma lista dos frameworks de auditoria pré-carregados no sistema (ex: NIST CSF 2.0, ISO 27001:2022, CIS Controls v8).

[SCREENSHOT: Tela de Listagem de Frameworks de Auditoria (/admin/audit/frameworks) mostrando cards para cada framework disponível.]

Clique no nome de um framework para ver seus detalhes e controles.

## 3. Página de Detalhes do Framework e Controles

Ao selecionar um framework (ex: clicando no card do NIST CSF 2.0), você será levado à página de detalhes daquele framework (ex: `/admin/audit/frameworks/:frameworkId`). Esta página combina informações sobre o framework, o score de conformidade da sua organização, e a lista de controles do framework com suas avaliações.

[SCREENSHOT: Página de Detalhes do Framework (ex: NIST CSF 2.0), mostrando o nome do framework no topo, a seção de "Resumo da Conformidade" (com score geral, controles avaliados/totais, etc.), e abaixo, a tabela de controles.]

### 3.1. Resumo da Conformidade

No topo desta página, você encontrará o "Resumo da Conformidade" para o framework selecionado, mostrando:
*   **Score Geral de Conformidade:** Uma métrica percentual (%) que indica o quão conforme sua organização está com os controles daquele framework, baseado nas avaliações preenchidas.
*   **Controles Avaliados:** O número de controles que já possuem uma avaliação registrada, em relação ao total de controles do framework.
*   **Contagem por Status:** Número de controles Conformes, Não Conformes e Parcialmente Conformes.

[SCREENSHOT: Detalhe da seção "Resumo da Conformidade" na página de detalhes do framework, mostrando os cards com os scores e contagens.]

### 3.2. Lista de Controles e Avaliações

Abaixo do resumo, uma tabela exibe todos os controles pertencentes ao framework selecionado, juntamente com o status da avaliação de cada um para a sua organização.

[SCREENSHOT: Tabela de Controles na página de detalhes do framework, mostrando colunas como ID do Controle, Descrição, Família, Status da Avaliação (com indicador visual), Score, e Ações.]

*   **Colunas:** ID do Controle, Descrição, Família, Status da Avaliação, Score.
*   **Filtros:** Você pode filtrar a lista de controles por Família (ex: "Access Control", "Protect") e por Status da Avaliação (Conforme, Não Conforme, Parcialmente Conforme, Não Avaliado).
    [SCREENSHOT: Detalhe dos filtros por Família e Status da Avaliação acima da tabela de controles.]
*   **Ação "Avaliar" / "Editar Avaliação":** Para cada controle, um botão permite iniciar uma nova avaliação ou editar uma existente. Clicar neste botão abrirá um modal com o formulário de avaliação.

## 4. Realizando ou Atualizando uma Avaliação de Controle

Ao clicar em "Avaliar" ou "Editar Avaliação" para um controle específico na tabela:

1.  Um **modal com o Formulário de Avaliação** será exibido.
    [SCREENSHOT: Modal do Formulário de Avaliação de Controle, mostrando o ID do Controle sendo avaliado, e os campos: Status (select), Score (input number), Data da Avaliação (date input), URL de Evidência (text input), e Upload de Arquivo de Evidência (file input).]
2.  **Preencha (ou atualize) os Campos do Formulário:**
    *   **Status:** Selecione o status da avaliação (Conforme, Não Conforme, Parcialmente Conforme). (Obrigatório)
    *   **Score:** (Opcional) Insira um score numérico (0-100). Este pode ser inferido automaticamente pelo sistema com base no Status, ou você pode definir um valor específico.
    *   **Data da Avaliação:** Defina a data em que a avaliação foi realizada (por padrão, será a data atual).
    *   **URL de Evidência (Texto):** Se sua evidência estiver hospedada em um link externo (ex: SharePoint, Confluence, Google Drive), cole a URL aqui.
    *   **Upload de Arquivo de Evidência:** Se sua evidência for um arquivo (ex: PDF, imagem, DOCX, XLSX), você pode fazer o upload diretamente. Clique para selecionar o arquivo ou arraste e solte. O sistema fará o upload para o armazenamento seguro configurado (Google Cloud Storage ou Amazon S3).
        [SCREENSHOT: Detalhe do campo de upload de arquivo no formulário de avaliação, talvez mostrando um arquivo selecionado ou a área de arrastar e soltar.]
3.  Clique em **"Salvar Avaliação"** (ou similar).

A avaliação será registrada (ou atualizada, pois a API faz um "upsert") e contribuirá para o score geral de conformidade do framework. A tabela de controles na página de detalhes do framework será atualizada para refletir a nova avaliação.

## 5. Visualizando Evidências Anexadas

Na tabela de controles da página de detalhes do framework, se uma avaliação tiver uma URL de evidência ou um arquivo anexado, um link ou ícone para "Ver Evidência" poderá estar disponível ao lado da ação de avaliação.

*   Se for uma URL, o link abrirá a URL em uma nova aba.
*   Se for um arquivo anexado, o link permitirá visualizar ou baixar o arquivo (dependendo da configuração do armazenamento e do navegador).

[SCREENSHOT: Linha de um controle na tabela de detalhes do framework, mostrando um link/ícone "Ver Evidência" clicável ao lado do botão "Editar Avaliação".]

---

Manter as avaliações de controle atualizadas é fundamental para entender a postura de conformidade da sua organização, identificar lacunas e demonstrar o cumprimento dos requisitos dos frameworks de segurança.
