package com.ubuntulink.app;

import android.app.Activity;
import android.content.Intent;
import android.os.Bundle;
import android.view.View;
import android.widget.Button;
import android.widget.EditText;
import android.widget.TextView;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/** Pairing screen: auto-discovery + manual address + 6-digit code. */
public class PairActivity extends Activity {

    private EditText etIp;
    private EditText etCode;
    private TextView tvStatus;
    private Button btnPair;
    private ExecutorService executor;
    private Prefs prefs;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_pair);

        prefs = new Prefs(this);
        executor = Executors.newSingleThreadExecutor();

        etIp = findViewById(R.id.et_ip);
        etCode = findViewById(R.id.et_code);
        tvStatus = findViewById(R.id.pair_status);
        btnPair = findViewById(R.id.btn_pair);

        findViewById(R.id.btn_discover).setOnClickListener(v -> onDiscover());
        btnPair.setOnClickListener(v -> onPair());

        // Try discovery immediately for zero-config UX.
        onDiscover();
    }

    private void setStatus(final String s) {
        runOnUiThread(() -> tvStatus.setText(s));
    }

    private void onDiscover() {
        setStatus("正在发现 Agent…");
        executor.execute(() -> {
            String url = ApiClient.discover();
            if (url != null) {
                runOnUiThread(() -> {
                    etIp.setText(url.replace("http://", "").replace(":38080", ""));
                    setStatus("已发现: " + url);
                });
            } else {
                setStatus("未发现，请手动输入地址（Tailscale 或局域网 IP）");
            }
        });
    }

    private void onPair() {
        final String host = etIp.getText().toString().trim();
        final String code = etCode.getText().toString().trim();
        if (host.isEmpty()) {
            setStatus("请先填写 Agent 地址");
            return;
        }
        if (code.length() != 6) {
            setStatus("请输入 6 位配对码");
            return;
        }
        btnPair.setEnabled(false);
        setStatus("配对中…");
        executor.execute(() -> {
            try {
                String baseUrl = ApiClient.baseUrl(host);
                String token = ApiClient.pair(baseUrl, code, prefs.getClientId(), "Android");
                prefs.savePairing(baseUrl, token);
                runOnUiThread(() -> {
                    startActivity(new Intent(PairActivity.this, MainActivity.class));
                    finish();
                });
            } catch (Exception e) {
                setStatus("配对失败: " + e.getMessage());
                runOnUiThread(() -> btnPair.setEnabled(true));
            }
        });
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        executor.shutdownNow();
    }
}
