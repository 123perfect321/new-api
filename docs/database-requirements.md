# new-api 网关 · 数据库需求文档

> 版本：v1.0
> 适用对象：数据库/运维同事、网关（我）开发
> 数据来源：本项目 `model/` 下的 GORM 模型 + 本地实际迁移出的完整 schema（38 张表）

---

## 1. 概述

new-api 网关是一个**无状态**服务，所有持久化数据落在数据库。按读写特征分为三类存储：

| 存储 | 选型 | 用途 | 是否必须 |
|---|---|---|---|
| 主数据库 | **PostgreSQL** | 用户/渠道/令牌/额度/配置等业务数据 | 必须 |
| 日志库 | **ClickHouse**（流量小可先共用 PG） | 每次模型调用的使用日志/审计日志 | 必须（可独立） |
| 缓存 | **Redis** | 令牌/渠道缓存、限流、会话、分布式锁 | 多实例必须 |
| 对象存储 | MinIO / S3（可选） | 文件/图片上传 | 推荐 |

本机开发用的是 SQLite（`SQLITE_PATH`），**上线前必须切换 PostgreSQL + Redis**，否则只能单实例运行。

---

## 2. 数据库选型与架构

### 2.1 主库 PostgreSQL

- 版本：PostgreSQL 15/16（与 Mars-Engine 同栈）
- 高可用：主从 + 自动故障切换（Patroni，或云托管 RDS）
- 备份：每日全量 + WAL 归档（PITR）
- 连接配置：`SQL_DSN=postgresql://user:password@host:5432/new-api`
- 隔离：与 Mars-Engine 复用集群，但**独立 database**（`new-api`）

### 2.2 日志库 ClickHouse

- 用途：`logs`、`audit_logs`、`perf_metrics` 等高写入表
- 连接配置：`LOG_SQL_DSN=clickhouse://user:password@host:9000/new_api_logs`
- 流量小可先不设 `LOG_SQL_DSN`（日志落主库），量上来再迁

### 2.3 Redis

- 用途：令牌缓存、渠道缓存、限流计数、会话、分布式锁、自动重试
- 连接配置：`REDIS_CONN_STRING=redis://:password@host:6379/0`
- 不配 Redis 只能用进程内缓存 → **只能单实例**

### 2.4 数据库用户/权限规划（给数据库同事）

| 数据库 | 建议账号 | 权限 |
|---|---|---|
| `new-api`（主库） | `newapi_app` | 仅本库 DML/DDL（GORM 迁移需要建表权限） |
| `new_api_logs`（日志库） | `newapi_log` | 仅本库 DML/DDL |
| Redis | 独立密码 | DB index 0 给 new-api |

---

## 3. 表结构总览

共 **38 张表**，按业务域分组如下：

| 业务域 | 表 |
|---|---|
| 用户与认证（13） | users, user_sessions, user_access_tokens, user_oauth_bindings, external_identity_claims, passkey_credentials, two_fas, two_fa_backup_codes, auth_flows, login_encryption_keys, custom_oauth_providers, authz_roles, casbin_rule |
| 渠道与模型（4） | channels, models, vendors, abilities |
| 令牌与计费（12） | tokens, quota_data, checkins, redemptions, top_ups, subscription_plans, subscription_orders, user_subscriptions, subscription_pre_consume_records, prefill_groups |
| 日志与监控（3） | logs, audit_logs, perf_metrics |
| 任务与异步（5） | tasks, system_tasks, system_task_locks, task_plugins, midjourneys |
| 系统配置（3） | options, setups, system_instances |

> 说明：GORM 会在启动时**自动建表/迁移**（AutoMigrate），无需手工执行 DDL。本文档字段结构用于理解数据模型、做容量/权限/索引规划。

---

## 4. 完整字段定义

### 4.1 用户与认证

#### users（用户）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | 自增主键 |
| username | text | 用户名（唯一 `uni_users_username`） |
| password | text | 密码（哈希） |
| display_name | text | 显示名 |
| role | int | 角色（1 普通用户，100 管理员） |
| status | int | 状态（1 启用） |
| email / github_id / discord_id / oidc_id / wechat_id / telegram_id / linux_do_id | text | 各 SSO/第三方登录标识 |
| access_token | char(32) | 个人访问 token（旧） |
| quota / used_quota | int | 额度 / 已用额度 |
| request_count | int | 请求次数 |
| group | varchar(64) | 分组（部门），默认 `default` |
| aff_code / aff_count / aff_quota / aff_history / inviter_id | - | 邀请返利相关 |
| setting | text | 用户级设置 JSON |
| remark | varchar(255) | 备注 |
| deleted_at | datetime | 软删除 |
| created_at / last_login_at / auth_version | bigint | 时间戳/认证版本 |

#### user_sessions（登录会话）

| 字段 | 类型 | 说明 |
|---|---|---|
| sid | varchar(64) PK | 会话 ID |
| user_id | int | 用户 ID |
| version / user_auth_version | bigint | 会话版本 / 用户认证版本（踢下线用） |
| status | varchar(16) | 会话状态 |
| refresh_hash / previous_refresh_hash | char(64) | refresh token 哈希 |
| login_method | varchar(32) | 登录方式 |
| ip / user_agent | - | 来源 |
| created_at / last_active_at / expires_at / revoked_at | bigint | 时间戳 |

#### user_access_tokens（个人访问令牌 PAT）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | |
| user_id | int | |
| name | varchar(64) | 令牌名 |
| token_hash | varchar(64) | 令牌哈希（不存明文） |
| token_hint | varchar(8) | 令牌前缀提示 |
| scopes | varchar(2048) | 权限范围 |
| expires_at / last_used_at / last_used_ip / created_at | bigint | 时间戳 |

#### user_oauth_bindings / external_identity_claims（OAuth/SSO 绑定）

- `user_oauth_bindings`：user_id + provider_id + provider_user_id，绑定第三方账号
- `external_identity_claims`：provider + subject + user_id，SSO 身份映射（OIDC 登录用）

#### passkey_credentials（WebAuthn/Passkey）

| 字段 | 说明 |
|---|---|
| user_id, rp_id, credential_id, public_key, attestation_type, aa_guid, sign_count, transports, attachment, last_used_at | Passkey 凭证 |

#### two_fas / two_fa_backup_codes（TOTP 两步验证）

- `two_fas`：user_id + secret + is_enabled + failed_attempts + locked_until（唯一 `uni_two_fas_user_id`）
- `two_fa_backup_codes`：user_id + code_hash + is_used

#### auth_flows（认证流程）

| 字段 | 说明 |
|---|---|
| token_hash, purpose, provider, intent, user_id, session_id, payload, created_at, expires_at, consumed_at | 一次性认证流程（登录/绑定） |

#### login_encryption_keys（登录加密密钥）

- slot + private_key_pem，用于前端登录密码加密

#### custom_oauth_providers（自定义 OAuth/OIDC 提供商）

| 字段 | 说明 |
|---|---|
| name, slug（唯一）, icon, enabled | 标识 |
| client_id, client_secret, authorization_endpoint, token_endpoint, user_info_endpoint, well_known, scopes, auth_style | OAuth/OIDC 端点 |
| user_id_field, username_field, display_name_field, email_field | 字段映射（JSONPath） |
| access_policy, access_denied_message | 准入策略（JSON） |

#### authz_roles / casbin_rule（授权）

- `authz_roles`：角色定义（key/name/description/built_in/enabled/sort）
- `casbin_rule`：Casbin 策略规则（ptype/v0~v5）

---

### 4.2 渠道与模型

#### channels（渠道，核心表）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | |
| type | int | 渠道类型（1 OpenAI, 4 Ollama, ...） |
| key | text | API Key（加密存储） |
| base_url | text | 代理地址 |
| models | text | 渠道可用模型（逗号分隔） |
| group | varchar(64) | 分组，默认 `default` |
| name / status / weight / priority / tag / remark | - | 名称/状态/权重/优先级/标签/备注 |
| test_model | text | 测试模型 |
| model_mapping | text | 模型映射 |
| balance / used_quota / balance_updated_time | - | 余额相关 |
| response_time / test_time | int | 响应耗时/最近测试时间 |
| auto_ban | int | 自动禁用开关 |
| status_code_mapping / param_override / header_override / setting / other_info / channel_info | json/text | 高级配置 |

#### models（模型列表）

| 字段 | 说明 |
|---|---|
| model_name（唯一 `uk_model_name_delete_at`）, description, icon, tags, vendor_id, endpoints, status, sync_official, name_rule, deleted_at | 模型元数据 |

#### vendors（模型厂商）

- name, description, icon, status, deleted_at

#### abilities（分组可用模型）

- group + model + channel_id（联合主键）+ enabled + priority + weight + tag

---

### 4.3 令牌与计费

#### tokens（API 令牌）

| 字段 | 说明 |
|---|---|
| id, user_id, key（唯一 `idx_tokens_key`）, status, name | 基础 |
| created_time, accessed_time, expired_time | 时间戳 |
| remain_quota, unlimited_quota, used_quota | 剩余额度/是否不限/已用 |
| model_limits_enabled, model_limits | 模型级限额 |
| allow_ips, group, cross_group_retry, auto_groups | IP 白名单/分组/跨组 |
| deleted_at | 软删除 |

#### quota_data（额度消耗明细）

| 字段 | 说明 |
|---|---|
| user_id, username, model_name, created_at, use_group, token_id, channel_id, node_name | 归属 |
| token_used, count, quota | token 数/次数/配额 |
| cache_read_tokens, cache_creation_tokens | 缓存读写 token |

#### checkins / redemptions / top_ups（签到/兑换码/充值）

- `checkins`：user_id + checkin_date + quota_awarded
- `redemptions`：key + quota + status + redeemed_time + used_user_id + expired_time
- `top_ups`：amount + money + trade_no（唯一）+ payment_method + status

#### subscription_*（订阅）

- `subscription_plans`：订阅计划（title, price_amount, currency, duration_unit, quota_reset_period, ...）
- `subscription_orders`：订阅订单（trade_no 唯一, payment_provider, status）
- `user_subscriptions`：用户订阅（plan_id, amount_total/used, start/end_time, status, upgrade_group, downgrade_group）
- `subscription_pre_consume_records`：预扣费记录

#### prefill_groups（预填分组）

- name + type + items(json) + description

---

### 4.4 日志与监控（重点：体量最大）

#### logs（使用日志，全库最大表）

| 字段 | 说明 |
|---|---|
| id, user_id, created_at, type | 主键/用户/时间/类型 |
| content | 日志内容 |
| username, token_name, model_name, channel_id, channel_name, group | 归属信息 |
| quota, prompt_tokens, completion_tokens, use_time, is_stream | 用量 |
| token_id, ip, request_id, upstream_request_id | 追踪 |
| other | 扩展 |

> **这是全库最大的表**：每次模型调用写入 1 行。企业级必须走 `LOG_SQL_DSN` 到 ClickHouse。索引见第 6 节。

#### audit_logs（审计日志）

| 字段 | 说明 |
|---|---|
| event_id, user_id, username, actor_role, category, action, auth_method, ip, user_agent, method, route, status, success, request_id, content, other | 安全审计事件 |

#### perf_metrics（性能指标）

- model_name, group, bucket_ts（时间桶）, request_count, success_count, total_latency_ms, ttft_sum_ms, output_tokens, generation_ms

---

### 4.5 任务与异步

#### tasks / system_tasks / system_task_locks（任务）

- `tasks`：异步任务（task_id, platform, user_id, group, channel_id, action, status, progress, data/json）
- `system_tasks`：系统任务（task_id, type, status, payload, state, result, error）
- `system_task_locks`：分布式锁（type 主键 + task_id + locked_by + locked_until）

#### task_plugins（任务插件）

- key, api_version, version, source, source_hash, enabled, active

#### midjourneys（MJ 绘图任务）

- code, user_id, action, mj_id, prompt, state, status, progress, image_url, quota, ...

---

### 4.6 系统配置

#### options（KV 配置，核心表）

- key（主键）+ value。存所有系统配置（ModelRatio、GroupRatio、站点设置等），value 是字符串/JSON。

#### setups（初始化记录）

- version + initialized_at

#### system_instances（多实例注册）

- node_name（主键）+ info + started_at + last_seen_at

---

## 5. 类型映射（SQLite → PostgreSQL）

GORM 迁移到 PostgreSQL 时类型变化如下，供数据库同事参考：

| SQLite | PostgreSQL |
|---|---|
| integer | BIGINT / INTEGER |
| text | TEXT |
| varchar(n) | VARCHAR(n) |
| char(n) | CHAR(n) |
| numeric（布尔） | BOOLEAN |
| real | DOUBLE PRECISION |
| json | JSONB |
| datetime | TIMESTAMP |
| bigint | BIGINT |
| decimal(10,6) | NUMERIC(10,6) |

---

## 6. 索引规划

现有索引（GORM 已建）：

- `users`：username 唯一
- `tokens`：key 唯一
- `channels`：name、tag
- `logs`：created_at+id、model_name、username、username+model_name、ip、request_id、upstream_request_id、user_id、group、token_id、channel_id、token_name、type+created_at 等
- `models`：model_name+deleted_at 唯一、vendor_id、deleted_at
- 其余见各表 DDL 中的 `CREATE INDEX`

**建议补充（企业级）**：

- `logs` 表若落 PostgreSQL，按 `created_at` 做**时间分区/分表**（或直接迁 ClickHouse）。
- `quota_data`、`audit_logs` 按 `created_at` 建索引（如量大）。
- 定期清理过期数据（日志、审计、会话、一次性 auth_flows）。

---

## 7. 容量估算（参考）

| 表 | 特征 | 量级 |
|---|---|---|
| logs | 每请求 1 行 | 日均 100 万调用 ≈ 300~500 MB/天，一年 100~180 GB |
| audit_logs | 管理操作 | 远小于 logs |
| quota_data | 每请求 1 行（若开启） | 与 logs 同量级 |
| users/tokens/channels/options | 配置类 | 几万行以内，几 MB~几十 MB |
| perf_metrics | 定时聚合 | 中等 |

> **结论**：`logs`（及可能的 `quota_data`）是唯一需要重点规划的写入热点，其余表都可忽略容量。

---

## 8. 部署 checklist（给运维）

1. 创建 PostgreSQL 库 `new-api` + 账号 `newapi_app`（建表权限）。
2. （可选）创建 ClickHouse 库 `new_api_logs` + 账号 `newapi_log`。
3. 准备 Redis（带密码），DB index 0 给 new-api。
4. 配置环境变量：

```env
SQL_DSN=postgresql://newapi_app:xxx@pg-host:5432/new-api
LOG_SQL_DSN=clickhouse://newapi_log:xxx@ch-host:9000/new_api_logs   # 可选
REDIS_CONN_STRING=redis://:xxx@redis-host:6379/0
SESSION_SECRET=<随机 64 位>
CRYPTO_SECRET=<随机 64 位>   # 渠道 key 等敏感字段加密用，必须固定
```

5. 首次启动：new-api 自动迁移建表 + 初始化 `options`（含默认 ModelRatio 等）。
6. 备份策略：主库每日全量 + WAL PITR；日志库按需保留（如 30~90 天）。

---

## 9. 注意事项

1. **不要用 SQLite 上生产**：文件锁、单实例、并发弱。
2. **`CRYPTO_SECRET` / `SESSION_SECRET` 必须固定且保密**：换了会导致渠道 key 解不开、所有会话失效。
3. **`logs` 必须分离**：这是唯一能把主库打爆的表。
4. 渠道 `key`、订阅/支付等敏感字段已加密存储（依赖 CRYPTO_SECRET），迁移库时不能只拷贝裸数据，需保持同一 CRYPTO_SECRET。
5. 多实例部署要求 Redis + PostgreSQL 都可用，且 `NODE_NAME` 每实例唯一（`system_instances` 用）。