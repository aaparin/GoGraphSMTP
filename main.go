// main.go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/emersion/go-smtp"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
	"gopkg.in/yaml.v3"
)

// Config represents the structure of the configuration file
type Config struct {
	Azure struct {
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		TenantID     string `yaml:"tenant_id"`
	} `yaml:"azure"`
	SMTP struct {
		Address string `yaml:"address"`
		Domain  string `yaml:"domain"`
	} `yaml:"smtp"`
	LogFile string `yaml:"log_file"`
}

// Backend implements the go-smtp Backend interface
type Backend struct {
	graphClient *msgraphsdk.GraphServiceClient
	config      Config
	logger      *log.Logger
}

// NewBackend creates a new backend with a configured Graph client
func NewBackend(config Config) (*Backend, error) {
	logFile, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	logger := log.New(logFile, "", 0)

	cred, err := azidentity.NewClientSecretCredential(
		config.Azure.TenantID,
		config.Azure.ClientID,
		config.Azure.ClientSecret,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %v", err)
	}

	graphClient, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, fmt.Errorf("failed to create graph client: %v", err)
	}

	return &Backend{
		graphClient: graphClient,
		config:      config,
		logger:      logger,
	}, nil
}

// NewSession creates a new SMTP session
func (bkd *Backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &Session{
		backend: bkd,
	}, nil
}

// Session represents an SMTP session
type Session struct {
	backend *Backend
	from    string
	to      []string
}

func (s *Session) AuthPlain(username, password string) error {
	s.from = username
	return nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	// Read the email data
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// Parse headers and body
	message := string(data)
	parts := strings.Split(message, "\r\n\r\n")
	headers := parseHeaders(parts[0])
	body := ""
	if len(parts) > 1 {
		body = parts[1]
	}

	// Create recipients
	var toRecipients []models.Recipientable
	for _, addr := range s.to {
		emailAddress := models.NewEmailAddress()
		emailAddress.SetAddress(&addr)

		recipient := models.NewRecipient()
		recipient.SetEmailAddress(emailAddress)
		toRecipients = append(toRecipients, recipient)
	}

	// Create the message body
	messageBody := models.NewItemBody()
	messageBody.SetContent(&body)

	contentType := models.TEXT_BODYTYPE // Default to plain text
	if strings.Contains(strings.ToLower(headers["Content-Type"]), "html") {
		contentType = models.HTML_BODYTYPE
	}
	messageBody.SetContentType(&contentType)

	// Handle attachments
	var attachments []models.Attachmentable
	if len(headers["Attachments"]) > 0 {
		attachmentPaths := strings.Split(headers["Attachments"], ",")
		for _, path := range attachmentPaths {
			data, err := os.ReadFile(strings.TrimSpace(path))
			if err != nil {
				s.backend.logger.Printf("Error reading attachment %s: %v\n", path, err)
				continue
			}

			attachment := models.NewFileAttachment()
			name := strings.TrimSpace(path)
			attachment.SetName(&name)
			attachment.SetContentBytes(data)
			attachments = append(attachments, attachment)
		}
	}

	// Create the message
	msg := models.NewMessage()
	subject := headers["Subject"]
	msg.SetSubject(&subject)
	msg.SetBody(messageBody)
	msg.SetToRecipients(toRecipients)
	msg.SetAttachments(attachments)

	// Send the email using Graph API
	requestBody := users.NewItemSendMailPostRequestBody()
	requestBody.SetMessage(msg)
	saveToSent := true
	requestBody.SetSaveToSentItems(&saveToSent)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = s.backend.graphClient.Users().
		ByUserId(s.from).
		SendMail().
		Post(ctx, requestBody, nil)

	if err != nil {
		s.backend.logger.Printf("from=<%s>, host=graph.microsoft.com, msgid=NA, errormsg=\"%v\"\n",
			s.from, err)
		return fmt.Errorf("failed to send email: %v", err)
	}

	recipients := strings.Join(s.to, ",")
	s.backend.logger.Printf("from=<%s>, host=graph.microsoft.com, msgid=NA, mailer=GoGraphSmtp, tls=on, recipients=%s\n",
		s.from, recipients)
	return nil
}

func (s *Session) Reset() {
	s.from = ""
	s.to = []string{}
}

func (s *Session) Logout() error {
	return nil
}

// Helper functions
func parseHeaders(headerData string) map[string]string {
	headers := make(map[string]string)
	lines := strings.Split(headerData, "\r\n")

	var currentHeader string
	var currentValue strings.Builder

	for _, line := range lines {
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			currentValue.WriteString(" " + strings.TrimSpace(line))
			continue
		}

		if currentHeader != "" {
			headers[currentHeader] = currentValue.String()
			currentValue.Reset()
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			currentHeader = strings.TrimSpace(parts[0])
			currentValue.WriteString(strings.TrimSpace(parts[1]))
		}
	}

	if currentHeader != "" {
		headers[currentHeader] = currentValue.String()
	}

	return headers
}

func loadConfig(filename string) (Config, error) {
	var config Config

	data, err := os.ReadFile(filename)
	if err != nil {
		return config, fmt.Errorf("error reading config file: %v", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("error parsing config file: %v", err)
	}

	return config, nil
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	backend, err := NewBackend(config)
	if err != nil {
		log.Fatalf("Failed to create backend: %v", err)
	}

	s := smtp.NewServer(backend)

	s.Addr = config.SMTP.Address
	s.Domain = config.SMTP.Domain
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	log.Printf("Starting SMTP server at %s", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
