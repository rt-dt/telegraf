package selfstat

import (
	"log"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
)

var (
	registry *rgstry
)

type Stat interface {
	Name() string
	FieldName() string
	Tags() map[string]string
	Key() uint64
	Incr(v int64)
	Set(v int64)
	Get() int64
}

func Register(measurement, field string, tags map[string]string) Stat {
	return registry.register(&stat{
		measurement: measurement,
		field:       field,
		metadata:    tags,
	})
}

func RegisterTiming(measurement, field string, tags map[string]string) Stat {
	return registry.register(&timingStat{
		measurement: measurement,
		field:       field,
		metadata:    tags,
	})
}

func Metrics() []telegraf.Metric {
	registry.mu.Lock()
	now := time.Now()
	metrics := make([]telegraf.Metric, len(registry.stats))
	i := 0
	for _, stats := range registry.stats {
		if len(stats) > 0 {
			var tags map[string]string
			var name string
			fields := map[string]interface{}{}
			for fieldname, stat := range stats {
				fields[fieldname] = stat.Get()
				tags = stat.Tags()
				name = stat.Name()
			}
			metric, err := telegraf.NewMetric(name, tags, fields, now)
			if err != nil {
				log.Printf("E! Error creating selfstat metric: %s", err)
				continue
			}
			metrics[i] = metric
			i++
		}
	}
	registry.mu.Unlock()
	return metrics
}

type rgstry struct {
	stats map[uint64]map[string]Stat
	mu    sync.Mutex
}

func (r *rgstry) register(s Stat) Stat {
	r.mu.Lock()
	defer r.mu.Unlock()
	if stats, ok := r.stats[s.Key()]; ok {
		// measurement exists
		if stat, ok := stats[s.FieldName()]; ok {
			// field already exists, so don't create a new one
			return stat
		}
		r.stats[s.Key()][s.FieldName()] = s
		return s
	} else {
		// creating a new unique metric
		r.stats[s.Key()] = map[string]Stat{s.FieldName(): s}
		return s
	}
}

func init() {
	registry = &rgstry{
		stats: make(map[uint64]map[string]Stat),
	}
}