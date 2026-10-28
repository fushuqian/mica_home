# RULES

项目：模块宿主（Modular Shell）—— 可插拔模块化的 Web 平台

## 工作规则
1. 老板是中国人，用中文沟通。
2. 说话精简干练，不过度解释。
3. 不擅自想象；需求模糊或信息不足时，主动问老板确认。
4. 桌面端优先 WinUI 3 / Windows App SDK；B/S 优先 Linux + Go + PostgreSQL + Nginx，开源免费第一。
5. 项目启动时自动创建并维护 RULES.md、QUEST.md、CHANGELOG.md、MEMORIES.md、IMPORTANT.md。
6. GitHub 关联项目，每次代码变更或版本发布后同步到 GitHub。
7. 所有需求记录到 QUEST.md，含描述、状态、优先级、时间。
8. 新工作上下文或连续对话超 10 轮后，重新读取规则和项目设定文件。
9. 修改概要写进 CHANGELOG.md，值得记住的经验写进 MEMORIES.md。
10. 为老板节约资源：优先免费/已付费工具，调用外部 API 前估算消耗。
11. 密钥、Token、关键配置记录到 IMPORTANT.md。
12. 不随意删除文件；暂时不用的移到 toberemoved/，项目结束后再清理。
13. 规则冲突时，以保护用户数据、代码安全和实现老板真实意图为优先。

## 技术约定（本项目）
- 宿主：Go 单二进制，容器化为 Docker 镜像
- 模块后端：WASM（wazero 运行时），不使用 Go `plugin` 包
- 模块前端：框架无关的远程 ESM，入口 `mount(el, api)`
- 数据：PostgreSQL，schema-per-module
- 模块仓库：OCI Registry
- 语言风格：前端原生 HTML/CSS/JS，无构建步骤优先
