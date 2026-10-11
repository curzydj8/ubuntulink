# 桌面流技术方案（阶段 4，2026-10-11）

## 调查结论

| 项目 | 结果 |
|---|---|
| xdg-desktop-portal ScreenCast | Wayland 官方捕获 API，需用户点选授权框 |
| GNOME Screenshot D-Bus | 被策略拒绝（`Screenshot is not allowed`），后台进程不可用 |
| GStreamer pipewiresrc | 可用 |
| GStreamer jpegenc | 可用 |
| H.264 编码器 | 无（无 x264enc/vaapih264enc/ffmpeg） |
| Go 外部依赖 | 模块代理不通，只能标准库 |

## 技术路线

**流格式：MJPEG over HTTP**（`multipart/x-mixed-replace; boundary=frame`）
- 理由：jpegenc 可用、Android 原生解 JPEG、无需编码协商、实现简单
- 代价：带宽高于 H.264；原型阶段可接受（Tailscale/LAN）
- H.264 作为 Phase 6 优化项（需 apt 装编码器）

**捕获：可插拔 `CaptureSource` 接口**
- Phase 4：`TestPatternSource`（验证流链路）
- 后续：`PortalSource`（portal D-Bus → PipeWire → 真实桌面，需用户会话测试）

**Android**：自定义 MJPEG 解析 + ImageView（WebView 无法加 Bearer 头）

## Portal 真实捕获流程（待用户会话验证）

1. Agent（用户会话内）经 D-Bus 调 `org.freedesktop.portal.Desktop.RequestScreenCast`
2. 系统弹框 → 用户选择屏幕
3. Portal 返回 PipeWire node
4. `gst-launch-1.0 pipewiresrc path=<node> ! videoconvert ! jpegenc ! multipartmux`
5. Agent 转发为 HTTP MJPEG 流

## 用户测试步骤（待执行）

1. 在 Ubuntu 桌面终端运行：`~/Projects/UbuntuLink/ubuntu-agent/agent -port 38080`
2. 看到配对码后，在手机 App 配对
3. 主屏上半应显示实时桌面（初版为测试图案，portal 接入后为真实桌面）
4. 记录帧率、延迟、CPU 占用
