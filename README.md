# GoIM
GoIM: A Decentralized, AI-Powered Chat System
GoIM 是一个基于 Go 语言实现的高性能、去中心化实时聊天室系统。项目深度融合了 NATS 消息系统，遵循 Web3 的点对点（P2P）和去中心化理念，构建了一个无需中央服务器即可进行消息路由的分布式通信网络。

更进一步，系统通过 langchain-go 框架集成了本地部署的 Llama 3.2 大语言模型，让每个用户都能在终端界面中与专属的 AI 助手进行实时、流畅的互动。

(在此处替换为您的终端界面截图或GIF动图)

✨ 核心特性 (Core Features)
去中心化消息路由 (Decentralized Messaging):

集成 NATS 作为分布式消息总线，实现高效、可靠的发布/订阅模型。

遵循 Web3 理念，节点之间可进行 P2P 通信，无单点故障，增强了系统的鲁棒性和抗审查性。

实时 AI 助手 (Real-Time AI Assistant):

通过 langchain-go 框架，无缝集成本地化部署的 Llama 3.2 大语言模型。

支持在聊天室内通过特定指令（如 @ai）与 AI 进行多轮对话，实现智能问答、内容创作等功能。

端到端安全加密 (End-to-End Security):

采用区块链核心的 ECDSA 签名算法进行用户身份验证，确保身份的唯一性和不可伪造性。

支持用户间分享 RSA 公钥，实现点对点的加密聊天，保障通信内容的私密性。

纯粹的终端体验 (Pure Terminal UI):

使用 tview 构建了功能完善、响应迅速的终端用户界面（TUI）。

支持实时消息提醒、用户列表切换和历史消息滚动查看，在终端中提供了现代化的聊天体验。

消息持久化 (Message Persistence):

结合本地缓存和持久化存储方案，确保了聊天记录的可靠保存和快速读取。

🏛️ 架构概述 (Architecture Overview)
GoIM 的核心是一个由 NATS 支持的去中心化网络。每个客户端启动后既是一个消息的生产者也是消费者，它们连接到同一个 NATS 网络（可以由多个 NATS 服务器集群组成）。

身份认证: 用户通过 ECDSA 私钥签名进行身份认证，确保了 Web3 世界中的身份所有权。

消息流转:

公共消息: 发布到聊天室的公共主题，所有订阅者都能收到。

加密私聊: 用户 A 获取用户 B 的 RSA 公钥后，将消息加密并发布到为他们二人设定的专属主题上，只有用户 B 的私钥能解密。

AI 交互: 发送到特定 AI 主题（如 chat.ai）的消息会被 AI 服务节点消费，该节点调用本地 Llama 3.2 模型生成回复，再将结果发布回用户的私有主题。

数据存储: 每个客户端负责将其收到的消息（公共和私有）持久化到本地存储中，以便在重启或切换视图时加载。

(您可以在此附上一张简单的架构图)

🛠️ 技术栈 (Tech Stack)
后端语言: Go

通信与消息队列: NATS, WebSocket (for potential future web clients)

AI 集成: Langchain-Go

大语言模型: Llama 3.2 (本地部署)

终端 UI: tview

加密算法: ECDSA, RSA

数据库: (例如: BadgerDB, SQLite, or simple file storage)

🚀 快速开始 (Getting Started)
先决条件
Go (版本 1.22+)

NATS Server

一个本地运行的 Llama 3.2 API 端点 (例如通过 Ollama 或 llama.cpp)
