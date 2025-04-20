package sunvoxgo

import (
	"errors"
)

type FutureError struct {
	errChan chan error
	error
	errorReturned bool
}

func newFutureError() *FutureError {
	return &FutureError{
		errChan: make(chan error, 1),
	}
}

var ErrorFunctionHasNotReturned = errors.New("function has not returned")

func (e *FutureError) Error() error {

	if len(e.errChan) > 0 {
		e.error = <-e.errChan
		e.errorReturned = true
	}

	if e.errorReturned {
		return e.error
	}

	return ErrorFunctionHasNotReturned
}

type VolumeFade struct {
	Channel     *SunvoxChannel
	startVolume float32
	endVolume   float32
	duration    float32

	// start time.Time
	percent float32
}

func NewVolumeFade(start, end, seconds float32, channel *SunvoxChannel) *VolumeFade {

	if start < 0 {
		start, _ = channel.Volume()
	}

	if end < 0 {
		end, _ = channel.Volume()
	}

	return &VolumeFade{
		startVolume: start,
		endVolume:   end,
		duration:    seconds,

		Channel: channel,
	}
}

func (f *VolumeFade) Restart() {
	f.percent = 0
}

func (f *VolumeFade) Update(dt float32) (float32, bool) {

	f.percent += dt / f.duration

	if f.percent > 1 {
		f.percent = 1
	}

	if f.percent < 0 {
		f.percent = 0
	}

	targetVolume := f.startVolume + (f.percent * (f.endVolume - f.startVolume))

	if f.Channel.IsValid() {
		f.Channel.SetVolume(targetVolume)
	}

	if f.percent >= 1 && f.endVolume <= 0 {
		f.Channel.StopAsync()
	}

	return targetVolume, f.percent >= 1
}
