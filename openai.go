package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OpenAIClient struct {
	apiKey string
	client *http.Client
}

type Transaction struct {
	Date         string  `json:"date"`
	Description  string  `json:"description"`
	Amount       float64 `json:"amount"`
	Category     string  `json:"category"`
	Confidence   int     `json:"confidence"`
	Timestamp    string  `json:"timestamp,omitempty"`
	BillingCycle string  `json:"billingCycle,omitempty"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

const systemPrompt = `You are a financial transaction parser for UAE-based transactions. Extract transaction details from SMS messages and convert ALL amounts to AED (UAE Dirham).

Parse the following message which may contain ONE or MORE transaction SMS messages and return ONLY a valid JSON array of transaction objects.

Each transaction object must have these exact fields:
- date: transaction date in YYYY-MM-DD format (infer current year if missing, use today's date if no date mentioned)
- description: merchant or transaction description
- amount: numeric value CONVERTED TO AED as a number (positive for expenses, negative for income/deposits)
- category: exactly ONE of these categories: "Food & Dining", "Transport", "Shopping", "Bills & Utilities", "Entertainment", "Health & Fitness", "Travel", "Cash Withdrawal", "Income/Transfer", "Unknown"
- confidence: number from 0-100

Currency Conversion Rules:
- If amount is in AED: keep as-is
- If amount is in USD: multiply by 3.67
- If amount is in EUR: multiply by 4.00
- If amount is in GBP: multiply by 4.70
- If amount is in SAR: multiply by 0.98
- Other currencies: use approximate current rates to convert to AED
- ALWAYS return amount in AED only

Parsing Rules:
- Return an ARRAY of transaction objects, even if there's only one transaction
- Only use "Unknown" category if confidence < 70
- Infer current year if not specified in SMS
- Extract numeric amount only, remove currency symbols
- Be conservative with category assignment
- Return ONLY the JSON array, no other text
- Each SMS in the message should be parsed as a separate transaction

Example response for multiple transactions:
[
  {
    "date": "2026-01-25",
    "description": "Starbucks Dubai Mall",
    "amount": 25.50,
    "category": "Food & Dining",
    "confidence": 95
  },
  {
    "date": "2026-01-25",
    "description": "Careem Ride",
    "amount": 35.00,
    "category": "Transport",
    "confidence": 98
  }
]`

func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (c *OpenAIClient) ParseTransactions(text string) ([]Transaction, error) {
	reqBody := openAIRequest{
		Model: "gpt-4o-mini",
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: text,
			},
		},
		Temperature: 0.3,
		MaxTokens:   1500,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	content := openAIResp.Choices[0].Message.Content

	var transactions []Transaction
	if err := json.Unmarshal([]byte(content), &transactions); err != nil {
		return nil, fmt.Errorf("failed to parse transactions from OpenAI response: %w (content: %s)", err, content)
	}

	return transactions, nil
}
