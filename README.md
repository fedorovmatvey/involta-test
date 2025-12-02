# Involta Test Task

## Запуск
```bash
make up
# или docker-compose up --build
```
Сервис будет доступен по адресу: `http://localhost:8080`

## Тестирование
```bash
make tests
# или вручную: go test ./internal/service/... -v
```

## Документация (Swagger)
После запуска доступна здесь:  
[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
```