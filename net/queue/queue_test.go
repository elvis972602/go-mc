package queue_test

import (
	"github.com/Tnze/go-mc/net/queue"
	"testing"
)

func BenchmarkLinkedListQueue_Push(b *testing.B) {
	q := queue.NewLinkedQueue[int]()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Push(i)
	}
}

func BenchmarkChannelQueue_Push(b *testing.B) {
	q := queue.NewChannelQueue[int](b.N)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Push(i)
	}
}

func BenchmarkLinkedListQueue_Pull(b *testing.B) {
	q := queue.NewLinkedQueue[int]()
	for i := 0; i < b.N; i++ {
		q.Push(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Pull()
	}
}

func BenchmarkChannelQueue_Pull(b *testing.B) {
	q := queue.NewChannelQueue[int](b.N)
	for i := 0; i < b.N; i++ {
		q.Push(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Pull()
	}
}

func BenchmarkLinkedListQueue_PushPull(b *testing.B) {
	q := queue.NewLinkedQueue[int]()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Push(0)
			q.Pull()
		}
	})
}

func BenchmarkChannelQueue_PushPull(b *testing.B) {
	q := queue.NewChannelQueue[int](b.N)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Push(0)
			q.Pull()
		}
	})
}
