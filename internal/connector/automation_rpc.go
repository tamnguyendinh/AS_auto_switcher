package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ModelOrAlias represents a model identifier or its alias.
type ModelOrAlias struct {
	Model string `json:"model"`
	Alias string `json:"alias,omitempty"`
}

// ClientModelConfig represents one model entry in clientModelConfigs from LanguageServer.
type ClientModelConfig struct {
	ModelOrAlias ModelOrAlias `json:"modelOrAlias"`
	Label        string       `json:"label,omitempty"`
	Description  string       `json:"description,omitempty"`
	Disabled     bool         `json:"disabled,omitempty"`
	IsSelected   bool         `json:"isSelected,omitempty"`
	Selected     bool         `json:"selected,omitempty"`
}

// CascadeModelConfigData is the response payload from GetCascadeModelConfigData.
type CascadeModelConfigData struct {
	DefaultOverrideModelConfig *ClientModelConfig  `json:"defaultOverrideModelConfig,omitempty"`
	ClientModelConfigs         []ClientModelConfig `json:"clientModelConfigs,omitempty"`
}

// GetCascadeModelConfig queries GetCascadeModelConfigData Connect-RPC endpoint.
func (c *Client) GetCascadeModelConfig(ctx context.Context) (*CascadeModelConfigData, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	resp, err := c.postJSON(ctx, EndpointCascadeModelConfigData, []byte("{}"))
	if err != nil {
		return nil, fmt.Errorf("failed calling GetCascadeModelConfigData: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetCascadeModelConfigData returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading GetCascadeModelConfigData response body: %w", err)
	}

	var data CascadeModelConfigData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed parsing GetCascadeModelConfigData JSON: %w", err)
	}
	return &data, nil
}

// ResolveActiveModel calls DiscoverActiveAgentModel and joins with GetCascadeModelConfig
// to dynamically determine the active model ID, friendly label, and whether it is disabled.
func (c *Client) ResolveActiveModel(ctx context.Context, pbtxtPath string) (modelID string, label string, disabled bool, err error) {
	modelID = c.DiscoverActiveAgentModel(ctx, pbtxtPath)
	if strings.TrimSpace(modelID) == "" {
		modelID = DefaultActiveAgentModel
	}

	cfgData, cfgErr := c.GetCascadeModelConfig(ctx)
	if cfgErr != nil || cfgData == nil {
		return modelID, modelID, false, nil
	}

	// 1. Check clientModelConfigs for direct model/alias match
	for _, cfg := range cfgData.ClientModelConfigs {
		if strings.EqualFold(cfg.ModelOrAlias.Model, modelID) || (cfg.ModelOrAlias.Alias != "" && strings.EqualFold(cfg.ModelOrAlias.Alias, modelID)) {
			lbl := cfg.Label
			if strings.TrimSpace(lbl) == "" {
				lbl = modelID
			}
			return modelID, lbl, cfg.Disabled, nil
		}
	}

	// 2. Check defaultOverrideModelConfig if model matches
	if cfgData.DefaultOverrideModelConfig != nil {
		moa := cfgData.DefaultOverrideModelConfig.ModelOrAlias
		if strings.EqualFold(moa.Model, modelID) || (moa.Alias != "" && strings.EqualFold(moa.Alias, modelID)) {
			lbl := cfgData.DefaultOverrideModelConfig.Label
			if strings.TrimSpace(lbl) == "" {
				lbl = modelID
			}
			return modelID, lbl, cfgData.DefaultOverrideModelConfig.Disabled, nil
		}
	}

	return modelID, modelID, false, nil
}
