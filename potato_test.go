package echoprobe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPotato(t *testing.T) {
	want := "potato"
	got := potato()

	assert.Equal(t, want, got, "potato() should return 'potato'")
}
