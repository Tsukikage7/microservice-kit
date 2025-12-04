package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_NilConfig(t *testing.T) {
	c, err := New(nil)

	assert.Nil(t, c)
	assert.ErrorIs(t, err, ErrNilConfig)
}

func TestNew_Success(t *testing.T) {
	cfg := &Config{
		Namespace: "test",
		Path:      "/metrics",
	}

	c, err := New(cfg)

	require.NoError(t, err)
	assert.IsType(t, &PrometheusCollector{}, c)
}

func TestMustNew_Success(t *testing.T) {
	cfg := &Config{
		Namespace: "test",
	}

	assert.NotPanics(t, func() {
		c := MustNew(cfg)
		assert.NotNil(t, c)
	})
}

func TestMustNew_NilConfig(t *testing.T) {
	assert.Panics(t, func() {
		MustNew(nil)
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "/metrics", cfg.Path)
	assert.Equal(t, "app", cfg.Namespace)
}
