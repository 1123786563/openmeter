# Credit Reservation v2 切换与删除运行手册

本手册落实《OpenMeter 唯一计费真相源与 WeKnora Credit 调用编排设计》，并与 WeKnora 的 `docs/openmeter-credit-reservation-v2-cutover.md` 共同维护。Reservation v2 将 OpenMeter 作为费率、余额、授信、预占、结算和 Ledger 的唯一货币真相源；WeKnora 仅编排调用和投递幂等命令。

## 阶段 0 基线（2026-08-09）

以下快照在两个隔离 worktree 中取得；后续阶段不得把未解释的失败归入本任务。

| 仓库 | worktree | HEAD | 工作区状态 |
| --- | --- | --- | --- |
| OpenMeter | `/Users/wuyongjun/trea/openmeter/.worktrees/openmeter-authoritative-credit-flow` | `6c3ec47f7222cbde62cf945504f95fbb2ac4bdff` | clean |
| WeKnora | `/Users/wuyongjun/trea/WeKnora/.worktrees/openmeter-authoritative-credit-flow` | `cb432a092477fc13cb58640989b93312480420c0` | 存在既有用户修改；本阶段提交不包含这些文件 |

WeKnora 基线时的既有修改为：`.env.example`、`.env.lite.example`、`config/config.yaml`、`docs/superpowers/plans/2026-08-09-global-agent-template-management.md`、`docs/superpowers/specs/2026-08-09-openmeter-authoritative-credit-flow-design.md`、删除的 `docs/superpowers/specs/2026-08-09-openmeter-model-usage-admission-design.md`、`internal/application/repository/billing_openmeter_operations_test.go`、`internal/application/service/billing/openmeter_settlement_test.go`、`internal/config/config.go`、`internal/types/billing/runtime_journal.go`、`internal/types/model.go`、`description.md`、`internal/modelgateway/`、`internal/types/model_catalog.go`、`internal/types/model_invocation_audit.go`、`migrations/sqlite/000041_widen_openmeter_batch_id.{up,down}.sql`、`migrations/versioned/000123_widen_openmeter_batch_id.{up,down}.sql`。

## 基线验证记录

### OpenMeter

命令：

```bash
cd /Users/wuyongjun/trea/openmeter/.worktrees/openmeter-authoritative-credit-flow
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./openmeter/ledger/... ./openmeter/aiusage/... ./api/v3/handlers/aiusage/...
```

结果：退出码 `1`。通过或无测试的包包括 `openmeter/ledger`、`routingrules`、`openmeter/aiusage`、`meterregistry`、`pricing`、`runtimeauthorization`、`settlement`、`signing`、`worker` 及若干 `[no test files]` 包。已存在的失败输出如下：

```text
api/v3/handlers/aiusage/handler_test.go:132:3: not enough arguments in call to New
have (func(context.Context) (string, error), a iusage.Service, runtimeauthorization.Service, nil)
want (func(ctx context.Context) (string, error), aiusage.Service, runtimeauthorization.Service, CreditBalanceReader, ratecard.Service, ...httptransport.HandlerOption)
...
FAIL github.com/openmeterio/openmeter/api/v3/handlers/aiusage [build failed]
```

数据库依赖的 ledger 和 AI Usage adapter 测试还一致地失败于：

```text
failed to connect to `user=postgres database=`: 127.0.0.1:5436 (127.0.0.1): dial error: dial tcp 127.0.0.1:5436: connect: connection refused
```

另有已存在的 `openmeter/aiusage/service` 失败：`TestSettle_BYOKMixedBatch` 返回 `validation error: line_item[0]: validation error: provider must not be empty for provider-managed resource`。

### WeKnora

命令：

```bash
cd /Users/wuyongjun/trea/WeKnora/.worktrees/openmeter-authoritative-credit-flow
go test ./internal/openmeter/... ./internal/application/service/billing/... ./internal/models/...
```

结果：退出码 `1`；三个模式均在测试执行前被 workspace 拒绝：

```text
pattern ./internal/openmeter/...: directory prefix internal/openmeter does not contain modules listed in go.work or their selected dependencies
pattern ./internal/application/service/billing/...: directory prefix internal/application/service/billing does not contain modules listed in go.work or their selected dependencies
pattern ./internal/models/...: directory prefix internal/models does not contain modules listed in go.work or their selected dependencies
```

## 阶段门与负责人签字

- [ ] 阶段 0：两仓 HEAD、工作区清单和上述基线失败已由 OpenMeter 与 WeKnora 技术负责人确认。
- [ ] 阶段 1：CREDIT 路由、唯一 Price Book 和显式企业授信已由 OpenMeter 负责人验收。
- [ ] 阶段 2：Reservation API 的并发、幂等和崩溃注入验收已由 OpenMeter 负责人签字。
- [ ] 阶段 3：WeKnora ProviderCallBillingSession、DirectCharge 和 UNKNOWN 核对流程已由 WeKnora 负责人签字。
- [ ] 阶段 4：租户级 `billing_engine` Canary 证明每个 call 只使用一个扣费引擎，且由值班/财务负责人签字。
- [ ] 阶段 5：全量切换、旧 Pending 排空或人工关闭、旧 Worker 停止已由双方负责人签字。
- [ ] 阶段 6：不可逆删除门全部通过，删除批准人确认只能 roll-forward。

## 切换快照命令

切换和删除前在生产副本执行；将每条命令的时间、操作者、结果和证据链接附入变更单。

```sql
-- WeKnora: 每个收费 tenant 必须有显式 v2 engine；先归档完整清单，再确认违规清单为 0 行。
SELECT ea.tenant_id, COALESCE(te.engine, '<missing>') AS billing_engine, te.updated_at, te.updated_by
FROM billing_external_accounts AS ea
LEFT JOIN billing_tenant_engines AS te ON te.tenant_id = ea.tenant_id
ORDER BY ea.tenant_id;
SELECT ea.tenant_id, COALESCE(te.engine, '<missing>') AS billing_engine
FROM billing_external_accounts AS ea
LEFT JOIN billing_tenant_engines AS te ON te.tenant_id = ea.tenant_id
WHERE COALESCE(te.engine, '') <> 'openmeter_reservation_v2'
ORDER BY ea.tenant_id;

-- WeKnora: 旧 pending 状态，以及本地 provider-call 执行证据的状态/年龄。
-- receipt 不是货币真相；它只用于关联 call、provider request 和 OpenMeter Reservation。
SELECT status, count(*) FROM billing_pending_usage GROUP BY status;
SELECT state, count(*), min(created_at) FROM billing_provider_call_receipts GROUP BY state;

-- OpenMeter: 旧 outbox 和权威 v2 Reservation 货币状态/年龄。
SELECT count(*) FROM ai_usage_outboxes WHERE published_at IS NULL OR dead_letter_reason IS NOT NULL;
SELECT status, count(*), min(created_at) FROM credit_reservations GROUP BY status;

-- CREDIT 迁移逐客户对平必须由迁移作业输出并归档。
```

## 删除前备份与恢复验证

在隔离的恢复目标执行以下命令；恢复目标必须为空的专用数据库或临时 SQLite 文件，绝不能是生产数据库。连接 URL、备份目录和文件路径只通过受控环境变量提供，禁止把凭据写入变更单或 shell 历史。归档每份备份的 SHA-256、恢复时间、操作者和 `psql`/`sqlite3` 输出。

```bash
# OpenMeter PostgreSQL。
: "${OPENMETER_DATABASE_URL:?set a secret-managed source database URL}"
: "${OPENMETER_RESTORE_DATABASE_URL:?set a dedicated empty restore database URL}"
: "${OPENMETER_BACKUP:?set an absolute backup file path}"
pg_dump --dbname="$OPENMETER_DATABASE_URL" --format=custom --file="$OPENMETER_BACKUP"
shasum -a 256 "$OPENMETER_BACKUP"
pg_restore --list "$OPENMETER_BACKUP" >/dev/null
pg_restore --dbname="$OPENMETER_RESTORE_DATABASE_URL" --clean --if-exists --exit-on-error "$OPENMETER_BACKUP"
psql "$OPENMETER_RESTORE_DATABASE_URL" -v ON_ERROR_STOP=1 -Atqc 'SELECT count(*) FROM credit_reservations;'

# WeKnora PostgreSQL。
: "${WEKNORA_DATABASE_URL:?set a secret-managed source database URL}"
: "${WEKNORA_RESTORE_DATABASE_URL:?set a dedicated empty restore database URL}"
: "${WEKNORA_POSTGRES_BACKUP:?set an absolute backup file path}"
pg_dump --dbname="$WEKNORA_DATABASE_URL" --format=custom --file="$WEKNORA_POSTGRES_BACKUP"
shasum -a 256 "$WEKNORA_POSTGRES_BACKUP"
pg_restore --list "$WEKNORA_POSTGRES_BACKUP" >/dev/null
pg_restore --dbname="$WEKNORA_RESTORE_DATABASE_URL" --clean --if-exists --exit-on-error "$WEKNORA_POSTGRES_BACKUP"
psql "$WEKNORA_RESTORE_DATABASE_URL" -v ON_ERROR_STOP=1 -Atqc 'SELECT count(*) FROM billing_tenant_engines;'

# WeKnora SQLite。恢复文件必须是不存在的临时文件。
: "${WEKNORA_SQLITE_DB:?set the source SQLite database path}"
: "${WEKNORA_SQLITE_BACKUP:?set an absolute backup file path}"
: "${WEKNORA_SQLITE_RESTORE_DB:?set an absolute nonexistent restore file path}"
sqlite3 "$WEKNORA_SQLITE_DB" ".backup $WEKNORA_SQLITE_BACKUP"
shasum -a 256 "$WEKNORA_SQLITE_BACKUP"
sqlite3 "$WEKNORA_SQLITE_BACKUP" 'PRAGMA integrity_check; PRAGMA foreign_key_check;'
sqlite3 "$WEKNORA_SQLITE_BACKUP" ".backup $WEKNORA_SQLITE_RESTORE_DB"
sqlite3 "$WEKNORA_SQLITE_RESTORE_DB" 'PRAGMA integrity_check; PRAGMA foreign_key_check;'
```

## 不可逆删除门

- [ ] 所有收费租户的 `billing_engine` 为 `openmeter_reservation_v2`
- [ ] 旧 `billing_pending_usage` 无 ready/accepted/unresolved 行
- [ ] 旧 OpenMeter `ai_usage_outboxes` 无未发布或 dead-letter 行
- [ ] ACTIVE/EXECUTING/UNKNOWN v2 Reservation 已核对且无超龄异常
- [ ] CREDIT 迁移逐客户对平为 0
- [ ] 已创建并验证 OpenMeter PostgreSQL 删除前备份
- [ ] 已创建并验证 WeKnora PostgreSQL/SQLite 删除前备份
- [ ] 回滚负责人明确接受删除后只能 roll-forward

删除门通过后，按外键依赖顺序删除旧 `ai_usage_*` 表、旧 API/Worker 和 WeKnora 旧 Runtime Journal 热路径。保留历史迁移和原生 Ledger、Credit Grant、Purchase、Customer、支付记录。
