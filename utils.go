package sunvoxgo

// type FutureError struct {
// 	errChan chan error
// 	error
// 	errorReturned bool
// }

// func newFutureError() *FutureError {
// 	return &FutureError{
// 		errChan: make(chan error, 1),
// 	}
// }

// var ErrorFunctionHasNotReturned = errors.New("function has not returned")

// func (e *FutureError) Error() error {

// 	if len(e.errChan) > 0 {
// 		e.error = <-e.errChan
// 		e.errorReturned = true
// 	}

// 	if e.errorReturned {
// 		return e.error
// 	}

// 	return ErrorFunctionHasNotReturned
// }

type VolumeFade struct {
	startVolume float32
	endVolume   float32
	duration    float32

	Channel *SunvoxChannel
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
		f.Channel.Stop()
	}

	return targetVolume, f.percent >= 1
}

type ControllerFade struct {
	start    int
	end      int
	duration float32

	Module     *SunvoxModule
	Controller int

	percent float32
}

func NewControllerFade(start, end int, seconds float32, module *SunvoxModule, controller int) *ControllerFade {

	if start < 0 {
		v, _ := module.ControllerValue(controller)
		start = v
	}

	if end < 0 {
		v, _ := module.ControllerValue(controller)
		end = v
	}

	return &ControllerFade{
		start:    start,
		end:      end,
		duration: seconds,

		Module:     module,
		Controller: controller,
	}
}

func (f *ControllerFade) Restart() {
	f.percent = 0
}

func (f *ControllerFade) Update(dt float32) (int, bool) {

	f.percent += dt / f.duration

	if f.percent > 1 {
		f.percent = 1
	}

	if f.percent < 0 {
		f.percent = 0
	}

	targetValue := int(float32(f.start) + (f.percent * (float32(f.end) - float32(f.start))))

	if f.Module.IsValid() {
		f.Module.SetControllerValue(f.Controller, targetValue)
	}

	return targetValue, f.percent >= 1
}

// cache is used to cache some relevant properties (pattern line number, for example) so we don't have to call the sunvox function to get that function unless it's necessary.
type cache map[int]map[string]any

func (c *cache) Get(index int, accessor string) any {

	if !cacheData {
		return nil
	}

	if _, ok := (*c)[index]; !ok {
		(*c)[index] = map[string]any{}
	}
	return (*c)[index][accessor]
}

func (c *cache) Set(index int, accessor string, value any) {

	if !cacheData {
		return
	}

	if _, ok := (*c)[index]; !ok {
		(*c)[index] = map[string]any{}
	}
	(*c)[index][accessor] = value
}

var patternCache = cache{}

// When enabled, some data will be cached when retrieved. This is good for performance, but I'll need to either make it possible to disable / invalidate the cache, or invalidate
// the cache when making some function calls, like modifying pattern size.
var cacheData = true
