# KFTurbo Control Server

A Go-based Discord bot primarily designed for managing Killing Floor servers, allowing authorized users to perform limited OS-level admin actions. While customizable for broader tasks across multiple servers, it adheres to strict security principles to prevent misuse.

## Features

- **Server Management**: Start, stop, restart, or reboot KF servers.
- **Hostname Validation**: Commands can target specific servers or all servers.
- **Multi-Server**: Can be run on multiple servers simultaniously. Only targetet servers will react.

## Security considerations

- **Hardcoded Commands**: OS-Commands are strictly hardcoded to prevent unauthorized or malicious actions. User input should never be trusted and passed directly.
- **Role & Channel Restrictions**: Only users with the correct role can execute commands, and only in the specified channel.

## Configuration
This bot uses a `config.json` file (see example file) for its settings. The file must be located in the same directory as the bot's executable. Below is an explanation of the required fields.

## `config.json` Structure

```json
{
  "token": "your-discord-bot-token",
  "valid_hostnames": ["hostname1", "hostname2"],
  "valid_channelIDs": ["channelID1", "channelID2"],
  "valid_roleIDs": ["roleID1", "roleID2"]
}
```

### Fields
- **`token`**: The Discord bot token used to authenticate the bot.
- **`valid_hostnames`**: A list of hostnames that the bot is allowed to respond to. Useful for ensuring commands are directed at the correct server(s).
- **`valid_channelIDs`**: A list of channel IDs where the bot is allowed to receive and respond to commands.
- **`valid_roleIDs`**: A list of role IDs allowed to execute bot commands.


### Notes
- Ensure that the bot has sufficient permissions in the specified channels and roles and you are using the correct token. Reference Discord's developer documentation if necessary.
- Protect the `config.json` file to avoid exposing sensitive information like the bot token.

## Usage
### Discord-Commands

- `/start <hostname|all>`: Start the KF server(s).
- `/stop <hostname|all>`: Stop the KF server(s).
- `/restart <hostname|all>`: Restart the KF server(s).
- `/reboot <hostname|all>`: Reboot the server machine(s) / OS.

### Example

Restart all servers:
```bash
/restart all
```

Restart specific server:
```bash
/restart serverhostname
```

### Bot logic and command flow
- Each bot instance reads its configuration from the `config.json` file, which specifies its valid hostnames, channels, and roles.
- When a message / command is received:
  1. The bot checks if the message comes from a whitelisted channel (`valid_channelIDs`).
  2. It verifies if the user has the required role (`valid_roleIDs`).
  3. It validates the hostname argument in the command against the server's own hostname or the keyword `all`.
- If all validations pass:
  - The command is executed on the target server or all servers if `all` is specified.
  - Logs and responses are generated to indicate command execution status.
- If validations fail:
  - The bot sends an error message to the user and logs the issue.

