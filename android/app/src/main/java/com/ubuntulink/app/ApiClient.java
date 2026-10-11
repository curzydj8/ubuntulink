package com.ubuntulink.app;

import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;

/**
 * Pure-Java HTTP client for the UbuntuLink agent API.
 * No Android dependencies except org.json (built into Android).
 */
public class ApiClient {

    public static final int PORT = 38080;
    public static final int TIMEOUT_MS = 8000;

    /** Well-known addresses for zero-config discovery. */
    public static final String[] DISCOVERY_HOSTS = {
            "100.105.107.86", // Tailscale (stable)
            "192.168.1.102",  // LAN (may change; manual entry as fallback)
    };

    public static String baseUrl(String host) {
        host = host.trim();
        // Allow "1.2.3.4" or "1.2.3.4:38080" or full URL.
        if (host.startsWith("http")) {
            return host;
        }
        if (host.contains(":")) {
            return "http://" + host;
        }
        return "http://" + host + ":" + PORT;
    }

    /** Try known hosts; return the first base URL whose /healthz answers. */
    public static String discover() {
        for (String host : DISCOVERY_HOSTS) {
            String url = baseUrl(host);
            try {
                String resp = http("GET", url + "/healthz", null, null);
                JSONObject o = new JSONObject(resp);
                if ("ok".equals(o.optString("status"))) {
                    return url;
                }
            } catch (Exception ignored) {
            }
        }
        return null;
    }

    public static String pair(String baseUrl, String code, String clientId, String clientName) throws Exception {
        JSONObject body = new JSONObject();
        body.put("code", code);
        body.put("client_id", clientId);
        body.put("client_name", clientName);
        String resp = http("POST", baseUrl + "/api/v1/pair", null, body.toString());
        JSONObject o = new JSONObject(resp);
        if (!o.has("token")) {
            throw new Exception(o.optString("error", "pairing failed"));
        }
        return o.getString("token");
    }

    public static JSONObject getMonitor(String baseUrl, String token) throws Exception {
        String resp = http("GET", baseUrl + "/api/v1/monitor", token, null);
        return new JSONObject(resp);
    }

    public static void unpair(String baseUrl, String token) throws Exception {
        http("POST", baseUrl + "/api/v1/unpair", token, null);
    }

    private static String http(String method, String urlStr, String token, String body) throws Exception {
        HttpURLConnection conn = null;
        try {
            URL url = new URL(urlStr);
            conn = (HttpURLConnection) url.openConnection();
            conn.setRequestMethod(method);
            conn.setConnectTimeout(TIMEOUT_MS);
            conn.setReadTimeout(TIMEOUT_MS);
            conn.setRequestProperty("Accept", "application/json");
            if (token != null) {
                conn.setRequestProperty("Authorization", "Bearer " + token);
            }
            if (body != null) {
                conn.setDoOutput(true);
                conn.setRequestProperty("Content-Type", "application/json");
                byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
                try (OutputStream os = conn.getOutputStream()) {
                    os.write(bytes);
                }
            }
            int code = conn.getResponseCode();
            InputStream is = (code >= 200 && code < 300) ? conn.getInputStream() : conn.getErrorStream();
            String resp = readAll(is);
            if (code == 401) {
                throw new AuthException(extractError(resp));
            }
            if (code == 429) {
                throw new Exception("too many attempts, try later");
            }
            if (code < 200 || code >= 300) {
                throw new Exception("HTTP " + code + ": " + extractError(resp));
            }
            return resp;
        } finally {
            if (conn != null) {
                conn.disconnect();
            }
        }
    }

    private static String extractError(String resp) {
        try {
            JSONObject o = new JSONObject(resp);
            return o.optString("error", "request failed");
        } catch (Exception e) {
            return "request failed";
        }
    }

    private static String readAll(InputStream is) throws Exception {
        if (is == null) {
            return "";
        }
        ByteArrayOutputStream bos = new ByteArrayOutputStream();
        byte[] buf = new byte[8192];
        int n;
        while ((n = is.read(buf)) != -1) {
            bos.write(buf, 0, n);
        }
        return bos.toString("UTF-8");
    }

    /** Thrown on 401: token invalid/expired or pairing code wrong. */
    public static class AuthException extends Exception {
        public AuthException(String msg) {
            super(msg);
        }
    }
}
