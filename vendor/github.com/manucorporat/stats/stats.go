package stats

import "sync"

type ValueType float64
type StatsType map[string]ValueType

type StatsCollector struct {
	lock  sync.RWMutex
	stats StatsType
}

func New() *StatsCollector {
	s := new(StatsCollector)
	s.Reset()
	return s
}

func (s *StatsCollector) Reset() {
	s.lock.Lock()
	s.stats = make(StatsType)
	s.lock.Unlock()
}

func (s *StatsCollector) Set(key string, value ValueType) {
	s.lock.Lock()
	s.stats[key] = value
	s.lock.Unlock()
}

func (s *StatsCollector) Add(key string, delta ValueType) (v ValueType) {
	s.lock.Lock()
	v = s.stats[key]
	v += delta
	s.stats[key] = v
	s.lock.Unlock()
	return
}

func (s *StatsCollector) Get(key string) (v ValueType) {
	s.lock.RLock()
	v = s.stats[key]
	s.lock.RUnlock()
	return
}

func (s *StatsCollector) Del(key string) {
	s.lock.Lock()
	delete(s.stats, key)
	s.lock.Unlock()
}

func (s *StatsCollector) Data() StatsType {
	cp := make(StatsType)
	s.lock.RLock()
	for key, value := range s.stats {
		cp[key] = value
	}
	s.lock.RUnlock()
	return cp
}

var defaultCollector = New()

func Reset() {
	defaultCollector.Reset()
}

func Set(key string, value ValueType) {
	defaultCollector.Set(key, value)
}

func Del(key string) {
	defaultCollector.Del(key)
}

func Add(key string, delta ValueType) ValueType {
	return defaultCollector.Add(key, delta)
}

func Get(key string) ValueType {
	return defaultCollector.Get(key)
}

func Data() StatsType {
	return defaultCollector.Data()
}
