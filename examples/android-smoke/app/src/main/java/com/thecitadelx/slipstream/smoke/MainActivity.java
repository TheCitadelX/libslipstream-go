package com.thecitadelx.slipstream.smoke;

import android.app.Activity;
import android.os.Bundle;
import android.util.Log;
import android.widget.TextView;

import com.thecitadelx.slipstream.mobile.Client;
import com.thecitadelx.slipstream.mobile.ClientConfig;
import com.thecitadelx.slipstream.mobile.Event;
import com.thecitadelx.slipstream.mobile.Mobile;

public final class MainActivity extends Activity {
    private static final String TAG = "SlipstreamSmoke";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        TextView text = new TextView(this);
        text.setTextSize(18);
        text.setPadding(32, 32, 32, 32);
        setContentView(text);

        try {
            ClientConfig config = new ClientConfig();
            config.setResolversCSV("1.1.1.1:53");
            config.setDomain("example.com");
            config.setAllowInsecure(true);
            config.setInitialPacketSize(1200);
            config.setEventQueueSize(8);

            Client client = Mobile.newClient(config);
            Event event = client.events().next(1000);
            boolean connected = client.connected();
            String eventMessage = event == null ? "no-event" : event.getLevel() + ":" + event.getMessage();
            String message = "SLIPSTREAM_SMOKE_OK connected=" + connected + " event=" + eventMessage;
            Log.i(TAG, message);
            text.setText(message);
        } catch (Exception e) {
            String message = "SLIPSTREAM_SMOKE_FAIL " + e.getMessage();
            Log.e(TAG, message, e);
            text.setText(message);
        }
    }
}
