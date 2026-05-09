@echo off
set VITE_DEV_PROXY_TARGET=http://127.0.0.1:18080
set VITE_DEV_PORT=5173
cd /d D:\projects\bot\sub2api\frontend
call conda run -n bot --no-capture-output pnpm dev --host 0.0.0.0
