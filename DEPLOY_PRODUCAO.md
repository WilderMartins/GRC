# Guia de Deploy em Produção - Phoenix GRC

Este guia fornece um checklist e instruções essenciais para implantar o Phoenix GRC em um ambiente de produção de forma segura e robusta.

## Checklist de Pré-Requisitos para Produção

Antes de colocar a aplicação no ar, garanta que os seguintes pontos foram atendidos:

- [ ] **Servidor de Produção:** Um servidor ou uma instância de nuvem (AWS EC2, Google Compute Engine, etc.) com Docker e Docker Compose instalados.
- [ ] **Domínio Registrado:** Um nome de domínio (ex: `grc.suaempresa.com`) apontando para o endereço IP do seu servidor.
- [ ] **Certificado SSL Válido:** Um certificado SSL/TLS para o seu domínio. Recomendamos o [Let's Encrypt](https://letsencrypt.org/), que é gratuito e amplamente aceito.
- [ ] **Solução de Backup:** Um plano e uma ferramenta para realizar backups regulares do banco de dados.
- [ ] **Serviço de E-mail Transacional:** Um serviço para enviar e-mails de notificação, como Amazon SES, SendGrid, etc.

---

## Passo 1: Configurando o Ambiente de Produção

### 1.1. Arquivo de Configuração `.env`

Copie o `.env.example` para `.env` e configure as seguintes variáveis para produção:

-   **`GIN_MODE`**: Mude para `release`.
    ```
    GIN_MODE=release
    ```
-   **`APP_ROOT_URL` e `FRONTEND_BASE_URL`**: Atualize para o seu domínio de produção, usando `https`.
    ```
    APP_ROOT_URL=https://grc.suaempresa.com
    FRONTEND_BASE_URL=https://grc.suaempresa.com
    ```
-   **`JWT_SECRET_KEY` e `ENCRYPTION_KEY_HEX`**: Certifique-se de que as chaves seguras geradas na primeira inicialização (ou novas chaves seguras) estão aqui. **Nunca use os valores padrão em produção.**

### 1.2. Permissões do Arquivo `.env`

É crucial restringir o acesso ao arquivo `.env` para que apenas o proprietário possa lê-lo.
```bash
chmod 600 .env
```

### 1.3. Gerenciamento de Segredos (Avançado)

Para segurança máxima, considere usar um serviço de gerenciamento de segredos como [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/), [Google Secret Manager](https://cloud.google.com/secret-manager), ou [HashiCorp Vault](https://www.vaultproject.io/). Isso evita armazenar segredos em arquivos de texto no servidor. A integração com essas ferramentas exigiria modificações no código da aplicação para buscar as variáveis de ambiente a partir desses serviços.

---

## Passo 2: Configurando HTTPS com Certificados Válidos

A configuração do Nginx está pronta para HTTPS, mas você precisa substituir os certificados autoassinados de desenvolvimento pelos seus certificados de produção.

1.  **Obtenha seus Certificados:**
    Use uma ferramenta como o [Certbot](https://certbot.eff.org/) para obter um certificado do Let's Encrypt. O Certbot pode automatizar a obtenção e a renovação dos certificados.

2.  **Substitua os Arquivos:**
    -   Substitua o arquivo `nginx/ssl/self-signed.crt` pelo seu arquivo de certificado de produção (geralmente `fullchain.pem`).
    -   Substitua o arquivo `nginx/ssl/self-signed.key` pela sua chave privada de produção (geralmente `privkey.pem`).

    **Atenção:** Os nomes dos arquivos devem corresponder aos que estão no `nginx/nginx.conf` (`self-signed.crt` e `self-signed.key`), ou você deve atualizar o `nginx.conf` para apontar para os novos nomes de arquivo.

3.  **Habilite o HSTS (Opcional, mas Recomendado):**
    Após confirmar que o HTTPS está funcionando perfeitamente, descomente a seguinte linha no seu `nginx/nginx.conf` para maior segurança:
    ```nginx
    add_header Strict-Transport-Security "max-age=15768000; includeSubDomains; preload" always;
    ```

---

## Passo 3: Estratégia de Backup do Banco de Dados

Os dados do PostgreSQL são o ativo mais crítico da aplicação. **Não confie apenas no volume do Docker.**

### Exemplo de Estratégia de Backup (Script Simples)

Você pode criar um script que roda diariamente via `cron` para fazer um dump do banco de dados e enviá-lo para um armazenamento em nuvem.

1.  **Crie um script `backup.sh`:**
    ```bash
    #!/bin/bash

    # Define variáveis
    DB_CONTAINER_NAME="phoenix_grc_db"
    DB_USER="admin"
    DB_NAME="phoenix_grc_dev"
    BACKUP_DIR="/path/to/your/backups"
    DATE=$(date +%Y-%m-%d_%H-%M-%S)

    # Cria o diretório de backup se não existir
    mkdir -p $BACKUP_DIR

    # Executa o dump do banco de dados
    docker exec $DB_CONTAINER_NAME pg_dump -U $DB_USER -d $DB_NAME | gzip > $BACKUP_DIR/phoenix_grc_backup_$DATE.sql.gz

    # (Opcional) Envia para a nuvem (ex: AWS S3)
    # aws s3 cp $BACKUP_DIR/phoenix_grc_backup_$DATE.sql.gz s3://your-s3-backup-bucket/

    # (Opcional) Remove backups locais antigos (ex: mais de 7 dias)
    find $BACKUP_DIR -type f -name "*.sql.gz" -mtime +7 -delete
    ```

2.  **Agende a execução com `cron`:**
    -   Abra o editor do cron: `crontab -e`
    -   Adicione uma linha para rodar o script todo dia às 2h da manhã:
        ```
        0 2 * * * /path/to/your/backup.sh
        ```

---

## Passo 4: Iniciando e Mantendo a Aplicação

1.  **Build e Start Inicial:**
    ```bash
    docker-compose build
    docker-compose up -d
    ```

2.  **Atualizações:**
    Para atualizar a aplicação com uma nova versão do código:
    ```bash
    # Baixa as últimas alterações
    git pull

    # Reconstrói as imagens e reinicia os serviços
    docker-compose build
    docker-compose up -d

    # Remove containers antigos e não utilizados
    docker image prune -f
    ```

3.  **Monitoramento:**
    -   Use o endpoint `/metrics` para integrar com um sistema de monitoramento como Prometheus e Grafana.
    -   Monitore os logs dos containers:
        ```bash
        docker-compose logs -f backend
        docker-compose logs -f nginx
        ```
