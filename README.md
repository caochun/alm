# ALM - 应用生命周期管理系统

一个用于管理企业第三方软件系统生命周期的平台。

## 设计理念

### 1. 领域驱动设计（DDD）

ALM采用领域驱动设计思想，将软件资产的生命周期管理抽象为领域模型：

- **SoftwareAsset（软件资产）**：聚合根，代表一个完整的软件系统，包含当前状态、具体资产和转换历史
- **StateMachineTemplate（状态机模板）**：定义软件资产可能经历的状态和状态转换规则
- **ConcreteAsset（具体资产）**：在生命周期各阶段产生的具体产物（源代码、构建产物、运行容器等）
- **StateTransition（状态转换）**：记录状态转换的完整历史，包括执行的动作、条件和结果

### 2. 状态机驱动

系统通过状态机来管理软件资产的生命周期：

- **可定制性**：通过DSL定义不同类型应用的状态机模板
- **规则验证**：状态转换前验证转换规则和前置条件
- **资产依赖**：明确每个状态转换的输入资产和输出资产，形成完整的依赖链
- **历史追溯**：完整记录所有状态转换历史，支持审计和回滚

### 3. DSL配置化

通过YAML DSL定义状态机模板，实现配置与代码分离：

- **状态定义**：定义生命周期中的各个状态及其期望的资产类型
- **转换规则**：定义状态间的转换条件、所需动作和生成的资产类型
- **动作定义**：定义工具动作（如git-clone、maven-build）和手动动作
- **资产类型**：定义资产类型的schema和验证规则

### 4. 工具集成

通过执行器（Executor）模式集成各种开发运维工具：

- **Git Executor**：执行代码克隆操作
- **Maven Executor**：执行构建操作
- **Terraform Executor**：执行基础设施部署
- **可扩展性**：通过Executor接口可以轻松集成新的工具

### 5. 资产依赖链

系统维护完整的资产依赖关系：

- **输入资产**：状态转换所需的输入资产（如构建需要源代码）
- **输出资产**：状态转换生成的输出资产（如构建生成JAR文件）
- **依赖追溯**：可以追溯每个资产的生成过程和依赖关系

## 系统架构

### 分层架构

```
┌─────────────────────────────────────────┐
│          Web UI (React)                 │
│  - 状态机可视化                          │
│  - 状态详情和操作                        │
│  - 文件浏览                              │
└─────────────────────────────────────────┘
                    ↕
┌─────────────────────────────────────────┐
│          RESTful API (Gin)              │
│  - 资产管理接口                          │
│  - 状态转换接口                          │
│  - 文件浏览接口                          │
└─────────────────────────────────────────┘
                    ↕
┌─────────────────────────────────────────┐
│          Application Layer              │
│  ┌──────────────┐  ┌─────────────────┐ │
│  │ AssetManager │  │ StateMachine    │ │
│  │              │  │ Engine          │ │
│  │ - 加载配置    │  │ - 状态转换      │ │
│  │ - 状态持久化  │  │ - 规则验证      │ │
│  │ - 资产管理    │  │ - 执行器调用    │ │
│  └──────────────┘  └─────────────────┘ │
│  ┌───────────────────────────────────┐ │
│  │ Executor Factory                  │ │
│  │ - Git Executor                    │ │
│  │ - Maven Executor                  │ │
│  │ - Terraform Executor              │ │
│  └───────────────────────────────────┘ │
└─────────────────────────────────────────┘
                    ↕
┌─────────────────────────────────────────┐
│          Domain Layer                    │
│  - SoftwareAsset (聚合根)                │
│  - StateMachineTemplate                  │
│  - State, StateTransition                │
│  - ConcreteAsset, AssetType              │
│  - Action, ActionExecutionResult         │
└─────────────────────────────────────────┘
                    ↕
┌─────────────────────────────────────────┐
│          DSL Layer                       │
│  - webapp.yaml (状态机模板)              │
│  - asset.yaml (应用配置)                  │
│  - Parser (DSL解析器)                    │
└─────────────────────────────────────────┘
```

### 核心组件

#### 1. Domain Layer（领域层）

位于 `domain/` 目录，包含核心业务模型：

- **SoftwareAsset**：软件资产聚合根，管理资产的状态和转换
- **StateMachineTemplate**：状态机模板，定义状态和转换规则
- **State**：生命周期状态
- **StateTransitionRule**：状态转换规则
- **ConcreteAsset**：具体资产实体
- **Action**：动作定义（工具动作/手动动作）

#### 2. DSL Layer（DSL层）

位于 `dsl/` 目录，提供配置化能力：

- **webapp.yaml**：Web应用的状态机模板定义
- **Parser**：YAML DSL解析器，将DSL转换为领域模型

#### 3. Application Layer（应用层）

位于 `internal/` 目录，包含应用服务：

- **manager/**：资产管理器
  - `AssetManager`：管理单个应用的资产，加载配置和状态
  - `AssetManagerFactory`：管理多个应用的资产管理器实例
  - `asset_persistence.go`：状态持久化（保存到 `.alm-state.json`）

- **engine/**：状态机引擎
  - `StateMachineEngine`：执行状态转换，验证规则，调用执行器

- **executor/**：工具执行器
  - `GitExecutor`：执行git clone
  - `MavenExecutor`：执行maven构建
  - `TerraformExecutor`：执行terraform部署
  - `ExecutorFactory`：创建执行器实例

- **api/**：Web API层
  - RESTful API接口
  - 文件浏览接口
  - 工作空间管理接口

#### 4. Interface Layer（接口层）

- **cmd/server/**：服务器入口，启动HTTP服务器

#### 5. Presentation Layer（表现层）

- **web/**：React前端应用
  - 状态机可视化（使用vis-network）
  - 状态详情和操作界面
  - 文件浏览界面
  - 执行结果模态框

### 数据流

1. **状态转换流程**：
   ```
   用户触发转换 → API接收请求 → StateMachineEngine验证规则 
   → 查找输入资产 → 调用Executor执行动作 → 创建输出资产 
   → 更新状态 → 持久化状态 → 返回结果
   ```

2. **状态加载流程**：
   ```
   应用启动 → AssetManager加载asset.yaml → 加载状态机模板 
   → 尝试加载.alm-state.json → 如果不存在则创建新资产 
   → 返回当前状态
   ```

3. **资产依赖链**：
   ```
   初始状态 → [git-clone] → 源代码资产 
   → [maven-build] → JAR文件资产（依赖源代码）
   → [terraform-deploy] → 容器资产（依赖JAR文件）
   ```

## 功能特性

- **领域模型驱动** - 基于DDD设计的核心领域模型
- **状态机管理** - 可定制的状态机模板（DSL定义）
- **工具集成** - 支持Git、Maven、Terraform等工具
- **资产依赖链** - 完整的资产依赖关系追溯
- **状态持久化** - 状态自动保存到文件，支持应用重启后恢复
- **Web UI** - 可视化的状态机管理和操作界面
- **文件浏览** - 在Web UI中浏览和查看workspace下的应用文件
- **执行反馈** - 实时显示动作执行过程和结果
- **RESTful API** - 完整的API接口

## 项目结构

```
alm/
├── domain/              # 领域模型（核心业务逻辑）
│   ├── software_asset.go
│   ├── state_machine_template.go
│   ├── state.go
│   ├── state_transition_rule.go
│   ├── concrete_asset.go
│   ├── action.go
│   └── ...
├── dsl/                 # 状态机DSL定义
│   ├── webapp.yaml      # Web应用状态机模板
│   └── parser.go        # DSL解析器
├── internal/
│   ├── manager/        # 资产管理器
│   │   ├── asset_manager.go
│   │   ├── asset_persistence.go
│   │   └── manager_factory.go
│   ├── engine/         # 状态机引擎
│   │   └── state_machine_engine.go
│   ├── executor/       # 工具执行器
│   │   ├── factory.go
│   │   ├── git_executor.go
│   │   ├── maven_executor.go
│   │   └── terraform_executor.go
│   └── api/            # Web API
│       ├── handler.go
│       ├── router.go
│       ├── workspace_handler.go
│       └── file_handler.go
├── cmd/server/         # 服务器入口
│   └── main.go
├── web/                # Web UI前端
│   ├── src/
│   │   ├── pages/      # 页面组件
│   │   ├── components/ # UI组件
│   │   └── services/   # API客户端
│   └── dist/           # 构建输出
└── workspace/          # 工作空间（应用目录）
    └── spring-petclinic/
        ├── asset.yaml           # 应用配置
        ├── .alm-state.json      # 状态持久化文件
        ├── source/              # 源代码目录
        ├── build/               # 构建产物目录
        └── deploy/              # 部署配置目录
```

## 快速开始

### 1. 构建前端

```bash
cd web
npm install
npm run build
```

### 2. 启动服务器（同时提供API和Web UI）

```bash
# 构建服务器
go build -o bin/alm-server ./cmd/server/

# 启动服务器（指定workspace根目录和web目录）
./bin/alm-server -workspace ./workspace -web ./web -port 8081
```

### 3. 访问Web UI

打开浏览器访问 `http://localhost:8081`

**注意**：如果不需要Web UI，可以省略 `-web` 参数，服务器将只提供API服务。

## 配置说明

### 应用配置（asset.yaml）

每个应用在workspace下需要创建 `asset.yaml` 配置文件：

```yaml
id: petclinic-001
name: Spring PetClinic
description: Spring Framework示例应用
state_machine_template: ../dsl/webapp.yaml  # 状态机模板路径

workspace:
  source_dir: source
  build_dir: build
  deploy_dir: deploy
  assets_dir: assets

application:
  git:
    repository: https://github.com/spring-projects/spring-petclinic.git
  maven:
    build_command: mvn clean package
  terraform:
    provider: docker
```

### 状态机模板（webapp.yaml）

定义应用的生命周期状态机：

```yaml
name: webapp-lifecycle
description: Web应用的标准生命周期状态机模板

asset_types:
  - id: source-code
    name: 源代码
    description: Git仓库克隆的源代码

states:
  - id: initial
    name: 初始状态
    description: 应用初始状态

transitions:
  - from: initial
    to: source-code
    action: git-clone
    conditions:
      repository: required
    input_asset_types: []
    generated_asset_types:
      - source-code
```

## API文档

详见 [API.md](./API.md)

## 使用示例

### 创建新应用

1. 在 `workspace/` 目录下创建应用目录
2. 创建 `asset.yaml` 配置文件
3. 在Web UI中即可看到新应用

### 执行状态转换

通过Web UI或API执行状态转换：

```bash
curl -X POST http://localhost:8081/api/v1/asset/transition?appPath=spring-petclinic \
  -H "Content-Type: application/json" \
  -d '{
    "toState": "source-code",
    "conditions": {
      "repository": "https://github.com/spring-projects/spring-petclinic.git"
    },
    "operator": "admin"
  }'
```

### 浏览应用文件

在Web UI中，选择应用后切换到"文件浏览"标签页，可以：
- 浏览应用目录结构
- 查看文件内容
- 导航到子目录

也可以通过API访问：

```bash
# 列出应用根目录文件
curl "http://localhost:8081/api/v1/files?appPath=spring-petclinic&path="

# 查看文件内容
curl "http://localhost:8081/api/v1/files/content?appPath=spring-petclinic&path=asset.yaml"
```

## 扩展开发

### 添加新的执行器

1. 在 `internal/executor/` 目录下创建新的执行器，实现 `Executor` 接口
2. 在 `ExecutorFactory` 中注册新执行器
3. 在DSL中定义对应的动作

示例：

```go
// internal/executor/custom_executor.go
type CustomExecutor struct {
    // ...
}

func (e *CustomExecutor) Execute(ctx *engine.ExecutionContext) (*engine.ExecutionResult, error) {
    // 实现执行逻辑
}
```

### 定义新的状态机模板

1. 创建新的YAML文件（如 `microservice.yaml`）
2. 定义资产类型、状态和转换规则
3. 在应用的 `asset.yaml` 中引用新模板

## 开发

### 后端开发

```bash
# 运行测试
go test ./...

# 构建
go build ./...
```

### 前端开发

```bash
cd web
npm run dev
```

## 许可证

MIT
