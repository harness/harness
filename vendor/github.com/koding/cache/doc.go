// Package cache provides basic caching mechanisms for Go(lang) projects.
//
// Currently supported caching algorithms:
//     MemoryNoTS: provides a non-thread safe in-memory caching system
//     Memory    : provides a thread safe in-memory caching system, built on top of MemoryNoTS cache
//     LRUNoTS   : provides a non-thread safe, fixed size in-memory caching system, built on top of MemoryNoTS cache
//     LRU       : provides a thread safe, fixed size in-memory caching system, built on top of LRUNoTS cache
//     MemoryTTL : provides a thread safe, expiring in-memory caching system,  built on top of MemoryNoTS cache
package cache
