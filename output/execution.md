# gb-534 执行与验证记录

## 验证摘要

| 项目 | 实测结果 |
| --- | --- |
| 项目编号 | `gb-534` |
| 项目 | `fermentation-kinetics-deviation-analysis` |
| 验证时间 | 2026-08-22 06:20-06:53 CST (UTC+08:00) |
| 前端 | `http://127.0.0.1:18534` |
| 后端 | `http://127.0.0.1:19534` |
| PostgreSQL | `127.0.0.1:57534` |
| runtime smoke | `http://127.0.0.1:20534/healthz` |
| Go 规模 | 4148 行功能代码，42 个功能 `.go` 文件 |
| 实现提交 | `52509d66806fee3d157d38bf0dd1263404d5c4a0` |
| 最终结论 | 构建、测试、启动、API、Browser 业务链、响应式布局与清理均通过 |

## 构建与测试

| 工作目录 | 命令 | 结果 |
| --- | --- | --- |
| 根目录 | `go work sync` | 通过 |
| 根目录 | `go build ./backend/...` | 通过 |
| 根目录 | `go vet ./backend/...` | 通过 |
| 根目录 | `go test ./backend/...` | 通过 |
| 根目录 | `go test -race ./backend/...` | 通过 |
| `backend/` | `go build ./...` | 通过 |
| `backend/` | `go vet ./...` | 通过 |
| `backend/` | `go test ./...` | 通过 |
| `backend/` | `go test -race ./...` | 通过 |
| `frontend/` | `npm ci` | 通过，使用 lockfile 干净安装 |
| `frontend/` | `npm run typecheck` | 通过 |
| `frontend/` | `npm test -- --run` | 3 个测试文件、7 项测试全部通过 |
| `frontend/` | `npm run build` | 通过，3784 modules transformed |
| `frontend/` | `npm audit --audit-level=moderate --registry=https://registry.npmjs.org` | `found 0 vulnerabilities` |
| 根目录 | `project_scale.py .` | `4148` 行 / `42` 文件，符合 2500-4200 行和 24-42 文件约束 |
| 根目录 | `runtime_smoke.py .` | SQLite 内存模式启动成功，`20534/healthz` 返回 HTTP 200 |
| 根目录 | `docker compose config --quiet` | 通过 |

Go 测试实际覆盖 `algorithm`、`constants` 状态机、`service` 权限/幂等/状态流和 `timeseries` 校验/标准化。前端测试覆盖共享阶段、偏差映射与审计证据快照展示。Vite 仅给出 ECharts 相关 chunk 大小建议，无构建错误。

## Compose 启动验证

1. 执行 `docker compose up -d --build` 真实构建并启动 PostgreSQL、Go 后端和 Nginx/Vue 前端。
2. `docker compose ps` 显示 `db`、`backend`、`frontend` 三个服务均为 `healthy`。
3. `GET http://127.0.0.1:19534/healthz` 返回 200 和 `status=healthy`。
4. `GET http://127.0.0.1:18534/api/healthz` 经 Nginx 代理返回 200 和 `status=healthy`。
5. 审计抽屉与 Tooltip 修复后分别重建前端；最终构建的三个服务再次全部 `healthy`，PostgreSQL 业务数据在复验期间保持。
6. 最终部署后日志扫描未发现应用 5xx、panic、上游超时或连接拒绝。

## 真实 API 冒烟

对 PostgreSQL Compose 环境执行了 41 个真实 HTTP 请求。下表按业务断言合并展示；角色登录、重复的详情/列表刷新也计入 41 次总数。

| 编号/请求 | 期望 | 实测 |
| --- | --- | --- |
| `GET /healthz` | 健康 | 200 |
| 未携带 token 读取罐体 | 认证拦截 | 401 `UNAUTHORIZED` |
| `admin/scientist/analyst/reviewer/auditor` 登录 | 五角色 JWT | 全部 200 |
| auditor 创建罐体 | RBAC 拒绝 | 403 `FORBIDDEN` |
| 罐体创建 / 详情 / 更新 / 列表 | 全链路 | 201 / 200 / 200 / 200 |
| 无效配方创建 | 阶段结束与目标时长冲突 | 422 `VALIDATION_FAILED` |
| 配方创建 / 详情 / 列表 | 全链路 | 201 / 200 / 200 |
| 配方 `draft -> validated -> published` | 两次合法迁移 | 全部 200 |
| 配方 `validated -> obsolete` | 非法迁移 | 409 `INVALID_STATE_TRANSITION` |
| 配方版本复制 | 生成新草稿 | 201，新 ID `4` |
| 不足 4 点的时序导入 | 质量校验 | 422 `VALIDATION_FAILED` |
| 时序导入 / 详情 / 列表 | 全链路 | 201 / 200 / 200 |
| 未 ready 时运行分析 | 前置状态校验 | 409 `INVALID_STATE_TRANSITION` |
| 时序 `imported -> validated -> normalized -> ready` | 三次合法迁移 | 全部 200 |
| 时序 `validated -> ready` | 跳步非法迁移 | 409 `INVALID_STATE_TRANSITION` |
| 分析运行 / 幂等重用 / 详情 / 列表 | 冻结结果链 | 201 / 200 / 200 / 200，分析 ID `2` |
| analyst 复核分析 | RBAC 拒绝 | 403 `FORBIDDEN` |
| 分析 `completed -> confirmed` | 跳步非法迁移 | 409 `INVALID_STATE_TRANSITION` |
| 分析 `completed -> reviewed -> confirmed` | 独立复核与确认 | 全部 200 |
| 冻结输入 replay | 确定性重放 | 200 |
| `GET /api/v1/audit-logs` | 审计投影 | 200 |
| `GET /api/v1/meta/enums` | 前后端共享枚举 | 200 |

API 夹具实际产生了罐体 `3` (`API-7351137`)、配方 `3`、复制配方 `4`、时序 `3` 和分析 `2`。后端日志中 `gb534-api-02/07/11/15/19/22/24/31/32` 分别保留了上表的 401/403/422/409 预期失败分支；无意外 5xx。

## Codex 内置 Browser 验证

本项只使用 Codex 内置 Browser，未使用外部 Chrome 或外部 Playwright。

1. 使用 `scientist` 通过 UI 登录，在罐体页创建 `BROWSER-534-0822`，列表刷新后可见。
2. 在配方页创建 `BROWSER-RECIPE-534`，完成草稿校验和发布状态变化。
3. 退出后使用 `analyst` 登录，导入 `BROWSER-RUN-534-0822`，依次完成校验、稳健缩放和就绪。
4. 运行真实偏差分析 `#3`，页面展示对齐曲线、偏差等级、阶段评分和解释。
5. 使用 `reviewer` 登录，对分析 `#3` 执行复核和确认，最终状态为 `confirmed`。
6. 使用 `auditor` 登录，进入审计中心，查看业务实体的操作人、request ID、耗时与前后 JSON 快照。
7. 在分析审计证据中点击“关联分析解释”，实际打开解释抽屉：显示 overall `0.137`、四阶段证据、input hash 和 `phase-dtw-v1.0.0`。
8. `/vessels`、`/recipes`、`/series`、`/analyses`、`/audit` 五页导航和 heading 均正确，页面真实请求的 11 个 `/api/v1` 资源均返回 200。
9. 修复版新 bundle 中，auditor 和 reviewer 分别打开 `/analyses` 与 `/vessels`，清空后的 Browser console 均为 `logs=[]`，原 `ElOnlyChild` 警告未再现。
10. 在 `390x844` 视口检查审计抽屉的标题、元数据、前后快照和滚动；`body` 与 `documentElement` 的 `scrollWidth` 均为 `390`，无水平溢出、文字遮挡或控件重叠。

### 截图

- [罐体总览桌面端](./vessels-desktop.jpg)
- [配方版本桌面端](./recipes-desktop.jpg)
- [时序工作台桌面端](./series-desktop.png)
- [偏差分析桌面端](./analysis-desktop.png)
- [已确认分析桌面端](./analysis-confirmed-desktop.png)
- [审计证据桌面端 1440x1000](./audit-evidence-desktop-1440x1000.png)
- [审计证据移动端 390x844](./audit-evidence-mobile-390x844.png)
- [审计前后快照移动端 390x844](./audit-after-mobile-390x844.png)

## 需求覆盖

| 需求 | 实际覆盖 |
| --- | --- |
| 四实体端到端分层 | 四实体均有 PostgreSQL 表、Go model/dto/repository/service/handler/router 与 Vue type/api/store/page 消费 |
| 五个核心页面 | 全部使用真实 `/api/v1`，Browser 完成导航、创建、导入、状态变化、分析和审计 |
| 共享组件与 hooks | `PhaseBadge`、`KineticsChart`、`AnalysisExplanationDrawer`、`AuditEvidenceDrawer`；`useAuth`、`useAnalysisRun` |
| JWT + RBAC | 五角色登录、路由/按钮权限、401/403 分支和发起人不能自确认均验证 |
| 时序与算法 | 排序去重、缺失证据、稳健缩放、阶段约束 DTW、多通道评分、原因规则、确定性 replay 均有实现与测试 |
| 状态机与幂等 | 合法流程、跳步 409、输入 hash + 算法版本幂等和不可覆盖历史均通过 |
| 审计与追踪 | 写操作前后快照、request ID、操作人、输入 hash、算法版本、耗时与结果解释可视化 |
| 中间件 | request ID、recovery、auth、RBAC、audit、error handler 独立分文件并在真实请求中生效 |
| 部署与文档 | Compose 三服务 healthcheck、PostgreSQL 正式模式、SQLite smoke、Nginx `/api` 代理与完整 README |
| 安全边界 | 仅离线决策支持，不接入设备，不生成或下发控制参数 |

## 修复摘要

1. Browser 首轮发现审计页对普通实体只有分析解释入口，无法阅读其前后快照。新增 `AuditEvidenceDrawer`，展示实体、动作、操作人、request ID、时间、算法、耗时、输入 hash、结果摘要和格式化前后 JSON；分析事件保留关联解释入口。
2. 依赖审计真实报告 ECharts 5.6 的 moderate XSS 公告，升级至 ECharts 6.1；官方 npm 安全端点复验为 0 vulnerabilities。
3. Browser console 发现 Element Plus `ElOnlyChild` 警告。根因是 `/analyses` 重放按钮和 `/vessels` 停用按钮将 `v-if` 放在 `el-tooltip` 唯一子节点上；将条件移到 Tooltip 本身后，类型检查、7 项前端测试、生产构建和 auditor/reviewer 内置 Browser 清空 console 复验均通过。

## 关闭与无残留验证

Browser 和 API 验证结束后执行：

```bash
docker compose down -v --remove-orphans
```

实测删除了三个容器、`fermentation-kinetics-deviation-analysis_default` 网络和 `fermentation-kinetics-deviation-analysis-postgres-data` 命名卷。关闭后再检查：

- `docker compose ps -a` 无服务。
- 项目 label 下容器、网络和命名卷查询均为空。
- `18534`、`19534`、`57534`、`20534` 均无 TCP LISTEN。
- runtime smoke 进程已退出。
