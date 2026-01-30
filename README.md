# Система обработки заказов 

![Go](https://img.shields.io/badge/Go-1.25-blue) ![Docker](https://img.shields.io/badge/Docker-Compose-blue) ![Kafka](https://img.shields.io/badge/Kafka-3.3-green) ![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-blue) ![Redis](https://img.shields.io/badge/Redis-7-red)

Order Service - это высокопроизводительный микросервис обработки заказов, написанный на Go с использованием принципов чистой архитектуры. Сервис принимает заказы через Kafka, валидирует их, сохраняет в PostgreSQL и кэширует в Redis для мгновенного доступа.

## Ключевые возможности

- **Чистая архитектура**: Четкое разделение на слои (Domain, Application, Infrastructure).
- **Kafka**: Реализован полноценный продюсер, который генерирует валидные и невалидные данные в Kafka, из которой читает консьюмер
- **Надежность**:
  - **Graceful Shutdown**: Корректное завершение работы сервера и консьюмеров.
  - **Restore Cache**: Автоматическое восстановление кэша из БД при старте сервиса.
  - **Retry Policy**: Повторные попытки при временных сбоях БД.
- **Валидация**: Строгая валидация входящих данных (структура, email, форматы телефонов).
- **Производительность**: Оптимизированные SQL-запросы с использованием индексов.
- **Инфраструктура**: Полная контейнеризация через Docker Compose.
- **Тесты**: Unit и интеграционные тесты.

## Архитектура

1.  **Domain**: Доменные сущности и интерфейсы репозиториев.
2.  **Application**: Бизнес-логика (сохранение заказа, получение, валидация).
3.  **Infrastructure**: Реализация работы с БД (Postgres), Кэшем (Redis), Брокером (Kafka) и HTTP (Gin)

## API Эндпоинты

Сервис предоставляет HTTP API:

- `GET /order/:order_uid` — Получить заказ по ID (из кэша или БД).

## Требования

- Docker & Docker Compose
- Golang 1.25

## Установка

1. **Клонируйте репозиторий**:
   ```bash
   git clone https://github.com/ekkapX/order_service.git
   cd order_service


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

### 1. Отправка заказа в Kafka
   Для отправки заказа используйте консоль Kafka:
   ```bash 
   docker exec -it l0-kafka kafka-console-producer --bootstrap-server kafka:9092 --topic orders
   ```

   Пример ввода
   ```json
   {
    "order_uid": "test456",
    "track_number": "TEST456",
    "entry": "test",
    "delivery": {
        "name": "John Doe",
        "phone": "+1234567890",
        "zip": "123456",
        "city": "Moscow",
        "address": "Lenin St 10",
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

## В ближайших планах (TODO)
1. Реализовать DLQ
2. Добавить Swagger/OpenAPI документацию к API
3. Добавить трейсинг и метрики
