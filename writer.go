package rotatelogs

import (
	"sync"
	"time"
)

type writeToFileFunc func(p []byte) (n int, err error)
type baseWriter struct {
	wf writeToFileFunc
}

type channelWriter struct {
	baseWriter

	syncChannel chan *syncDone
	logChannel  chan []byte
	logWait     sync.WaitGroup
}

type syncDone struct {
	doneChan chan struct{}
}

type fileWriter struct {
	baseWriter
}

func (s *syncDone) waitDone() bool {
	select {
	case <-s.doneChan:
		return true
	case <-time.After(DoneTimeout):
		return false
	}
}

func (s *syncDone) done() {
	s.doneChan <- struct{}{}
}

func (cw *channelWriter) Write(p []byte) (n int, err error) {
	cw.logChannel <- p
	return len(p), nil
}

func (cw *channelWriter) Sync() error {
	sDone := syncDone{doneChan: make(chan struct{}, 1)}
	cw.syncChannel <- &sDone
	sDone.waitDone()

	return nil
}

func (cw *channelWriter) Close() error {
	close(cw.syncChannel)
	cw.logWait.Wait()

	return nil
}

func (cw *fileWriter) Write(p []byte) (n int, err error) {
	return cw.wf(p)
}

func (cw *fileWriter) Sync() error {
	return nil
}

func (cw *fileWriter) Close() error {
	return nil
}

func newChannelWriter(w writeToFileFunc, logChannelLen int) *channelWriter {
	var writer channelWriter
	writer.wf = w
	writer.logChannel = make(chan []byte, logChannelLen)
	writer.syncChannel = make(chan *syncDone, 1)
	writer.logWait.Add(1)
	go func() {
		defer writer.logWait.Done()
		for writer.write() {
		}
	}()
	return &writer
}

func newFileWriter(w writeToFileFunc) *fileWriter {
	var writer fileWriter
	writer.wf = w
	return &writer
}

func (cw *channelWriter) write() bool {
	var byteLog []byte
	select {
	case sDone := <-cw.syncChannel:
		for i := 0; i < len(cw.logChannel); i++ {
			byteLog = <-cw.logChannel
			cw.wf(byteLog)
		}
		
		if sDone == nil {
			return false
		}
		sDone.done()
	case byteLog = <-cw.logChannel:
		cw.wf(byteLog)
	}

	return true
}
