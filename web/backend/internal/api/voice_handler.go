package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/hume-evi/web/internal/db"
)

type CreateVoiceRequest struct {
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	Prompt           string  `json:"prompt"`
	VoiceDescription string  `json:"voice_description"`
	EVIVersion       string  `json:"evi_version"`
	Temperature      float64 `json:"temperature"`
}

type UpdateVoiceRequest struct {
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	Prompt           string  `json:"prompt"`
	VoiceDescription string  `json:"voice_description"`
	EVIVersion       string  `json:"evi_version"`
	Temperature      float64 `json:"temperature"`
}

type HumeTTSRequest struct {
	Utterances []struct {
		Text        string `json:"text"`
		Description string `json:"description"`
	} `json:"utterances"`
	NumGenerations int    `json:"num_generations"`
	Version        string `json:"version"`
}

type HumeTTSResponse struct {
	Generations []struct {
		GenerationID string `json:"generation_id"`
		Audio        string `json:"audio"`
	} `json:"generations"`
}

type HumeVoiceResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type HumeConfigRequest struct {
	Name              string `json:"name"`
	EVIVersion        string `json:"eviVersion"`
	VersionDescription string `json:"versionDescription,omitempty"`
	Prompt            *HumePromptReference `json:"prompt,omitempty"`
	Voice             *HumeVoiceReference `json:"voice"`
	LanguageModel     *HumeLanguageModel  `json:"languageModel"`
	EventMessages     *HumeEventMessages  `json:"eventMessages,omitempty"`
}

type HumePromptReference struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

type HumeVoiceReference struct {
	Provider string  `json:"provider"`
	Name     *string `json:"name,omitempty"`
	ID       *string `json:"id,omitempty"`
}

type HumeLanguageModel struct {
	ModelProvider string  `json:"modelProvider"`
	ModelResource string  `json:"modelResource"`
	Temperature   float64 `json:"temperature"`
}

type HumeEventMessages struct {
	OnNewChat struct {
		Enabled bool   `json:"enabled"`
		Text    string `json:"text"`
	} `json:"onNewChat"`
	OnInactivityTimeout struct {
		Enabled bool   `json:"enabled"`
		Text    string `json:"text"`
	} `json:"onInactivityTimeout"`
	OnMaxDurationTimeout struct {
		Enabled bool   `json:"enabled"`
		Text    string `json:"text"`
	} `json:"onMaxDurationTimeout"`
}

type HumeConfigResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type HumePromptRequest struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

type HumePromptResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Text    string `json:"text"`
	Version int    `json:"version"`
}

type HumePromptListResponse struct {
	ResultsPage struct {
		Results []HumePromptResponse `json:"results"`
	} `json:"results_page"`
}

func (s *Server) listVoicesHandler(w http.ResponseWriter, r *http.Request) {
	voices, err := s.db.ListVoices(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list voices: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(voices)
}

func (s *Server) getVoiceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid voice ID", http.StatusBadRequest)
		return
	}

	voice, err := s.db.GetVoice(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Voice not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(voice)
}

func (s *Server) createVoiceHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateVoiceRequest
	bodyBytes, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v, body: %s", err, string(bodyBytes)), http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Prompt == "" {
		http.Error(w, fmt.Sprintf("Name and prompt are required. Got name: %q, prompt: %q", req.Name, req.Prompt), http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.EVIVersion == "" {
		req.EVIVersion = "3"
	}
	if req.Temperature == 0 {
		req.Temperature = 1.0
	}

	ctx := r.Context()

	// Step 1: Create voice via TTS API if voice_description is provided
	// Note: TTS voices and EVI voices are separate - we'll store the TTS voice ID
	// but use a default EVI voice name for the config
	var humeVoiceID string
	if req.VoiceDescription != "" {
		voiceID, err := s.createHumeVoice(ctx, req.VoiceDescription)
		if err != nil {
			log.Printf("Warning: Failed to create Hume TTS voice: %v, continuing with default voice", err)
			// Continue without custom voice - use default
		} else {
			humeVoiceID = voiceID
		}
	}

	// Step 2: Create prompt via Hume API
	var promptRef *HumePromptReference
	promptID, promptVersion, err := s.createHumePrompt(ctx, req.Name, req.Prompt)
	if err != nil {
		log.Printf("Error creating Hume prompt: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create Hume prompt: %v", err), http.StatusInternalServerError)
		return
	}
	promptRef = &HumePromptReference{
		ID:      promptID,
		Version: promptVersion,
	}

	// Step 3: Create EVI config
	// Note: EVI configs use voice names, not TTS voice IDs
	// We'll use a default voice name for now
	configID, err := s.createHumeConfig(ctx, req, promptRef)
	if err != nil {
		log.Printf("Error creating Hume config: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create Hume config: %v", err), http.StatusInternalServerError)
		return
	}

	// Step 4: Save to database
	voice := &db.Voice{
		Name:                  req.Name,
		Description:           req.Description,
		Prompt:                req.Prompt,
		VoiceDescription:      req.VoiceDescription,
		HumeVoiceID:           humeVoiceID,
		HumeConfigID:          configID,
		EVIVersion:            req.EVIVersion,
		LanguageModelProvider: "ANTHROPIC",
		LanguageModelResource: "claude-3-7-sonnet-latest",
		Temperature:           req.Temperature,
	}

	created, err := s.db.CreateVoice(ctx, voice)
	if err != nil {
		log.Printf("Error saving voice to database: %v", err)
		http.Error(w, fmt.Sprintf("Failed to save voice: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (s *Server) updateVoiceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid voice ID", http.StatusBadRequest)
		return
	}

	var req UpdateVoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get existing voice
	existing, err := s.db.GetVoice(r.Context(), id)
	if err != nil {
		http.Error(w, "Voice not found", http.StatusNotFound)
		return
	}

	// Update fields
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Prompt != "" {
		existing.Prompt = req.Prompt
	}
	if req.VoiceDescription != "" {
		existing.VoiceDescription = req.VoiceDescription
	}
	if req.EVIVersion != "" {
		existing.EVIVersion = req.EVIVersion
	}
	if req.Temperature != 0 {
		existing.Temperature = req.Temperature
	}

	updated, err := s.db.UpdateVoice(r.Context(), id, existing)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update voice: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (s *Server) deleteVoiceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid voice ID", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteVoice(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete voice: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// createHumeVoice creates a custom voice via Hume TTS API
func (s *Server) createHumeVoice(ctx context.Context, voiceDescription string) (string, error) {
	ttsReq := HumeTTSRequest{
		Utterances: []struct {
			Text        string `json:"text"`
			Description string `json:"description"`
		}{
			{
				Text:        "Hello, this is a test of the voice generation system.",
				Description: voiceDescription,
			},
		},
		NumGenerations: 1,
		Version:        "1",
	}

	reqBody, err := json.Marshal(ttsReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.hume.ai/v0/tts", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("X-Hume-Api-Key", s.config.HumeAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Hume API error: %s", string(body))
	}

	var ttsResp HumeTTSResponse
	if err := json.NewDecoder(resp.Body).Decode(&ttsResp); err != nil {
		return "", err
	}

	if len(ttsResp.Generations) == 0 {
		return "", fmt.Errorf("no generations returned")
	}

	generationID := ttsResp.Generations[0].GenerationID

	// Save the voice
	saveReq := map[string]string{
		"generation_id": generationID,
		"name":          fmt.Sprintf("Voice-%d", time.Now().Unix()),
	}

	saveBody, err := json.Marshal(saveReq)
	if err != nil {
		return "", err
	}

	saveHTTPReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.hume.ai/v0/tts/voices", bytes.NewBuffer(saveBody))
	if err != nil {
		return "", err
	}

	saveHTTPReq.Header.Set("X-Hume-Api-Key", s.config.HumeAPIKey)
	saveHTTPReq.Header.Set("Content-Type", "application/json")

	saveResp, err := client.Do(saveHTTPReq)
	if err != nil {
		return "", err
	}
	defer saveResp.Body.Close()

	if saveResp.StatusCode != http.StatusOK && saveResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(saveResp.Body)
		return "", fmt.Errorf("Hume API error saving voice: %s", string(body))
	}

	var voiceResp HumeVoiceResponse
	if err := json.NewDecoder(saveResp.Body).Decode(&voiceResp); err != nil {
		return "", err
	}

	return voiceResp.ID, nil
}

// createHumePromptWithName creates a prompt with a specific name (internal helper)
// If retryOnConflict is true and we get a 409, it will try with a modified name
func (s *Server) createHumePromptWithName(ctx context.Context, name, text string, retryOnConflict bool) (string, int, error) {
	promptReq := HumePromptRequest{
		Name: name,
		Text: text,
	}

	reqBody, err := json.Marshal(promptReq)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal prompt request: %w", err)
	}

	log.Printf("Creating Hume prompt with request: %s", string(reqBody))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.hume.ai/v0/evi/prompts", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", 0, err
	}

	httpReq.Header.Set("X-Hume-Api-Key", s.config.HumeAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Handle 409 Conflict - config or prompt already exists
		if resp.StatusCode == http.StatusConflict {
			// Parse JSON error response to extract message
			var errorResp map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
				if message, ok := errorResp["message"].(string); ok {
					// Error format: "Prevented duplicate key for prompt (ID: <uuid>, NAME: <name>)"
					if strings.Contains(message, "ID:") {
						// Extract UUID from error message
						startIdx := strings.Index(message, "ID: ")
						if startIdx != -1 {
							startIdx += 4 // Move past "ID: "
							endIdx := strings.Index(message[startIdx:], ",")
							if endIdx == -1 {
								endIdx = strings.Index(message[startIdx:], ")")
							}
							if endIdx != -1 {
								promptID := strings.TrimSpace(message[startIdx : startIdx+endIdx])
								log.Printf("Extracted prompt ID from error: %s", promptID)
								// Get the prompt to find its version
								promptIDParsed, promptVersion, err := s.getHumePrompt(ctx, promptID)
								if err == nil {
									log.Printf("Found existing Hume prompt: ID=%s, Version=%d", promptIDParsed, promptVersion)
									return promptIDParsed, promptVersion, nil
								}
								// If we can't retrieve it (404/unauthorized), we can't use it
								// Create a new prompt with a modified name to avoid conflict
								log.Printf("Warning: Could not retrieve prompt %s: %v, creating new prompt with modified name", promptID, err)
								newName := fmt.Sprintf("%s (%d)", name, time.Now().Unix())
								return s.createHumePromptWithName(ctx, newName, text, false)
							}
						}
					}
				}
			}
			// Fallback: list prompts and find by name
			log.Printf("Attempting to find prompt by name: %s", name)
			promptID, promptVersion, err := s.findHumePromptByName(ctx, name)
			if err == nil {
				log.Printf("Found existing Hume prompt by name: ID=%s, Version=%d", promptID, promptVersion)
				return promptID, promptVersion, nil
			}
			log.Printf("Error finding prompt by name: %v", err)
			// If retryOnConflict is true, try creating with a modified name
			if retryOnConflict {
				newName := fmt.Sprintf("%s (%d)", name, time.Now().Unix())
				log.Printf("Retrying with modified name: %s", newName)
				return s.createHumePromptWithName(ctx, newName, text, false)
			}
			return "", 0, fmt.Errorf("prompt already exists but could not retrieve it: %s", string(bodyBytes))
		}
		return "", 0, fmt.Errorf("Hume API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var promptResp HumePromptResponse
	if err := json.Unmarshal(bodyBytes, &promptResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode prompt response: %v, body: %s", err, string(bodyBytes))
	}

	if promptResp.ID == "" {
		return "", 0, fmt.Errorf("prompt ID not found in response: %s", string(bodyBytes))
	}

	log.Printf("Created Hume prompt: ID=%s, Version=%d", promptResp.ID, promptResp.Version)
	return promptResp.ID, promptResp.Version, nil
}

// createHumePrompt creates a prompt via Hume API, handling conflicts by creating with modified name
func (s *Server) createHumePrompt(ctx context.Context, name, text string) (string, int, error) {
	return s.createHumePromptWithName(ctx, name, text, true)
}

// getHumePrompt retrieves a prompt by ID and returns its latest version
func (s *Server) getHumePrompt(ctx context.Context, promptID string) (string, int, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.hume.ai/v0/evi/prompts/%s", promptID), nil)
	if err != nil {
		return "", 0, err
	}

	httpReq.Header.Set("X-Hume-Api-Key", s.config.HumeAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("Hume API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var promptResp HumePromptResponse
	if err := json.Unmarshal(bodyBytes, &promptResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode prompt response: %v", err)
	}

	return promptResp.ID, promptResp.Version, nil
}

// findHumePromptByName lists prompts and finds one by name
func (s *Server) findHumePromptByName(ctx context.Context, name string) (string, int, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.hume.ai/v0/evi/prompts?page_size=100", nil)
	if err != nil {
		return "", 0, err
	}

	httpReq.Header.Set("X-Hume-Api-Key", s.config.HumeAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("Hume API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Try different response structures
	var listResp HumePromptListResponse
	if err := json.Unmarshal(bodyBytes, &listResp); err != nil {
		// Try alternative structure - maybe it's just an array
		var prompts []HumePromptResponse
		if err2 := json.Unmarshal(bodyBytes, &prompts); err2 == nil {
			log.Printf("Found %d prompts in array format", len(prompts))
			for _, prompt := range prompts {
				if prompt.Name == name {
					return prompt.ID, prompt.Version, nil
				}
			}
		} else {
			// Try direct results structure
			var directResp struct {
				Results []HumePromptResponse `json:"results"`
			}
			if err3 := json.Unmarshal(bodyBytes, &directResp); err3 == nil {
				log.Printf("Found %d prompts in direct results format", len(directResp.Results))
				for _, prompt := range directResp.Results {
					if prompt.Name == name {
						return prompt.ID, prompt.Version, nil
					}
				}
			} else {
				log.Printf("Failed to parse prompt list response. Body: %s", string(bodyBytes))
				return "", 0, fmt.Errorf("failed to decode prompt list response: %v, body: %s", err, string(bodyBytes))
			}
		}
	} else {
		log.Printf("Found %d prompts in ResultsPage format", len(listResp.ResultsPage.Results))
		// Find prompt by name
		for _, prompt := range listResp.ResultsPage.Results {
			if prompt.Name == name {
				return prompt.ID, prompt.Version, nil
			}
		}
	}

	return "", 0, fmt.Errorf("prompt with name %q not found in list", name)
}

// createHumeConfig creates an EVI configuration via Hume API
func (s *Server) createHumeConfig(ctx context.Context, req CreateVoiceRequest, promptRef *HumePromptReference) (string, error) {
	configReq := HumeConfigRequest{
		Name:       req.Name,
		EVIVersion: req.EVIVersion,
		Voice: &HumeVoiceReference{
			Provider: "HUME_AI",
		},
		LanguageModel: &HumeLanguageModel{
			ModelProvider: "ANTHROPIC",
			ModelResource: "claude-3-7-sonnet-latest",
			Temperature:  req.Temperature,
		},
		EventMessages: &HumeEventMessages{},
	}

	// Include prompt reference if provided
	if promptRef != nil {
		configReq.Prompt = promptRef
	}

	// EVI configs can use voice IDs or voice names from the Hume voice library
	// Using a specific voice ID
	defaultVoiceID := "5add9038-28df-40a6-900c-2f736d008ab3"
	configReq.Voice.ID = &defaultVoiceID
	configReq.Voice.Name = nil

	reqBody, err := json.Marshal(configReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config request: %w", err)
	}
	
	log.Printf("Creating Hume config with request: %s", string(reqBody))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.hume.ai/v0/evi/configs", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("X-Hume-Api-Key", s.config.HumeAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Handle 409 Conflict - config name already exists
		if resp.StatusCode == http.StatusConflict {
			// Try creating with a modified name
			log.Printf("Config name conflict, retrying with modified name")
			req.Name = fmt.Sprintf("%s (%d)", req.Name, time.Now().Unix())
			return s.createHumeConfig(ctx, req, promptRef)
		}
		return "", fmt.Errorf("Hume API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var configResp HumeConfigResponse
	if err := json.Unmarshal(bodyBytes, &configResp); err != nil {
		return "", fmt.Errorf("failed to decode config response: %v, body: %s", err, string(bodyBytes))
	}

	if configResp.ID == "" {
		return "", fmt.Errorf("config ID is empty in response: %s", string(bodyBytes))
	}

	return configResp.ID, nil
}

// updateHumeConfig creates a new version of an existing EVI config with updated prompt
func (s *Server) updateHumeConfig(ctx context.Context, configID string, req CreateVoiceRequest, promptRef *HumePromptReference) error {
	configReq := HumeConfigRequest{
		Name:              req.Name,
		EVIVersion:        req.EVIVersion,
		VersionDescription: fmt.Sprintf("Updated with prompt: %s", req.Name),
		Voice: &HumeVoiceReference{
			Provider: "HUME_AI",
		},
		LanguageModel: &HumeLanguageModel{
			ModelProvider: "ANTHROPIC",
			ModelResource: "claude-3-7-sonnet-latest",
			Temperature:  req.Temperature,
		},
		EventMessages: &HumeEventMessages{},
	}

	// Include prompt reference if provided
	if promptRef != nil {
		configReq.Prompt = promptRef
	}

	// EVI configs can use voice IDs or voice names from the Hume voice library
	// Using a specific voice ID
	defaultVoiceID := "5add9038-28df-40a6-900c-2f736d008ab3"
	configReq.Voice.ID = &defaultVoiceID
	configReq.Voice.Name = nil

	reqBody, err := json.Marshal(configReq)
	if err != nil {
		return fmt.Errorf("failed to marshal config request: %w", err)
	}

	log.Printf("Updating Hume config %s with request: %s", configID, string(reqBody))

	// Create a new version by POSTing to /v0/evi/configs/{id}/versions
	// Note: This creates a new version of the existing config
	httpReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://api.hume.ai/v0/evi/configs/%s/versions", configID), bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	httpReq.Header.Set("X-Hume-Api-Key", s.config.HumeAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Hume API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var configResp HumeConfigResponse
	if err := json.Unmarshal(bodyBytes, &configResp); err != nil {
		return fmt.Errorf("failed to decode config response: %v, body: %s", err, string(bodyBytes))
	}

	log.Printf("Updated Hume config: ID=%s, Version=%d", configResp.ID, configResp.Version)
	return nil
}

// Note: updateHumeConfig is kept for potential future use but currently not working
// because the /versions endpoint returns 404 for configs created via UI

// syncVoiceToHume creates a prompt and creates a new config for an existing voice
// Note: We create a new config instead of updating because the /versions endpoint
// doesn't seem to be available for configs created via the UI
func (s *Server) syncVoiceToHume(ctx context.Context, voice *db.Voice) error {
	if voice.Prompt == "" {
		return fmt.Errorf("voice has no prompt text")
	}

	// Create prompt in Hume
	promptID, promptVersion, err := s.createHumePrompt(ctx, voice.Name, voice.Prompt)
	if err != nil {
		return fmt.Errorf("failed to create prompt: %w", err)
	}

	promptRef := &HumePromptReference{
		ID:      promptID,
		Version: promptVersion,
	}

	// Create new config with prompt (instead of updating existing)
	// Use a modified name to avoid conflicts with existing configs
	configName := fmt.Sprintf("%s (%d)", voice.Name, time.Now().Unix())
	req := CreateVoiceRequest{
		Name:        configName,
		Description: voice.Description,
		Prompt:      voice.Prompt,
		EVIVersion:  voice.EVIVersion,
		Temperature: voice.Temperature,
	}

	newConfigID, err := s.createHumeConfig(ctx, req, promptRef)
	if err != nil {
		return fmt.Errorf("failed to create new config: %w", err)
	}

	// Store old config ID for logging
	oldConfigID := voice.HumeConfigID

	// Update the voice record with the new config ID
	voice.HumeConfigID = newConfigID
	_, err = s.db.UpdateVoice(ctx, voice.ID, voice)
	if err != nil {
		log.Printf("Warning: Created new config %s but failed to update database: %v", newConfigID, err)
		// Don't fail the sync if DB update fails - config was created successfully
	}

	log.Printf("Created new config %s for voice %s (replacing old config %s)", newConfigID, voice.Name, oldConfigID)
	return nil
}

func (s *Server) syncVoiceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid voice ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get voice from database
	voice, err := s.db.GetVoice(ctx, id)
	if err != nil {
		http.Error(w, "Voice not found", http.StatusNotFound)
		return
	}

	// Sync to Hume
	if err := s.syncVoiceToHume(ctx, voice); err != nil {
		log.Printf("Error syncing voice to Hume: %v", err)
		http.Error(w, fmt.Sprintf("Failed to sync voice to Hume: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "synced", "voice_id": id.String()})
}

func (s *Server) syncAllVoicesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all voices
	voices, err := s.db.ListVoices(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list voices: %v", err), http.StatusInternalServerError)
		return
	}

	results := make([]map[string]interface{}, 0)
	for _, voice := range voices {
		if voice.Prompt == "" || voice.HumeConfigID == "" {
			results = append(results, map[string]interface{}{
				"voice_id": voice.ID.String(),
				"name":     voice.Name,
				"status":    "skipped",
				"reason":   "missing prompt or config ID",
			})
			continue
		}

		if err := s.syncVoiceToHume(ctx, &voice); err != nil {
			results = append(results, map[string]interface{}{
				"voice_id": voice.ID.String(),
				"name":     voice.Name,
				"status":    "error",
				"error":    err.Error(),
			})
		} else {
			results = append(results, map[string]interface{}{
				"voice_id": voice.ID.String(),
				"name":     voice.Name,
				"status":    "synced",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "completed",
		"results": results,
	})
}

