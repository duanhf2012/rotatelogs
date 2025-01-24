package rotatelogs

import (
	"fmt"
	"time"
)

type IWriter interface {
	Write(p []byte) (n int, err error)
	Sync() error
	Close() error
}

type Option interface {
	Configure(*RotateLogs) error
}

type OptionFn func(*RotateLogs) error

func (o OptionFn) Configure(rl *RotateLogs) error {
	return o(rl)
}

const (
	MinRotationTime = 5 * time.Minute
	MinAge          = 1 * time.Hour
	DefaultFileExt  = ".log"
	DoneTimeout     = 15 * time.Second
)

// WithChannelLen 日志Channel长度,不设置则直接写入到文件
func WithChannelLen(channelLen int) Option {
	return OptionFn(func(rl *RotateLogs) error {
		if channelLen <= 0 {
			return fmt.Errorf("channel length must be greater than zero")
		}
		rl.logChannelLen = channelLen
		return nil
	})
}

// WithFileExt 指定文件扩展名,默认.log
func WithFileExt(fileExt string) Option {
	return OptionFn(func(rl *RotateLogs) error {
		rl.fileExt = fileExt
		return nil
	})
}

// WithRotationTime 分割间隔时间，超过则创建新文件
func WithRotationTime(t time.Duration) Option {
	return OptionFn(func(rl *RotateLogs) error {
		if t < MinRotationTime {
			return fmt.Errorf("rotationTime is too small")
		}

		rl.rotationTime = t
		return nil
	})
}

// WithRotateMaxSize 文件最大Byte，超过则创建新文件
func WithRotateMaxSize(maxSize int64) Option {
	return OptionFn(func(rl *RotateLogs) error {
		rl.rotateLogFileMaxSize = maxSize
		return nil
	})
}

// WithMaxAge 最大保留时长,超过则删除文件
func WithMaxAge(maxAge time.Duration) Option {
	return OptionFn(func(rl *RotateLogs) error {
		if maxAge < MinAge {
			return fmt.Errorf("maxAge is too small")
		}

		rl.maxAge = maxAge
		return nil
	})
}

// WithMaxFileNum 最大保留文件数量,超过则删除文件
func WithMaxFileNum(maxNum int) Option {
	return OptionFn(func(rl *RotateLogs) error {
		rl.maxNum = maxNum
		return nil
	})
}
