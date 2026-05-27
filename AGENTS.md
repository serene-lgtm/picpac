# PackMate Agent 指南

## 项目概述

picpac是一款个人物品管理类型的手机端app，旨在提供一个互联网sandbox帮助经常出行的朋友们先在手机上规划好需要的物品，以便在实际打包的时候有条不紊地check done。该project是picpac app的后端。

## 技术栈

- 语言：Go
- Web框架：Gin
- 接口层：Restful
- 数据库：MongoDB
- 客户端：Flutter

## 🚨 核心原则（最高优先级）

1. 严格三层架构：Handler → Service → Repository，禁止跨层调用。
2. 所有公开函数必须有单行注释，且注释必须以函数名开头。
3. 错误只向上传递，只允许在 Service 或 Handler 的最顶层记录日志。
4. 所有 delete 操作默认必须是逻辑删除，除非用户明确要求物理删除。

## API 文档规则

- 每次新增、删除、修改 API 后，必须同步更新根目录的 `docs/api.md`。
- `docs/api.md` 面向前端 agent 和前端开发者，内容必须优先描述正式接口约定。
- 如果代码实现与 `docs/api.md` 不一致，修正文档属于交付的一部分，不允许留下未同步状态。

## 代码组织约定

新增功能时，除非仓库已经形成稳定模式，否则优先按以下结构组织：

- `cmd/`：应用入口
- `internal/domain/`：核心领域实体
- `internal/handler/`：HTTP 处理层
- `internal/service/`：业务逻辑层
- `internal/repository/`：持久化抽象及 MongoDB 实现
- `internal/dto/`：请求和响应结构

必须遵守：
- Handler 负责 HTTP 参数解析、请求形态校验、调用 Service、返回响应。
- Service 负责业务逻辑编排、业务规则校验和防绕过兜底校验。
- Repository 负责数据访问。
- 数据模型必须按 `DTO → Domain Model → Mongo Document` 分层转换，不允许直接把 HTTP DTO 或 Mongo Document 泄漏到不属于它的层。
- Domain Model 字段发生变化时，必须先询问用户是否需要同步调整相关 DTO 和 Mongo Document；如果变化影响正式 API 请求或响应，必须同步更新 `docs/api.md`。
- 不允许把业务逻辑堆在 Handler 中。
- 不允许把存储实现细节泄漏到 Handler 中。
- Delete 接口和 `DeleteByID` 等底层删除能力默认只更新状态为 deleted，不允许直接物理删除 Mongo 文档。
- Repository 的 Mongo 实现应只在 Repository 内部操作 Mongo Document，并在 Repository 边界完成 Domain Model 与 Mongo Document 的互转；只有出现复杂查询、跨集合组合或明显复用需求时才额外拆 DAO。

## Struct Tag 规范
所有 Mongo Document 字段必须同时包含 json 和 bson tag；Domain Model 不允许保留 bson tag；HTTP DTO 只保留 json 或 form tag：
`json:"<蛇形命名>" bson:"<缩写>"`

### 命名规则
| tag | 规则 | 示例 |
|:---|:---|:---|
| json | 蛇形（snake_case） | `max_token`、`user_id` |
| bson | 2-5 字母缩写 | `mt`、`uid`、`act` |

### 示例
\`\`\`go
type Task struct {
    ID        string `json:"id" bson:"_id"`
    MaxToken  int    `json:"max_token" bson:"mt"`
    UserID    int64  `json:"user_id" bson:"uid"`
    IsActive  bool   `json:"is_active" bson:"act"`
}
\`\`\`

## 质量与提交流程

- 提交前必须保证代码可以成功构建。
- 提交前必须保证所有相关测试通过。
- Go 代码格式必须正确。
- Go lint 必须通过。
- 不允许遗留编译告警、死代码或未使用导入。
- 如果某项检查当前无法执行，必须在最终说明中明确指出。

完成工作前最低验证要求：

- `go test ./...`
- `gofmt` 或 `go fmt` 处理所有改动过的 Go 文件
- `golangci-lint run`，或仓库约定的 Go lint 命令

## 测试要求

- 只要有非平凡行为变更，就应补充或更新测试。
- 优先为 service、handler、utils 编写单元测试。
- 涉及接口行为或存储交互变化时，应补充集成风格测试。
- 至少覆盖成功路径、非法输入、关键失败路径。

## 工作方式

- 以小步、聚焦的方式修改代码。
- 除非有明确收益，否则优先遵循现有模式。
- 避免为形式上的“优雅”引入过重抽象，注释只解释不明显的逻辑。
- 配置优先使用环境变量或配置结构管理，新代码必须继续遵守三层架构。
- 对语义重复的函数优先收敛为通用接口，底层能力命名保持业务中立。

在认为工作完成之前，至少确认：

1. 代码可以成功构建。
2. 相关测试已经通过。
3. lint 和格式检查已经通过。
4. 若有正式接口变更，`docs/api.md` 已同步更新并说明前端需要关注的字段。
