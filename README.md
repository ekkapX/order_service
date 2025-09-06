# Система обработки заказов L0

![Go](https://img.shields.io/badge/Go-1.24-blue) ![Docker](https://img.shields.io/badge/Docker-Compose-blue) ![Kafka](https://img.shields.io/badge/Kafka-3.3-green) ![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-blue) ![Redis](https://img.shields.io/badge/Redis-7-red)

L0 — это система обработки заказов, написанная на Go, которая принимает заказы через Kafka, сохраняет их в PostgreSQL и кэширует в Redis. Проект предоставляет HTTP API и веб-интерфейс для получения деталей заказов. Данные сохраняются персистентно, а кэш восстанавливается после перезапуска сервера, что обеспечивает надёжность и быстрый доступ к данным.

## Возможности

- **Потребитель Kafka**: Обрабатывает заказы из топика `orders` и сохраняет их в PostgreSQL и Redis.
- **Хранилище PostgreSQL**: Персистентное хранение заказов, информации о доставке, оплате и товарах.
- **Кэш Redis**: Быстрый доступ к заказам через кэширование, с восстановлением из PostgreSQL при запуске.
- **HTTP API**: Эндпоинт `GET /order/:order_uid` для получения деталей заказа.
- **Веб-интерфейс**: Позволяет ввести `order_uid` и отобразить детали заказа в формате JSON.
- **Контейнеризация**: Полностью контейнеризировано с помощью Docker Compose.
- **Надёжность**: Использует `wait-for-it.sh` для ожидания готовности Kafka, исключая проблемы синхронизации.
- **Безопасность**: Настроен Gin в режиме `Release` с отключением доверия прокси для продакшен-безопасности.

## Архитектура

Система состоит из следующих компонентов:
- **Kafka**: Брокер сообщений для получения заказов.
- **PostgreSQL**: Персистентное хранилище для заказов, доставки, оплаты и товаров.
- **Redis**: Кэш в памяти для быстрого доступа к заказам.
- **Go-сервер**: Обрабатывает HTTP API и предоставляет веб-интерфейс.
- **Docker Compose**: Оркестрирует Kafka, Zookeeper, PostgreSQL, Redis и сервер.

## Требования

- [Docker](https://www.docker.com/get-started) и [Docker Compose](https://docs.docker.com/compose/install/)
- [Go](https://golang.org/dl/) 1.24 или выше (для локальной разработки)
- [Git](https://git-scm.com/downloads)

## Установка

1. **Клонируйте репозиторий**:
   ```bash
   git clone https://github.com/<your-username>/l0.git
   cd l0

2. **Скачайте wait-for-it.sh**:
   ```bash
    curl -o wait-for-it.sh https://raw.githubusercontent.com/vishnubob/wait-for-it/master/wait-for-it.sh
    chmod +x wait-for-it.sh

## Настройка и запуск

1. **Запустите сервисы с помощью Docker Compose**:
   ```bash
   docker-compose -f compose.yaml up -d --build

2. **Проверьте запущенные сервисы**:
   ```bash
   docker ps

3. **Откройте веб-интерфейс**:
   Перейдите по адресу http://localhost:8080 в браузере.
