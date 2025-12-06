# Terraform 手动执行指南

## 前置条件

1. 确保已安装 Terraform
2. 确保已安装 Docker 并运行
3. 确保 JAR 文件已构建完成

## 执行步骤

### 1. 进入部署目录

```bash
cd /home/chun/Develop/alm/workspace/spring-petclinic/deploy
```

### 2. 初始化 Terraform（首次执行或配置变更后）

```bash
terraform init
```

这会下载所需的 Docker provider。

### 3. 查看执行计划（可选）

```bash
terraform plan
```

这会显示 Terraform 将要执行的操作，但不会实际执行。

### 4. 执行部署

```bash
terraform apply
```

或者自动确认（无需交互）：

```bash
terraform apply -auto-approve
```

### 5. 查看容器状态

```bash
docker ps | grep petclinic
```

### 6. 查看容器日志

```bash
docker logs petclinic-001
```

### 7. 测试应用

应用会在容器的 8080 端口运行，映射到主机的 8080 端口：

```bash
curl http://localhost:8080
```

### 8. 销毁资源（清理）

```bash
terraform destroy
```

或者自动确认：

```bash
terraform destroy -auto-approve
```

## 常用命令

- `terraform init` - 初始化工作目录
- `terraform plan` - 查看执行计划
- `terraform apply` - 应用配置
- `terraform destroy` - 销毁资源
- `terraform show` - 显示当前状态
- `terraform output` - 显示输出值

## 故障排查

### 如果 Docker 镜像拉取失败

检查镜像名称是否正确，可以尝试：

```bash
docker pull eclipse-temurin:17-jre
```

### 如果端口已被占用

修改 `main.tf` 中的端口映射：

```hcl
ports {
  internal = 8080
  external = 8081  # 改为其他端口
}
```

### 如果容器启动失败

查看容器日志：

```bash
docker logs petclinic-001
```

检查 JAR 文件路径是否正确：

```bash
ls -lh /home/chun/Develop/alm/workspace/spring-petclinic/source/spring-petclinic/target/spring-petclinic-4.0.0-SNAPSHOT.jar
```

