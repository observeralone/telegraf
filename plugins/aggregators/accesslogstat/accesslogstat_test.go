package easeaccesslogstat

import (
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/testutil"
)

// Create a valuecounter with config
func NewTestNewAccessLogStat() telegraf.Aggregator {
	vc := NewAccessLogStat()
	return vc
}

var m1, _ = metric.New("m1",
	map[string]string{"foo": "bar"},
	map[string]interface{}{
		"status_code":   "200",
		"request_time":  "10",
		"request_size":  "1024",
		"response_size": "1024",
	},
	time.Now(),
)

func TestBasic(t *testing.T) {
	vc := NewTestNewAccessLogStat()
	acc := testutil.Accumulator{}

	vc.Add(m1)
	vc.Push(&acc)

	// expectedFields := map[string]interface{}{
	// 	"status_code": 2,
	// 	"status_OK":  1,
	// }
	// expectedTags := map[string]string{
	// 	"foo": "bar",
	// }
	// acc.AssertContainsTaggedFields(t, "m1", expectedFields, expectedTags)
}
