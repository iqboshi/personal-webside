# 张孟庆个人主页

Go 后端 + Vue 3 + Element Plus 前端的个人主页，用来记录项目、文章和最近在读内容。
项目文章由 Go 后端从本地 SQLite 数据库读取，首次启动会自动创建 `data/homepage.db`，并把 `content/articles/*.json` 同步进数据库。
项目正文使用 `blocks_json` 存储富文本块，支持段落、列表、代码块、图片/GIF 和链接卡片。

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
