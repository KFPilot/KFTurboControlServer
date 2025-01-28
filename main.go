package main

import (
	"encoding/json"
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

type Config struct {
	Token           string   `json:"token"`
	ValidHostnames  []string `json:"valid_hostnames"`
	ValidChannelIDs []string `json:"valid_channelIDs"`
	ValidRoleIDs    []string `json:"valid_roleIDs"`
}

var token string
var hostname string
var config Config

func init() {
	configFilePath := "config.json"

	configData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Fatalf("[x] Error reading config file: %v", err)
	}

	err = json.Unmarshal(configData, &config)
	if err != nil {
		log.Fatalf("[x] Error parsing config file: %v", err)
	}

	validateConfig()

	hostname, err = os.Hostname()
	if err != nil {
		fmt.Println("[x] Error getting hostname:", err)
		return
	}
}

func main() {
	sess, err := discordgo.New("Bot " + config.Token)
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
			fmt.Printf("[-] Invalid command: '%s' is not recognized\n", args[0])
			s.ChannelMessageSend(m.ChannelID, "Invalid command, try /help")
			return
		}

		// This looks cancer but if we use proper indentation, it'll translate over to Discord
		if args[0] == "/help" {
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

		validationResponse, err := validateMessageParameters(m.ChannelID, m.Member.Roles, args[1])
		if err != nil {
			fmt.Printf("[-] Message verification failed: %v\n", err)
			s.ChannelMessageSend(m.ChannelID, validationResponse)
			return
		}

		switch args[0] {
		case "/start":
			execResponse := handleCommand(args[0], "/home/steamcmd/manage_kfserver.sh", "start-server")
			s.ChannelMessageSend(m.ChannelID, execResponse)
		case "/stop":
			execResponse := handleCommand(args[0], "/home/steamcmd/manage_kfserver.sh", "stop-server")
			s.ChannelMessageSend(m.ChannelID, execResponse)
		case "/restart":
			execResponse := handleCommand(args[0], "/home/steamcmd/manage_kfserver.sh", "restart-server")
			s.ChannelMessageSend(m.ChannelID, execResponse)
		case "/reboot":
			execResponse := handleCommand(args[0], "reboot", "")
			s.ChannelMessageSend(m.ChannelID, execResponse)
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

	fmt.Println("[+] Bot online and waiting for commands!")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func validateConfig() {
	if config.Token == "" {
		log.Fatal(config.Token)
		log.Fatal("[x] Token in config file is empty")
	}
	if len(config.ValidHostnames) == 0 {
		log.Fatal("[x] ValidHostnames in config file is empty")
	}
	if len(config.ValidChannelIDs) == 0 {
		log.Fatal("[x] ValidChannelIDs in config file is empty")
	}
	if len(config.ValidRoleIDs) == 0 {
		log.Fatal(".ValidRoleIDs in config file is empty")
	}
}

func validateMessageParameters(channelID string, userRoles []string, hostnameArg string) (validationResponse string, err error) {

	isValidChannel := false
	for _, validChannel := range config.ValidChannelIDs {
		if channelID == validChannel {
			isValidChannel = true
			break
		}
	}
	if !isValidChannel {
		validationResponse := "We are not within a whitelisted channel"
		err := fmt.Errorf("[-] Message from within invalid channel: '%s'", channelID)
		return validationResponse, err
	}

	hasValidRole := false
	for _, userRole := range userRoles {
		for _, validRole := range config.ValidRoleIDs {
			if userRole == validRole {
				hasValidRole = true
				break
			}
		}
		if hasValidRole {
			break
		}
	}
	if !hasValidRole {
		validationResponse := "You don't have permission to do this"
		err := fmt.Errorf("[-] Message from user with no valid role membership: '%v'", userRoles)
		return validationResponse, err
	}

	isValidHostname := false
	for _, validHostname := range config.ValidHostnames {
		if hostnameArg == validHostname {
			isValidHostname = true
			break
		}
	}

	if !isValidHostname {
		validationResponse := "Invalid hostname"
		err := fmt.Errorf("[-] Message contained invalid hostname argument: '%s'", hostnameArg)
		return validationResponse, err
	}

	if hostnameArg != hostname && hostnameArg != "all" {
		validationResponse := ""
		err := fmt.Errorf("[!] Hostname mismatch: '%s' does not match servers hostname '%s'. Command probably is not intended for us.", hostnameArg, hostname)
		return validationResponse, err
	}

	return "", nil
}

func isValidCommand(validCommands map[string]struct{}, command string) bool {
	_, exists := validCommands[command]
	return exists
}

func handleCommand(command string, path string, arg string) string {
	fmt.Printf("[+] Received command: '%s' '%s'\n", command, arg)
	err := execCommand(path, arg)
	if err != nil {
		fmt.Printf("[!] Command failed: %v\n", err)
		return "Command execution failed"
	}

	return "Command execution succesful"
}

func execCommand(path string, args ...string) error {
	cmd := exec.Command(path, args...)
	output, err := cmd.CombinedOutput()

	fmt.Printf("[+] Executing system command:: %s %s\n", path, strings.Join(args, " "))

	if err != nil {
		return err
	}
	fmt.Printf("[*] Command output: %s\n", string(output))

	return nil
}
