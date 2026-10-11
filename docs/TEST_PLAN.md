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

## 后续阶段测试占位

- 阶段 1：`go build ./...`、`go test ./...`、本机运行数据交叉验证记录。
- 阶段 2：配对成功/失败/过期/撤销、安全负例测试记录。
- 阶段 3：`./gradlew assembleDebug`、APK 安装启动、真机联调记录。
- 阶段 4：帧率/延迟/CPU 实测表、Wayland 兼容记录。
- 阶段 5：两种真实网络环境跨网测试记录、连接路径记录。
- 阶段 6：长稳/弱网/前后台/锁屏测试记录。
- 阶段 7：发布检查表、已知限制清单。
