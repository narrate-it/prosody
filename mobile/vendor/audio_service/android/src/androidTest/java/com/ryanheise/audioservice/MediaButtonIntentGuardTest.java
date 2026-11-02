package com.ryanheise.audioservice;

import android.app.Service;
import android.app.Activity;
import android.app.Instrumentation;
import android.content.Intent;
import android.os.BadParcelableException;
import android.os.Bundle;
import android.os.Parcelable;
import android.view.KeyEvent;
import android.support.v4.media.session.MediaSessionCompat;
import java.lang.reflect.Field;

/** Device-side tests; no media service is started and no credentials are used. */
@SuppressWarnings("deprecation")
public class MediaButtonIntentGuardTest extends Instrumentation {
    @Override
    public void onCreate(Bundle arguments) {
        super.onCreate(arguments);
        start();
    }

    @Override
    public void onStart() {
        Bundle results = new Bundle();
        try {
            testMissingNullAndWrongTypeRejectedAtBothEntryPoints();
            testUnparcelablePayloadIsRejected();
            testValidEventsAndCustomIntentsRemainOnUpstreamPath();
            results.putString("stream", "Media-button guard checks passed\n");
            finish(Activity.RESULT_OK, results);
        } catch (Throwable failure) {
            results.putString("stream", "Media-button guard checks failed: " + failure.getClass().getName());
            finish(Activity.RESULT_CANCELED, results);
        }
    }

    private static void assertTrue(boolean value) {
        if (!value) throw new AssertionError("Expected rejection");
    }

    private static void assertFalse(boolean value) {
        if (value) throw new AssertionError("Expected upstream dispatch");
    }

    private static void assertEquals(int expected, int actual) {
        if (expected != actual) throw new AssertionError("Unexpected service result");
    }
    public void testMissingNullAndWrongTypeRejectedAtBothEntryPoints() throws Exception {
        Intent[] invalid = new Intent[] {
            new Intent(Intent.ACTION_MEDIA_BUTTON),
            new Intent(Intent.ACTION_MEDIA_BUTTON).putExtra(Intent.EXTRA_KEY_EVENT, (Parcelable) null),
            new Intent(Intent.ACTION_MEDIA_BUTTON).putExtra(Intent.EXTRA_KEY_EVENT, new Bundle())
        };
        for (Intent intent : invalid) {
            assertTrue(MediaButtonIntentGuard.rejects(intent));
            // If either entry point forwards this, an unattached service/context
            // or AndroidX dispatch will fail rather than silently pass the test.
            new MediaButtonReceiver().onReceive(getContext(), intent);
            assertServiceRejects(intent);
        }
    }

    public void testUnparcelablePayloadIsRejected() throws Exception {
        Intent malformed = new Intent(Intent.ACTION_MEDIA_BUTTON) {
            @Override
            public <T extends Parcelable> T getParcelableExtra(String name) {
                throw new BadParcelableException("synthetic malformed parcel");
            }
        };
        assertTrue(MediaButtonIntentGuard.rejects(malformed));
        new MediaButtonReceiver().onReceive(getContext(), malformed);
        assertServiceRejects(malformed);
    }

    private void assertServiceRejects(Intent intent) throws Exception {
        AudioService service = new AudioService();
        MediaSessionCompat session = new MediaSessionCompat(getTargetContext(), "prosody-intent-guard-test");
        try {
            // A null session would make AndroidX silently return before decoding
            // the event and mask a missing service guard. Use a real session.
            Field field = AudioService.class.getDeclaredField("mediaSession");
            field.setAccessible(true);
            field.set(service, session);
            assertEquals(Service.START_NOT_STICKY, service.onStartCommand(intent, 0, 1));
        } finally {
            session.release();
        }
    }

    public void testValidEventsAndCustomIntentsRemainOnUpstreamPath() {
        Intent valid = new Intent(Intent.ACTION_MEDIA_BUTTON).putExtra(
            Intent.EXTRA_KEY_EVENT, new KeyEvent(KeyEvent.ACTION_DOWN, KeyEvent.KEYCODE_MEDIA_PLAY));
        assertFalse(MediaButtonIntentGuard.rejects(valid));
        assertFalse(MediaButtonIntentGuard.rejects(new Intent(MediaButtonReceiver.ACTION_NOTIFICATION_DELETE)));
        assertFalse(MediaButtonIntentGuard.rejects(null));
    }
}
