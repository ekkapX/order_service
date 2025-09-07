# Система обработки заказов L0

![Go](https://img.shields.io/badge/Go-1.24-blue) ![Docker](https://img.shields.io/badge/Docker-Compose-blue) ![Kafka](https://img.shields.io/badge/Kafka-3.3-green) ![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-blue) ![Redis](https://img.shields.io/badge/Redis-7-red)

L0 — это система обработки заказов, написанная на Go в рамках выполнения нулевого уровня Техношколы ВБ, которая принимает заказы через Kafka, сохраняет их в PostgreSQL и кэширует в Redis. Проект предоставляет HTTP API и веб-интерфейс для получения деталей заказов. Данные сохраняются персистентно, а кэш восстанавливается после перезапуска сервера, что обеспечивает надёжность и быстрый доступ к данным.

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
   git clone https://github.com/ekkapX/l0.git
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

## Работа с системой

### Отправка тестового заказа в Kafka
   Для тестирования отправки заказа используйте консоль Kafka:
   ```bash 
   docker exec -it l0-kafka kafka-console-producer --bootstrap-server kafka:9092 --topic orders
   ```

   Пример ввода
   ```json
   {
    "order_uid": "test456",
    "track_number": "TEST456",
    "order_entry": "test",
    "delivery": {
        "name": "John Doe",
        "phone": "+1234567890",
        "zip": "123456",
        "city": "Moscow",
        "adress": "Lenin St 10",
        "region": "Central",
        "email": "john@example.com"
    },
    "payment": {
        "transaction": "test456",
        "currency": "USD",
        "provider": "wbpay",
        "amount": 2000,
        "payment_dt": 1637907728,
        "bank": "sber",
        "delivery_cost": 1000,
        "goods_total": 1000,
        "custom_fee": 0
    },
    "items": [
        {
            "chrt_id": 9934931,
            "track_number": "TEST456",
            "price": 500,
            "rid": "item456",
            "name": "Lipstick",
            "sale": 20,
            "size": "0",
            "total_price": 400,
            "nm_id": 2389213,
            "brand": "Maybelline",
            "status": 202
        }
    ],
    "locale": "en",
    "customer_id": "test2",
    "delivery_service": "meest",
    "shardkey": "8",
    "sm_id": 98,
    "date_created": "2021-11-26T06:22:20Z",
    "oof_shard": "2"
   }
   ```

### Проверьте веб интерфейсы
- Откройте http://localhost:8080/ в браузере.
- Введите order_uid (например, test789, test123, test456) для просмотра деталей заказа.
- Введите неверный order_uid (например, invalid123) для проверки обработки ошибок.

### Проверьте API
```bash
curl http://localhost:8080/order/test789
curl http://localhost:8080/order/test456
```
Проверка на несуществующий заказ 
```bash
curl http://localhost:8080/order/invalid123
```
Ожидаемый ответ - {"error":"order not found"}

## Возможные улучшения
1. Добавить валидацию входных данных при обработке заказа
2. Добавить unit- и integration-тесты
3. Добавить Swagger/OpenAPI документацию к API

## Благодарности

- Проект создан как учебный для изучения распределительных систем и микросервисов
- Это мой первый подобный опыт и я рад, что довел его до работающего состояния.
- Спасибо за испытание, Техношкола WB!