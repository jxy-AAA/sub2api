@echo off
set DATA_DIR=D:\projects\bot\sub2api\.runtime\data
set AUTO_SETUP=true
set SERVER_HOST=0.0.0.0
set SERVER_PORT=18080
set SERVER_MODE=debug
set DATABASE_HOST=127.0.0.1
set DATABASE_PORT=5432
set DATABASE_USER=postgres
set DATABASE_PASSWORD=postgres
set DATABASE_DBNAME=sub2api
set DATABASE_SSLMODE=disable
set REDIS_HOST=127.0.0.1
set REDIS_PORT=6379
set REDIS_PASSWORD=
set ADMIN_EMAIL=admin@sub2api.local
set ADMIN_PASSWORD=admin123456
set TZ=Asia/Shanghai
cd /d D:\projects\bot\sub2api\backend
call conda run -n bot --no-capture-output cmd.exe /d /c "set GOTOOLCHAIN=auto&&set GOPROXY=https://goproxy.cn,direct&&go run ./cmd/server"
