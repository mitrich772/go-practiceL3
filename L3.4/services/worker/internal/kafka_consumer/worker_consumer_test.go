package kafka_consumer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"contracts/dto"
)

func TestParseJob_Valid(t *testing.T) {
	job := dto.ImageJob{
		ID:           "abc-123",
		OriginalPath: "originals/abc-123.jpg",
		Mode:         "resize",
		Width:        100,
		Height:       200,
	}

	data, err := json.Marshal(job)
	require.NoError(t, err)

	var parsed dto.ImageJob
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "abc-123", parsed.ID)
	assert.Equal(t, "originals/abc-123.jpg", parsed.OriginalPath)
	assert.Equal(t, "resize", parsed.Mode)
	assert.Equal(t, 100, parsed.Width)
	assert.Equal(t, 200, parsed.Height)
}

func TestParseJob_InvalidJSON(t *testing.T) {
	garbage := []byte(`{invalid json!!!`)

	var parsed dto.ImageJob
	err := json.Unmarshal(garbage, &parsed)
	require.Error(t, err)
}

func TestParseJob_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "missing ID",
			json: `{"original_path":"originals/x.jpg","mode":"resize"}`,
		},
		{
			name: "missing OriginalPath",
			json: `{"id":"abc","mode":"resize"}`,
		},
		{
			name: "missing Mode",
			json: `{"id":"abc","original_path":"originals/x.jpg"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var parsed dto.ImageJob
			err := json.Unmarshal([]byte(tc.json), &parsed)
			require.NoError(t, err)

			hasMissing := parsed.ID == "" || parsed.OriginalPath == "" || parsed.Mode == ""
			assert.True(t, hasMissing)
		})
	}
}
