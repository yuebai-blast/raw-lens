# 随机 id + 通用确认弹框 设计文档

日期：2026-06-23

## 目标

两个改动：

1. 请求记录不再用自增整数 id，改用随机字符串 id。
2. 删除确认从原生 `window.confirm` 换成 Vue 确认弹框组件，并做成通用可复用。

## 一、随机 id

### Schema

`captured_request` 拆成内部序号 + 对外标识两列：

- `seq INTEGER PRIMARY KEY AUTOINCREMENT`：内部序号，**仅**用于排序与保留策略，不对外暴露。
- `id TEXT NOT NULL UNIQUE`：对外公开标识，12 位小写 hex（`crypto/rand` 6 字节 → hex）。

`selectCols` 保持不含 `seq`（对外查询不需要）；新建表带 `UNIQUE(id)`（自带索引）。去掉原 `idx_captured_request_id_desc`（排序改走主键 `seq`）。

### store（`internal/store/store.go`）

- `CapturedRequest.ID` 由 `int64` 改为 `string`。
- 新增 `newID()`：`crypto/rand` 取 6 字节 → `hex.EncodeToString` 得 12 位小写 hex。
- `Add`：生成随机 id 后 INSERT；若命中 `id` 的 UNIQUE 冲突，重新生成重试（最多若干次兜底，正常永不触发）。返回 string id（出错返回空串）。
- `List`：`ORDER BY seq DESC`，仍反转成旧→新返回（语义不变）。
- `prune`：改按 `seq` 算——`DELETE ... WHERE seq <= (SELECT MAX(seq)) - max`。
- `Get(id string)` / `SetName(id string, name string)` / `Delete(id string)`：参数改 string，`WHERE id = ?`。
- `scanRow`：`cr.ID` 扫描为 string。

### 迁移（老库平滑升级，不丢数据）

`New()` 建表后按顺序执行两步迁移：

1. **补 name 列**（沿用已有逻辑）：缺 `name` 列则 `ALTER TABLE captured_request ADD COLUMN name TEXT NOT NULL DEFAULT ''`。先做这步，保证后续重建的 SELECT 能安全引用 `name`。
2. **重建为随机 id 结构**：检测缺 `seq` 列（即旧的 `id INTEGER PRIMARY KEY AUTOINCREMENT` 结构）时，在事务内重建表：
   - 建新表（`seq` 自增主键 + `id TEXT UNIQUE` + 其余列）；
   - `INSERT INTO 新表 (id, captured_at, ...) SELECT lower(hex(randomblob(6))), captured_at, ... FROM captured_request ORDER BY id`——按旧 id 顺序灌入，`seq` 自然续上，旧数据保留、时序不变，随机 id 由 SQLite `randomblob(6)` 现场生成（与 Go 端同为 12 位 hex）；
   - `DROP TABLE captured_request`；`ALTER TABLE 新表 RENAME TO captured_request`。

   已是新结构（有 `seq` 列）则跳过。

### dashboard（`internal/dashboard/dashboard.go`）

- `summaryDTO.ID` 改为 `string`（`detailDTO` 继承）。
- `GET/PATCH/DELETE /api/requests/{id}` 不再 `strconv.ParseInt`，直接 `r.PathValue("id")` 取字符串。
- 结构性「bad id → 400」语义取消：任意非空 id 串都是合法形状，查不到即 `GET` 返回 404、`PATCH/DELETE` 无操作返回 204。相应去掉原 `TestPatchBadID`，改测 PATCH 未知 id → 204。

### 其它调用方

- `internal/capture` 与 `cmd/rawlens` 中接收 `store.Add` 返回值（原 `int64`）的地方改为 `string`（如日志打印请求 id）。

## 二、通用确认弹框

- 新增 `frontend/src/components/ConfirmDialog.vue`：纯展示组件，`Teleport` 到 body，遮罩 + 终端风格面板，props `{ open, title, message, confirmText?, cancelText? }`，emits `confirm` / `cancel`；ESC 取消、Enter 确认、点遮罩取消。
- 新增 `frontend/src/stores/confirm.ts`（Pinia）：暴露 `confirm({ title, message }): Promise<boolean>`，内部持有 `{ open, title, message, resolve }`；`resolve(true/false)` 后关闭。
- `App.vue` 挂一个 `<ConfirmDialog>`，绑定 confirm store 状态，`@confirm`/`@cancel` 调 store 的 resolve。
- 调用方：
  - `SignalLog.vue` 删除：`if (await confirm({ title: '删除记录', message: '确认删除记录 #<id>？' })) await store.remove(id)`。
  - `Masthead.vue` PURGE：`if (await confirm({ title: '清空全部', message: '确认清空所有抓包记录？' })) await store.clear()`（原为无确认直接清空）。

## 三、前端类型与 store 跟随

- `types/api.ts`：`Summary.id` 改 `string`。
- `stores/captures.ts`：`activeId: string | null`、`newIds`/`knownIds` 改 `Set<string>`、`fetchDetail`/`setName`/`remove` 参数改 `string`。fetch URL 拼接对字符串 id 天然适用。
- `LogItem.vue` 仍显示 `#{{ id }}`（现在是随机串）。

## 测试

- Go store：id 随机且唯一、`ORDER BY seq` 时序正确、`prune` 按 seq、`SetName`/`Delete` 用 string id、**重建迁移**（旧 int-id 库升级后数据条数与时序保留、id 变为 12 位 hex 串、可继续 Add/SetName/Delete）。
- Go dashboard：路由用 string id；PATCH 未知 id → 204；GET 未知 id → 404。
- 前端：所有 fixture 的 id 改字符串；`captures.spec` 跟随；新增 `ConfirmDialog` 组件测试（open 时渲染、点确认/取消 emit 对应事件）与 `confirm` store 测试（`confirm()` 返回 Promise，resolve 后关闭）。

## 不做（YAGNI）

- 不保留数字序号展示习惯。
- 确认弹框只做确认/取消，不扩展多按钮、输入框等。
