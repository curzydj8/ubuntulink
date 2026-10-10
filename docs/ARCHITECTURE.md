# UbuntuLink 架构（阶段 0，2026-10-11）

## 1. 总览

```
┌──────────────┐      tailnet/LAN (加密)       ┌───────────────────┐
│ Android App  │ ◄──────────────────────────► │ Ubuntu Go Agent   │
│ (Kotlin)     │   配对/认证/监控/远控/信令      │ (用户会话内服务)   │
└──────────────┘                              └─────────┬─────────┘
                                                      │ portal API
                                                      ▼
                                             ┌───────────────────┐
                                             │ Wayland/GNOME 50  │
                                             │ ScreenCast+Remote │
                                             │ Desktop (PipeWire)│
                                             └───────────────────┘
```

公网信令/中继服务（`server/`）：阶段 5 按需设计，第一版不部署。

## 2. Ubuntu Go Agent

- 语言：Go（阶段 1 安装工具链后开发）。
- 运行形态：**systemd user service**（`systemd --user`），随图形会话启动，
  继承 `WAYLAND_DISPLAY`/`XDG_RUNTIME_DIR`，以普通用户身份运行（非 root）。
- 职责划分：
  - 监控模块：只读采集（/proc、/sys），低权限。
  - 配对/会话模块：设备身份、配对码、会话认证。
  - 桌面模块：经 xdg-desktop-portal 请求 ScreenCast（视频流），
    系统弹框由用户授权；仅授权会话可观看。**不实现输入注入（只读）。**
- 不提供任何远程操作接口：无鼠标/键盘注入，无任意命令执行（第一版明确不做）。
- 配置文件 + 结构化日志 + 日志轮转；`/healthz` 健康检查；版本信息。

## 3. Android 客户端

- Kotlin（Jetpack Compose 在阶段 3 评估；若构建/体积代价大可退回 View 体系，
  以 translator-app 的 View 方案为保底）。
- 通信：Tailscale 内优先明文 WS（WireGuard 已加密）或 WSS；局域网直连用 WSS。
- 本地存储：Room（设备列表、偏好、非敏感历史）；长期凭据进 Android Keystore。
- 主屏：上半桌面视频流（H.264 解码、缩放、方向适配），下半监控面板；
  一键切换全屏桌面 / 完整监控面板。**无触控映射、无键盘输入（只读）。**

## 4. 连接模型

```
配对：二维码/短码（有有效期+尝试次数限制）→ 双方交换设备身份+长期密钥
会话：每次连接做认证握手 → 监控通道 / 桌面观看通道
路径优先级：Tailscale P2P > LAN 直连 > 中继（无服务器=不可用，明确标记）
断线：心跳超时 → 冻结观看会话并提示；重连需重新认证
撤销：任一端可撤销配对，立即失效全部会话
```

## 5. 协议（protocol/，阶段 1/2 定义）

至少覆盖：设备注册与配对、在线状态/心跳、认证与会话建立、
监控数据请求/响应、连接协商、桌面观看会话建立/关闭、
错误码、协议版本兼容、权限拒绝与会话撤销。
`API_SPEC.md` 在协议定稿后补。

## 6. 模块与扩展预留

- `ubuntu-agent/internal/`：monitor、pairing、session、desktop、config、log。
- `android/`：device list、pairing、detail、monitor、remote desktop、settings、diagnostics。
- 预留：多设备管理（设备列表已支持多条目）、文件传输（独立通道+授权）、
  系统诊断（agent 诊断接口）。
