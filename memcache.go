package cache

import (
	"container/list"
	"errors"
	"sync"
	"sync/atomic"
)

const ErrMaxsize = "Must provide a nonnegative size"

// An AtomicInt is an int64 to be accessed atomically.
type AtomicInt int64

// EvictCallback is used to get a callback when a cache entry is evicted
type EvictCallback func(key interface{}, value interface{})

// MemCache is an LRU cache. It is safe for concurrent access.
type MemCache struct {
	mutex       sync.RWMutex
	maxItemSize int
	cacheList   *list.List
	cache       map[interface{}]*list.Element
	hits, gets  AtomicInt
	onEvict     EvictCallback
}

// Map中具体的存储结构
type entry struct {
	key   interface{}
	value interface{}
}

//return status of chache
type CacheStatus struct {
	Gets        int64
	Hits        int64
	MaxItemSize int
	CurrentSize int
}

//this is a interface which defines some common functions
type Cache interface {
	Put(key string, value interface{})
	Get(key string) (interface{}, bool)
	Delete(key string)
	Status() *CacheStatus
}

/*
创建一个MemCache结构，传入参数maxItemSize表明该结构的最大大小.为0表示不限大小，非0的情况下cache大小超过maxItemSize将触发swap

onEvict为提供的callback函数，在发生swap时被触发

	NewMemCache If maxItemSize is zero, the cache has no limit.
	if maxItemSize is not zero, when cache's size beyond maxItemSize,start to swap
*/
func NewMemCache(maxItemSize int, onEvict EvictCallback) (*MemCache, error) {
	if maxItemSize < 0 {
		return nil, errors.New(ErrMaxsize)
	}
	return &MemCache{
		maxItemSize: maxItemSize,
		cacheList:   list.New(),
		cache:       make(map[interface{}]*list.Element),
		onEvict:     onEvict,
	}, nil
}

//返回cache的当前状态
//	Status return the status of cache
func (c *MemCache) Status() *CacheStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return &CacheStatus{
		MaxItemSize: c.maxItemSize,
		CurrentSize: c.cacheList.Len(),
		Gets:        c.gets.Get(),
		Hits:        c.hits.Get(),
	}
}

//根据一个key获取其值，如果命中将改变其在LRU中的优先顺序，
//因此在测试中不要用其取值
//	Get value with key
func (c *MemCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	c.gets.Add(1)
	if ele, hit := c.cache[key]; hit {
		c.hits.Add(1)
		c.cacheList.MoveToFront(ele)
		return ele.Value.(*entry).value, true
	}
	return nil, false
}

//为key设置一个值，在队列长度大于maxItemSize的情况下触发swap并计数
//	Put a value with key
func (c *MemCache) Put(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ele, ok := c.cache[key]; ok {
		c.cacheList.MoveToFront(ele)
		ele.Value.(*entry).value = value
		return
	}

	ele := c.cacheList.PushFront(&entry{key: key, value: value})
	c.cache[key] = ele

	if c.maxItemSize != 0 && c.cacheList.Len() > c.maxItemSize {
		c.RemoveOldest()
	}
}

//删除cache中的一个key
//	Delete delete the key
func (c *MemCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ele, ok := c.cache[key]; ok {
		c.cacheList.Remove(ele)
		key := ele.Value.(*entry).key
		delete(c.cache, key)
		return
	}
}

//根据最近最少使用原则删除最久未使用的key, 即去除队尾
//	RemoveOldest remove the oldest key
func (c *MemCache) RemoveOldest() {
	ele := c.cacheList.Back()
	if ele != nil {
		c.cacheList.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		if c.onEvict != nil {
			c.onEvict(kv.key, kv.value)
		}
	}
}

// 自动将n增加到i
//	Add atomically adds n to i.
func (i *AtomicInt) Add(n int64) {
	atomic.AddInt64((*int64)(i), n)
}

// 获取i的值
//	Get atomically gets the value of i.
func (i *AtomicInt) Get() int64 {
	return atomic.LoadInt64((*int64)(i))
}
