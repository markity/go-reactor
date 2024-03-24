package kvcontext

import "sync"

type keyValueContext struct {
	mu sync.RWMutex

	kv map[string]interface{}
}

type KVContext interface {
	Set(key string, value interface{})
	Delete(key string)
	Get(key string) (value interface{}, exists bool)
}

func (c *keyValueContext) Set(key string, value interface{}) {
	c.mu.Lock()
	if c.kv == nil {
		c.kv = make(map[string]interface{})
	}

	c.kv[key] = value
	c.mu.Unlock()
}

func (c *keyValueContext) Delete(key string) {
	c.mu.Lock()
	delete(c.kv, key)
	c.mu.Unlock()
}

func (c *keyValueContext) Get(key string) (value interface{}, exists bool) {
	c.mu.RLock()
	value, exists = c.kv[key]
	c.mu.RUnlock()
	return
}

func (c *keyValueContext) reset() {
	c.mu.Lock()
	c.kv = nil
	c.mu.Unlock()
}

func NewContext() KVContext {
	return &keyValueContext{
		kv: make(map[string]interface{}),
	}
}
