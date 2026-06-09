package crdt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPheromoneSummary_Merge(t *testing.T) {
	a := NewPheromoneSummary()
	b := NewPheromoneSummary()

	now := time.Now()
	a.Set("SAFE", "task-1", "issuer-a", now.Add(-1*time.Hour), 1.0)
	b.Set("DANGER", "task-1", "issuer-b", now, 1.0) // newer

	merged := a.Merge(b)
	entry := merged.Records["task-1_issuer-b"]
	require.Equal(t, "DANGER", entry.Kind)
	require.Equal(t, "issuer-b", entry.Issuer)
}

func TestPheromoneSummary_NewerWins(t *testing.T) {
	now := time.Now()
	a := NewPheromoneSummary()
	b := NewPheromoneSummary()

	a.Set("TODO", "task-x", "i1", now.Add(-2*time.Hour), 0.5)
	b.Set("SAFE", "task-x", "i1", now.Add(-1*time.Hour), 1.0)

	merged := a.Merge(b)
	require.Equal(t, "SAFE", merged.Records["task-x_i1"].Kind)
}

func TestCapabilitySummary_Merge(t *testing.T) {
	a := NewCapabilitySummary()
	b := NewCapabilitySummary()

	a.AddTier("EXECUTOR_BASIC")
	a.AddAccelerator("npu-0")
	b.AddTier("EXECUTOR_WASM")
	b.AddAccelerator("npu-0") // duplicate
	b.AddTransportProfile("fast-wide")
	b.QueueDepth = 20

	a.QueueDepth = 10

	merged := a.Merge(b)
	require.True(t, merged.Tiers["EXECUTOR_BASIC"])
	require.True(t, merged.Tiers["EXECUTOR_WASM"])
	require.True(t, merged.Accelerators["npu-0"])
	require.True(t, merged.TransportProfs["fast-wide"])
	require.Equal(t, uint32(20), merged.QueueDepth) // max
}

func TestSensorAggregate_Merge(t *testing.T) {
	now := time.Now()
	a := NewSensorAggregate()
	b := NewSensorAggregate()

	a.Set("temp_c", 25.5, "celsius", now.Add(-1*time.Hour))
	b.Set("temp_c", 26.1, "celsius", now) // newer

	merged := a.Merge(b)
	require.Equal(t, 26.1, merged.Values["temp_c"].Val)
}

func TestSensorAggregate_LWWSameKey(t *testing.T) {
	now := time.Now()
	a := NewSensorAggregate()
	b := NewSensorAggregate()

	a.Set("cpu_percent", 80.0, "%", now.Add(-1*time.Hour))
	b.Set("cpu_percent", 95.0, "%", now) // newer

	merged := a.Merge(b)
	require.Equal(t, 95.0, merged.Values["cpu_percent"].Val)
}

func TestFamily(t *testing.T) {
	require.NotNil(t, Family("pheromone_summary"))
	require.NotNil(t, Family("capability_summary"))
	require.NotNil(t, Family("sensor_aggregate"))
	require.Nil(t, Family("code_repo"))      // blocked per AllowNaiveCRDT
	require.Nil(t, Family("website"))         // blocked
	require.Nil(t, Family("binary_artifact")) // blocked
}

func TestPheromoneSummary_SizeBytes(t *testing.T) {
	s := NewPheromoneSummary()
	s.Set("SAFE", "task-1", "issuer", time.Now(), 1.0)
	size := s.SizeBytes()
	require.Greater(t, size, 0)
}
