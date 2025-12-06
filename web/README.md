# ALM Web UI

ALM系统的Web用户界面，用于可视化和管理软件资产的生命周期。

## 功能特性

1. **应用列表** - 显示workspace下的所有应用
2. **状态机可视化** - 以graph形式展示状态机和当前状态
3. **状态详情** - 查看当前状态下的concrete assets和可触发的actions
4. **状态转换** - 触发action执行状态转换

## 安装和启动

### 安装依赖

```bash
cd web
npm install
```

### 启动开发服务器

```bash
npm run dev
```

前端将在 `http://localhost:3000` 启动，API请求会自动代理到后端服务器（`http://localhost:8081`）。

### 构建生产版本

```bash
npm run build
```

## 使用说明

1. 启动后端服务器（见项目根目录README）
2. 启动前端开发服务器
3. 在浏览器中访问 `http://localhost:3000`
4. 选择应用查看其状态机
5. 点击状态节点查看详情
6. 触发action执行状态转换

## 技术栈

- React 18
- React Router
- Vite
- vis-network (状态机可视化)
- Axios (HTTP客户端)

