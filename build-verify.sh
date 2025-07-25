#!/bin/bash

# Este script executa um build limpo de todos os serviços Docker.
# Ele primeiro remove todos os caches de build para garantir que estamos
# construindo do zero, e então tenta construir as imagens.

set -e

echo "------------------------------------------------"
echo "Limpando o cache do Docker Builder..."
echo "------------------------------------------------"
sudo docker builder prune -a -f

echo ""
echo "------------------------------------------------"
echo "Iniciando o build dos serviços (sem cache)..."
echo "------------------------------------------------"
sudo docker compose build --no-cache

echo ""
echo "------------------------------------------------"
echo "Build concluído com sucesso!"
echo "------------------------------------------------"
