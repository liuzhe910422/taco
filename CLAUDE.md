# Taco - 智能动漫生成系统

## 项目概述

Taco 是一个自动根据小说生成动漫的智能系统，通过 AI 技术将文字小说转换为视觉化的动漫内容（图配文+声音形式）。

## 技术栈

- **后端**: Go 1.21
- **前端**: 原生 JavaScript + HTML + CSS
- **AI 服务**: 支持多种 LLM、图像生成和语音合成模型

## 项目结构

```
/workspace/
├── backend/              # Go 后端服务
│   ├── main.go          # 主服务器代码（HTTP API、AI 调用等）
│   └── config/          # 后端配置文件
├── web/                 # 前端静态文件
│   ├── index.html       # 主页
│   ├── characters.html  # 角色管理页面
│   ├── scenes.html      # 场景管理页面
│   ├── scene_detail.html # 场景详情页面
│   ├── playback.html    # 播放页面
│   └── *.js             # 对应的 JavaScript 文件
├── config/              # 配置文件目录
│   ├── characters.json  # 角色配置
│   └── scenes.json      # 场景配置
├── uploads/             # 上传的小说文件
├── generated/           # AI 生成的内容
│   ├── images/          # 生成的图片
│   └── audio/           # 生成的音频
├── test/                # 测试文件
├── go.mod               # Go 模块定义
└── taco-server          # 编译后的服务器可执行文件
```

## 核心功能模块

### 1. 小说处理
- 上传和解析小说文本
- 使用 LLM 提取角色信息
- 使用 LLM 分析场景内容

### 2. 角色生成
- 基于 LLM 分析生成角色描述
- 为每个角色生成一致的视觉形象
- 角色信息存储在 `config/characters.json`

### 3. 场景生成
- 场景分镜和对话提取
- 为每个场景生成对应图片（使用图像生成 API）
- 为场景生成配音（使用语音合成 API）
- 场景信息存储在 `config/scenes.json`

### 4. Web 界面
- 小说上传界面
- 角色管理和查看
- 场景列表和详情
- 动漫播放功能

## 开发指南

### 启动服务器

```bash
# 运行编译后的服务器
./taco-server

# 或者从源码运行
cd backend
go run main.go
```

服务器默认监听在 `:8080` 端口。

### API 配置

在 `backend/config/config.json` 中配置 AI 服务：

```json
{
  "llm": {
    "model": "模型名称",
    "baseUrl": "API 基础 URL",
    "apiKey": "API 密钥"
  },
  "image": {
    "model": "图像模型名称",
    "baseUrl": "API 基础 URL",
    "apiKey": "API 密钥"
  },
  "voice": {
    "model": "语音模型名称",
    "baseUrl": "API 基础 URL",
    "apiKey": "API 密钥"
  }
}
```

### 主要 API 端点

后端提供以下 HTTP API（参见 `backend/main.go`）：

- `POST /upload` - 上传小说文件
- `GET /config` - 获取配置
- `POST /config` - 更新配置
- `GET /characters` - 获取角色列表
- `POST /characters/generate` - 生成角色
- `GET /scenes` - 获取场景列表
- `POST /scenes/generate` - 生成场景
- `POST /scenes/generate-images` - 为场景生成图片
- `POST /scenes/generate-audio` - 为场景生成音频
- `/generated/*` - 静态文件服务（图片、音频）

### 代码规范

- Go 代码遵循标准 Go 格式化规范（使用 `go fmt`）
- 前端代码使用原生 JavaScript，避免引入额外框架
- API 调用采用标准 REST 风格
- 错误处理使用 HTTP 状态码和 JSON 响应

### 文件大小限制

- 上传文件最大 32MB（见 `maxFileSize` 常量）

## 工作流程

1. **上传小说** → 用户通过 Web 界面上传小说文本
2. **提取角色** → 使用 LLM 分析小说，提取主要角色信息
3. **生成角色图片** → 为每个角色生成视觉形象
4. **场景分析** → 使用 LLM 将小说分解为多个场景
5. **生成场景内容** → 为每个场景生成图片和音频
6. **播放预览** → 在 Web 界面中以图配文+声音形式播放

## 注意事项

- 需要配置有效的 API 密钥才能使用 AI 功能
- 生成的内容会保存在 `generated/` 目录中
- 角色和场景的配置持久化在 `config/` 目录中
- 服务器日志会输出到 `server.log` 文件

## 贡献建议

- 添加新功能前，先了解现有的 API 结构
- 修改配置结构时，需要同时更新前后端代码
- 测试 AI 功能时，确保 API 配置正确且有足够的配额
