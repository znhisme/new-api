package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestFilterPricingHidesCompactModels(t *testing.T) {
	pricing := []model.Pricing{
		{ModelName: "gpt-5.4", EnableGroup: []string{"default"}},
		{ModelName: "gpt-5.4-openai-compact", EnableGroup: []string{"default"}},
		{ModelName: "gpt-5.4-mini", EnableGroup: []string{"default"}},
	}

	filtered := filterPricingByUsableGroups(pricing, map[string]string{"default": "default"})

	require.Len(t, filtered, 2)
	require.Equal(t, "gpt-5.4", filtered[0].ModelName)
	require.Equal(t, "gpt-5.4-mini", filtered[1].ModelName)
}
