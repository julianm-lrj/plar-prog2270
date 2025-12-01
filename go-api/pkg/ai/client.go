package ai

import (
	"context"
	"log"
	"os"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

var client *openai.Client
var isInitialized bool

// InitializeAIService initializes the Azure OpenAI client with environment variables
func InitializeAIService() {
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	apiKey := os.Getenv("AZURE_OPENAI_API_KEY")

	if endpoint == "" || apiKey == "" {
		log.Println("AI service disabled - Azure OpenAI credentials not provided")
		log.Println("Required: AZURE_OPENAI_ENDPOINT and AZURE_OPENAI_API_KEY environment variables")
		isInitialized = false
		return
	}

	clientValue := openai.NewClient(
		option.WithBaseURL(endpoint),
		option.WithAPIKey(apiKey),
	)
	client = &clientValue

	isInitialized = true
	log.Println("AI service initialized with Azure OpenAI")
}

// IsEnabled returns whether the AI service is properly initialized
func IsEnabled() bool {
	return isInitialized && client != nil
}

// GetClient returns the OpenAI client instance
func GetClient() *openai.Client {
	if !IsEnabled() {
		return nil
	}
	return client
}

// generateCompletion is a helper function to generate AI completions
func generateCompletion(ctx context.Context, systemMessage, userMessage string) (string, error) {
	if !IsEnabled() {
		return "", &AIError{Message: "AI service is not enabled"}
	}

	deploymentName := os.Getenv("AZURE_OPENAI_DEPLOYMENT_NAME")
	if deploymentName == "" {
		deploymentName = "gpt-35-turbo" // Default deployment name
	}

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(deploymentName),
		Messages: []openai.ChatCompletionMessageParamUnion{
			{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(systemMessage),
					},
				},
			},
			{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(userMessage),
					},
				},
			},
		},
		MaxTokens:   openai.Int(1500),  // Limit response length
		Temperature: openai.Float(0.7), // Balanced creativity
	})

	if err != nil {
		log.Printf("AI API Error: %v", err)
		return "", &AIError{Message: "Failed to generate AI response", Cause: err}
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", &AIError{Message: "AI returned empty response"}
	}

	return resp.Choices[0].Message.Content, nil
}

// AIError represents an AI service error
type AIError struct {
	Message string
	Cause   error
}

func (e *AIError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}
