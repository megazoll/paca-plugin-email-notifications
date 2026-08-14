package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	plugin "github.com/Paca-AI/plugin-sdk-go"
)

// OutboundEmail encapsulates content to be delivered.
type OutboundEmail struct {
	FromEmail string
	FromName  string
	ToEmail   string
	Subject   string
	BodyHTML  string
	BodyText  string
}

// httpRequest executes an outbound HTTP request via plugin.Fetch.
func httpRequest(method, endpoint string, headers map[string]string, body string) error {
	resp, err := plugin.Fetch(method, endpoint, headers, body)
	if err != nil {
		return err
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.Status, resp.Body)
	}
	return nil
}

// SendEmail dispatches the email using the configured HTTP provider.
func SendEmail(settings *SMTPSettings, email OutboundEmail) error {
	from := strings.TrimSpace(settings.FromEmail)
	if from == "" {
		from = strings.TrimSpace(email.FromEmail)
	}
	if from == "" {
		from = "no-reply@paca.local"
	}

	fromName := strings.TrimSpace(settings.FromName)
	if fromName == "" {
		fromName = strings.TrimSpace(email.FromName)
	}
	if fromName == "" {
		fromName = "PACA Notifications"
	}

	provider := settings.Provider
	if provider == "" {
		provider = ProviderYandexPostbox
	}

	switch provider {
	case ProviderYandexPostbox:
		return sendYandexPostbox(settings, email, from, fromName)
	case ProviderResend:
		return sendResend(settings, email, from, fromName)
	case ProviderSendGrid:
		return sendSendGrid(settings, email, from, fromName)
	case ProviderMailgun:
		return sendMailgun(settings, email, from, fromName)
	case ProviderPostmark:
		return sendPostmark(settings, email, from, fromName)
	case ProviderBrevo:
		return sendBrevo(settings, email, from, fromName)
	case ProviderWebhook:
		return sendWebhook(settings, email, from, fromName)
	default:
		return sendYandexPostbox(settings, email, from, fromName)
	}
}

func sendYandexPostbox(settings *SMTPSettings, email OutboundEmail, from, fromName string) error {
	endpoint := strings.TrimSpace(settings.Endpoint)
	if endpoint == "" {
		endpoint = "https://postbox.cloud.yandex.net/v2/email/outbound-emails"
	}

	fromHeader := from
	if fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", fromName, from)
	}

	payload := map[string]any{
		"FromEmailAddress": fromHeader,
		"Destination": map[string]any{
			"ToAddresses": []string{email.ToEmail},
		},
		"Content": map[string]any{
			"Simple": map[string]any{
				"Subject": map[string]string{
					"Data":    email.Subject,
					"Charset": "UTF-8",
				},
				"Body": map[string]any{
					"Html": map[string]string{
						"Data":    email.BodyHTML,
						"Charset": "UTF-8",
					},
					"Text": map[string]string{
						"Data":    email.BodyText,
						"Charset": "UTF-8",
					},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal yandex postbox request: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	apiKey := strings.TrimSpace(settings.APIKey)
	if apiKey != "" {
		var accessKey, secretKey string
		if strings.Contains(apiKey, ":") {
			parts := strings.SplitN(apiKey, ":", 2)
			accessKey = strings.TrimSpace(parts[0])
			secretKey = strings.TrimSpace(parts[1])
		} else if strings.Contains(apiKey, "\n") {
			parts := strings.SplitN(apiKey, "\n", 2)
			accessKey = strings.TrimSpace(parts[0])
			secretKey = strings.TrimSpace(parts[1])
		} else if strings.Contains(apiKey, " ") && !strings.HasPrefix(apiKey, "Bearer ") && !strings.HasPrefix(apiKey, "AWS4-") {
			parts := strings.SplitN(apiKey, " ", 2)
			accessKey = strings.TrimSpace(parts[0])
			secretKey = strings.TrimSpace(parts[1])
		}

		if accessKey != "" && secretKey != "" {
			authHeader, amzDate, err := signAWSSigV4(endpoint, bodyBytes, accessKey, secretKey, "ru-central1", "ses", time.Now().UTC())
			if err == nil {
				headers["X-Amz-Date"] = amzDate
				headers["Authorization"] = authHeader
			}
		} else if strings.HasPrefix(apiKey, "t1.") || strings.HasPrefix(apiKey, "AQVN") {
			headers["X-YaCloud-SubjectToken"] = apiKey
			headers["Authorization"] = "Bearer " + apiKey
		} else if strings.HasPrefix(apiKey, "Bearer ") || strings.HasPrefix(apiKey, "AWS4-") {
			headers["Authorization"] = apiKey
		} else {
			headers["Authorization"] = "Bearer " + apiKey
		}
	}

	err = httpRequest("POST", endpoint, headers, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("yandex postbox api: %w", err)
	}
	return nil
}

func signAWSSigV4(reqURL string, bodyBytes []byte, accessKey, secretKey, region, service string, t time.Time) (string, string, error) {
	u, err := url.Parse(reqURL)
	if err != nil {
		return "", "", err
	}

	host := u.Host
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}

	amzDate := t.UTC().Format("20060102T150405Z")
	dateStamp := t.UTC().Format("20060102")

	payloadSum := sha256.Sum256(bodyBytes)
	payloadHash := hex.EncodeToString(payloadSum[:])

	canonicalHeaders := fmt.Sprintf("content-type:application/json\nhost:%s\nx-amz-date:%s\n", host, amzDate)
	signedHeaders := "content-type;host;x-amz-date"

	canonicalRequest := fmt.Sprintf("POST\n%s\n\n%s\n%s\n%s", path, canonicalHeaders, signedHeaders, payloadHash)
	reqSum := sha256.Sum256([]byte(canonicalRequest))
	reqHash := hex.EncodeToString(reqSum[:])

	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, credentialScope, reqHash)

	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))

	signature := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", accessKey, credentialScope, signedHeaders, signature)

	return authHeader, amzDate, nil
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sendResend(settings *SMTPSettings, email OutboundEmail, from, fromName string) error {
	endpoint := strings.TrimSpace(settings.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.resend.com/emails"
	}

	fromHeader := from
	if fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", fromName, from)
	}

	payload := map[string]any{
		"from":    fromHeader,
		"to":      []string{email.ToEmail},
		"subject": email.Subject,
		"html":    email.BodyHTML,
		"text":    email.BodyText,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend request: %w", err)
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + strings.TrimSpace(settings.APIKey),
	}

	err = httpRequest("POST", endpoint, headers, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("resend api: %w", err)
	}
	return nil
}

func sendSendGrid(settings *SMTPSettings, email OutboundEmail, from, fromName string) error {
	endpoint := strings.TrimSpace(settings.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.sendgrid.com/v3/mail/send"
	}

	payload := map[string]any{
		"personalizations": []map[string]any{
			{
				"to": []map[string]string{
					{"email": email.ToEmail},
				},
			},
		},
		"from": map[string]string{
			"email": from,
			"name":  fromName,
		},
		"subject": email.Subject,
		"content": []map[string]string{
			{"type": "text/plain", "value": email.BodyText},
			{"type": "text/html", "value": email.BodyHTML},
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sendgrid request: %w", err)
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + strings.TrimSpace(settings.APIKey),
	}

	err = httpRequest("POST", endpoint, headers, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("sendgrid api: %w", err)
	}
	return nil
}

func sendMailgun(settings *SMTPSettings, email OutboundEmail, from, fromName string) error {
	endpoint := strings.TrimSpace(settings.Endpoint)
	if endpoint == "" {
		return fmt.Errorf("mailgun requires endpoint URL (e.g. https://api.mailgun.net/v3/<domain>/messages)")
	}

	fromHeader := from
	if fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", fromName, from)
	}

	payload := map[string]any{
		"from":    fromHeader,
		"to":      email.ToEmail,
		"subject": email.Subject,
		"html":    email.BodyHTML,
		"text":    email.BodyText,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal mailgun request: %w", err)
	}

	apiKey := strings.TrimSpace(settings.APIKey)
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("api:"+apiKey))
	if strings.HasPrefix(apiKey, "Basic ") || strings.HasPrefix(apiKey, "Bearer ") {
		authHeader = apiKey
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": authHeader,
	}

	err = httpRequest("POST", endpoint, headers, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("mailgun api: %w", err)
	}
	return nil
}

func sendPostmark(settings *SMTPSettings, email OutboundEmail, from, fromName string) error {
	endpoint := strings.TrimSpace(settings.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.postmarkapp.com/email"
	}

	fromHeader := from
	if fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", fromName, from)
	}

	payload := map[string]any{
		"From":     fromHeader,
		"To":       email.ToEmail,
		"Subject":  email.Subject,
		"HtmlBody": email.BodyHTML,
		"TextBody": email.BodyText,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal postmark request: %w", err)
	}

	headers := map[string]string{
		"Content-Type":            "application/json",
		"X-Postmark-Server-Token": strings.TrimSpace(settings.APIKey),
	}

	err = httpRequest("POST", endpoint, headers, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("postmark api: %w", err)
	}
	return nil
}

func sendBrevo(settings *SMTPSettings, email OutboundEmail, from, fromName string) error {
	endpoint := strings.TrimSpace(settings.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.brevo.com/v3/smtp/email"
	}

	payload := map[string]any{
		"sender": map[string]string{
			"name":  fromName,
			"email": from,
		},
		"to": []map[string]string{
			{"email": email.ToEmail},
		},
		"subject":     email.Subject,
		"htmlContent": email.BodyHTML,
		"textContent": email.BodyText,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal brevo request: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"api-key":      strings.TrimSpace(settings.APIKey),
	}

	err = httpRequest("POST", endpoint, headers, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("brevo api: %w", err)
	}
	return nil
}

func sendWebhook(settings *SMTPSettings, email OutboundEmail, from, fromName string) error {
	endpoint := strings.TrimSpace(settings.Endpoint)
	if endpoint == "" {
		return fmt.Errorf("webhook provider requires an endpoint URL")
	}

	payload := map[string]any{
		"from":      from,
		"from_name": fromName,
		"to":        email.ToEmail,
		"subject":   email.Subject,
		"html":      email.BodyHTML,
		"text":      email.BodyText,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook request: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	apiKey := strings.TrimSpace(settings.APIKey)
	if apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}

	err = httpRequest("POST", endpoint, headers, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("webhook delivery: %w", err)
	}
	return nil
}
