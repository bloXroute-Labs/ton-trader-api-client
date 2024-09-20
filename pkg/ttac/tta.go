package ttac

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// TTASubmitRequest represents the request body for the /api/v2/submit endpoint.
type TTASubmitRequest struct {
	Transaction struct {
		Content string `json:"content"`
	} `json:"transaction"`
	Wallet string `json:"wallet"`
}

// TTASubmitResponse represents the response from the /api/v2/submit endpoint.
type TTASubmitResponse struct {
	MsgBodyHash string `json:"msg_body_hash"`
}

// ErrorResponse represents the error response from the API.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// submitTransaction sends a POST request to submit a signed TON transaction with a context.
func submitTransaction(ctx context.Context, baseURL, authHeader string, submitReq *TTASubmitRequest, timeout time.Duration) (*TTASubmitResponse, error) {
	// Marshal the request body into JSON
	reqBody, err := json.Marshal(submitReq)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal request")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request with the provided context
	url := fmt.Sprintf("%s/api/v2/submit", baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create HTTP request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	// Perform the HTTP request
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		// Check if the error is due to context cancellation or deadline exceeded
		if ctx.Err() == context.Canceled {
			log.Error().Err(ctx.Err()).Msg("Request was canceled")
			return nil, fmt.Errorf("request was canceled: %w", ctx.Err())
		} else if ctx.Err() == context.DeadlineExceeded {
			log.Error().Err(ctx.Err()).Msg("Request deadline exceeded")
			return nil, fmt.Errorf("request deadline exceeded: %w", ctx.Err())
		}

		log.Error().Err(err).Msg("Failed to send HTTP request")
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read response body")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the status code
	log.Info().Int("status_code", resp.StatusCode).Msg("API response received")

	// Check the status code and handle errors
	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			log.Error().Err(err).Msg("Failed to parse error response")
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
		log.Error().Int("code", errorResp.Code).Str("message", errorResp.Message).Msg("TTA API error")
		return nil, fmt.Errorf("API error: %s (code: %d)", errorResp.Message, errorResp.Code)
	}

	// Parse the successful response
	var submitResp TTASubmitResponse
	if err := json.Unmarshal(body, &submitResp); err != nil {
		log.Error().Err(err).Msg("Failed to parse successful response")
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &submitResp, nil
}
