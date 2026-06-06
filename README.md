# 张孟庆个人主页

Go 后端 + Vue 3 + Element Plus 前端的个人主页，用来记录项目、文章和最近在读内容。
项目文章由 Go 后端从本地 SQLite 数据库读取，首次启动会自动创建 `data/homepage.db`，并把 `content/articles/*.json` 同步进数据库。
项目正文使用 `blocks_json` 存储富文本块，支持段落、列表、代码块、图片/GIF 和链接卡片。
签到日历和每日笔记由 Go API 写入 SQLite；访客只能查看，管理员登录后才能签到和编辑笔记。

## 本地运行

启动后端 API：

```powershell
go run ./cmd/server
```

默认数据库路径是 `data/homepage.db`，也可以手动指定：

```powershell
go run ./cmd/server -db data/homepage.db
```

文章内容目录也可以手动指定：

```powershell
go run ./cmd/server -content content/articles
```

开启管理员写入能力需要设置环境变量：

```powershell
$env:ADMIN_USERNAME = "admin"
$env:ADMIN_PASSWORD = "你的管理员密码"
$env:SESSION_SECRET = "一段足够长的随机字符串"
go run ./cmd/server
```

也可以不放明文密码，改用 SHA-256：

```powershell
$env:ADMIN_USERNAME = "admin"
$env:ADMIN_PASSWORD_SHA256 = "<管理员密码的 sha256 hex>"
$env:SESSION_SECRET = "一段足够长的随机字符串"
go run ./cmd/server
```

## 添加项目文章

新增项目时，在 `content/articles` 里新增一个 `.json` 文件即可。文件名建议带排序前缀，例如：

```text
content/articles/
  001-remote-sensing-data-workflow-platform.json
  002-your-next-project.json
```

文章字段示例：

```json
{
  "slug": "your-next-project",
  "category": "项目",
  "date": "2026-06-04",
  "title": "你的项目标题",
  "excerpt": "一句话说明项目解决的问题。",
  "tags": ["Go", "Vue", "SQLite"],
  "blocks": [
    { "type": "heading", "level": 2, "text": "项目定位" },
    { "type": "paragraph", "text": "这里写项目正文。" },
    { "type": "code", "language": "go", "filename": "main.go", "code": "package main" }
  ]
}
```

后端启动时会读取这些文件，内容变化后自动覆盖同步到 SQLite；前端仍然只通过 `/api/articles` 和 `/api/articles/{slug}` 读取数据库。

启动前端开发服务：

```powershell
cd frontend
npm install
npm run dev
```

生产构建：

```powershell
cd frontend
npm run build
cd ..
go run ./cmd/server
```

构建后，Go 服务会自动托管 `frontend/dist`。

## 线上部署结构

当前 GitHub Pages 地址是静态前端，不能直接运行 Go 服务或写 SQLite。线上要让签到和每日笔记真正可用，需要拆成两部分：

1. GitHub Pages：继续托管 `frontend/dist`。
2. Go API：部署到支持后端服务的平台，例如 Render、Railway、Fly.io 或自己的 VPS，并给 SQLite 配一个持久化目录。

前端会读取 `VITE_API_BASE_URL`。例如 Go API 部署到：

```text
https://personal-webside-api.onrender.com
```

就在 GitHub 仓库的 `Settings -> Secrets and variables -> Actions -> Variables` 里添加：

```text
VITE_API_BASE_URL=https://personal-webside-api.onrender.com
```

然后重新运行 `Deploy Pages` 工作流，线上 Pages 就会把 `/api/checkins`、`/api/admin/login`、`/api/admin/checkins/{date}` 请求发到这个 Go API。

## Go API 部署环境变量

后端服务至少需要这些环境变量：

```text
ADMIN_USERNAME=admin
ADMIN_PASSWORD=<管理员密码>
SESSION_SECRET=<一段足够长的随机字符串>
DB_PATH=/app/data/homepage.db
CONTENT_DIR=/app/content/articles
CORS_ALLOWED_ORIGINS=https://iqboshi.github.io
```

如果使用 Dockerfile 部署，记得把持久盘挂载到 `/app/data`，这样 `homepage.db` 才不会在重启或重新部署后丢失。

本地 Docker 运行示例：

```powershell
docker build -t personal-webside-api .
docker run --rm -p 8080:8080 `
  -e ADMIN_USERNAME=admin `
  -e ADMIN_PASSWORD=your-password `
  -e SESSION_SECRET=replace-with-long-random-secret `
  -e CORS_ALLOWED_ORIGINS=http://127.0.0.1:5173,https://iqboshi.github.io `
  -v ${PWD}/data:/app/data `
  personal-webside-api
```
