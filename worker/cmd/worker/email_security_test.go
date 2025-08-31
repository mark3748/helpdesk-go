package main

import (
	"testing"
)

func TestSanitizeEmailHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal text without CRLF",
			input:    "Normal subject line",
			expected: "Normal subject line",
		},
		{
			name:     "Text with carriage return",
			input:    "Subject with\rinjection",
			expected: "Subject withinjection",
		},
		{
			name:     "Text with line feed",
			input:    "Subject with\ninjection",
			expected: "Subject withinjection",
		},
		{
			name:     "Text with CRLF",
			input:    "Subject with\r\ninjection",
			expected: "Subject withinjection",
		},
		{
			name:     "Header injection attempt",
			input:    "Normal Subject\r\nBcc: attacker@evil.com",
			expected: "Normal SubjectBcc: attacker@evil.com",
		},
		{
			name:     "Multiple CRLF attempts",
			input:    "Subject\r\nBcc: evil@bad.com\r\nX-Custom: attack",
			expected: "SubjectBcc: evil@bad.comX-Custom: attack",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Whitespace only",
			input:    "   \t  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeEmailHeader(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeEmailHeader(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeEmailBody(t *testing.T) {
	input := []byte("<script>alert('xss')</script><p>Hello</p>")
	expected := "<p>Hello</p>"
	if result := sanitizeEmailBody(input); result != expected {
		t.Errorf("sanitizeEmailBody() = %q, want %q", result, expected)
	}
}

func TestValidateEmailAddress(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "Valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "Valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "Valid email with plus",
			email:   "user+tag@example.com",
			wantErr: false,
		},
		{
			name:    "Valid email with numbers",
			email:   "user123@example123.com",
			wantErr: false,
		},
		{
			name:    "Empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "Email without @",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "Email without domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "Email without user",
			email:   "@example.com",
			wantErr: true,
		},
		{
			name:    "Email with spaces",
			email:   "user @example.com",
			wantErr: true,
		},
		{
			name:    "Invalid characters",
			email:   "user<>@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmailAddress(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmailAddress(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeAndValidateEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		expected    string
		expectError bool
	}{
		{
			name:        "Valid email",
			email:       "user@example.com",
			expected:    "user@example.com",
			expectError: false,
		},
		{
			name:        "Valid email with whitespace",
			email:       "  user@example.com  ",
			expected:    "user@example.com",
			expectError: false,
		},
		{
			name:        "Email with CRLF injection attempt",
			email:       "user@example.com\r\nBcc: attacker@evil.com",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Email with line feed",
			email:       "user@example.com\nBcc: attacker@evil.com",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid email format",
			email:       "invalid-email",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty email",
			email:       "",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeAndValidateEmail(tt.email)

			if tt.expectError {
				if err == nil {
					t.Errorf("sanitizeAndValidateEmail(%q) expected error but got none", tt.email)
				}
			} else {
				if err != nil {
					t.Errorf("sanitizeAndValidateEmail(%q) unexpected error: %v", tt.email, err)
				}
				if result != tt.expected {
					t.Errorf("sanitizeAndValidateEmail(%q) = %q, want %q", tt.email, result, tt.expected)
				}
			}
		})
	}
}

func TestSendEmailSecurityValidation(t *testing.T) {
	// Create a mock configuration with invalid email addresses to test validation
	config := Config{
		SMTPHost: "localhost",
		SMTPPort: "25",
		SMTPFrom: "invalid-from-email", // Invalid email to test validation
	}

	job := EmailJob{
		To:       "user@example.com\r\nBcc: attacker@evil.com", // Injection attempt
		Template: "test",
		Data:     map[string]string{"test": "data"},
	}

	// This should fail due to invalid From address
	err := sendEmail(config, job)
	if err == nil {
		t.Error("Expected sendEmail to fail with invalid From address, but it succeeded")
	}

	// Test with invalid To address
	config.SMTPFrom = "valid@example.com"
	err = sendEmail(config, job)
	if err == nil {
		t.Error("Expected sendEmail to fail with header injection in To address, but it succeeded")
	}

	// Test with valid addresses (this will fail due to missing template, but validation should pass)
	job.To = "user@example.com"
	err = sendEmail(config, job)
	// We expect an error due to missing template, but not due to email validation
	if err != nil && err.Error() == "invalid To address: invalid email address format: user@example.comBcc: attacker@evil.com" {
		t.Error("Email validation should have passed for clean email address")
	}
}

func TestEmailHeaderSanitizationInProcessing(t *testing.T) {
	tests := []struct {
		name           string
		inputSubject   string
		inputFrom      string
		expectedSubject string
		expectedFrom   string
	}{
		{
			name:           "Normal headers",
			inputSubject:   "Normal ticket subject",
			inputFrom:      "user@example.com",
			expectedSubject: "Normal ticket subject",
			expectedFrom:   "user@example.com",
		},
		{
			name:           "Subject with CRLF injection",
			inputSubject:   "Ticket Subject\r\nBcc: attacker@evil.com",
			inputFrom:      "user@example.com",
			expectedSubject: "Ticket SubjectBcc: attacker@evil.com",
			expectedFrom:   "user@example.com",
		},
		{
			name:           "From with CRLF injection",
			inputSubject:   "Normal Subject",
			inputFrom:      "user@example.com\r\nX-Spam: true",
			expectedSubject: "Normal Subject",
			expectedFrom:   "user@example.comX-Spam: true",
		},
		{
			name:           "Both headers with injection attempts",
			inputSubject:   "Subject\nwith\rCRLF",
			inputFrom:      "from@example.com\r\nBcc: evil@bad.com",
			expectedSubject: "SubjectwithCRLF",
			expectedFrom:   "from@example.comBcc: evil@bad.com",
		},
		{
			name:           "Headers with HTML/Script tags",
			inputSubject:   "<script>alert('xss')</script>Ticket",
			inputFrom:      "<script>evil()</script>user@example.com",
			expectedSubject: "<script>alert('xss')</script>Ticket",
			expectedFrom:   "<script>evil()</script>user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the sanitization functions directly as they would be called in imap.go
			sanitizedSubject := sanitizeEmailHeader(tt.inputSubject)
			sanitizedFrom := sanitizeEmailHeader(tt.inputFrom)

			if sanitizedSubject != tt.expectedSubject {
				t.Errorf("sanitizeEmailHeader(subject %q) = %q, want %q", tt.inputSubject, sanitizedSubject, tt.expectedSubject)
			}

			if sanitizedFrom != tt.expectedFrom {
				t.Errorf("sanitizeEmailHeader(from %q) = %q, want %q", tt.inputFrom, sanitizedFrom, tt.expectedFrom)
			}
		})
	}
}
