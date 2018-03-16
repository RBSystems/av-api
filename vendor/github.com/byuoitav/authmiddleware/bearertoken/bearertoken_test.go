package bearertoken

import "testing"

func BenchmarkGetToken(b *testing.B) {
	for n := 0; n < b.N; n++ {
		GetToken()
	}
}
