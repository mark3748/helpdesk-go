# Discord Bot Integration

The Discord integration lets users create tickets with slash commands and keeps
Discord ticket threads synchronized with Helpdesk comments.

## Discord Application Setup

1. Create an application in the [Discord Developer Portal](https://discord.com/developers/applications).
2. Add a bot to the application and copy its token.
3. Under **Bot → Privileged Gateway Intents**, enable **Message Content Intent**.
4. Invite the bot to the target server with the `bot` and `applications.commands` scopes.
5. Grant the bot permission to:
   - View the ticket channel
   - Send messages
   - Read message history
   - Create public threads
   - Send messages in threads
6. Enable Discord developer mode, then copy the server ID and the channel ID
   that should contain ticket threads.

## Configuration

Administrators can save the bot token, server ID, and ticket channel ID from
**Admin Settings → Discord Bot**. The token is never returned to the browser,
and leaving the token field blank preserves the currently saved token.

Saved Discord settings override non-empty worker environment variables. Restart
the worker after saving because the Discord gateway connection and slash
commands are initialized when the worker starts.

The equivalent worker environment variables are:

```text
DISCORD_BOT_TOKEN
DISCORD_GUILD_ID
DISCORD_CHANNEL_ID
```

For Helm deployments, keep the token in `secrets.data` and the IDs in `env`:

```yaml
env:
  DISCORD_GUILD_ID: "123456789012345678"
  DISCORD_CHANNEL_ID: "123456789012345678"

secrets:
  enabled: true
  data:
    DISCORD_BOT_TOKEN: "replace-with-the-bot-token"
```

The same values can be added to `helm/local-values.yaml` when testing with
Tilt. After changing the file or saving values in the UI, restart the worker
resource so it reconnects with the new configuration.

## Commands And Behavior

- `/create-ticket` opens a modal and creates a Helpdesk ticket plus a public
  Discord thread in the configured channel.
- Replies in a mapped Discord thread are added to the Helpdesk ticket.
- Comments added in the internal Helpdesk UI are posted to the mapped Discord
  thread.
- `/link-email` and `/verify-email` link a Discord user to an existing
  requester after email verification.

Email linking is registered only when SMTP Host and From Address are configured
in Mail Settings or through `SMTP_HOST` and `SMTP_FROM`. Redis is also required
to queue verification emails.

## Troubleshooting

- If slash commands do not appear, verify the guild ID, confirm the bot was
  invited with the `applications.commands` scope, and restart the worker.
- If ticket creation fails, verify the channel ID and the bot's thread
  permissions.
- If thread replies do not synchronize, verify Message Content Intent is
  enabled in the Developer Portal.
- Check worker logs for Discord connection, command registration, and
  permission errors.
