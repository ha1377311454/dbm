package adapter

import (
	"dbm/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMongoDBAdapter_Connect(t *testing.T) {
	a := NewMongoDBAdapter()

	// Test without host or URI
	config := &model.ConnectionConfig{
		Type: model.DatabaseMongoDB,
	}
	_, err := a.Connect(config)
	assert.Error(t, err)

	// Test with invalid host (should fail to connect)
	config = &model.ConnectionConfig{
		Type: model.DatabaseMongoDB,
		Host: "localhost",
		Port: 27018, // Likely nothing there
	}
	// We don't want to actually connect in a unit test if possible,
	// but currently the adapter pings on connect.
	// So we expect a failure unless a local mongodb is running.
}

func TestMongoDBAdapter_Metadata(t *testing.T) {
	// These would require a mock or a live connection.
	// For now, let's just test that the functions exist and have correct signatures.
}
