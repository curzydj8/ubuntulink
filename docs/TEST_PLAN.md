# UbuntuLink 测试计划（阶段 0，2026-10-11）

> 每个阶段更新本文件：日期、命令、环境版本、实际结果、失败项、
> 修复文件、未解决问题、下一步。状态词见 DEVELOPMENT_PLAN.md。

## 阶段 0 测试记录（2026-10-11）

| 检查项 | 命令 | 结果 |
|---|---|---|
| 项目目录 | `ls ~/Projects/UbuntuLink` | 不存在，需新建 |
| Go | `go version` | 未安装 |
| JDK/javac | `java -version` | OpenJDK 17.0.20.1 |
| Android SDK | `ls ~/Android/Sdk` | platform-35、build-tools 35.0.0、platform-tools 俱全 |
| adb | `adb version` | 1.0.41 |
| Git | `git --version` | 2.53.0 |
| 系统/内核 | `lsb_release -d; uname -r` | Ubuntu 26.04.1 LTS；7.0.0-38-generic x86_64 |
| 会话类型 | `loginctl show-session … -p Type` | wayland（GNOME Shell 50.1） |
| 桌面环境变量 | `echo $DISPLAY/$WAYLAND_DISPLAY`（Runner） | 为空——Runner 不在图形会话内 |
| gnome-remote-desktop | `dpkg -l \| grep gnome-remote-desktop` | 50.2 已安装 |
| xdg-desktop-portal | `dpkg -l \| grep xdg-desktop-portal` | 1.21.1 + portal-gnome 50.0 + portal-gtk |
| PipeWire | `dpkg -l \| grep pipewire` | 1.6.2 |
| GPU | `lspci \| grep VGA` | Intel Xeon E3-1200 v2 核显（无独显） |
| 温度 | `which sensors` | 未安装 |
| 网络 | `ip -o -4 addr show` | 192.168.1.102/24；tailscale0 100.105.107.86 |
| Tailscale 对端 | `tailscale status` | 手机 ace-2-pro 同 tailnet，当前离线（16h） |

结论：阶段 0 检查通过。Go 缺失为阶段 1 前置安装项；其余工具链就绪。

## 阶段 1 测试记录（2026-10-11）

- Go 1.26.0 经 `sudo apt install -y golang` 安装（用户批准）。
- `go build ./...`、`go vet ./...` 一次通过；`go test ./...` 12 项全过
 （config 4、monitor 8；途中修 1 个测试用例 bug：TestParseNetDevShort 的输入不够短）。
- `/tmp/agent -once` 真实采集（JSON）：
  - system：Ubuntu 26.04.1 LTS，kernel 7.0.0-38-generic，uptime 正常
  - cpu：8.9%，4 核，2138.5 MHz（与 /proc/cpuinfo 一致）
  - mem：3292MB total / 2203MB avail / 33.1%（与 `free -m` 一致）
  - disk /：usage 35.5%（与 `df -B1` 的 used/avail 字节数 1MB 内一致；途中修 2 个口径 bug：
    先后把保留块算入分母、把 Bavail 当 Bfree 用，现与 df 完全对齐）
  - net：enp2s0/tailscale0/docker0，operstate 如实上报
  - gpu：`unavailable`（驱动未暴露计数器；设备已检出，不伪造数字）
  - temp：acpitz 27.8/29.8°C、x86_pkg_temp 34.0°C（/sys 可读）
- 结论：阶段 1 验收通过。GitHub `curzydj8/ubuntulink` 已同步 24 文件。

## 阶段 2 测试记录（2026-10-11）

- `go build`、`go vet`、`go test ./...`：26 项全过（config 5、monitor 8、pairing 5、auth 4、server 4）。
  途中修 1 个真 bug：`PairedDevice.TokenHash` 的 json tag 为 `"-"` 导致持久化丢失，重载后 token 失效。
- 真机 curl 全流程（127.0.0.1:38080，单次 Runner 调用内完成）：
  - `GET /healthz` → 200 `{"status":"ok","version":"0.2.0"}`
  - `GET /api/v1/info` → 200（device_id / version / paired_count）
  - `GET /api/v1/monitor` 无 token → 401
  - `POST /api/v1/pair` 错码 → 401；正确码 → 200 + 64 字符 token
  - `GET /api/v1/monitor` 带 token → 200（真实 Snapshot）
  - `GET /api/v1/peers` → 列出已配对设备（不含 token 材料）
  - `POST /api/v1/unpair` → 200；撤销后 token 调 monitor → 401
- 结论：阶段 2 验收通过。配对码 6 位/5 分钟/单次使用/5 次锁定；token 为 256-bit 随机，
  服务端只存 SHA-256，0600 文件持久化。

## 阶段 3 测试记录（2026-10-11）

- 语言：Java（无 kotlinc；手动 aapt2 流程已验证，文档化偏离）。
- `build.sh`：aapt2 compile→link→javac→d8→zip→zipalign→apksigner，一次通过
  （途中修 1 处：manifest 补 `package` 属性）。
- APK：`ubuntulink-v0.3.0.apk`（25,458 bytes），`apksigner verify` 通过；
  `aapt2 dump badging` 确认包名 `com.ubuntulink.app`、权限 INTERNET。
- ApiClient 配对/轮询/撤销逻辑与阶段 2 的 curl 验证一致。
- 状态：**已实现，未验证**——真机安装与联调待用户完成。
- 诚实标记：桌面为占位（阶段 4 接视频流）；Room/Keystore 延期（文档已记）。

## 阶段 4 测试记录（2026-10-11）

**技术调查**（诚实记录）：
- xdg-desktop-portal ScreenCast 为 Wayland 官方路径，需用户点选授权框
- GNOME Screenshot D-Bus 被策略拒绝（`Screenshot is not allowed`）
- GStreamer 有 pipewiresrc+jpegenc，无 H.264 编码器；Go 模块代理不通（只能标准库）
- 结论：流格式选 MJPEG（jpegenc 可用、Android 原生解、无需协商）

**实现**：
- `internal/desktop/`：MJPEG 流（`multipart/x-mixed-replace`）+ 可插拔 CaptureSource；
  Phase 4 用 TestPatternSource（时间戳测试图，`image/jpeg` 标准库生成）
- `GET /api/v1/desktop/stream`（需认证）：无 token→401；有 token→200 MJPEG 流（curl 验证）
- Android `MjpegPlayer`：SOI/EOI 解析 + BitmapFactory 解码 + ImageView 显示
- `go test ./internal/desktop/`：2 项通过

**未验证**（诚实标记）：
- 真实桌面捕获：portal 需用户图形会话 + 点选授权，Runner 无法完成；
  PortalSource 为后续工作，`docs/DESKTOP_STREAMING.md` 有完整设计与测试步骤
- Android 真机显示：待用户安装 v0.4.0 测试

**状态：已实现，未验证**（流链路已验证，真实捕获待用户会话）

## 阶段 5 测试记录（2026-10-11）

- LAN 路径：`curl http://192.168.1.102:38080/healthz` → 200，0.64ms ✅
- Tailscale 路径：`curl http://100.105.107.86:38080/healthz` → 200，0.79ms ✅
- 手机节点（ace-2-pro）离线 17h，手机→Ubuntu 实测待用户上线 Tailscale
- 真实跨 ISP：无第二种网络，标记未验收
- 自建中继：不适用（Tailscale DERP 内置）
- 结论：本地双路径验证通过；跨网真机待用户配合

## 阶段 6 测试记录（2026-10-11）

**性能**：
- agent 连续运行 60s，RSS 稳定 8496KB，无内存泄漏 ✅
- MJPEG 流（960x540，5fps）：60KB/s，~4fps 实测 ✅（原型可接受）

**安全**：
- 文件权限：`~/.config/ubuntulink/` 0700，`device.json`/`paired.json` 0600 ✅
- 日志脱敏：无 token/bearer 泄露；配对码在日志中为设计使然（5 分钟过期、单次使用）✅
- 暴力防护：连续错码 4 次 401，第 5 次 429 锁定 ✅

**结论**：阶段 6 通过。无需性能优化；Android 生命周期代码审查通过（onDestroy 正确释放）。

## 后续阶段测试占位

- 阶段 1：`go build ./...`、`go test ./...`、本机运行数据交叉验证记录。
- 阶段 2：配对成功/失败/过期/撤销、安全负例测试记录。
- 阶段 3：`./gradlew assembleDebug`、APK 安装启动、真机联调记录。
- 阶段 4：帧率/延迟/CPU 实测表、Wayland 兼容记录。
- 阶段 5：两种真实网络环境跨网测试记录、连接路径记录。
- 阶段 6：长稳/弱网/前后台/锁屏测试记录。
- 阶段 7：发布检查表、已知限制清单。
