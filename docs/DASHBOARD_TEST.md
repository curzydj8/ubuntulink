# 动态机器信息页面 — 测试验证方案

> 范围：Go agent 新增端点 `GET /dashboard`（HTML）与
> `GET /api/v1/dashboard/metrics`（JSON，无需认证）。
> HTML 源码位置：`ubuntu-agent/web/dashboard.html`。
> 本文档每一步都有明确通过标准；任一步失败即打回实现。

---

## 1. 后端单元测试清单

运行方式（在 `ubuntu-agent/` 目录下）：

```bash
go test ./internal/server/ -run 'Dashboard' -v
```

### 1.1 `TestDashboardPageNoAuth`

- **测什么**：未携带任何 token 时 `GET /dashboard` 仍返回 200。
- **预期**：`StatusCode == 200`；`Content-Type` 以 `text/html` 开头；
  body 包含 `<html` 且包含标识字符串 `ubuntulink-dashboard`（实现方必须在
  HTML 根元素上加 `id="ubuntulink-dashboard"`，否则此测试无法定位页面）。
- **不通过**：返回 401/404，或 Content-Type 不是 HTML。

### 1.2 `TestDashboardPageServesEmbeddedHTML`

- **测什么**：页面内容与 `web/dashboard.html` 源码一致（防止 serve 了空模板或旧缓存）。
- **预期**：body 与仓库中 `web/dashboard.html` 文件内容逐字节一致
  （实现必须用 `//go:embed`，禁止运行时读磁盘——否则文件缺失会导致线上 500）。
- **不通过**：body 为空、长度 < 1KB，或与源码文件不一致。

### 1.3 `TestDashboardMetricsNoAuth`

- **测什么**：`GET /api/v1/dashboard/metrics` 无 token 返回 200。
- **预期**：`StatusCode == 200`；`Content-Type` 为 `application/json`；
  body 可被 `json.Unmarshal` 解析为 `map[string]any` 且不报错。
- **不通过**：401（误加了 auth 中间件）、非 JSON。

### 1.4 `TestDashboardMetricsFields`

- **测什么**：metrics JSON 包含全部必需字段且数值合法。
- **预期字段**（顶层 key，缺一不可）：
  `cpu`, `memory`, `disk`, `network`, `temperature`, `timestamp`
- **数值合法性**：
  - `cpu.percent` 在 [0, 100] 之间，或字段明确标记 `unavailable`
  - `memory.total_mb > 0`
  - `timestamp` 与当前时间差 < 60 秒（防止返回缓存快照冒充实时数据）
- **不通过**：缺字段、percent 为负数/NaN、timestamp 陈旧。

### 1.5 `TestDashboardMetricsNoSensitiveFields`

- **测什么**：metrics 不泄露敏感信息（安全红线，见第 5 节）。
- **预期**：将 body 转小写后，不包含以下任一子串：
  `token`, `bearer`, `secret`, `password`, `private_key`, `paired`
- **不通过**：包含任一子串。**此测试失败 = 实现直接打回，不许合入。**

### 1.6 `TestDashboardMetricsMethodNotAllowed`

- **测什么**：`POST /api/v1/dashboard/metrics` 被拒绝。
- **预期**：`StatusCode == 405`。
- **不通过**：返回 200（说明 handler 没做方法校验，有被误用的风险）。

### 1.7 `TestDashboardPathTraversal`

- **测什么**：路径穿越打不到文件系统。
- **预期**：`GET /dashboard/%2e%2e/%2e%2e/etc/passwd` 返回 404；
  `GET /dashboard/../../go.mod` 返回 404。
  （实现必须只 serve 嵌入的单文件，不允许任何路径拼接逻辑。）
- **不通过**：返回 200 或 500（说明有路径拼接）。

### 1.8 `TestDashboardMetricsNoQueryExecution`

- **测什么**：查询参数不被执行、不影响输出。
- **预期**：`GET /api/v1/dashboard/metrics?cmd=id` 与无参数请求返回
  完全相同的 body（实现必须忽略所有 query 参数）。
- **不通过**：body 不同（说明 query 参数参与了逻辑）。

---

## 2. API 集成测试步骤

在 Ubuntu 上执行（agent 以测试端口启动，避免与生产 38080 冲突）：

```bash
cd ~/Projects/UbuntuLink/ubuntu-agent
go build -o /tmp/agent-dash ./cmd/ubuntulink-agent
/tmp/agent-dash -port 38091 > /tmp/dash-test.log 2>&1 &
sleep 2
```

### 2.1 页面端点

```bash
curl -s -o /tmp/dash.html -w "HTTP %{http_code} | %{content_type} | %{size_download}B | %{time_total}s\n" \
  http://127.0.0.1:38091/dashboard
```

- **通过标准**：`HTTP 200`、`text/html`、大小 > 5KB（空页面通常 < 1KB）、
  耗时 < 1s。
- **附加检查**：

```bash
grep -c 'id="ubuntulink-dashboard"' /tmp/dash.html   # 必须 >= 1
grep -ci 'setInterval\|setTimeout' /tmp/dash.html    # 必须 >= 1（有轮询逻辑）
grep -c '<canvas\|<svg' /tmp/dash.html               # 必须 >= 1（有图表元素）
```

### 2.2 metrics 端点（无认证）

```bash
curl -s -w "\nHTTP %{http_code} | %{time_total}s\n" \
  http://127.0.0.1:38091/metrics 2>/dev/null
curl -s http://127.0.0.1:38091/api/v1/dashboard/metrics | python3 -m json.tool | head -40
```

- **通过标准**：HTTP 200、耗时 < 0.5s、输出为合法 JSON、
  顶层包含 `cpu/memory/disk/network/temperature/timestamp`。
- **注意**：第一条 curl 故意打错路径（`/metrics`），预期 404——确认
  metrics 只在完整路径下暴露，不在短路径下误暴露。

### 2.3 轮询稳定性（2 秒间隔 × 30 次）

```bash
for i in $(seq 1 30); do
  code=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:38091/api/v1/dashboard/metrics)
  [ "$code" != "200" ] && echo "FAIL at request $i: $code" && break
  sleep 2
done; echo "poll loop done"
```

- **通过标准**：60 秒内 30 次全部 200，无一次失败、无连接重置。

### 2.4 清理

```bash
pkill -f "/tmp/agent-dash"; echo "cleaned"
```

---

## 3. 前端手动验证清单

在桌面 Chrome/Firefox 和手机浏览器各打开一次
`http://<ubuntu-ip>:38080/dashboard`，逐项打勾：

### 3.1 数据展示

- [ ] 页面加载后 3 秒内出现 CPU / 内存 / 磁盘 / 网络 / 温度数值，
      不全是 `--` 或 `加载中`
- [ ] 等待 10 秒，至少看到一次数值变化（证明不是静态快照；
      可用 `curl` 对比两次 metrics 的 `timestamp` 不同来交叉确认）
- [ ] 不可用的指标显示"不可用"字样，**不显示 0 冒充**（如 GPU/温度缺失时）

### 3.2 图表绘制

- [ ] CPU/内存有趋势图（canvas 或 svg 非空白；右键检查元素确认
      `<canvas>` 宽高 > 0）
- [ ] 图表随轮询追加数据点，不闪烁、不重置整图
- [ ] 连续观察 2 分钟，图表无内存泄漏迹象（页面不卡顿；
      桌面端可开 DevTools → Performance 观察 JS heap 不持续爬升）

### 3.3 断线重连

- [ ] 在 Ubuntu 上 `pkill` 掉 agent → 页面 5 秒内显示"连接断开，正在重连"
      之类的明确提示（而不是静默卡死在旧数据）
- [ ] 重启 agent → 页面自动恢复数据，**无需手动刷新**
- [ ] 重连后图表不断裂成两截或可接受断裂但有标注（任选其一，行为需一致）

### 3.4 手机适配

- [ ] 在手机浏览器打开，无横向滚动条
- [ ] 字体 ≥ 12px 可读；图表在 360px 宽下完整显示
- [ ] 锁屏 1 分钟再解锁，页面自动恢复轮询（不需手动刷新）
- [ ] 切后台 5 分钟再切回，无白屏、无需重载

### 3.5 无认证访问确认

- [ ] 浏览器开隐身窗口直接打开 `/dashboard`，**不提示登录/配对**，
      直接看到数据（这是产品需求：局域网内免配对查看）

---

## 4. 性能基准

### 4.1 页面加载

```bash
curl -s -o /dev/null -w "time_total=%{time_total}s size=%{size_download}B\n" \
  http://127.0.0.1:38091/dashboard
```

| 指标 | 通过标准 | 说明 |
|---|---|---|
| 首字节+传输总耗时 | < 1.0s（LAN） | 纯静态 HTML，无服务端渲染 |
| HTML 体积 | < 150KB | 禁止引入 jQuery/Bootstrap 等重型库；只允许原生 JS + 轻量 canvas 绘图 |
| 无外部请求 | 用 DevTools Network 确认 | 页面**不许**引用 CDN/外部字体/统计脚本（内网离线环境必须可用） |

### 4.2 2 秒轮询的 CPU 开销

```bash
# 启动 agent，记录基线
/tmp/agent-dash -port 38091 > /dev/null 2>&1 &
AGENT_PID=$!; sleep 5
ps -o %cpu= -p $AGENT_PID | tr -d ' '   # 基线 CPU
# 模拟一个浏览器客户端以 2s 间隔轮询 60 秒
for i in $(seq 1 30); do curl -s http://127.0.0.1:38091/api/v1/dashboard/metrics > /dev/null; sleep 2; done &
sleep 65
ps -o %cpu=,rss= -p $AGENT_PID          # 轮询期间 CPU 与内存
kill $AGENT_PID
```

- **通过标准**：轮询期间 agent 平均 CPU < 5%（i3-3220 单核视角）；
  RSS 增长 < 5MB（无泄漏）。
- **不通过则要求**：降低采样频率、给 metrics 加 1 秒服务端缓存、
  或把网络/磁盘等重采集项改为惰性更新——三选一，不许"先这样以后优化"。

### 4.3 并发

```bash
seq 1 20 | xargs -P20 -I{} curl -s -o /dev/null -w "%{http_code}\n" \
  http://127.0.0.1:38091/api/v1/dashboard/metrics | sort | uniq -c
```

- **通过标准**：20 并发全部 200，无 500/超时（家庭场景：2~3 个设备同时看）。

---

## 5. 安全检查

> metrics 端点**无需认证**，所以它是暴露面最大的端点。以下任一条失败，
> 功能不许上线。

### 5.1 无敏感信息泄露

```bash
curl -s http://127.0.0.1:38091/api/v1/dashboard/metrics > /tmp/m.json
for kw in token bearer secret password private_key paired apikey; do
  grep -qi "$kw" /tmp/m.json && echo "LEAK: $kw" || true
done; echo "leak scan done"
```

- **通过标准**：无任何 `LEAK` 输出。
- **特别注意**：`paired_devices` 数量可以给，但**不许**给设备 token 哈希、
  client_id、IP 等可识别信息。

### 5.2 无路径遍历

```bash
for p in "/dashboard/%2e%2e/%2e%2e/etc/passwd" "/dashboard/..%2f..%2fetc%2fpasswd" \
         "/dashboard/%2e%2e/go.mod" "/api/v1/dashboard/metrics/../../etc/passwd"; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:38091$p")
  echo "$p -> $code"   # 全部必须是 404
done
```

- **通过标准**：四个全部 404。出现 200 = 直接打回。

### 5.3 无命令执行面

- [ ] `grep -rn "exec.Command\|os/exec\|syscall" ubuntu-agent/internal/server/`
      返回空（server 包不许调 exec）
- [ ] metrics handler 不读取任何 query 参数（见 1.8 测试）
- [ ] metrics 输出中不包含可执行路径以外的绝对路径；
      允许的例外只有磁盘挂载点（如 `/`, `/boot`）——用以下命令确认无
      `/home/` 用户目录泄露：

```bash
grep -o '"/home/[^"]*"' /tmp/m.json && echo "LEAK: home path" || echo "home path clean"
```

### 5.4 与认证端点的隔离

```bash
# 未配对 token 打需认证的端点必须 401，但打 metrics 必须 200
curl -s -o /dev/null -w "monitor(no token)=%{http_code}\n" http://127.0.0.1:38091/api/v1/monitor
curl -s -o /dev/null -w "metrics(no token)=%{http_code}\n" http://127.0.0.1:38091/api/v1/dashboard/metrics
```

- **通过标准**：第一行 401，第二行 200——证明 metrics 的免认证是
  **有意的单独配置**，不是 auth 中间件被全局关掉了。

---

## 6. 验收汇总表

| # | 项 | 命令/位置 | 通过标准 |
|---|---|---|---|
| 1.1–1.8 | 单元测试 | `go test ./internal/server/ -run Dashboard -v` | 8 个全过 |
| 2.1 | 页面集成 | curl /dashboard | 200 + HTML + 标识齐全 |
| 2.2 | metrics 集成 | curl metrics + json.tool | 200 + 字段齐全 |
| 2.3 | 轮询稳定 | 30 次 × 2s | 全 200 |
| 3.1–3.5 | 前端手动 | 浏览器 | 逐项打勾 |
| 4.1 | 加载性能 | curl time_total | < 1s, < 150KB, 无外部请求 |
| 4.2 | 轮询开销 | ps %cpu/rss | CPU < 5%, RSS 增长 < 5MB |
| 4.3 | 并发 | 20 并发 curl | 全 200 |
| 5.1–5.4 | 安全 | 泄露/遍历/隔离检查 | 零失败项 |

**上线门槛**：上表全部通过。安全类（第 5 节）任一项失败即一票否决。
