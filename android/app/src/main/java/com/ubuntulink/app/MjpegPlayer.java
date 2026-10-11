package com.ubuntulink.app;

import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.os.Handler;
import android.os.Looper;
import android.widget.ImageView;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;

/**
 * MJPEG stream player: parses multipart/x-mixed-replace and
 * decodes JPEG frames into an ImageView.
 * Runs network I/O on a background thread, posts bitmaps to the UI thread.
 */
public class MjpegPlayer {

    private final ImageView target;
    private final Handler uiHandler;
    private volatile boolean running;
    private Thread worker;

    public MjpegPlayer(ImageView target) {
        this.target = target;
        this.uiHandler = new Handler(Looper.getMainLooper());
    }

    public interface StatusListener {
        void onStatus(boolean connected, String message);
    }

    public synchronized void start(final String streamUrl, final String token,
                                   final StatusListener listener) {
        stop();
        running = true;
        worker = new Thread(() -> playLoop(streamUrl, token, listener));
        worker.setDaemon(true);
        worker.start();
    }

    public synchronized void stop() {
        running = false;
        if (worker != null) {
            worker.interrupt();
            worker = null;
        }
    }

    private void playLoop(String streamUrl, String token, StatusListener listener) {
        while (running) {
            HttpURLConnection conn = null;
            try {
                URL url = new URL(streamUrl);
                conn = (HttpURLConnection) url.openConnection();
                conn.setRequestProperty("Authorization", "Bearer " + token);
                conn.setConnectTimeout(8000);
                conn.setReadTimeout(15000);
                int code = conn.getResponseCode();
                if (code != 200) {
                    report(listener, false, "HTTP " + code);
                    sleepQuiet(3000);
                    continue;
                }
                report(listener, true, "● 已连接");
                parseStream(conn.getInputStream(), listener);
            } catch (Exception e) {
                if (running) {
                    report(listener, false, "○ 未连接");
                    sleepQuiet(3000);
                }
            } finally {
                if (conn != null) {
                    conn.disconnect();
                }
            }
        }
    }

    private void report(StatusListener listener, boolean connected, String msg) {
        if (listener != null) {
            uiHandler.post(() -> listener.onStatus(connected, msg));
        }
    }

    private void sleepQuiet(long ms) {
        try {
            Thread.sleep(ms);
        } catch (InterruptedException ignored) {
        }
    }

    /**
     * Minimal multipart parser: finds JPEG SOI (FF D8) .. EOI (FF D9)
     * markers and decodes each frame. Tolerant of boundary variations.
     */
    private void parseStream(InputStream is, StatusListener listener) throws Exception {
        ByteArrayOutputStream frameBuf = new ByteArrayOutputStream();
        boolean inFrame = false;
        int prev = -1;
        byte[] buf = new byte[16384];
        int n;
        while (running && (n = is.read(buf)) != -1) {
            for (int i = 0; i < n; i++) {
                int b = buf[i] & 0xFF;
                if (!inFrame) {
                    // Look for SOI: FF D8
                    if (prev == 0xFF && b == 0xD8) {
                        inFrame = true;
                        frameBuf.reset();
                        frameBuf.write(0xFF);
                        frameBuf.write(0xD8);
                    }
                } else {
                    frameBuf.write(b);
                    // Look for EOI: FF D9
                    if (prev == 0xFF && b == 0xD9) {
                        inFrame = false;
                        decodeAndShow(frameBuf.toByteArray());
                        frameBuf.reset();
                    }
                    // Sanity cap: drop absurdly large frames.
                    if (frameBuf.size() > 8 * 1024 * 1024) {
                        inFrame = false;
                        frameBuf.reset();
                    }
                }
                prev = b;
            }
        }
    }

    private void decodeAndShow(byte[] jpeg) {
        final Bitmap bmp = BitmapFactory.decodeByteArray(jpeg, 0, jpeg.length);
        if (bmp != null && running) {
            uiHandler.post(() -> {
                if (running) {
                    target.setImageBitmap(bmp);
                }
            });
        }
    }
}
