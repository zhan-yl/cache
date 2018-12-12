package cache

import (
	"fmt"
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

func assertEqual(t *testing.T, got, want interface{}) {
	//错误定位
	t.Helper()
	if got != want {
		t.Fatalf("got '%v' want '%v'", got, want)
	}
}

func TestPut(t *testing.T) {
	t.Run("LRU Put测试：maxSize错误", func(t *testing.T) {
		_, err := NewMemCache(-10, nil)
		assertEqual(t, err.Error(), ErrMaxsize)
	})

	t.Run("LRU Put测试：插值错误", func(t *testing.T) {
		evictCounter := 0
		onEvicted := func(k interface{}, v interface{}) {
			evictCounter++
		}
		cache, err := NewMemCache(0, onEvicted)
		assertEqual(t, err, nil)

		values := []string{"test1", "test2", "test3"}
		key := "key1"
		for _, v := range values {
			cache.Put(key, v)
			val, ok := cache.Get(key)
			assertEqual(t, ok, true)
			assertEqual(t, val, v)
		}
		assertEqual(t, evictCounter, 0)
	})
}

func TestGet(t *testing.T) {
	t.Run("LRU Get测试", func(t *testing.T) {
		cache, err := NewMemCache(0, nil)
		assertEqual(t, err, nil)

		for _, tt := range getTests {
			cache.Put(tt.keyToAdd, testInt)
			val, ok := cache.Get(tt.keyToGet)
			assertEqual(t, ok, tt.expectedOk)
			if ok {
				assertEqual(t, val, testInt)
			}
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("LRU Delete测试", func(t *testing.T) {
		cache, err := NewMemCache(0, nil)
		assertEqual(t, err, nil)

		cache.Put(testKey, testInt)
		val, ok := cache.Get(testKey)
		assertEqual(t, ok, true)
		assertEqual(t, val, testInt)

		cache.Delete(testKey)
		_, ok = cache.Get(testKey)
		assertEqual(t, ok, false)
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
		assertEqual(t, err, nil)

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
		assertEqual(t, status.CurrentSize, currentSize)
		assertEqual(t, status.MaxItemSize, maxSize)
		assertEqual(t, status.Gets, gets)
		assertEqual(t, status.Hits, hits)
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
			fmt.Printf("The item will be swapped out, key=%v, value=%v\n", k, v)
			evictCounter++
		}
		cache, err := NewMemCache(maxSize, onEvicted)
		assertEqual(t, err, nil)

		for i, key := range keys {
			cache.Put(key, testInt)
			if i == 3 {
				status := cache.Status()
				assertEqual(t, status.CurrentSize, maxSize)

				//check key&value
				_, ok1 := cache.cache["2"]
				_, ok2 := cache.cache["3"]
				assertEqual(t, ok1, true)
				assertEqual(t, ok2, true)

				//check LRU order
				for e, j := cache.cacheList.Front(), 0; e != nil; e, j = e.Next(), j+1 {
					assertEqual(t, e.Value.(*entry).key, keysorder1[j])
				}
			}
			if i == 5 {
				//check key&value
				_, ok1 := cache.cache["1"]
				_, ok2 := cache.cache["2"]
				_, ok3 := cache.cache["4"]

				assertEqual(t, ok1, true)
				assertEqual(t, ok2, true)
				assertEqual(t, ok3, true)

				//check LRU order
				for e, j := cache.cacheList.Front(), 0; e != nil; e, j = e.Next(), j+1 {
					assertEqual(t, e.Value.(*entry).key, keysorder2[j])
				}
			}
		}

		status := cache.Status()
		assertEqual(t, status.CurrentSize, maxSize)

		//check key&value
		_, ok1 := cache.cache["3"]
		_, ok2 := cache.cache["5"]
		_, ok3 := cache.cache["6"]
		assertEqual(t, ok1, true)
		assertEqual(t, ok2, true)
		assertEqual(t, ok3, true)

		//check LRU order
		for e, j := cache.cacheList.Front(), 0; e != nil; e, j = e.Next(), j+1 {
			assertEqual(t, e.Value.(*entry).key, keysorder3[j])
		}

		//swap 5 times
		assertEqual(t, evictCounter, 5)
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
		assertEqual(t, err, nil)

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
			assertEqual(t, val, 3000000)
			assertEqual(t, status.Gets, int64(3000001))
			assertEqual(t, status.Hits, int64(3000001))
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
