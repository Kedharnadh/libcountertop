package noise

import "testing"

func BenchmarkSounds(b *testing.B) {
	for _, name := range Names() {
		b.Run(name, func(b *testing.B) {
			fill := New(name, rate)
			buf := make([]float32, blockSize)

			b.ResetTimer()
			for range b.N {
				fill(buf)
			}

			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*blockSize)*rate/1e7,
				"%-of-a-core")
		})
	}
}
