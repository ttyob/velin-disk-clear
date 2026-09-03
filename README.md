# Velin Disk Clear

Velin Disk Clear 是 Windows 磁盘清理管家，使用 Go、Wails、React 和 TypeScript 开发。

应用数据统一保存在程序/项目目录下的 `data/`：SQLite 数据库、WAL 文件、清理历史和凭据都不会写入其他外部目录。首次启动会将旧版本用户配置目录迁移到该位置。

当前已实现磁盘概览、42 条异步扫描规则、文件与文件夹结果视图、大文件分析、分页文件说明、按卷回收站清理、下载目录候选扫描、永久删除清理计划、清理历史、SQLite 持久化，以及 OpenAI-compatible Cleaning Agent。Agent 由 Go 扫描器生成真实文件快照，向模型提供路径、用途、大小和规则说明，模型只能返回受限分析；所有删除都需要用户勾选和确认，Agent 不能直接执行。AI 页面支持预置目标的智能扫描，非磁盘清理问题会被拒绝。

回收站通过 Windows Shell API 统计并清空，默认不勾选；执行前会复核项目数和占用大小，内容变化时要求重新扫描。下载目录只列出旧安装包、大压缩包和大文件候选，始终默认不勾选，由用户逐项确认后永久删除。推荐自动勾选的项目仅限超过保留期的临时文件、旧维护日志和可重建的浏览器缓存。

## 开发

环境要求：Go 1.25+、Node.js、Windows WebView2。Linux 可运行后端测试和浏览器模拟界面，但 Win32 磁盘、DPAPI、系统设置入口需在 Windows 验证。

```bash
go test -race ./...
cd frontend && npm install && npm run build
go run github.com/wailsapp/wails/v2/cmd/wails@v2.15.0 dev
```

浏览器预览（调用 Go API）：

```bash
# 终端 1：启动 Go 预览 API（使用真实 SQLite 和扫描引擎）
go run . --dev-api --dev-api-addr 0.0.0.0:8788

# 终端 2：启动 Vite 页面
cd frontend && npm run dev
```

浏览器预览地址为 `http://127.0.0.1:5173`。Vite 会将同源 `/api` 请求代理到 Go `:8788`，因此从服务器 IP 或域名访问时也不会把 API 请求错误发到客户端的 `127.0.0.1`。预览页面的规则、磁盘扫描、清理计划、模型配置和 Cleaning Agent 请求都会通过 Go API 处理；模型配置写入 Go 使用的 SQLite 数据库，不再使用前端模拟数据。若 Go API 使用其他地址，可设置 `VITE_GO_API_URL`，例如 `http://127.0.0.1:8788/api`，然后重启 Vite。

产品与安全边界见 [需求文档](docs/requirements.md)，内置规则目录见 [扫描规则](docs/builtin-rules.md)。
