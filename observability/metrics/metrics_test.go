package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics_NilConfig(t *testing.T) {
	c, err := NewMetrics(nil)

	assert.Nil(t, c)
	assert.ErrorIs(t, err, ErrNilConfig)
}

func TestNewMetrics_Success(t *testing.T) {
	cfg := &Config{
		Namespace: "test",
		Path:      "/metrics",
	}

	c, err := NewMetrics(cfg)

	require.NoError(t, err)
	assert.IsType(t, &PrometheusCollector{}, c)
}

func TestMustNewMetrics_Success(t *testing.T) {
	cfg := &Config{
		Namespace: "test",
	}

	assert.NotPanics(t, func() {
		c := MustNewMetrics(cfg)
		assert.NotNil(t, c)
	})
}

func TestMustNewMetrics_NilConfig(t *testing.T) {
	assert.Panics(t, func() {
		MustNewMetrics(nil)
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "/metrics", cfg.Path)
	assert.Equal(t, "app", cfg.Namespace)
}
