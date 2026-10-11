package com.ubuntulink.app;

import android.content.Context;
import android.content.SharedPreferences;

import java.util.UUID;

/**
 * Phase 3 storage: SharedPreferences.
 * (Single-device v1; Room deferred to a later phase if multi-device lands.)
 * Token is an opaque Bearer token, not a password; SharedPreferences is
 * app-private. Keystore migration is a Phase 6 hardening item.
 */
public class Prefs {

    private static final String NAME = "ubuntulink";
    private static final String KEY_BASE_URL = "base_url";
    private static final String KEY_TOKEN = "token";
    private static final String KEY_CLIENT_ID = "client_id";

    private final SharedPreferences sp;

    public Prefs(Context ctx) {
        sp = ctx.getSharedPreferences(NAME, Context.MODE_PRIVATE);
    }

    public boolean isPaired() {
        return getToken() != null && getBaseUrl() != null;
    }

    public String getBaseUrl() {
        return sp.getString(KEY_BASE_URL, null);
    }

    public String getToken() {
        return sp.getString(KEY_TOKEN, null);
    }

    public String getClientId() {
        String id = sp.getString(KEY_CLIENT_ID, null);
        if (id == null) {
            id = UUID.randomUUID().toString();
            sp.edit().putString(KEY_CLIENT_ID, id).apply();
        }
        return id;
    }

    public void savePairing(String baseUrl, String token) {
        sp.edit()
                .putString(KEY_BASE_URL, baseUrl)
                .putString(KEY_TOKEN, token)
                .apply();
    }

    public void clear() {
        sp.edit().remove(KEY_BASE_URL).remove(KEY_TOKEN).apply();
    }
}
