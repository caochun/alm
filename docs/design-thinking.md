# ALM 设计思考过程

> 记录 2026-02-27 的设计讨论，供后续参考。

---

## 起点：原有设计的局限

原始设计是一个**单体状态机模型**：`webapp.yaml` 定义了一条从源码到部署的线性流水线（git-clone → maven-build → terraform-deploy），由 `SoftwareAsset` 聚合根驱动状态转换。

这个设计对单一单体应用可行，但面临两个扩展问题：
1. **服务化应用**：一个应用由多个微服务组成，单一状态机无法描述多服务的组合关系
2. **基础设施**：部署时需要数据库、缓存、消息队列等资源，原设计没有对这些建模

---

## 第一次推导：Stakeholder 分析

讨论服务化应用架构时，提出了"应用架构定义语言"的想法。最初的设计思路是扩展现有 DSL，加入服务列表和基础设施声明。

**转折点**：引入了 Stakeholder 视角。

> "这个项目主要面向两类 Stakeholder：应用开发者关注如何把代码变成可部署制品，基础设施运营者关注如何为制品提供运行环境。"

这一视角带来了本质性的设计转变：不是扩展一个 DSL，而是**定义两个不同的 DSL**，对应两类人的不同关注点。

### 两类 Stakeholder 的边界

```
应用开发者关心：                    基础设施运营者关心：
───────────────────────             ─────────────────────────────
代码 → JAR → 容器镜像               虚拟机/容器 + MySQL + Redis
（资产转换流水线）                   （运行环境供给）
```

**关键洞察**：两者的接口只有一个——**可部署制品的类型**。开发者生产它，运营者消费它。

原有 `webapp.yaml` 实际上把两件事混在了一起：`git-clone + maven-build`（开发者的事）和 `terraform-deploy`（运营者的事）。

---

## 第二次推导：多种制品形态

一个自然的问题：如果不同服务产出不同类型的制品怎么办？例如：
- Java 服务 → jar-file（部署到 VM）或 docker-image（部署到容器）
- Node.js 前端 → static-bundle

**原来的线性流水线设计无法处理这个问题**，因为它只有一个出口。

解法：**流水线是 DAG，有多个可能的出口（deliverables）**。

```
source-code → jar-file → docker-image
                ↑              ↑
          可以在这里停       也可以到这里停
```

部署环境通过 `accepts` 声明需要哪种制品形态，引擎按需规划执行路径：
- 如果 `accepts: jar-file`，流水线执行到 jar-file 阶段停止
- 如果 `accepts: docker-image`，继续执行 docker-build 阶段

**契约设计**：`pipeline.deliverables` ↔ `serviceDeploySpec.accepts` 在解析阶段校验，类型不匹配直接报错，不等到运行时。

---

## 第三次推导：基础设施资源定义

基础设施资源的建模经历了两个层次的思考：

### 第一层：资源量

基础设施资源需要定义资源量（CPU、内存、存储）和运行时参数（端口、数据库名）：

```yaml
dependencies:
  - name: mysql
    type: mysql:8.0
    resources:
      cpu: "2"
      memory: 2Gi
      storage: 20Gi
    config:
      port: 3306
      database: mall_dev
```

**设计决策**：`resources`（多少）和 `config`（怎么跑）分开，职责不同。

### 第二层：供给方式

一个问题：DSL 只说明了"需要什么"，但没说明"如何供给"。同样一个 MySQL 实例：
- 本地开发：`docker run mysql:8.0`
- 测试环境：Terraform 创建云上 RDS
- 生产：已有实例，只需连接信息

**解法**：在 `InfraResource` 中增加 `provision` 块，`via` 字段决定使用哪种供给机制。

```yaml
provision:
  via: docker        # or: terraform | helm | external
  image: mysql:8.0
  env:
    MYSQL_ROOT_PASSWORD: secret
```

类比：这和 Pipeline 中 `action.type = tool/manual/http` 是一样的设计模式——`via` 决定执行器，其余字段是参数。

不同的 `via` 强制要求不同的字段：
- `docker` → 必须有 `image`
- `terraform` → 必须有 `module`
- `helm` → 必须有 `chart`
- `external` → 必须有 `endpoint`（已有实例，只提供连接信息，引擎不做任何操��）

---

## 最终三层 DSL 体系

```
Layer 1: AppPipeline       （应用开发者写）
         描述"代码如何变成制品"
         关键概念：stages（DAG）、deliverables（多出口）

Layer 2: AppArchitecture   （应用开发者写）
         描述"应用由哪些服务组成"
         关键概念：services、depends_on（拓扑排序）

Layer 3: DeploymentEnv     （基础设施运营者写）
         描述"制品如何运行"
         关键概念：services.accepts、dependencies（+provision）、bindings、network
```

### 层间关系

```
AppPipeline ←── AppArchitecture.service.pipeline（引用）
                        ↓
                 AppArchitecture
                        ↓
AppPipeline.deliverables ↔ DeploymentEnv.service.accepts（契约）
                        ↓
                 DeploymentEnv
```

AppArchitecture 是纯逻辑定义，与环境无关；DeploymentEnv 是环境相关的，同一 AppArchitecture 可以有多套 DeploymentEnv（dev/staging/prod）。

---

## 关键设计决策汇总

| 问题 | 决策 | 理由 |
|------|------|------|
| 为什么不扩展原有状态机？ | 彻底重构 | 原设计把两类 Stakeholder 的关注点混在一起，扩展会越来越复杂 |
| 流水线是线性还是 DAG？ | DAG + 多 deliverable | 同一服务可能需要在不同环境交付不同形态的制品 |
| 类型契约何时校验？ | 解析阶段 | 早失败比晚失败好，运行时失败代价更高 |
| resources 和 config 为何分开？ | 职责分离 | resources 是"多少"，config 是"怎么跑"，变化频率和维护者不同 |
| provision 为何在 DeploymentEnv 而非独立文件？ | 供给方式是环境相关的 | 同一基础设施在不同环境用不同方式供给（dev 用 docker，prod 用 terraform） |
| external 的 endpoint 为什么是必填？ | 强制显式声明 | 避免隐式依赖，让"已有实例"这个事实在 DSL 中显式表达 |
| AppArchitecture 的拓扑排序在哪里做？ | 解析阶段 | 循环依赖是配置错误，应该在加载时就暴露 |

---

## 被否定的方向

### "基础设施也用生命周期模板管理"

最初的复杂设计方案中，考虑过让 MySQL、Redis 等基础设施资源也有自己的状态机模板（`mysql-lifecycle.yaml`），像服务一样管理它们的生命周期（initial → provisioned → initialized → running）。

**为什么放弃**：用户反馈这样"逻辑上有点复杂了"。基础设施资源的供给方式（docker/terraform/helm）本身已经是成熟的工具，不需要再抽象一层生命周期。`provision` 块直接指定工具和参数，更简单直接。

---

## 待解决的问题（设计时已知）

1. **Binding 中的变量插值**：`${mysql.host}` 中的 `host` 从哪里来？基础设施运行后才有这个值，引擎需要在运行时解析。
2. **部分失败处理**：多服务部署时某个服务失败，是回滚还是允许重试？
3. **基础设施健康检查**：供给完成后，如何验证基础设施已就绪再部署服务？
4. **跨环境配置差异**：同一个 binding env 模板，不同环境的 secret（如数据库密码）如何管理？是否引入 secrets 管理机制？

---

## 对后续引擎实现的建议

根据三层 DSL 的设计，引擎应该分两个阶段：

**Build Phase（Pipeline 执行引擎）**
- 输入：AppArchitecture + Pipelines + 目标制品类型
- 对每个服务调用 `pipeline.GetStagesFor(accepts)` 获取需要执行的 stages
- 按 stages 顺序执行各 action
- 输出：每个服务的可部署制品（路径/镜像 tag 等）

**Deploy Phase（DeploymentEnv 执行引擎）**
- 输入：DeploymentEnv + Build Phase 的输出
- Step 1：按 InfraProvision.Via 供给所有 dependencies
- Step 2：等待基础设施健康检查通过
- Step 3：渲染 Binding（将基础设施运行时属性插值到 env vars）
- Step 4：按 AppArchitecture.TopologicalOrder() 顺序部署服务
- Step 5：配置 Network / Ingress
