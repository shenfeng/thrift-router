package main

import (
	"fmt"
	"testing"
	"time"
)

type UserProfile struct {
	Uid  int32
	Name string
	i    int
	j    int
	s    string
}

func NewUserProfile() *UserProfile {
	return &UserProfile{}
}

func init() {
}

func TestSetGetDel(t *testing.T) {
	cache := NewCache(100)
	const Size = 100
	keys := make([]string, Size)

	for i := 0; i < Size; i++ {
		dl := NewUserProfile()
		dl.Uid = int32(i)
		dl.Name = "type1"
		keys[i] = fmt.Sprintf("key-%d-no", i)
		cache.Setex(keys[i], 1, dl)
	}

	for i := 0; i < Size; i++ {
		v, ok := cache.Get(keys[i])
		if !ok {
			t.Error("cache should not miss")
		} else if v.(*UserProfile).Uid != int32(i) {
			t.Error("cache should return the value puts in")
		}
	}

	// update
	for i := 0; i < Size; i++ {
		dl := NewUserProfile()
		dl.Uid = int32(i + Size)
		dl.Name = "type1"
		cache.Setex(keys[i], 1, dl)
	}

	for i := 0; i < Size; i++ {
		v, ok := cache.Get(keys[i])
		if !ok {
			t.Error("cache should not miss")
		} else if v.(*UserProfile).Uid != int32(i+Size) {
			t.Error("cache should return the updated value")
		}
	}

	cache.Delete(keys[0])
	_, ok := cache.Get(keys[0])
	if ok {
		t.Error("delete failed")
	}

	time.Sleep(time.Second * 2)
	for i := 0; i < Size; i++ {
		_, ok := cache.Get(keys[i])
		if ok {
			t.Error("cache should expired")
		}
	}

}

func BenchmarkSetex(b *testing.B) {
	cache := NewCache(100000)
	for i := 0; i < b.N; i++ {
		cache.Setex(fmt.Sprintf("key-%d", i), 100, NewUserProfile())
	}
}

func BenchmarkSetUpdate(b *testing.B) {
	const Size = 100000
	cache := NewCache(Size)
	keys := make([]string, Size)
	for i := 0; i < Size; i++ {
		key := fmt.Sprintf("key--------%d----------", i)
		cache.Setex(key, 100, NewUserProfile())
		keys[i] = key
	}
	for i := 0; i < b.N; i++ {
		cache.Setex(keys[i%Size], 100, NewUserProfile())
	}
}

func BenchmarkGet(b *testing.B) {
	const Size = 100000
	cache := NewCache(Size)
	keys := make([]string, Size)
	for i := 0; i < Size; i++ {
		key := fmt.Sprintf("key--------%d----------", i)
		cache.Setex(key, 100, NewUserProfile())
		keys[i] = key
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(keys[i%Size])
	}
}
