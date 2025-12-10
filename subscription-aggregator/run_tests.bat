@echo off
chcp 65001 >nul
setlocal

echo.
echo 🐳 Запуск тестовой БД...
docker-compose -f docker-compose.test.yml up -d

timeout /t 10 >nul

echo 🚀 Запуск тестов...
go test -v ./e2e/...

echo 🧹 Очистка...
docker-compose -f docker-compose.test.yml down -v

pause