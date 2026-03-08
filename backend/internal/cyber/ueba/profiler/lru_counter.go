package profiler

import (
	"container/list"
	"sort"
	"time"
)

type LRUFrequencySet struct {
	items    map[string]*LRUItem
	order    *list.List
	capacity int
}

type LRUItem struct {
	Key       string
	Frequency float64
	LastSeen  time.Time
	Element   *list.Element
}

func NewLRUFrequencySet(capacity int) *LRUFrequencySet {
	return &LRUFrequencySet{
		items:    make(map[string]*LRUItem, capacity),
		order:    list.New(),
		capacity: capacity,
	}
}

func (s *LRUFrequencySet) Access(key string, timestamp time.Time, alpha float64) {
	if key == "" || s.capacity <= 0 {
		return
	}
	if item, ok := s.items[key]; ok {
		item.Frequency = EMA(item.Frequency, 1, alpha)
		item.LastSeen = timestamp
		s.order.MoveToFront(item.Element)
		return
	}
	if len(s.items) >= s.capacity {
		s.evictLowestFrequency()
	}
	element := s.order.PushFront(key)
	s.items[key] = &LRUItem{
		Key:       key,
		Frequency: alpha,
		LastSeen:  timestamp,
		Element:   element,
	}
}

func (s *LRUFrequencySet) Values() []*LRUItem {
	values := make([]*LRUItem, 0, len(s.items))
	for element := s.order.Front(); element != nil; element = element.Next() {
		key := element.Value.(string)
		values = append(values, s.items[key])
	}
	return values
}

func (s *LRUFrequencySet) evictLowestFrequency() {
	if len(s.items) == 0 {
		return
	}
	values := s.Values()
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Frequency == values[j].Frequency {
			return values[i].LastSeen.Before(values[j].LastSeen)
		}
		return values[i].Frequency < values[j].Frequency
	})
	victim := values[0]
	s.order.Remove(victim.Element)
	delete(s.items, victim.Key)
}
