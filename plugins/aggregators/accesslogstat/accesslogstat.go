package easeaccesslogstat

// accesslogstat.go

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/aggregators"
	metrics "github.com/rcrowley/go-metrics"
)

type AccessLogStat struct {
	cache           map[uint64]aggregate
	RequestTimeUnit string `toml:"request_time_unit"`
}

func NewAccessLogStat() telegraf.Aggregator {
	mm := &AccessLogStat{}
	mm.Reset()
	mm.cache = make(map[uint64]aggregate)
	return mm
}

type aggregate struct {
	meter    metrics.Meter
	errMeter metrics.Meter
	sample   metrics.Sample
	counter  *counter
	count    int64
	name     string
	tags     map[string]string
}

// The accumulated value from monitoring to the current time
type counter struct {
	count    int64
	reqSize  uint64
	respSize uint64
}

var sampleConfig = `
  ## [[inputs.tail]]
  ## ## file(s) to tail:
  ## files = ["/Users/ease/go/access.log"]
  ## from_beginning = true
  ##
  ## #name of the "Metric" (which I want to see in Grafana eventually)
  ## name_override = "magicparser"
  ##
  ## grok_patterns = ["\\[%{NON_SPACE:timestamp:ts-rfc3339}\\] \\[%{NON_SPACE:request_time:float}\\] \\[%{NON_SPACE:origin:tag} %{NON_SPACE:service:tag} %{NON_SPACE:host_ipv4:tag} %{NON_SPACE:host_name:tag} %{NON_SPACE:client_ip:tag} %{NON_SPACE:status_code} %{NON_SPACE:request_size:int} %{NON_SPACE:response_size:int} %{NON_SPACE:trace_id:drop}\\]"]
  ##   grok_custom_patterns = '''NON_SPACE [^ ]*'''
  ## data_format = "grok"
  ##
  ## [[aggregators.accesslogstat]]
  ##   period = "30s"
  ##
  ## [[outputs.file]]
  ##   files = ["/Users/ease/go/stdout"]
  ##   data_format = "json"

  ## General Aggregator Arguments:
  ## The period on which to flush & clear the aggregator.
  period = "30s"
  ## If true, the original metric will be dropped by the
  ## aggregator and will not get sent to the output plugins.
  drop_original = false
  request_time_unit = "s"
`

func (m *AccessLogStat) SampleConfig() string {
	return sampleConfig
}

func (m *AccessLogStat) Description() string {
	return "Keep the aggregate min/max of each metric passing through."
}

func (m *AccessLogStat) Add(in telegraf.Metric) {
	id := in.HashID()
	if _, ok := m.cache[id]; !ok {
		// hit an uncached metric, create caches for first time:
		a := aggregate{
			name:     in.Name(),
			tags:     in.Tags(),
			meter:    metrics.NewMeter(),
			errMeter: metrics.NewMeter(),
			// https://github.com/rcrowley/go-metrics/blob/3113b8401b8a98917cde58f8bbd42a1b1c03b1fd/sample_test.go#L65
			sample: metrics.NewExpDecaySample(1028, 0.015),
			counter: &counter{
				count:    0,
				reqSize:  0,
				respSize: 0,
			},
		}
		m.cache[id] = a
		m.add(&a, in)
	} else {
		a := m.cache[id]
		m.add(&a, in)
	}
}

//todo pase time log_timestamp

func (m *AccessLogStat) add(a *aggregate, in telegraf.Metric) {
	a.meter.Mark(1)
	a.counter.count++
	statusCodeName := "status_code"
	requestTimeName := "request_time"
	requestSizeName := "request_size"
	responseSizeName := "response_size"
	if in.HasField(statusCodeName) {
		statusCode, _ := in.GetField(statusCodeName)
		if convertInt(statusCode) > 400 {
			a.errMeter.Mark(1)
		}
	}
	if in.HasField(requestTimeName) {
		requestTime, _ := in.GetField(requestTimeName)
		//todo change to time.Duration
		var requestTimeUnit = "s"
		if m.RequestTimeUnit != "" {
			requestTimeUnit = m.RequestTimeUnit
		}
		duration := convertDuration(requestTimeUnit, requestTime)
		a.sample.Update(duration)
	}

	if in.HasField(requestSizeName) {
		requestSize, _ := in.GetField(requestSizeName)
		a.counter.reqSize += uint64(convertInt(requestSize))
	}
	if in.HasField(responseSizeName) {
		responseSize, _ := in.GetField(responseSizeName)
		a.counter.respSize += uint64(convertInt(responseSize))
	}
}

func convertDuration(unit string, in interface{}) int64 {
	str := fmt.Sprintf("%v%v", in, unit)
	duration, err := time.ParseDuration(str)
	if err != nil {
		log.Fatal("E! ParseDuration " + str + " fail! " + err.Error())
		return 0
	}
	return int64(duration) / 1e6
}

func convertInt(in interface{}) int64 {
	switch v := in.(type) {
	case string:
		result, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Fatal("E! string.parseInt(" + v + ") fail! " + err.Error())
			return 0
		}
		return result

	default:
		return in.(int64)
	}
}

func nanoToMilli(f float64) float64 {
	return f / 1000000
}

func (m *AccessLogStat) Push(acc telegraf.Accumulator) {
	for _, aggregate := range m.cache {

		meter := aggregate.meter.Snapshot()
		errMeter := aggregate.errMeter.Snapshot()
		m1, m5, m15 := meter.Rate1(), meter.Rate5(), meter.Rate15()
		m1Err, m5Err, m15Err := errMeter.Rate1(), errMeter.Rate5(), errMeter.Rate15()
		m1ErrPercent, m5ErrPercent, m15ErrPercent := 0.0, 0.0, 0.0
		if m1 > 0 {
			m1ErrPercent = m1Err / m1
		}
		if m5 > 0 {
			m1ErrPercent = m5Err / m5
		}
		if m15 > 0 {
			m1ErrPercent = m15Err / m15
		}

		percentiles := aggregate.sample.Percentiles([]float64{
			0.25, 0.5, 0.75,
			0.95, 0.98, 0.99,
			0.999,
		})

		for i, p := range percentiles {
			percentiles[i] = nanoToMilli(p)
		}

		counter := aggregate.counter

		fields := map[string]interface{}{
			"count":         counter.count,
			"m1":            m1,
			"m5":            m5,
			"m15":           m15,
			"errCount":      errMeter.Count(),
			"m1err":         m1Err,
			"m5err":         m5Err,
			"m15err":        m15Err,
			"m1ErrPercent":  m1ErrPercent,
			"m5ErrPercent":  m5ErrPercent,
			"m15ErrPercent": m15ErrPercent,
			"p25":           percentiles[0],
			"p50":           percentiles[1],
			"p75":           percentiles[2],
			"p95":           percentiles[3],
			"p98":           percentiles[4],
			"p99":           percentiles[5],
			"p999":          percentiles[6],
			"min":           aggregate.sample.Min(),
			"mean":          aggregate.sample.Mean(),
			"max":           aggregate.sample.Max(),
			"reqSize":       counter.reqSize,
			"respSize":      counter.respSize,
		}
		acc.AddFields(aggregate.name, fields, aggregate.tags)
	}
}

func (m *AccessLogStat) Reset() {
	for _, aggregate := range m.cache {
		aggregate.sample.Clear()
	}
}

func init() {
	aggregators.Add("accesslogstat", func() telegraf.Aggregator {
		return NewAccessLogStat()
	})
}
