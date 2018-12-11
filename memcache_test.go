package cache

import (
	"strconv"
	"sync"
	"testing"
)

var getTests = []struct {
	name       string
	keyToAdd   string
	keyToGet   string
	expectedOk bool
}{
	{"string_hit", testKey, testKey, true},
	{"string_miss", testKey, "nonsense", false},
}

const testKey = "mykey"
const testInt = 1234

var m sync.Mutex

func TestPut(t *testing.T) {
	t.Run("LRU Put测试：maxSize错误", func(t *testing.T) {
		_, err := NewMemCache(-10, nil)
		if err.Error() != ErrMaxsize {
			t.Fatalf("err: %v", err)
		}
	})

	t.Run("LRU Put测试：插值错误", func(t *testing.T) {
		evictCounter := 0
		onEvicted := func(k interface{}, v interface{}) {
			evictCounter++
		}
		cache, err := NewMemCache(0, onEvicted)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		values := []string{"test1", "test2", "test3"}
		key := "key1"
		for _, v := range values {
			cache.Put(key, v)
			val, ok := cache.Get(key)
			if !ok {
				t.Fatalf("expect key:%v, value:%v", key, v)
			} else if ok && val != v {
				t.Fatalf("expect key:%v, value:%v, get value:%v", key, v, val)
			}
		}
		if evictCounter != 0 {
			t.Fatalf("evictCounter is incorrect :%d", evictCounter)
		}
	})
}

func TestGet(t *testing.T) {
	t.Run("LRU Get测试", func(t *testing.T) {
		cache, err := NewMemCache(0, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		for _, tt := range getTests {
			cache.Put(tt.keyToAdd, testInt)
			val, ok := cache.Get(tt.keyToGet)

			if ok != tt.expectedOk {
				t.Fatalf("%s: val:%v cache hit = %v; want %v", tt.name, val, ok, tt.expectedOk)
			} else if ok && val != testInt {
				t.Fatalf("%s expected get to return %v but got %v", tt.name, testInt, val)
			}
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("LRU Delete测试", func(t *testing.T) {
		cache, err := NewMemCache(0, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		cache.Put(testKey, testInt)
		if val, ok := cache.Get(testKey); !ok {
			t.Fatal("TestRemove returned no match")
		} else if val != testInt {
			t.Fatalf("TestRemove failed. Expected %d, got %v", testInt, val)
		}

		cache.Delete(testKey)
		if _, ok := cache.Get(testKey); ok {
			t.Fatal("TestRemove returned a removed item")
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("LRU Status测试", func(t *testing.T) {
		keys := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}

		var gets int64
		var hits int64
		var maxSize int
		var currentSize int
		maxSize = 5
		evictCounter := 0
		onEvicted := func(k interface{}, v interface{}) {
			evictCounter++
		}
		cache, err := NewMemCache(maxSize, onEvicted)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		for _, key := range keys {
			cache.Put(key, testInt)
			currentSize++
		}
		currentSize -= evictCounter

		newKeys := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}

		for _, newKey := range newKeys {
			_, ok := cache.Get(newKey)
			if ok == true {
				hits++
			}
			gets++
		}
		t.Logf("evict:%v, gets:%v, hits:%v, hitratio:%.2f%%, maxSize:%v, currentSize:%v", evictCounter, gets, hits, float64(hits)*100.0/float64(gets), maxSize, currentSize)
		status := cache.Status()
		if status.CurrentSize != currentSize || status.MaxItemSize != maxSize ||
			status.Gets != gets || status.Hits != hits {
			t.Fatalf("get status maxSize:%v, currentSize:%v, nget:%v, nhit:%v",
				status.MaxItemSize, status.CurrentSize, status.Gets, status.Hits)
		}
	})
}

// 在测试的过程中不要使用Get检查值，否则会导致LRU访问顺序发生变化
func TestLRU(t *testing.T) {
	t.Run("LRU多次操作测试", func(t *testing.T) {
		keys := []string{"1", "2", "3", "4", "2", "1", "3", "5", "6", "5", "6"}
		keysorder1 := []string{"4", "3", "2"}
		keysorder2 := []string{"1", "2", "4"}
		keysorder3 := []string{"6", "5", "3"}
		maxSize := 3
		evictCounter := 0
		onEvicted := func(k interface{}, v interface{}) {
			evictCounter++
		}
		cache, err := NewMemCache(maxSize, onEvicted)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		for i, key := range keys {
			cache.Put(key, testInt)
			if i == 3 {
				status := cache.Status()
				if status.CurrentSize != maxSize {
					t.Fatalf("expected maxSize %v,currentSize:%v", maxSize, status.CurrentSize)
				}
				//check key&value
				_, ok1 := cache.cache["2"]
				_, ok2 := cache.cache["3"]
				if !(ok1 && ok2) {
					t.Fatalf("expected remains key 2:%v,3:%v", ok1, ok2)
				}

				//check LRU order
				for e, j := cache.cacheList.Front(), 0; e != nil; e, j = e.Next(), j+1 {
					if e.Value.(*entry).key != keysorder1[j] {
						t.Fatalf("expected key %v, got:%v", keysorder1[j], e.Value.(*entry).key)
					}
				}
			}
			if i == 5 {
				//check key&value
				_, ok1 := cache.cache["1"]
				_, ok2 := cache.cache["2"]
				_, ok3 := cache.cache["4"]

				if !(ok1 && ok2 && ok3) {
					t.Fatalf("expected remains key 1:%v 2:%v,4:%v", ok1, ok2, ok3)
				}

				//check LRU order
				for e, j := cache.cacheList.Front(), 0; e != nil; e, j = e.Next(), j+1 {
					if e.Value.(*entry).key != keysorder2[j] {
						t.Fatalf("expected key %v, got:%v", keysorder2[j], e.Value.(*entry).key)
					}
				}
			}
		}

		status := cache.Status()
		if status.CurrentSize != maxSize {
			t.Fatalf("expected maxSize %v,currentSize:%v", maxSize, status.CurrentSize)
		}
		//check key&value
		_, ok1 := cache.cache["3"]
		_, ok2 := cache.cache["5"]
		_, ok3 := cache.cache["6"]
		if !(ok1 && ok2 && ok3) {
			t.Fatalf("expected remains key 3:%v,5:%v, 6:%v", ok1, ok2, ok3)
		}
		//check LRU order
		for e, j := cache.cacheList.Front(), 0; e != nil; e, j = e.Next(), j+1 {
			if e.Value.(*entry).key != keysorder3[j] {
				t.Fatalf("expected key %v, got:%v", keysorder3[j], e.Value.(*entry).key)
			}
		}

		//swap 5 times
		if evictCounter != 5 {
			t.Fatalf("evictCounter is incorrect :%d", evictCounter)
		}
	})
}

func safethread(c *MemCache, w chan bool) {
	for i := 0; i < 1000000; i++ {
		//需要在取值与重新设值之间进行封锁，否则不同的routine之间会互相影响
		m.Lock()
		if val, ok := c.Get(testKey); ok {
			c.Put(testKey, val.(int)+1)
		}
		m.Unlock()
	}
	w <- true
	return
}

//测试线程安全性，对value进行并发设值，同时统计hit,get
func TestSafeThread(t *testing.T) {
	t.Run("线程安全性测试", func(t *testing.T) {
		cache, err := NewMemCache(0, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		cache.Put(testKey, 0)
		w1, w2, w3 := make(chan bool), make(chan bool), make(chan bool)
		go safethread(cache, w1)
		go safethread(cache, w2)
		go safethread(cache, w3)
		<-w1
		<-w2
		<-w3
		if val, ok := cache.Get(testKey); ok {
			status := cache.Status()
			if !(val == 3000000 && status.Gets == 3000001 && status.Hits == 3000001) {
				t.Fatalf("get status val:%v, maxSize:%v, currentSize:%v, nget:%v, nhit:%v",
					val, status.MaxItemSize, status.CurrentSize, status.Gets, status.Hits)
			}
		}
	})
}

func BenchmarkPutGet(b *testing.B) {
	cache, err := NewMemCache(0, nil)
	if err != nil {
		b.Fatalf("err: %v", err)
	}
	for i := 0; i < b.N; i++ {
		cache.Put(strconv.Itoa(i), i)
	}
	for i := 0; i < b.N; i++ {
		if val, ok := cache.Get(strconv.Itoa(i)); !ok {
			b.Fatalf("Can not get the key %d", i)
		} else if ok && val != i {
			b.Fatalf("Got incorrect value key %s, expect value %d, got value %d", strconv.Itoa(i), i, val)
		}
	}
}
