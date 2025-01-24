package rotatelogs

import (
	"testing"
	"time"
)

func TestLog(t *testing.T) {
	logs, err := NewRotateLogs("d:\\log", "20060102\\gamelog_20060102_150405.log", //) //WithRotateMaxSize(1024),
		WithRotationTime(10*time.Second), WithChannelLen(10), WithMaxAge(50*time.Second))

	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1000000; i++ {
		time.Sleep(200 * time.Millisecond)
		for range 10 {
			logs.Write([]byte("111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"))
		}
	}
	logs.Sync()
	logs.Close()
}

func TestLog2(t *testing.T) {
	logs, err := NewRotateLogs("d:\\log", "20060102\\gamelog_20060102_150405.log", //) //WithRotateMaxSize(1024),
		WithRotationTime(10*time.Second), WithChannelLen(10), WithMaxFileNum(4))

	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1000000; i++ {
		time.Sleep(200 * time.Millisecond)
		for range 10 {
			logs.Write([]byte("111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"))
		}
	}
	logs.Sync()
	logs.Close()
}
