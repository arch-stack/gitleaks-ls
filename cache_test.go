package main

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache_GetPut(t *testing.T) {
	cache := NewCache()

	content := "const key = \"AKIATESTKEYEXAMPLE7A\""
	findings := []Finding{
		{RuleID: "aws-access-key", Description: "AWS Access Key"},
	}

	// Initially empty
	_, ok := cache.Get(content)
	assert.False(t, ok, "Cache should be empty initially")

	// Put and get
	cache.Put(content, findings)
	result, ok := cache.Get(content)
	assert.True(t, ok, "Should find cached entry")
	assert.Len(t, result, 1)
	assert.Equal(t, "aws-access-key", result[0].RuleID)
}

func TestCache_DifferentContent(t *testing.T) {
	cache := NewCache()

	content1 := "secret1"
	content2 := "secret2"

	findings1 := []Finding{{RuleID: "rule1"}}
	findings2 := []Finding{{RuleID: "rule2"}}

	cache.Put(content1, findings1)
	cache.Put(content2, findings2)

	result1, ok := cache.Get(content1)
	assert.True(t, ok)
	assert.Equal(t, "rule1", result1[0].RuleID)

	result2, ok := cache.Get(content2)
	assert.True(t, ok)
	assert.Equal(t, "rule2", result2[0].RuleID)
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache()

	cache.Put("content1", []Finding{{RuleID: "rule1"}})
	cache.Put("content2", []Finding{{RuleID: "rule2"}})

	assert.Equal(t, 2, cache.Size())

	cache.Clear()

	assert.Equal(t, 0, cache.Size())
	_, ok := cache.Get("content1")
	assert.False(t, ok, "Cache should be empty after clear")
}

func TestCache_Size(t *testing.T) {
	cache := NewCache()

	assert.Equal(t, 0, cache.Size())

	cache.Put("a", []Finding{})
	assert.Equal(t, 1, cache.Size())

	cache.Put("b", []Finding{})
	assert.Equal(t, 2, cache.Size())

	// Same content doesn't increase size
	cache.Put("a", []Finding{{RuleID: "updated"}})
	assert.Equal(t, 2, cache.Size())
}

func TestCache_EmptyFindings(t *testing.T) {
	cache := NewCache()

	content := "clean code with no secrets"
	cache.Put(content, []Finding{})

	result, ok := cache.Get(content)
	assert.True(t, ok, "Should cache empty findings too")
	assert.Empty(t, result)
}

func TestCache_Concurrent(t *testing.T) {
	cache := NewCache()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			content := string(rune('a' + n%26))
			cache.Put(content, []Finding{{RuleID: "rule"}})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			content := string(rune('a' + n%26))
			cache.Get(content)
		}(i)
	}

	wg.Wait()
	// No race conditions or panics
}
