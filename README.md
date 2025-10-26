# Taco - 智能动漫生成系统

## 项目简介

Taco 是一个自动根据小说生成动漫的智能系统。通过先进的 AI 技术，系统能够将文字小说转换为视觉化的动漫内容，以图配文+声音的形式呈现。

## 演示视频

项目演示视频位于：`taco/demo.mp4`

## 核心功能

### 1. 小说处理
- 上传和解析小说文本
- 使用 LLM 提取角色信息
- 使用 LLM 分析场景内

### 2. 角色生成与一致性保持
- 基于 LLM 分析生成角色描述
- 为每个角色生成一致的视觉形象
- 确保角色的外观、特征在不同场景中保持统一
- 角色信息持久化存储在 `config/characters.json`

### 3. 场景生成
- 场景分镜和对话提取
- 为每个场景生成对应图片（使用图像生成 API）
- 为场景生成配音（使用语音合成 API）
- 场景信息存储在 `config/scenes.json`

### 4. 多模态内容生成
- **图像生成**：根据小说情节生成对应的场景和角色画面
- **文字展示**：保留关键对话和叙述文字
- **语音合成**：为角色对话和旁白配音

### 5. Web 界面
- 小说上传界面
- 角色管理和查看
- 场景列表和详情
- 动漫播放功能

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

## 快速开始

### 前置要求

- Go 1.21 或更高版本
- 有效的 AI 服务 API 密钥（LLM、图像生成、语音合成）

### 安装和配置

1. 克隆仓库
```bash
git clone https://github.com/liuzhe910422/taco.git
cd taco
```

2. 配置 AI 服务

配置文件位于 `taco/config/config.json`，需要配置以下内容：

```json
{
  "novelFile": "/path/to/your/novel.txt",
  "llm": {
    "model": "gpt-4.1-nano",
    "baseUrl": "https://api.apiqik.com",
    "apiKey": "your-llm-api-key"
  },
  "image": {
    "model": "qwen-image",
    "baseUrl": "https://api.apiqik.com",
    "apiKey": "your-image-api-key"
  },
  "imageEdit": {
    "model": "qwen-image-edit",
    "baseUrl": "https://dashscope.aliyuncs.com",
    "apiKey": "your-image-edit-api-key"
  },
  "voice": {
    "model": "qwen3-tts-flash",
    "baseUrl": "https://dashscope.aliyuncs.com",
    "apiKey": "your-voice-api-key",
    "voice": "Cherry",
    "language": "Chinese",
    "outputDir": ""
  },
  "videoModel": "pika-labs",
  "characterCount": 3,
  "sceneCount": 2,
  "animeStyle": "可爱Q版风格"
}
```

**配置说明：**
- `novelFile`: 小说文件路径
- `llm`: 大语言模型配置（用于角色提取和场景分析）
- `image`: 图像生成模型配置
- `imageEdit`: 图像编辑模型配置
- `voice`: 语音合成配置，包括音色(voice)、语言(language)等
- `videoModel`: 视频生成模型
- `characterCount`: 提取的角色数量
- `sceneCount`: 生成的场景数量
- `animeStyle`: 动漫风格设定

3. 启动服务器

```bash
# 方式一：运行编译后的服务器
./taco-server

# 方式二：从源码运行
cd backend
go run main.go
```

服务器默认监听在 `:8080` 端口。

4. 访问 Web 界面

在浏览器中打开 `http://localhost:8080` 即可使用。

## 使用流程

1. **上传小说** → 通过 Web 界面上传小说文本文件
2. **提取角色** → 系统使用 LLM 自动分析小说，提取主要角色信息
3. **生成角色图片** → 为每个角色生成统一的视觉形象
4. **场景分析** → 系统使用 LLM 将小说分解为多个场景
5. **生成场景内容** → 为每个场景生成对应的图片和音频
6. **播放预览** → 在 Web 界面中以图配文+声音形式播放生成的动漫

## API 文档

后端提供以下 HTTP API 端点（详见 `backend/main.go`）：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/upload` | POST | 上传小说文件 |
| `/config` | GET | 获取配置 |
| `/config` | POST | 更新配置 |
| `/characters` | GET | 获取角色列表 |
| `/characters/generate` | POST | 生成角色 |
| `/scenes` | GET | 获取场景列表 |
| `/scenes/generate` | POST | 生成场景 |
| `/scenes/generate-images` | POST | 为场景生成图片 |
| `/scenes/generate-audio` | POST | 为场景生成音频 |
| `/generated/*` | GET | 静态文件服务（图片、音频） |

## 技术特点

- 基于 AI 的小说理解和场景提取
- 智能角色识别和外观生成
- 角色一致性维护算法
- 文本到图像的转换技术
- 文本到语音的合成技术
- 轻量级实现，降低技术复杂度

## 应用场景

- 小说可视化
- 轻量级动漫制作
- 内容创作辅助
- 故事快速原型设计

## 开发指南

### 代码规范

- Go 代码遵循标准 Go 格式化规范（使用 `go fmt`）
- 前端代码使用原生 JavaScript，避免引入额外框架
- API 调用采用标准 REST 风格
- 错误处理使用 HTTP 状态码和 JSON 响应

### 文件大小限制

- 上传文件最大 32MB

### 注意事项

- 需要配置有效的 API 密钥才能使用 AI 功能
- 生成的内容会保存在 `generated/` 目录中
- 角色和场景的配置持久化在 `config/` 目录中
- 服务器日志会输出到 `server.log` 文件

## 贡献指南

欢迎贡献！在提交 PR 前，请注意：

- 添加新功能前，先了解现有的 API 结构
- 修改配置结构时，需要同时更新前后端代码
- 测试 AI 功能时，确保 API 配置正确且有足够的配额
- 遵循项目的代码规范和开发指南

详细开发指南请参考 `CLAUDE.md` 文件。

## 许可证

MIT License
