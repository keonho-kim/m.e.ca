// Purpose: Reuse immutable parsed call arguments across planning and preparation within one request.
// Pattern: Executor-owned bounded content-addressed cache.
// Usage: Local planning and preparation call parsedCall; lifecycle boundaries clear retained arguments.
// Related: internal/tools/local/parallel.go, internal/tools/local/prepare.go, internal/tools/local/catalog/arguments.go.
package local

import (
	"crypto/sha256"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
	"slices"
	"sync"
)

const argumentCacheBytes = 8 << 20

type argumentKey struct {
	name   string
	digest [32]byte
}
type parsedArguments struct {
	spec  toolcatalog.Specification
	input toolcatalog.Arguments
	bytes int
}
type argumentCache struct {
	mu         sync.Mutex
	values     map[argumentKey]parsedArguments
	order      []argumentKey
	bytes      int
	generation uint64
}

func (c *argumentCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values = nil
	c.order = nil
	c.bytes = 0
	c.generation++
}
func cloneArguments(input toolcatalog.Arguments) toolcatalog.Arguments {
	input.Paths = slices.Clone(input.Paths)
	input.Edits = slices.Clone(input.Edits)
	return input
}
func (e *Executor) parsedCall(call toolapi.Call) (toolcatalog.Specification, toolcatalog.Arguments, error) {
	if err := call.CheckIdentity(); err != nil {
		return toolcatalog.Specification{}, toolcatalog.Arguments{}, err
	}
	key := argumentKey{call.Name, sha256.Sum256(call.Arguments)}
	cache := &e.arguments
	cache.mu.Lock()
	known, ok := cache.values[key]
	generation := cache.generation
	cache.mu.Unlock()
	if ok {
		return known.spec, cloneArguments(known.input), nil
	}
	spec, input, err := toolcatalog.Parse(call)
	if err != nil {
		return spec, input, err
	}
	size := 2*len(call.Arguments) + 512
	if size <= argumentCacheBytes {
		cache.mu.Lock()
		if _, ok := cache.values[key]; !ok && cache.generation == generation {
			for cache.bytes+size > argumentCacheBytes && len(cache.order) > 0 {
				old := cache.order[0]
				cache.order = cache.order[1:]
				cache.bytes -= cache.values[old].bytes
				delete(cache.values, old)
			}
			if cache.values == nil {
				cache.values = make(map[argumentKey]parsedArguments)
			}
			cache.values[key] = parsedArguments{spec, input, size}
			cache.order = append(cache.order, key)
			cache.bytes += size
		}
		cache.mu.Unlock()
	}
	return spec, cloneArguments(input), nil
}
