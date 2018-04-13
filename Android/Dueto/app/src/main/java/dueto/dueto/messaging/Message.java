package dueto.dueto.messaging;

import org.json.JSONException;
import org.json.JSONObject;

/**
 *
 */

public class Message
{
    private String message, time, user;
    private int type;
    private JSONObject json;

    public static final int SENT = 0, RECEIVED = 1;

    public Message(JSONObject jsonMessage, int type)
    {
        this.type = type;
        try {
            message = jsonMessage.getString("Message");
            time = jsonMessage.getString("Time");
            user = jsonMessage.getString("Artist");
            this.json = jsonMessage;
        }catch (JSONException j)
        {
            message = "Message cannot be retrieved at the moment";
            time = "N/A";
        }
    }

    public String getMessage() {
        return message;
    }

    public String getTime() {
        return time;
    }

    public int getType() {
        return type;
    }

    public JSONObject toJSON()
    {
        return json;
    }
}
