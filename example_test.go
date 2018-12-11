package cache

import (
	"fmt"
)

func ExampleMemCache_Put() {
	var cache Cache
	cache = NewMemCache(0)
	cache.Put("key", 1)
	if val, ok := cache.Get("key"); ok {
		fmt.Println(val)
	}
	// Output:
	// 1
}

func ExampleMemCache_Status() {
	var cache Cache
	cache = NewMemCache(0)
	cache.Put("key", 1)
	if val, ok := cache.Get("key"); ok {
		fmt.Println(val)
	}
	status := cache.Status()
	fmt.Println(status)
	// Output:
	// 1
	// &{1 1 0 1}
}
