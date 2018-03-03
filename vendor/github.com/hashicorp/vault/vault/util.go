package vault

import (
	"crypto/rand"
	"fmt"
)

// memzero is used to zero out a byte buffer. This specific format is optimized
// by the compiler to use memclr to improve performance. See this code review:
// https://codereview.appspot.com/137880043
//
// Use of memzero is not a guarantee against memory analysis as described in
// the Vault threat model:
// https://www.vaultproject.io/docs/internals/security.html .  Vault does not
// provide guarantees against memory analysis or raw memory dumping by
// operators, however it does minimize this exposure by zeroing out buffers
// that contain secrets as soon as they are no longer used.  Starting with Go
// 1.5, the garbage collector was changed to become a "generational copying
// garbage collector."  This change to the garbage collector makes it
// impossible for Vault to guarantee a buffer with a secret has not been
// copied during a garbage collection.  It is therefore possible that secrets
// may be exist in memory that have not been wiped despite a pending memzero
// call.  Over time any copied data with a secret will be reused and the
// memory overwritten thereby mitigating some of the risk from this threat
// vector.
func memzero(b []byte) {
	if b == nil {
		return
	}
	for i := range b {
		b[i] = 0
	}
}

// randbytes is used to create a buffer of size n filled with random bytes
func randbytes(n int) []byte {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("failed to generate %d random bytes: %v", n, err))
	}
	return buf
}
