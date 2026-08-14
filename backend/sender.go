package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
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
		if strings.HasPrefix(apiKey, "t1.") || strings.HasPrefix(apiKey, "AQVN") || !strings.Contains(apiKey, " ") {
			headers["X-YaCloud-SubjectToken"] = apiKey
			headers["Authorization"] = "Bearer " + apiKey
		} else {
			headers["Authorization"] = apiKey
		}
	}

	_, err = doFetch(fetchHostRequest{
		Method:  "POST",
		URL:     endpoint,
		Headers: headers,
		Body:    string(bodyBytes),
	})
	if err != nil {
		return fmt.Errorf("yandex postbox api: %w", err)
	}
	return nil
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

	_, err = doFetch(fetchHostRequest{
		Method:  "POST",
		URL:     endpoint,
		Headers: headers,
		Body:    string(bodyBytes),
	})
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

	_, err = doFetch(fetchHostRequest{
		Method:  "POST",
		URL:     endpoint,
		Headers: headers,
		Body:    string(bodyBytes),
	})
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

	_, err = doFetch(fetchHostRequest{
		Method:  "POST",
		URL:     endpoint,
		Headers: headers,
		Body:    string(bodyBytes),
	})
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

	_, err = doFetch(fetchHostRequest{
		Method:  "POST",
		URL:     endpoint,
		Headers: headers,
		Body:    string(bodyBytes),
	})
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

	_, err = doFetch(fetchHostRequest{
		Method:  "POST",
		URL:     endpoint,
		Headers: headers,
		Body:    string(bodyBytes),
	})
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

	_, err = doFetch(fetchHostRequest{
		Method:  "POST",
		URL:     endpoint,
		Headers: headers,
		Body:    string(bodyBytes),
	})
	if err != nil {
		return fmt.Errorf("webhook delivery: %w", err)
	}
	return nil
}
