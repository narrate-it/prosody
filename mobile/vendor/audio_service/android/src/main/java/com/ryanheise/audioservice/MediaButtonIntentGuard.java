package com.ryanheise.audioservice;

import android.content.Intent;
import android.view.KeyEvent;

/** Prosody maintained patch: validate external media-button payloads before AndroidX dispatch. */
final class MediaButtonIntentGuard {
    private MediaButtonIntentGuard() {}

    @SuppressWarnings("deprecation")
    static boolean rejects(Intent intent) {
        // Keep restart/custom intents on the upstream path.
        if (intent == null || !Intent.ACTION_MEDIA_BUTTON.equals(intent.getAction())) {
            return false;
        }
        try {
            Object event = intent.getParcelableExtra(Intent.EXTRA_KEY_EVENT);
            return !(event instanceof KeyEvent);
        } catch (RuntimeException invalidParcel) {
            // Untrusted parcels can fail deserialization or contain the wrong type.
            // Do not log their contents or let them reach asynchronous dispatch.
            return true;
        }
    }
}
