# Тестовое задание в компанию Infotecs

Необходимо разработать приложение EWallet реализующее систему обработки транзакций платёжной системы. Приложение должно быть реализовано в виде HTTP сервера, реализующее REST API. Сервер должен реализовать 4 метода и их логику:

- Создание кошелька
- Перевод средств с одного кошелька на другой
- Получение истории входящих и исходящих транзакций
- Получение текущего состояния кошелька

Подробное описание в файле openapi.yaml

## Запуск

Выполнить скрипт createdb.sql:

```sql
psql -U <user> -d <database> -h localhost -p 5432 -a -f createdb.sql
```

Запустить сервер:

```bash
DB_URL="postgres://<user>:<password>@localhost:5432/<database>" go run main.go
```

Для проверки работоспособности подготовлен файл test.sh с curl-запросами.

```bash
./test.sh
```

## TODO

- Тесты
- Docker
