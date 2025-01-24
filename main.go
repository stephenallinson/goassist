package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"
)

const (
	openAIEndpoint        = "https://api.openai.com/v1/chat/completions"
	importantInfoFilePath = "/home/stephen/important_information.txt"
)

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message ChatMessage `json:"message"`
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set the OPENAI_API_Key in your Environment Variables")
	}

	messages := loadImportantInformation()

	// Simple Front-End
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		scanner.Scan()
		input := scanner.Text()

		if input == "exit" {
			summary, err := summarizeConversation(apiKey, messages)
			if err != nil {
				fmt.Println("Error summarizing conversation:", err)
			} else {
				fmt.Println("Conversation Summary:\n", summary)
			}
			writeImportantInformation(summary)
			break
		}

		messages = append(messages, ChatMessage{Role: "user", Content: input})

		response, err := getChatGPTResponse(apiKey, messages)
		if err != nil {
			fmt.Println("Error getting response:", err)
			continue
		}
		fmt.Println("Bot:", response)
		messages = append(messages, ChatMessage{Role: "assistant", Content: response})

		logConversation(input, response)

	}
}

func getChatGPTResponse(apiKey string, messages []ChatMessage) (string, error) {
	message := ChatRequest{
		Model:    "gpt-3.5-turbo", // Specify model type here
		Messages: messages,
	}

	requestBody, err := json.Marshal(message)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", openAIEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var chatResponse ChatResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return "", err
	}

	return chatResponse.Choices[0].Message.Content, nil
}

func loadImportantInformation() []ChatMessage {
	var messages []ChatMessage
	file, err := os.Open(importantInfoFilePath)
	if err != nil {
		fmt.Println("Error opening important information file:", err)
		return messages // Return empty slice if the file can't be opened
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" { // Check for non-empty lines
			messages = append(
				messages,
				ChatMessage{Role: "system", Content: line},
			) // Add as a system message
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading important information file:", err)
	}

	return messages
}

func summarizeConversation(apiKey string, messages []ChatMessage) (string, error) {
	// Prepare to generate a summary
	var summaryMessages []ChatMessage
	for _, msg := range messages {
		summaryMessages = append(summaryMessages, msg)
	}

	// Add a prompt to the summary request
	summaryMessages = append(summaryMessages, ChatMessage{
		Role:    "user",
		Content: "This is a conversation between an AI chat bot and a human, please extract the important information within the conversation in a format best suited for a chatbot, remove any duplicated information",
	})

	message := ChatRequest{
		Model:    "gpt-3.5-turbo",
		Messages: summaryMessages,
	}

	requestBody, err := json.Marshal(message)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", openAIEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var chatResponse ChatResponse
	if err := json.Unmarshal(body, &chatResponse); err != nil {
		return "", err
	}

	if len(chatResponse.Choices) > 0 {
		return chatResponse.Choices[0].Message.Content, nil
	}

	return "No summary returned", nil
}

func writeImportantInformation(summary string) {
	// Write important information to a file
	usr, _ := user.Current()
	filePath := usr.HomeDir + "/important_information.txt"

	// Open or create the log file
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer file.Close()

	historyEntry := fmt.Sprintf("%s\n", summary)

	if _, err := file.WriteString(historyEntry); err != nil {
		fmt.Println("Error writing to log file:", err)
	}
}

func logConversation(userMessage, botResponse string) {
	// Get current user's home directory
	usr, _ := user.Current()
	filePath := usr.HomeDir + "/conversations.log"

	// Open or create the log file
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer file.Close()

	// Log the conversation with timestamps
	logEntry := fmt.Sprintf(
		"%s\nYou: %s\nBot: %s\n\n",
		time.Now().Format(time.RFC822),
		userMessage,
		botResponse,
	)
	if _, err := file.WriteString(logEntry); err != nil {
		fmt.Println("Error writing to log file:", err)
	}
}
