# KFTurbo Control Server

A Go-based Discord bot primarily designed for managing Killing Floor servers, allowing authorized users to perform limited OS-level admin actions. While customizable for broader tasks across multiple servers, it adheres to strict security principles to prevent misuse.

## Features

- **Server Management**: Start, stop, restart, or reboot KF servers.
- **Hostname Validation**: Commands can target specific servers or all servers.
- **Multi-Server**: Can be run on multiple servers simultaniously. Only targetet servers will react.

## Security considerations

- **Hardcoded Commands**: OS-Commands are strictly hardcoded to prevent unauthorized or malicious actions. User input should never be trusted and passed directly.
- **Role & Channel Restrictions**: Only users with the correct role can execute commands, and only in the specified channel.


## Usage
### Discord-Commands

- `/start <hostname|all>`: Start the KF server(s).
- `/stop <hostname|all>`: Stop the KF server(s).
- `/restart <hostname|all>`: Restart the KF server(s).
- `/reboot <hostname|all>`: Reboot the server machine(s) / OS.
- `/help`: Show command syntax and examples.

### Example

Restart all servers:
```bash
/restart all
```

Restart specific server:
```bash
/restart serverhostname
```
