package zigbolt

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
)

// macOS limits POSIX shm names to ~31 chars; keep them short and unique.
func shmName(tag string) string {
	return fmt.Sprintf("/zb-go-%s-%d", tag, os.Getpid()%100000)
}

func TestVersion(t *testing.T) {
	major, minor, patch := Version()
	if major != 0 || minor != 2 || patch != 1 {
		t.Fatalf("expected core version 0.2.1, got %d.%d.%d", major, minor, patch)
	}
}

func TestPublishPollRoundtrip(t *testing.T) {
	name := shmName("rt")

	producer, err := CreateChannel(name, DefaultTermLength)
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	defer producer.Close()

	consumer, err := OpenChannel(name, DefaultTermLength)
	if err != nil {
		t.Fatalf("OpenChannel: %v", err)
	}
	defer consumer.Close()

	want := []byte("hello-from-go")
	if err := producer.Publish(want, 7); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	var got []byte
	var gotType int32
	for i := 0; i < 100 && got == nil; i++ {
		consumer.Poll(func(data []byte, msgTypeId int32) {
			got = append([]byte(nil), data...)
			gotType = msgTypeId
		}, 16)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if gotType != 7 {
		t.Fatalf("got msgTypeId %d, want 7", gotType)
	}
}

// Concurrent polls on independent channels must not serialize or deadlock
// (the old implementation used one package-global handler + mutex).
func TestConcurrentPollIndependentChannels(t *testing.T) {
	const perChannel = 50
	var wg sync.WaitGroup

	for c := 0; c < 2; c++ {
		name := shmName(fmt.Sprintf("cc%d", c))

		producer, err := CreateChannel(name, DefaultTermLength)
		if err != nil {
			t.Fatalf("CreateChannel(%s): %v", name, err)
		}
		defer producer.Close()

		consumer, err := OpenChannel(name, DefaultTermLength)
		if err != nil {
			t.Fatalf("OpenChannel(%s): %v", name, err)
		}
		defer consumer.Close()

		for i := 0; i < perChannel; i++ {
			if err := producer.Publish([]byte(fmt.Sprintf("msg-%d", i)), int32(c)); err != nil {
				t.Fatalf("Publish: %v", err)
			}
		}

		wg.Add(1)
		go func(ch *IpcChannel, wantType int32) {
			defer wg.Done()
			received := 0
			for spin := 0; spin < 10000 && received < perChannel; spin++ {
				ch.Poll(func(data []byte, msgTypeId int32) {
					if msgTypeId != wantType {
						t.Errorf("channel %d received foreign msgTypeId %d", wantType, msgTypeId)
					}
					received++
				}, 16)
			}
			if received != perChannel {
				t.Errorf("channel %d received %d/%d messages", wantType, received, perChannel)
			}
		}(consumer, int32(c))
	}

	wg.Wait()
}

// A reentrant Poll (polling channel B from inside channel A's handler) must
// not deadlock and must keep delivering A's remaining fragments.
func TestNestedPoll(t *testing.T) {
	nameA := shmName("na")
	nameB := shmName("nb")

	pubA, err := CreateChannel(nameA, DefaultTermLength)
	if err != nil {
		t.Fatalf("CreateChannel(A): %v", err)
	}
	defer pubA.Close()
	subA, err := OpenChannel(nameA, DefaultTermLength)
	if err != nil {
		t.Fatalf("OpenChannel(A): %v", err)
	}
	defer subA.Close()

	pubB, err := CreateChannel(nameB, DefaultTermLength)
	if err != nil {
		t.Fatalf("CreateChannel(B): %v", err)
	}
	defer pubB.Close()
	subB, err := OpenChannel(nameB, DefaultTermLength)
	if err != nil {
		t.Fatalf("OpenChannel(B): %v", err)
	}
	defer subB.Close()

	if err := pubA.Publish([]byte("outer-1"), 1); err != nil {
		t.Fatal(err)
	}
	if err := pubA.Publish([]byte("outer-2"), 1); err != nil {
		t.Fatal(err)
	}
	if err := pubB.Publish([]byte("inner"), 2); err != nil {
		t.Fatal(err)
	}

	var outer, inner []string
	for spin := 0; spin < 100 && len(outer) < 2; spin++ {
		subA.Poll(func(data []byte, _ int32) {
			outer = append(outer, string(data))
			subB.Poll(func(d []byte, _ int32) {
				inner = append(inner, string(d))
			}, 16)
		}, 16)
	}

	if len(outer) != 2 || outer[0] != "outer-1" || outer[1] != "outer-2" {
		t.Fatalf("outer messages = %v, want [outer-1 outer-2]", outer)
	}
	if len(inner) != 1 || inner[0] != "inner" {
		t.Fatalf("inner messages = %v, want [inner]", inner)
	}
}

func TestPublishOnClosedChannel(t *testing.T) {
	name := shmName("cl")
	ch, err := CreateChannel(name, DefaultTermLength)
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	ch.Close()
	if err := ch.Publish([]byte("too late"), 1); err == nil {
		t.Fatal("expected error publishing on a closed channel")
	}
}
