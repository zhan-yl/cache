package cache

import (
	"fmt"
)

func ExampleMemCache_Put() {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		evictCounter++
	}
	cache, _ := NewMemCache(1, onEvicted)
	cache.Put("key1", 2)
	cache.Put("key", 1)
	if val, ok := cache.Get("key"); ok {
		fmt.Println(val)
	}
	fmt.Println(evictCounter)
	// Output:
	// 1
	// 1
}

func ExampleMemCache_Status() {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		evictCounter++
	}
	cache, _ := NewMemCache(0, onEvicted)
	cache.Put("key", 1)
	if val, ok := cache.Get("key"); ok {
		fmt.Println(val)
	}
	status := cache.Status()
	fmt.Println(status)
	fmt.Println(evictCounter)
	// Output:
	// 1
	// &{1 1 0 1}
	// 0
}
