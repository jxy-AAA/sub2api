# 合规数据导出说明

本文档说明当前仓库中与合规审计、root 管理员 JSON 导出、以及 trace 导出任务相关的最小配置、状态机和验证方式。

## 1. 登录协议与导出前提

登录协议不是 `config.yaml` 环境项，而是数据库中的系统设置。当前约定的关键项包括：

- `login_agreement_enabled`
- `login_agreement_mode`
- `login_agreement_updated_at`
- `login_agreement_documents`

`service-specific-terms` 中应明确说明：

- 平台会按最小必要原则保留账户标识、请求时间、模型或渠道、请求/响应元数据，以及导出所需上下文。
- 用户同意协议后，`root` 管理员可在审计、申诉处理、监管配合或安全调查场景下导出留存数据。
- 导出格式为 JSON，访问必须遵循最小权限原则。

## 2. 现有同步导出接口

当前 root/admin 同步导出接口包括：

- `GET /api/v1/admin/accounts/data`
- `GET /api/v1/admin/proxies/data`

认证方式二选一：

- `Authorization: Bearer <admin-jwt>`
- `x-api-key: <admin-api-key>`

返回体中的 `data` 载荷固定包含：

- `type`
- `version`
- `exported_at`
- `accounts`
- `proxies`

## 3. Trace 异步导出任务

Trace 导出走异步任务接口：

- `GET /api/v1/admin/traces/export-tasks`
- `POST /api/v1/admin/traces/export-tasks`
- `GET /api/v1/admin/traces/export-tasks/:id`
- `POST /api/v1/admin/traces/export-tasks/:id/cancel`
- `GET /api/v1/admin/traces/export-tasks/:id/download`

这些接口要求 root 管理员 JWT，不接受 admin API key。

### 3.1 状态机

任务状态固定为：

- `pending`：任务已创建，等待后台执行器抢占
- `running`：执行器已开始导出，持续刷新 `total_records` / `processed_records`
- `succeeded`：导出文件写入完成，可下载
- `failed`：执行失败，`error_message` 记录失败原因
- `canceled`：root 管理员取消，任务停止执行

状态转移规则：

- `pending -> running`：执行器通过数据库原子抢占
- `running -> succeeded`：JSON 文件落盘并写回 `file_path` / `file_size_bytes`
- `running -> failed`：查询、编码、写盘、超限或超时失败
- `pending|running -> canceled`：root 管理员取消

### 3.2 导出数据源与过滤条件

任务数据源为 `model_trace_captures`。

支持的过滤条件与管理后台列表保持一致，包括：

- `model`
- `user_id`
- `api_key_id`
- `capture_rule_id`
- `start_time` / `end_time`
- `keyword`
- `min_*_tokens` / `max_*_tokens`

如果任务创建时未指定 `end_time`，执行器会用任务 `started_at` 作为导出快照上界，避免导出过程中新增 trace 持续进入结果集。

### 3.3 JSON 文件契约

当前 `format` 固定为 `json_array`，下载文件是一个 JSON 数组；数组元素来自 `ModelTraceCapture.Export(...)`。

默认字段包括：

- 基础标识：`id`, `task_id`, `request_id`, `response_id`
- 归属信息：`user_id`, `api_key_id`, `group_id`, `account_id`, `capture_rule_id`
- 模型信息：`protocol`, `model`, `requested_model`, `upstream_model`
- token / 状态：`input_tokens`, `output_tokens`, `total_tokens`, `upstream_status_code`
- 结构化内容：`system`, `user`, `tool`, `assistant`, `prompt`, `candidates`, `tools`, `signature`, `meta`
- 完整性字段：`dedupe_hash`, `prompt_hash`, `created_at`

`include_raw=false` 时，不包含以下字段：

- `raw_request`
- `raw_response`
- `raw_request_text`
- `raw_response_text`

`include_raw=true` 时，以上四个字段会一并导出。

## 4. 运行配置

新增配置段：

```yaml
trace_export:
  enabled: true
  export_dir: "./data/trace-exports"
  poll_interval_seconds: 15
  batch_size: 200
  task_timeout_seconds: 3600
  cleanup_batch_size: 50
  max_records_per_task: 100000
```

含义：

- `enabled`：后台执行器总开关
- `export_dir`：导出文件落盘目录
- `poll_interval_seconds`：轮询 `pending` 任务的间隔
- `batch_size`：单批读取 `model_trace_captures` 的页大小
- `task_timeout_seconds`：任务最大执行时长；超时的 `running` 任务会被标记为 `failed`
- 清理策略：后台任务每周按 UTC 自然周清理上周及更早的受管导出文件
- `cleanup_batch_size`：单次清理的文件元数据批量大小
- `max_records_per_task`：单任务允许导出的最大 trace 数

对应环境变量：

- `TRACE_EXPORT_ENABLED`
- `TRACE_EXPORT_EXPORT_DIR`
- `TRACE_EXPORT_POLL_INTERVAL_SECONDS`
- `TRACE_EXPORT_BATCH_SIZE`
- `TRACE_EXPORT_TASK_TIMEOUT_SECONDS`
- `TRACE_EXPORT_CLEANUP_BATCH_SIZE`
- `TRACE_EXPORT_MAX_RECORDS_PER_TASK`

## 5. 保留与清理

执行器会定期扫描已结束任务：

- 若 `file_path` 指向受管目录下的导出文件，且 `finished_at` 早于当前 UTC 自然周起点，则在每周清理时删除文件。
- 删除成功后会清空任务上的 `file_path` 和 `file_size_bytes`，保留任务元数据与状态审计信息。

因此，历史任务列表会保留，但过期任务的下载可能返回“文件不存在”。

## 6. 本地验证

后端单测：

```bash
cd backend
go test ./internal/config ./internal/repository ./internal/service ./internal/handler/admin -run "TraceExport|TraceExportTask"
```

重点覆盖：

- 配置默认值、环境变量和校验
- 任务状态推进：`pending -> running -> succeeded/failed/canceled`
- JSON schema contract：`include_raw` 开关前后字段集变化
- repository 原子状态变更与过期文件清理辅助方法
