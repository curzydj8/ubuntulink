package com.ubuntulink.app;

import android.app.Activity;
import android.content.Intent;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.view.View;
import android.widget.Button;
import android.widget.FrameLayout;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;
import android.widget.Toast;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.Locale;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Main screen: top = live desktop (Phase 4 video), bottom = monitoring panel.
 * Read-only: no input is ever sent to the computer.
 */
public class MainActivity extends Activity {

    private static final int POLL_MS = 5000;

    private static final int MODE_SPLIT = 0;
    private static final int MODE_DESKTOP = 1;
    private static final int MODE_MONITOR = 2;

    private Prefs prefs;
    private ExecutorService executor;
    private Handler handler;
    private Runnable pollTask;
    private int mode = MODE_SPLIT;

    private FrameLayout desktopContainer;
    private ScrollView monitorContainer;
    private TextView monitorText;
    private TextView desktopStatus;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        prefs = new Prefs(this);
        if (!prefs.isPaired()) {
            startActivity(new Intent(this, PairActivity.class));
            finish();
            return;
        }

        setContentView(R.layout.activity_main);

        desktopContainer = findViewById(R.id.desktop_container);
        monitorContainer = findViewById(R.id.monitor_container);
        monitorText = findViewById(R.id.monitor_text);
        desktopStatus = findViewById(R.id.desktop_status);

        findViewById(R.id.btn_split).setOnClickListener(v -> setMode(MODE_SPLIT));
        findViewById(R.id.btn_full_desktop).setOnClickListener(v -> setMode(MODE_DESKTOP));
        findViewById(R.id.btn_full_monitor).setOnClickListener(v -> setMode(MODE_MONITOR));
        findViewById(R.id.btn_unpair).setOnClickListener(v -> onUnpair());

        executor = Executors.newSingleThreadExecutor();
        handler = new Handler(Looper.getMainLooper());
        pollTask = new Runnable() {
            @Override
            public void run() {
                fetchMonitor();
                handler.postDelayed(this, POLL_MS);
            }
        };
        handler.post(pollTask);
    }

    private void setMode(int m) {
        mode = m;
        LinearLayout.LayoutParams desktopLp =
                (LinearLayout.LayoutParams) desktopContainer.getLayoutParams();
        LinearLayout.LayoutParams monitorLp =
                (LinearLayout.LayoutParams) monitorContainer.getLayoutParams();
        switch (m) {
            case MODE_DESKTOP:
                desktopContainer.setVisibility(View.VISIBLE);
                monitorContainer.setVisibility(View.GONE);
                desktopLp.weight = 1;
                break;
            case MODE_MONITOR:
                desktopContainer.setVisibility(View.GONE);
                monitorContainer.setVisibility(View.VISIBLE);
                monitorLp.weight = 1;
                break;
            default:
                desktopContainer.setVisibility(View.VISIBLE);
                monitorContainer.setVisibility(View.VISIBLE);
                desktopLp.weight = 1;
                monitorLp.weight = 1;
                break;
        }
        desktopContainer.setLayoutParams(desktopLp);
        monitorContainer.setLayoutParams(monitorLp);
    }

    private void fetchMonitor() {
        final String baseUrl = prefs.getBaseUrl();
        final String token = prefs.getToken();
        executor.execute(() -> {
            try {
                JSONObject snap = ApiClient.getMonitor(baseUrl, token);
                final String text = formatSnapshot(snap);
                runOnUiThread(() -> {
                    monitorText.setText(text);
                    desktopStatus.setText("● 已连接");
                });
            } catch (ApiClient.AuthException e) {
                // Token dead: back to pairing.
                runOnUiThread(() -> {
                    Toast.makeText(this, "授权失效，请重新配对", Toast.LENGTH_LONG).show();
                    prefs.clear();
                    startActivity(new Intent(this, PairActivity.class));
                    finish();
                });
            } catch (Exception e) {
                final String msg = e.getMessage();
                runOnUiThread(() -> {
                    monitorText.setText("连接失败: " + msg + "\n\n5 秒后重试…");
                    desktopStatus.setText("○ 未连接");
                });
            }
        });
    }

    private String formatSnapshot(JSONObject s) {
        StringBuilder b = new StringBuilder();
        try {
            JSONObject sys = s.optJSONObject("system");
            if (sys != null) {
                b.append("系统 ").append(sys.optString("os_name")).append(" ")
                        .append(sys.optString("kernel")).append("\n");
                b.append("运行 ").append(formatUptime(sys.optLong("uptime_seconds"))).append("\n\n");
            }
            JSONObject cpu = s.optJSONObject("cpu");
            if (cpu != null) {
                b.append(String.format(Locale.US, "CPU  %.1f%%（%d 核",
                        cpu.optDouble("usage_percent"), cpu.optInt("cores")));
                if ("available".equals(cpu.optString("freq_status"))) {
                    b.append(String.format(Locale.US, " %.0fMHz", cpu.optDouble("freq_mhz")));
                }
                b.append("）\n");
            }
            JSONObject mem = s.optJSONObject("memory");
            if (mem != null) {
                b.append(String.format(Locale.US, "内存 %.1f%%（%d/%dMB）\n",
                        mem.optDouble("usage_percent"),
                        mem.optLong("used_bytes") / 1048576,
                        mem.optLong("total_bytes") / 1048576));
            }
            JSONArray disks = s.optJSONArray("disks");
            if (disks != null) {
                for (int i = 0; i < disks.length(); i++) {
                    JSONObject d = disks.getJSONObject(i);
                    if (!"available".equals(d.optString("available"))) {
                        continue;
                    }
                    b.append(String.format(Locale.US, "磁盘 %s %.1f%%（%d/%dGB）\n",
                            d.optString("mountpoint"), d.optDouble("usage_percent"),
                            d.optLong("used_bytes") / 1073741824,
                            d.optLong("total_bytes") / 1073741824));
                }
            }
            JSONArray net = s.optJSONArray("network");
            if (net != null) {
                for (int i = 0; i < net.length(); i++) {
                    JSONObject n = net.getJSONObject(i);
                    b.append(String.format(Locale.US, "网络 %s %s ↑%s/s ↓%s/s\n",
                            n.optString("iface"), n.optString("operstate"),
                            formatRate(n.optLong("tx_bytes_per_sec")),
                            formatRate(n.optLong("rx_bytes_per_sec"))));
                }
            }
            JSONArray temps = s.optJSONArray("temperatures");
            if (temps != null) {
                b.append("温度 ");
                for (int i = 0; i < temps.length(); i++) {
                    JSONObject t = temps.getJSONObject(i);
                    if (!"available".equals(t.optString("available"))) {
                        continue;
                    }
                    if (i > 0) {
                        b.append("，");
                    }
                    b.append(String.format(Locale.US, "%s %.1f°C",
                            t.optString("label"), t.optDouble("celsius")));
                }
                b.append("\n");
            }
            JSONObject gpu = s.optJSONObject("gpu");
            if (gpu != null) {
                b.append("GPU  ").append("available".equals(gpu.optString("available"))
                        ? gpu.optString("name") : "不可用").append("\n");
            }
            b.append("\n更新于 ").append(s.optString("timestamp"));
        } catch (Exception e) {
            b.append("\n（解析异常: ").append(e.getMessage()).append("）");
        }
        return b.toString();
    }

    private String formatRate(long bps) {
        if (bps < 1024) {
            return bps + "B";
        } else if (bps < 1048576) {
            return String.format(Locale.US, "%.1fKB", bps / 1024.0);
        } else {
            return String.format(Locale.US, "%.1fMB", bps / 1048576.0);
        }
    }

    private String formatUptime(long sec) {
        long h = sec / 3600, m = (sec % 3600) / 60;
        if (h > 24) {
            return (h / 24) + "天" + (h % 24) + "小时";
        }
        return h + "小时" + m + "分";
    }

    private void onUnpair() {
        final String baseUrl = prefs.getBaseUrl();
        final String token = prefs.getToken();
        executor.execute(() -> {
            try {
                ApiClient.unpair(baseUrl, token);
            } catch (Exception ignored) {
            }
            runOnUiThread(() -> {
                prefs.clear();
                startActivity(new Intent(this, PairActivity.class));
                finish();
            });
        });
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        if (handler != null) {
            handler.removeCallbacks(pollTask);
        }
        if (executor != null) {
            executor.shutdownNow();
        }
    }
}
