package perfmetrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHourlyStatusSeriesReturnsContinuousHoursAndMarksMissingData(t *testing.T) {
	const endTs int64 = 20*3600 + 123
	buckets := map[int64]counters{
		18 * 3600: {requestCount: 4, successCount: 3},
		20 * 3600: {requestCount: 2, successCount: 2},
	}

	series := hourlyStatusSeries(buckets, 3, endTs)

	require.Len(t, series, 3)
	assert.Equal(t, int64(18*3600), series[0].Ts)
	require.NotNil(t, series[0].SuccessRate)
	assert.Equal(t, 75.0, *series[0].SuccessRate)
	assert.Nil(t, series[1].SuccessRate)
	assert.Zero(t, series[1].RequestCount)
	require.NotNil(t, series[2].SuccessRate)
	assert.Equal(t, 100.0, *series[2].SuccessRate)
}
