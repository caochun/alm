# Spring PetClinic 软件资产工作空间

这个目录用于管理 Spring PetClinic 应用的生命周期。

## 目录结构

```
spring-petclinic/
├── asset.yaml          # 软件资产配置文件
├── README.md          # 本文件
├── source/            # 源代码（git clone的结果，不纳入版本控制）
├── build/             # 构建产物（不纳入版本控制）
├── deploy/            # 部署配置和Terraform文件
└── assets/            # 其他资产（SBOM、测试报告等，不纳入版本控制）
```

## 状态机

使用的状态机模板：`../../dsl/state_machine.yaml`

状态流程：
- 初始状态 → 获取源代码 → 构建 → 部署运行

## 使用说明

1. 通过 `asset.yaml` 配置软件资产的基本信息
2. 使用 ALM 系统管理状态转换
3. 源代码、构建产物等会存储在对应的子目录中

## 注意事项

- `source/`、`build/`、`assets/` 目录已加入 `.gitignore`，不会纳入版本控制
- 只有配置文件和部署配置（Terraform文件）会纳入版本控制

