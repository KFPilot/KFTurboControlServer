package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sort"
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

var hostname string
var config Config

type CommandSpec struct {
	Name        string
	Description string
	Path        string
	Args        []string
}

var commandSpecs = map[string]CommandSpec{
	"start": {
		Name:        "start",
		Description: "Start the KF server",
		Path:        "/home/steamcmd/manage_kfserver.sh",
		Args:        []string{"start-server"},
	},
	"stop": {
		Name:        "stop",
		Description: "Stop the KF server",
		Path:        "/home/steamcmd/manage_kfserver.sh",
		Args:        []string{"stop-server"},
	},
	"restart": {
		Name:        "restart",
		Description: "Restart the KF server",
		Path:        "/home/steamcmd/manage_kfserver.sh",
		Args:        []string{"restart-server"},
	},
	"reboot": {
		Name:        "reboot",
		Description: "Reboot the OS on the server",
		Path:        "reboot",
	},
}

var slashCommandDefs = []*discordgo.ApplicationCommand{
	{
		Name:        "start",
		Description: "Start the KF server",
		Options:     hostnameOption(),
	},
	{
		Name:        "stop",
		Description: "Stop the KF server",
		Options:     hostnameOption(),
	},
	{
		Name:        "restart",
		Description: "Restart the KF server",
		Options:     hostnameOption(),
	},
	{
		Name:        "reboot",
		Description: "Reboot the server machine / OS",
		Options:     hostnameOption(),
	},
	{
		Name:        "commands",
		Description: "Show the commands available to you in this channel",
	},
}

func init() {
	configFilePath := "config.json"

	configData, err := os.ReadFile(configFilePath)
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

	sess.AddHandler(onInteractionCreate)

	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = sess.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer sess.Close()

	err = registerSlashCommands(sess)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("[+] Bot online and waiting for slash commands!")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func validateConfig() {
	if config.Token == "" {
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

func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" || i.Member == nil {
		return
	}

	switch i.Type {
	case discordgo.InteractionApplicationCommandAutocomplete:
		handleAutocomplete(s, i)
	case discordgo.InteractionApplicationCommand:
		handleApplicationCommand(s, i)
	}
}

func handleApplicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name
	if name == "commands" {
		handleCommandsListing(s, i)
		return
	}

	spec, found := commandSpecs[name]
	if !found {
		_ = respondImmediate(s, i, "Unknown command.", true)
		return
	}

	err := isValidSource(i.ChannelID, i.Member.Roles)
	if err != nil {
		_ = respondImmediate(s, i, err.Error(), true)
		return
	}

	hostnameArg := getStringOption(i.ApplicationCommandData().Options, "hostname")
	if hostnameArg == "" {
		_ = respondImmediate(s, i, "Missing required option: hostname", true)
		return
	}

	err = validateHostnameParameter(hostnameArg)
	if err != nil {
		_ = respondImmediate(s, i, err.Error(), true)
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("[x] Failed to defer interaction response: %v", err)
		return
	}

	fmt.Printf("[+] %s requested by %s targeting '%s'\n", name, i.Member.User.ID, hostnameArg)
	result := executeSpec(spec)
	msg := fmt.Sprintf("`%s` on `%s` (target: `%s`)\n%s", spec.Name, hostname, hostnameArg, result)
	_, err = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: msg,
	})
	if err != nil {
		log.Printf("[x] Failed sending followup message: %v", err)
	}
}

func handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name := i.ApplicationCommandData().Name
	if _, found := commandSpecs[name]; !found {
		return
	}

	partial := strings.ToLower(getFocusedStringOption(i.ApplicationCommandData().Options))
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(config.ValidHostnames)+1)

	if strings.Contains("all", partial) {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  "all",
			Value: "all",
		})
	}

	for _, h := range config.ValidHostnames {
		if partial == "" || strings.Contains(strings.ToLower(h), partial) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  h,
				Value: h,
			})
		}
		if len(choices) == 25 {
			break
		}
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		log.Printf("[x] Failed autocomplete response: %v", err)
	}
}

func handleCommandsListing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	canUseErr := isValidSource(i.ChannelID, i.Member.Roles)
	available := sortedCommandNames()

	var sb strings.Builder
	sb.WriteString("Available commands:\n")
	for _, cmd := range available {
		sb.WriteString(fmt.Sprintf("- `/%s <hostname|all>`: %s\n", cmd, commandSpecs[cmd].Description))
	}
	sb.WriteString("- `/commands`: Show this list\n")

	if canUseErr != nil {
		sb.WriteString(fmt.Sprintf("\nYou cannot execute control commands here yet: %s", canUseErr.Error()))
	} else {
		sb.WriteString("\nYou can execute these commands in this channel.")
	}

	_ = respondImmediate(s, i, sb.String(), true)
}

func validateHostnameParameter(hostnameArg string) (err error) {
	isValidHostname := false
	for _, validHostname := range config.ValidHostnames {
		if hostnameArg == validHostname {
			isValidHostname = true
			break
		}
	}

	if !isValidHostname {
		err := fmt.Errorf("[-] Message contained invalid hostname argument: '%s'", hostnameArg)
		return err
	}

	if hostnameArg != hostname && hostnameArg != "all" {
		err := fmt.Errorf("[!] Hostname mismatch: '%s' does not match servers hostname '%s'. Command probably is not intended for us.", hostnameArg, hostname)
		return err
	}

	return nil
}

func isValidSource(channelID string, userRoles []string) (err error) {
	isValidChannel := false
	for _, validChannel := range config.ValidChannelIDs {
		if channelID == validChannel {
			isValidChannel = true
			break
		}
	}

	if !isValidChannel {
		err := fmt.Errorf("[-] Message from within invalid channel: '%s'", channelID)
		return err
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
		err := fmt.Errorf("[-] Message from user with no valid role membership: '%v'", userRoles)
		return err
	}

	return nil
}

func execCommand(path string, args ...string) (string, error) {
	cmd := exec.Command(path, args...)
	output, err := cmd.CombinedOutput()

	fmt.Printf("[+] Executing system command:: %s %s\n", path, strings.Join(args, " "))
	fmt.Printf("[*] Command output: %s\n", string(output))

	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

func executeSpec(spec CommandSpec) string {
	output, err := execCommand(spec.Path, spec.Args...)
	if err != nil {
		return fmt.Sprintf("Failed while running `%s %s`\n```%s```", spec.Path, strings.Join(spec.Args, " "), truncateOutput(output))
	}

	return fmt.Sprintf("Executing `%s %s` finished successfully.\n```%s```", spec.Path, strings.Join(spec.Args, " "), truncateOutput(output))
}

func truncateOutput(out string) string {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return "(no output)"
	}
	if len(trimmed) > 1700 {
		return trimmed[:1700] + "\n... (truncated)"
	}
	return trimmed
}

func registerSlashCommands(s *discordgo.Session) error {
	existing, err := s.ApplicationCommands(s.State.User.ID, "")
	if err != nil {
		return fmt.Errorf("failed to list existing commands: %w", err)
	}

	byName := make(map[string]*discordgo.ApplicationCommand, len(existing))
	for _, cmd := range existing {
		byName[cmd.Name] = cmd
	}

	for _, cmd := range slashCommandDefs {
		if old, exists := byName[cmd.Name]; exists {
			_, err = s.ApplicationCommandEdit(s.State.User.ID, "", old.ID, cmd)
		} else {
			_, err = s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		}
		if err != nil {
			return fmt.Errorf("failed to register command '%s': %w", cmd.Name, err)
		}
	}

	return nil
}

func hostnameOption() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Type:         discordgo.ApplicationCommandOptionString,
			Name:         "hostname",
			Description:  "Target hostname, or 'all'",
			Required:     true,
			Autocomplete: true,
		},
	}
}

func getStringOption(options []*discordgo.ApplicationCommandInteractionDataOption, key string) string {
	for _, option := range options {
		if option.Name == key {
			return option.StringValue()
		}
	}
	return ""
}

func getFocusedStringOption(options []*discordgo.ApplicationCommandInteractionDataOption) string {
	for _, option := range options {
		if option.Focused {
			return option.StringValue()
		}
	}
	return ""
}

func respondImmediate(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) error {
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   flags,
		},
	})
}

func sortedCommandNames() []string {
	names := make([]string, 0, len(commandSpecs))
	for k := range commandSpecs {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
