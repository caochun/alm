# ALM — 应用生命周期管理系统

ALM（Application Lifecycle Management）是一个面向企业的应用生命周期管理平台，通过 DSL 驱动的方式将应用从源码到生产部署的全过程标准化、可配置化。

---

## 设计目标

企业中管理大量第三方和内部软件系统时，面临两类核心诉求：

1. **应用开发者**：如何将代码从仓库变成可以交付的制品（JAR、Docker 镜像、静态包），流程是否可标准化、可复用？
2. **基础设施运营者**：如何为制品提供运行所需的计算资源和支撑服务（数据库、缓存、消息队列、Ingress），如何定义资源规格、如何供给这些基础设施？

ALM 的目标是：**用 DSL 将这两类关注点分离建模，并由引擎统一编排执行**。

---

## 设计理念

### Stakeholder 驱动的关注点分离

ALM 围绕两类 Stakeholder 设计，每类只需关注自己的领域：

```
应用开发者                           基础设施运营者
──────────────────────────           ──────────────────────────────────
定义：应用由哪些服务组成               定义：服务运行在什么计算环境中
定义：每个服务如何从代码变成制品         定义：需要哪些支撑服务（DB/Cache/MQ）
关注：构建工具、依赖关系、制品类型       关注：资源规格、供给方式、网络拓扑、环境差异
```

两者通过**可部署制品类型**（Deliverable）形成唯一接口契约：

```
Pipeline ──[produces]──► docker-image ◄──[accepts]── DeploymentEnv
                               ↑
                           契约接口（解析阶段校验类型匹配）
```

开发者只管左边，运营者只管右边，系统在解析阶段校验两侧是否匹配，不等到运行时才报错。

---

### 三层 DSL 体系

#### Layer 1 — AppPipeline（应用开发者写）

描述"如何把源码变成可部署制品"，是一个**有向无环图（DAG）**，而非线性流水线。

- 支持多个 `deliverables`（出口），同一条流水线可以在 `jar-file` 处停止，也可以继续走到 `docker-image`
- 部署环境通过 `accepts` 声明需要哪种形态，引擎按需规划执行路径，不多跑也不少跑

```yaml
kind: AppPipeline
name: java-webapp-pipeline

stages:
  - id: source-code
    produces: source-code
    action: { type: tool, command: git, args: [clone, "${repository}", "${outputDir}"] }

  - id: jar-file
    requires: [source-code]
    produces: jar-file
    action: { type: tool, command: mvn, args: [clean, package, -DskipTests] }

  - id: docker-image
    requires: [jar-file]
    produces: docker-image
    action: { type: tool, command: docker, args: [build, -t, "${imageName}:${imageTag}", .] }

deliverables: [jar-file, docker-image]   # 两个可能的出口
```

#### Layer 2 — AppArchitecture（应用开发者写）

描述"应用由哪些服务组成以及它们的依赖关系"，是**纯逻辑定义，与环境无关**。

- 每个服务声明使用哪条 Pipeline 模板和对应的代码仓库
- `depends_on` 描述服务间的部署依赖，引擎通过拓扑排序决定执行顺序
- 一份 AppArchitecture 可以对应多套 DeploymentEnv（dev / staging / prod）

```yaml
kind: AppArchitecture
name: mall-platform

services:
  - name: user-service
    pipeline: java-webapp-pipeline
    repository: https://github.com/example/user-service.git

  - name: order-service
    pipeline: java-webapp-pipeline
    repository: https://github.com/example/order-service.git
    depends_on: [user-service]           # order 必须在 user 之后部署
```

#### Layer 3 — DeploymentEnv（基础设施运营者写）

描述"在某个环境中如何运行这个应用"，包含**计算资源、支撑服务、网络**三个维度。

**服务部署规格**：每个服务声明接受什么制品类型、运行在什么计算环境，以及资源量（CPU、内存、实例数）。

**基础设施依赖**：声明需要哪些支撑服务（MySQL、Redis、Kafka 等），包含：
- `type`：软件类型和版本
- `provision`：如何供给（`docker` / `terraform` / `helm` / `external`）
- `resources`：资源规格（CPU、内存、存储）
- `config`：运行时参数（端口、数据库名等）

**Bindings**：将支撑服务的连接信息注入到服务运行时环境变量，支持 `${infra.field}` 插值。

**网络**：Ingress 定义，包含 IP/端口绑定、TLS 配置、路由规则及资源规格。

```yaml
kind: DeploymentEnv
name: mall-dev
environment: development
app: mall-platform

services:
  - name: order-service
    accepts: docker-image              # 与 pipeline deliverables 形成契约
    compute:
      type: docker-container
      resources: { cpu: "1", memory: 512Mi, replicas: 1 }

dependencies:
  - name: mysql
    type: mysql:8.0
    provision:
      via: docker                      # 本地开发用 Docker 供给
      image: mysql:8.0
      env: { MYSQL_ROOT_PASSWORD: secret }
    resources: { cpu: "2", memory: 2Gi, storage: 20Gi }
    config: { port: 3306, database: mall_dev }

bindings:
  - service: order-service
    env:
      SPRING_DATASOURCE_URL: "jdbc:mysql://${mysql.host}:${mysql.config.port}/mall_orders"

network:
  ingress:
    - name: public-gateway
      type: nginx
      bind: { ip: "0.0.0.0", http: 80, https: 443 }
      routes:
        - { path: /api/orders, service: order-service, port: 8080 }
      resources: { cpu: "0.5", memory: 256Mi }
```

同一套 AppArchitecture，不同环境只需换一个 DeploymentEnv 文件：

```
deploy/
  dev.yaml      ← 本地 Docker，小规格
  staging.yaml  ← 云上测试，中等规格
  prod.yaml     ← 生产 Kubernetes，高可用
```

---

## 基础设施供给方式（provision.via）

| via | 含义 |
|-----|------|
| `docker` | 引擎调用 Docker 启动容器，需提供 `image` |
| `terraform` | 执行 Terraform 模块，需提供 `module` 路径 |
| `helm` | Helm 安装 Chart，需提供 `chart` |
| `external` | 已有实例，不做供给操作，需提供 `endpoint` |

---

## 项目结构

```
alm/
├── domain/                        # 领域模型
│   ├── pipeline.go                # Pipeline、Stage、PipelineAction
│   ├── app_architecture.go        # AppArchitecture、ServiceSpec（含拓扑排序）
│   ├── deployment_env.go          # DeploymentEnv、InfraResource、InfraProvision、Binding、Network
│   ├── resource.go                # ResourceSpec、VolumeSpec
│   └── errors.go
│
├── dsl/                           # DSL 解析层
│   ├── pipeline_parser.go         # 解析 AppPipeline YAML → domain.Pipeline
│   ├── app_arch_parser.go         # 解析 AppArchitecture YAML → domain.AppArchitecture
│   ├── deploy_env_parser.go       # 解析 DeploymentEnv YAML → domain.DeploymentEnv
│   ├── loader.go                  # LoadPipelinesFromDir()
│   ├── validator.go               # 跨模型一致性校验
│   └── templates/                 # 内置 Pipeline 模板
│       ├── java-webapp-pipeline.yaml
│       └── nodejs-spa-pipeline.yaml
│
├── cmd/
│   └── validate/main.go           # DSL 校验工具
│
└── workspace/                     # 应用工作目录
    ├── spring-petclinic/          # 单服务示例（Java）
    │   ├── app-arch.yaml
    │   └── deploy/local.yaml
    └── mall-platform/             # 多服务微服务示例
        ├── app-arch.yaml
        └── deploy/dev.yaml
```

---

## DSL 校验工具

提供命令行工具用于在部署前验证三层 DSL 的一致性：

```bash
go run ./cmd/validate \
  -arch      workspace/mall-platform/app-arch.yaml \
  -deploy    workspace/mall-platform/deploy/dev.yaml \
  -pipelines dsl/templates
```

输出示例：

```
Pipelines loaded (2):
  ✓ java-webapp-pipeline  deliverables: [jar-file docker-image]
  ✓ nodejs-spa-pipeline   deliverables: [static-bundle]

AppArchitecture: mall-platform
  Deployment order (4 services):
    1. user-service      pipeline: java-webapp-pipeline   depends_on: none
    2. product-service   pipeline: java-webapp-pipeline   depends_on: none
    3. order-service     pipeline: java-webapp-pipeline   depends_on: [user-service product-service]
    4. frontend          pipeline: nodejs-spa-pipeline    depends_on: [user-service product-service order-service]

Validating...
  ✓ All checks passed
```

校验内容包括：
- `accepts` 的制品类型是否在对应 Pipeline 的 `deliverables` 中
- `bindings` 和网络路由引用的服务是否在 AppArchitecture 中存在
- 服务依赖图是否存在循环

---

## 当前状态与路线图

| 层次 | 状态 |
|------|------|
| DSL 定义（三层模型） | ✅ 已完成 |
| DSL 解析器 | ✅ 已完成 |
| 跨模型校验 | ✅ 已完成 |
| Pipeline 执行引擎（Build Phase） | 🚧 待实现 |
| DeploymentEnv 执行引擎（Deploy Phase） | 🚧 待实现 |
| 工具执行器（Git / Maven / Docker / Terraform） | 🚧 待实现 |
| REST API | 🚧 待实现 |
| Web 管理界面 | 🚧 待实现 |

---

## 快速开始

```bash
# 克隆项目
git clone <repo-url>
cd alm

# 验证示例应用（spring-petclinic）
go run ./cmd/validate \
  -arch      workspace/spring-petclinic/app-arch.yaml \
  -deploy    workspace/spring-petclinic/deploy/local.yaml \
  -pipelines dsl/templates

# 验证多服务示例（mall-platform）
go run ./cmd/validate \
  -arch      workspace/mall-platform/app-arch.yaml \
  -deploy    workspace/mall-platform/deploy/dev.yaml \
  -pipelines dsl/templates
```

---

## 许可证

MIT
