package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompactModelBillingInheritsBaseModel(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, UpdateModelPriceByJSONString(`{"gpt-5.4":0.42,"*-openai-compact":9.9}`))
	require.NoError(t, UpdateModelRatioByJSONString(`{"gpt-5.4":3.14,"*-openai-compact":8.8}`))
	require.NoError(t, UpdateCompletionRatioByJSONString(`{"gpt-5.4":6.28}`))
	require.NoError(t, UpdateCacheRatioByJSONString(`{"gpt-5.4":0.5}`))
	require.NoError(t, UpdateCreateCacheRatioByJSONString(`{"gpt-5.4":1.5}`))

	compact := "gpt-5.4-openai-compact"
	price, ok := GetModelPrice(compact, false)
	require.True(t, ok)
	require.Equal(t, 0.42, price)

	ratio, ok, matchName := GetModelRatio(compact)
	require.True(t, ok)
	require.Equal(t, 3.14, ratio)
	require.Equal(t, "gpt-5.4", matchName)

	require.Equal(t, 6.28, GetCompletionRatio(compact))

	cacheRatio, ok := GetCacheRatio(compact)
	require.True(t, ok)
	require.Equal(t, 0.5, cacheRatio)

	createCacheRatio, ok := GetCreateCacheRatio(compact)
	require.True(t, ok)
	require.Equal(t, 1.5, createCacheRatio)
}

func TestCompactModelBillingFallsBackToWildcard(t *testing.T) {
	restoreRatioSettings(t)

	require.NoError(t, UpdateModelPriceByJSONString(`{"*-openai-compact":9.9}`))
	require.NoError(t, UpdateModelRatioByJSONString(`{"*-openai-compact":8.8}`))

	compact := "gpt-5.4-openai-compact"
	price, ok := GetModelPrice(compact, false)
	require.True(t, ok)
	require.Equal(t, 9.9, price)

	ratio, ok, matchName := GetModelRatio(compact)
	require.True(t, ok)
	require.Equal(t, 8.8, ratio)
	require.Equal(t, CompactWildcardModelKey, matchName)
}

func TestWithCompactModelSuffixDoesNotDuplicateSuffix(t *testing.T) {
	require.Equal(t, "gpt-5.4-openai-compact", WithCompactModelSuffix("gpt-5.4"))
	require.Equal(t, "gpt-5.4-openai-compact", WithCompactModelSuffix("gpt-5.4-openai-compact"))
}

func restoreRatioSettings(t *testing.T) {
	t.Helper()

	modelPriceJSON := ModelPrice2JSONString()
	modelRatioJSON := ModelRatio2JSONString()
	completionRatioJSON := CompletionRatio2JSONString()
	cacheRatioJSON := CacheRatio2JSONString()
	createCacheRatioJSON := CreateCacheRatio2JSONString()

	t.Cleanup(func() {
		require.NoError(t, UpdateModelPriceByJSONString(modelPriceJSON))
		require.NoError(t, UpdateModelRatioByJSONString(modelRatioJSON))
		require.NoError(t, UpdateCompletionRatioByJSONString(completionRatioJSON))
		require.NoError(t, UpdateCacheRatioByJSONString(cacheRatioJSON))
		require.NoError(t, UpdateCreateCacheRatioByJSONString(createCacheRatioJSON))
	})
}
