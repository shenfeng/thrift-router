package main

import (
	"container/list"
	"fmt"
	// "log"
	"sync"
	"time"
)

type LRUCache struct {
	mu sync.Mutex

	// list & table of *entry objects
	list  *list.List
	table map[string]*list.Element

	miss    int64 //  miss count
	hits    int64 //  hits count
	expires int64

	// max items allow to store in the cache
	capacity int
}

type entry struct {
	key    string
	value  []byte
	expire time.Time
}

func NewCache(capacity int) *LRUCache {
	return &LRUCache{
		list:     list.New(),
		table:    make(map[string]*list.Element, int(float64(capacity)*1.3)),
		capacity: capacity,
	}
}

func (lru *LRUCache) Clear() {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.list.Init()
	lru.table = make(map[string]*list.Element)
	lru.miss, lru.hits = 0, 0
}

func (lru *LRUCache) String() string {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	p := float64(lru.hits) / float64(lru.miss+lru.hits+lru.expires) * 100.0
	return fmt.Sprintf("size: %d, capacity: %d, hits: %d, miss: %d, expired: %d, %.4f%%",
		len(lru.table), lru.capacity, lru.hits, lru.miss, lru.expires, p)
}

func (lru *LRUCache) Delete(key string) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	ele := lru.table[key]
	if ele == nil {
		return false
	}

	lru.list.Remove(ele)
	delete(lru.table, key)
	return true
}

// ~600W per seconds when no contention, capacity 100000
func (lru *LRUCache) Get(key string) ([]byte, bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	element := lru.table[key]
	if element == nil {
		lru.miss += 1
		return nil, false
	}

	if element.Value.(*entry).expire.Before(time.Now()) { //  expired
		lru.expires += 1
		lru.list.Remove(element)
		delete(lru.table, key)
		return nil, false
	}
	// log.Println(element.Value)
	lru.hits += 1
	lru.list.MoveToFront(element)
	return element.Value.(*entry).value, true
}

func (lru *LRUCache) Setex(key string, seconds int, value []byte) {
	lru.mu.Lock()
	defer lru.mu.Unlock()
	expire := time.Now().Add(time.Duration(seconds) * time.Second)
	if element := lru.table[key]; element != nil {
		lru.list.MoveToFront(element)
		e := element.Value.(*entry)
		e.value = value
		e.expire = expire
	} else {
		lru.addNew(key, expire, value)
	}
}

func (lru *LRUCache) addNew(key string, expire time.Time, value []byte) {
	// protected by lock in Set()
	element := lru.list.PushFront(&entry{key: key, value: value, expire: expire})
	lru.table[key] = element

	if len(lru.table) > lru.capacity {
		toRemove := lru.list.Back()
		lru.list.Remove(toRemove)

		e := toRemove.Value.(*entry)
		delete(lru.table, e.key)
	}
}
