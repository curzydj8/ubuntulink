# UbuntuLink

用手机远程查看、操作自己授权的 Ubuntu 电脑，并实时监控硬件与系统状态。

## 当前阶段：阶段 0 —— 环境检查与项目初始化（2026-10-11）

本阶段只做检查与文档，不写业务代码。

## 项目结构（目标）

```
UbuntuLink/
├── README.md
├── LICENSE
├── .gitignore
├── docs/
│   ├── PRODUCT_SPEC.md
│   ├── ARCHITECTURE.md
│   ├── SECURITY.md
│   ├── API_SPEC.md            # 阶段 1/2 定义协议后补
│   ├── DEVELOPMENT_PLAN.md
│   ├── TEST_PLAN.md
│   └── TROUBLESHOOTING.md     # 遇到问题后补
├── android/                   # Kotlin 安卓客户端（阶段 3）
├── ubuntu-agent/              # Go 轻量 agent（阶段 1）
├── server/                    # 信令/中继服务（阶段 5，按需）
├── protocol/                  # 协议定义与版本兼容（阶段 1/2）
└── tests/                     # 集成/安全/验收测试
```

## 阶段 0 已确认的环境事实（2026-10-11 实测）

| 项目 | 实测结果 |
|---|---|
| 系统 | Ubuntu 26.04.1 LTS，内核 7.0.0-38-generic，x86_64 |
| 桌面 | GNOME Shell 50.1，**Wayland** 会话 |
| Go | 未安装（阶段 1 需安装） |
| JDK/javac | 17.0.20.1 |
| Android SDK | platform-35 / build-tools 35.0.0 / platform-tools（adb 1.0.41） |
| Git | 2.53.0 |
| 显卡 | Intel Xeon E3-1200 v2 核显（无独显；GPU 监控受限） |
| 远控基础 | gnome-remote-desktop 50.2（已装）；xdg-desktop-portal 1.21.1 + portal-gnome 50.0（ScreenCast/RemoteDesktop 可用）；PipeWire 1.6.2 |
| 温度传感器 | `sensors` 命令未安装（待验证 /sys 热区） |
| 网络 | LAN 192.168.1.102/24；Tailscale 100.105.107.86（手机 ace-2-pro 同 tailnet，当前离线） |

关键约束：构建 Runner 跑在后台服务，不在图形会话内（无 DISPLAY/WAYLAND_DISPLAY）——
Go agent 必须以用户会话内服务运行才能捕获真实桌面。

## 技术路线（阶段 0 结论）

- **远程桌面**：优先评估 xdg-desktop-portal ScreenCast（PipeWire 录屏）+ RemoteDesktop（输入注入），
  Wayland 原生、无需 root、系统弹框授权；备选 GNOME Remote Desktop（RDP）。阶段 4 实测二选一。
- **连接**：Tailscale（已有 tailnet，双端已在网）→ 零配置加密 P2P；局域网 mDNS 发现为辅；
  公网中继暂无服务器，阶段 5 明确标记未验收。
- **零配置要求**：手机 Tailscale 在线后，App 通过 tailnet 直连 agent，无需手动填 IP。

## 开发纪律

- 每阶段先声明目标/范围/验收标准，完成后如实记录命令与结果。
- 不伪造测试、日志、设备信息；不支持的指标标"不可用"，不用 0 冒充。
- 不提交密钥、令牌、私钥到 Git。
- 不擅自改系统网络/桌面/防火墙/SSH 配置。
