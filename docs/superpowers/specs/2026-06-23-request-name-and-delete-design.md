# 请求命名 + 单条删除 设计文档

日期：2026-06-23

## 目标

给抓包列表加两个小功能：

1. 可以给每条请求记录单独设置一个名称（备注）。
2. 可以单独删除某一条请求记录。

## 数据层（`internal/store`）

- `captured_request` 表新增列 `name TEXT NOT NULL DEFAULT ''`，comment：用户给该请求起的备注名（可空，默认空串）。
- **迁移**：`New()` 在执行建表 `schema` 后，查 `PRAGMA table_info(captured_request)`，若不含 `name` 列则执行 `ALTER TABLE captured_request ADD COLUMN name TEXT NOT NULL DEFAULT ''`，让老库平滑升级。
- `CapturedRequest` 结构体新增字段 `Name string`。
- `selectCols`、`scanRow`、`Add` 的 `INSERT` 同步加上 `name`（新抓到的请求 name 为空串）。
- 新增方法：
  - `SetName(id int64, name string)`：`UPDATE captured_request SET name = ? WHERE id = ?`，出错记日志。
  - `Delete(id int64)`：`DELETE FROM captured_request WHERE id = ?`，出错记日志（不存在不视为错误，幂等）。

## API 层（`internal/dashboard`）

- `PATCH /api/requests/{id}`，请求体 `{"name":"..."}`。名称做去首尾空格 + 上限 200 字符截断，再调 `st.SetName` → 返回 204。id 非法返回 400。
- `DELETE /api/requests/{id}` → 调 `st.Delete` → 返回 204（记录不存在也返回 204，保持幂等）。
- `summaryDTO`、`detailDTO` 新增 `Name string \`json:"name"\``，列表与详情都带回名称。
- 鉴权：`isGated` 已按 `/api/requests/` 前缀拦截**任意方法**，新增的 PATCH/DELETE 自动受面板鉴权保护，无需改动；抓包端口不受影响（不变量保持）。

## 前端（`frontend/src`）

- `types/api.ts`：`Summary` 接口新增 `name: string`（`Detail` 继承自 `Summary`，自动带上）。
- `stores/captures.ts` 新增 action：
  - `setName(id, name)`：`PATCH /api/requests/{id}`；成功后同步更新本地 `list` 中该项与 `current` 的 `name`。
  - `remove(id)`：`DELETE /api/requests/{id}`；成功后从 `list` 移除该项；若删的是 `activeId`，清空 `current`/`activeId` 并路由回 `/`。
  - 两者 401 复用 `handleUnauthorized`。
- `RequestDetail.vue`：详情头加一个可编辑名称输入框（占位提示「未命名 / 点击命名」），失焦或回车时若值有变则提交 `store.setName`。
- `LogItem.vue`：当 `item.name` 非空时，在 target 上方显示该名称（截断省略）。
- `LogItem.vue`：列表项 hover 时显示一个小删除按钮，点击触发 `window.confirm` 确认后调用 `store.remove(item.id)`；`@click.stop` 避免冒泡到选中逻辑。

## 测试

- Go：`internal/store` 加 `SetName`、`Delete` 的单元测试（含删不存在、设名后 List/Get 能读回）。
- 前端：`stores/captures.spec.ts` 补 `setName`/`remove` 的行为；`LogItem.spec.ts` 补名称显示与删除按钮。

## 不做（YAGNI）

- 名称不参与 `prune` 保留策略：命名过的请求仍可能因超出 `max` 条数被删，可接受。
- 不做置顶/锁定、不做批量删除。
