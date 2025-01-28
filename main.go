package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var token string
var hostname string

const tokenFilePath = "token.txt"

func init() {
	content, err := ioutil.ReadFile(tokenFilePath)
	if err != nil {
		log.Fatalf("Error reading token file %v", err)
	}

	token = strings.TrimSpace(string(content))

	if token == "" {
		log.Fatal("Token file is empty")
	}

	hostname, err = os.Hostname()
	if err != nil {
		fmt.Println("Error getting hostname:", err)
		return
	}
}

func main() {
	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	sess.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		validCommandPrefixes := map[string]struct{}{
			"/start":   {},
			"/stop":    {},
			"/restart": {},
			"/reboot":  {},
			"/help":    {},
		}

		args := strings.Split((m.Content), " ")

		if !isValidCommand(validCommandPrefixes, args[0]) {
			fmt.Printf("Invalid command: '%s' is not recognized\n", args[0])
			s.ChannelMessageSend(m.ChannelID, "Invalid command")
			return
		}

		// This looks cancer but if we use proper indentation, it'll translate over to Discord
		if args[0] == "help" {
			helpMessage := `:eyes:

**Syntax:**
- **start**: Starts the KFServer(s), if not already running.
- **stop**: Stops the KFserver(s), if not already stopped.
- **restart**: Restarts the KFserver(s). Will start it if not running.
- **reboot**: Reboots the OS of the specified server(s).

**Arguments:**
- **hostname**: The specific server you want to control.
- **all**: Applies the command to all servers.

**Example Commands:**
- **To start a specific server**: start newyork
- **To restart all servers**: restart all
`
			s.ChannelMessageSend(m.ChannelID, helpMessage)

			return
		}

		if len(args) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Invalid number of arguments. Try 'help'")

			return
		}

		if args[1] != hostname && args[1] != "all" {
			fmt.Printf("Hostname mismatch: '%s' does not match servers hostname '%s'\n. Command probably is not intended for us.", args[1], hostname)
			return
		}

		allowedChannel := "1331284270311801024"
		if m.ChannelID != allowedChannel {
			fmt.Printf("Received message from invalid channel: '%s'", m.ChannelID)
			s.ChannelMessageSend(m.ChannelID, ":middle_finger:")
			return
		}

		allowedRole := "1330951734243233804"
		validUser := false
		for _, roleID := range m.Member.Roles {
			if roleID == allowedRole {
				validUser = true
				break
			}
		}

		if !validUser {
			fmt.Printf("User has incorrect role memberships: '%s'", m.Member.Roles)
			s.ChannelMessageSend(m.ChannelID, ":middle_finger:")
			return
		}

		switch args[0] {
		case "/start":
			fmt.Printf("Received command: '%s'", args[0])
			runCommand("/home/steamcmd/manage_kfserver.sh", "start-server")
			s.ChannelMessageSend(m.ChannelID, ":thumbsup:")
		case "/stop":
			fmt.Printf("Received command: '%s'", args[0])
			runCommand("/home/steamcmd/manage_kfserver.sh", "stop-server")
			s.ChannelMessageSend(m.ChannelID, ":thumbsup:")
		case "/restart":
			fmt.Printf("Received command: '%s'", args[0])
			runCommand("/home/steamcmd/manage_kfserver.sh", "restart-server")
			s.ChannelMessageSend(m.ChannelID, ":thumbsup:")
		case "/reboot":
			fmt.Printf("Received command: '%s'", args[0])
			runCommand("reboot")
			s.ChannelMessageSend(m.ChannelID, ":thumbsup:")
		default:
			return
		}
	})

	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = sess.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close()

	fmt.Println("Bot online!")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func isValidCommand(validCommands map[string]struct{}, command string) bool {
	_, exists := validCommands[command]
	return exists
}

func runCommand(command string, args ...string) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()

	fmt.Printf("Executing system command:: %s %s\n", command, strings.Join(args, " "))

	if err != nil {
		fmt.Printf("Error executing command: %v\n", err)
		return
	}
	fmt.Printf("Command output: %s\n", string(output))
}
