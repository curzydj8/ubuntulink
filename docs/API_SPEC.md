# UbuntuLink API 规格 v1（阶段 2，2026-10-11）

Base URL: `http://<agent>:38080`
传输：Tailscale 内 HTTP（WireGuard 加密）；LAN 直连的 TLS 在阶段 5/6 落实。
认证：`Authorization: Bearer <token>`（配对后签发，256-bit 随机，服务端只存 SHA-256）。

## 公开接口

### GET /healthz
`{"status":"ok","version":"0.2.0"}`

### GET /api/v1/info
发现用最小信息：`{"device_id":"…","agent_version":"0.2.0","paired_count":1}`

### POST /api/v1/pair
Body: `{"code":"123456","client_id":"…","client_name":"…"}`

| 结果 | HTTP | error_code |
|---|---|---|
| 成功 | 200 | —（返回 `{"token":"…","device_id":"…"}`） |
| 码错误 | 401 | invalid_code |
| 码过期 | 410 | expired |
| 超次锁定 | 429 | locked |
| body 非法 | 400 | bad_request |

配对码：6 位数字，5 分钟有效，单次使用（成功后轮换），5 次错锁 5 分钟。

## 需认证接口

### GET /api/v1/monitor
返回完整 Snapshot JSON（见阶段 1 类型定义）。无 token/错 token → 401 `unauthorized`。

### POST /api/v1/unpair
撤销调用者自己的 token → `{"status":"unpaired"}`。撤销后 token 立即失效。

### GET /api/v1/peers
`{"peers":[{"client_id":"…","client_name":"…","paired_at":"…"}]}`（不含 token 材料）。

### GET /api/v1/desktop/stream
MJPEG 视频流（需认证）。`Content-Type: multipart/x-mixed-replace; boundary=frame`，
每帧为 JPEG。Phase 4 为测试图案源；portal 真实捕获接入后为实时桌面。

### GET /dashboard（公开，无需认证）
嵌入的机器信息动态页面（HTML）。每 2 秒轮询下方 metrics 接口刷新。

### GET /api/v1/dashboard/metrics（公开，无需认证）
只读系统指标 JSON，供 dashboard 页面使用。只含 CPU、内存、磁盘（根分区优先）、
网络、温度；**不含**设备 ID、hostname、系统信息、GPU、token 等敏感字段。
不可用指标序列化为 `null`（页面显示"不可用"，不用 0 冒充）。

## 错误体
`{"error_code":"…","error":"…"}`

## 版本兼容
- `agent_version` 在 /info 与 /healthz 中声明；客户端不匹配大版本时应提示升级。
- v1 接口保持向后兼容；破坏性变更升 v2 路径。
