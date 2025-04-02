package sunvoxgo

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	InitFlagNoDebugOutput = 1 << iota
	InitFlagUserAudioCallback
	InitFlagAudioInt16
	InitFlagAudioFloat32
	InitFlagOneThread
)

const (
	ModuleFlagExists = 1 << iota
	ModuleFlagGenerator
	ModuleFlagEffect
	ModuleFlagMute
	ModuleFlagSolo
	ModuleFlagBypass
)

const (
	NoteCommandNoteOff     = 128 + iota
	NoteCommandAllNotesOff // send "note off" to all modules;
	NoteCommandCleanSynths // stop all modules - clear their internal buffers and put them into standby mode;
	NoteCommandStop
	NoteCommandPlay
	NoteCommandSetPitch    // set the pitch specified in column XXYY, where 0x0000 - highest possible pitch, 0x7800 - lowest pitch (note C0); one semitone = 0x100;
	NoteCommandCleanModule // stop the module - clear its internal buffers and put it into standby mode.
)

type InitConfig struct {
	SampleRate  int
	Flags       uint32
	ExtraString string
}

func NewInitConfig() *InitConfig {
	return &InitConfig{}
}

// The preferred buffer size to initialize the engine with; note that the engine may not be able to initialize with this exact buffer size.
func (i *InitConfig) Buffer(bufferSize int) *InitConfig {
	if len(i.ExtraString) > 0 {
		i.ExtraString += "|"
	}
	i.ExtraString += "buffer=" + strconv.Itoa(bufferSize)
	return i
}

// The audio driver to be used; can be something like alsa on Linux, maybe dsound on Windows?
func (i *InitConfig) Driver(driverName string) *InitConfig {
	if len(i.ExtraString) > 0 {
		i.ExtraString += "|"
	}
	i.ExtraString += "audiodriver=" + driverName
	return i
}

// The device to be used; something like "hw:0,0" on Linux for the first audio device
func (i *InitConfig) Device(deviceName string) *InitConfig {
	if len(i.ExtraString) > 0 {
		i.ExtraString += "|"
	}
	i.ExtraString += "audiodevice=" + deviceName
	return i
}

// TODO: Maybe replace all ints with int32s for functions below?

/*
Init initializes the Sunvox engine.

Arguments:

config - string with additional configuration in the following format: "option_name=value|option_name=value"; or NULL for auto config;

example: "buffer=1024|audiodriver=alsa|audiodevice=hw:0,0";

sample_rate - desired sample rate (Hz); min - 44100; the actual rate may be different, if SV_INIT_FLAG_OFFLINE is not set;

channels - only 2 supported now;

flags - set of flags SV_INIT_FLAG_*;

Returns the version or an error string otherwise
*/
var initEngine func(config string, sampleRate int, flags uint32) int32
var deinitEngine func() int32

// Opens a project slot; any number from 0 to 15 (that hasn't been used before).
var openSlot func(projectNum int) int32
var closeSlot func(projectNum int) int32

// Returns the sample rate or an error code if negative.
var getSampleRate func() int32

// Loads a file from a given path. Success is 0, negative is an error code.
var loadFile func(slotNum int, fp string) int32

// Loads a Sunvox file from memory. Success is 0, negative is an error code.
var loadFileFromMemory func(slotNum int, data []byte, dataSize uint32) int32

// Slot functions

var setSlotVolume func(slotNum int, volume int) int32
var getCurrentLine func(slotNum int) int32
var getCurrentSignalLevel func(slotNum, channel int) uint8 // Ranges from 0 - 255
var getSongName func(slotNum int) string
var setSongName func(slotNum int, name string) int32
var getSongBPM func(slotNum int) int32
var getSongTPL func(slotNum int) int32
var getLengthFrames func(slotNum int) uint32
var getLengthLines func(slotNum int) uint32

var play func(slotNum int) int32
var playFromBeginning func(slotNum int) int32
var pause func(slotNum int) int32
var resume func(slotNum int) int32
var stop func(slotNum int) int32
var rewind func(slotNum int, lineNum int) int32

var setAutostop func(slotNum int, autoStop int) int32
var getAutostop func(slotNum int) int32
var endOfSong func(slotNum int) int32
var findPattern func(slotNum int, patternName string) int32
var lock func(slotNum int) int32
var unlock func(slotNum int) int32

// Pattern functions

var getNumberOfPatternSlots func(slotNum int) int32          // Number of patterns in the slot (project)
var getPatternX func(slotNum, patternNum int) int32          // Line number of the pattern
var getPatternY func(slotNum, patternNum int) int32          // Y coordinate of the pattern
var getPatternTrackCount func(slotNum, patternNum int) int32 // Number of tracks in the pattern
var getPatternLineCount func(slotNum, patternNum int) int32  // Number of lines in the pattern
var getPatternName func(slotNum, patternNum int) string
var setPatternMute func(slotNum, patternNum, muted int32) int32
var getPatternData func(slotNum, patternNum int) *SunvoxPatternNoteData

// Module functions

var getNumberOfModuleSlots func(slotNum int) int32 // Number of module slots (not the number of actual modules)
var findModule func(slotNum int, name string) int32
var getModuleFlags func(slotNum, moduleNum int) int32
var getModuleName func(slotNum, moduleNum int) string
var getModuleCtlName func(slotNum, moduleNum, ctrlNum int) string
var getNumberOfModuleCtls func(slotNum, moduleNum int) int32
var connectModule func(slotNum, sourceMod, destMod int) int32
var disconnectModule func(slotNum, sourceMod, destMod int) int32

// scaled:
// 0 - real value (0,1,2...) as it is stored inside the controller; but the value displayed in the program interface may be different - you can use scaled=2 to get the displayed value;
// 1 - scaled (0x0000...0x8000) if the controller type = 0, or the real value if the controller type = 1; this value can be used in the pattern column XXYY;
// 2 - final value displayed in the program interface - in most cases it is identical to the real value (scaled=0), and sometimes it has an additional offset.
var getModuleCtlValue func(slotNum, moduleNum, ctlNum, scaled int) int32
var getModuleCtlMin func(slotNum, moduleNum, ctlNum, scaled int) int32
var getModuleCtlMax func(slotNum, moduleNum, ctlNum, scaled int) int32

// TODO: Implement the below functions
var getModuleFinetuneRelativeNote func(slotNum, moduleNum int) uint32
var setModuleFinetune func(slotNum, moduleNum int, finetune int) int32
var setModuleRelativeNote func(slotNum, moduleNum int, finetune int) int32
var getModuleScope func(slotNum, moduleNum, channelNum int, destinationBuffer uintptr, sampleCount uint32) int32 // Gets the currently playing audio through a module
//

var getTicks func() uint32
var getTicksPerSecond func() uint32

var setModuleCtlValue func(slotNum, moduleNum, ctlNum, val, scaled int) int32

var setEventT func(slotNum, set int, timestamp uint32) int32
var sendEvent func(slotNum, trackNum, note, vel, module, ctl, ctlVal int) int32

type SunvoxEngine struct {
	Initialized   bool
	MajorVersion  int
	MinorVersion  int
	MinorVersion2 int

	// channelIndex int
	Channels map[int]*SunvoxChannel // A map of channel indices to SunvoxChannels, on which one can playback audio.
}

var engine = &SunvoxEngine{
	Channels: map[int]*SunvoxChannel{},
}

// Engine returns the running Sunvox instance; each process can run only one.
func Engine() *SunvoxEngine {
	return engine
}

// Init initializes the SunvoxEngine using the passed shared library filepath.
// The path is, by default, relative to the executable, in the current working directory.
// config is an InitConfig object that controls how the engine is initialized.
// The function automatically loads libraries using the OS and architecture hierarchy from the original
// DLL / library download.
func (e *SunvoxEngine) Init(libraryPath string, config *InitConfig) error {

	// If already initialized, return nothing; it can only be running once per process
	if e.Initialized {
		return nil
	}

	lib, err := loadLibrary(libraryPath)
	if err != nil {
		return err
	}

	purego.RegisterLibFunc(&initEngine, lib, "sv_init")
	purego.RegisterLibFunc(&deinitEngine, lib, "sv_deinit")
	purego.RegisterLibFunc(&openSlot, lib, "sv_open_slot")
	purego.RegisterLibFunc(&closeSlot, lib, "sv_close_slot")
	purego.RegisterLibFunc(&getSampleRate, lib, "sv_get_sample_rate")
	purego.RegisterLibFunc(&loadFile, lib, "sv_load")
	purego.RegisterLibFunc(&loadFileFromMemory, lib, "sv_load_from_memory")
	purego.RegisterLibFunc(&setSlotVolume, lib, "sv_volume")
	purego.RegisterLibFunc(&getCurrentLine, lib, "sv_get_current_line")
	purego.RegisterLibFunc(&getCurrentSignalLevel, lib, "sv_get_current_signal_level")
	purego.RegisterLibFunc(&getSongName, lib, "sv_get_song_name")
	purego.RegisterLibFunc(&setSongName, lib, "sv_set_song_name")
	purego.RegisterLibFunc(&getSongBPM, lib, "sv_get_song_bpm")
	purego.RegisterLibFunc(&getSongTPL, lib, "sv_get_song_tpl")

	purego.RegisterLibFunc(&rewind, lib, "sv_rewind")
	purego.RegisterLibFunc(&play, lib, "sv_play")
	purego.RegisterLibFunc(&playFromBeginning, lib, "sv_play_from_beginning")
	purego.RegisterLibFunc(&pause, lib, "sv_pause")
	purego.RegisterLibFunc(&resume, lib, "sv_resume")
	purego.RegisterLibFunc(&stop, lib, "sv_stop")
	purego.RegisterLibFunc(&getAutostop, lib, "sv_get_autostop")
	purego.RegisterLibFunc(&setAutostop, lib, "sv_set_autostop")
	purego.RegisterLibFunc(&endOfSong, lib, "sv_end_of_song")
	purego.RegisterLibFunc(&findPattern, lib, "sv_find_pattern")
	purego.RegisterLibFunc(&lock, lib, "sv_lock_slot")
	purego.RegisterLibFunc(&unlock, lib, "sv_unlock_slot")
	purego.RegisterLibFunc(&getLengthFrames, lib, "sv_get_song_length_frames")
	purego.RegisterLibFunc(&getLengthLines, lib, "sv_get_song_length_lines")
	purego.RegisterLibFunc(&setEventT, lib, "sv_set_event_t")
	purego.RegisterLibFunc(&sendEvent, lib, "sv_send_event")
	purego.RegisterLibFunc(&getPatternData, lib, "sv_get_pattern_data")

	purego.RegisterLibFunc(&getNumberOfPatternSlots, lib, "sv_get_number_of_patterns")
	purego.RegisterLibFunc(&getPatternX, lib, "sv_get_pattern_x")
	purego.RegisterLibFunc(&getPatternY, lib, "sv_get_pattern_y")
	purego.RegisterLibFunc(&getPatternTrackCount, lib, "sv_get_pattern_tracks")
	purego.RegisterLibFunc(&getPatternLineCount, lib, "sv_get_pattern_lines")
	purego.RegisterLibFunc(&getPatternName, lib, "sv_get_pattern_name")
	purego.RegisterLibFunc(&setPatternMute, lib, "sv_pattern_mute")

	purego.RegisterLibFunc(&getNumberOfModuleSlots, lib, "sv_get_number_of_modules")
	purego.RegisterLibFunc(&connectModule, lib, "sv_connect_module")
	purego.RegisterLibFunc(&disconnectModule, lib, "sv_disconnect_module")
	purego.RegisterLibFunc(&findModule, lib, "sv_find_module")
	purego.RegisterLibFunc(&getModuleFlags, lib, "sv_get_module_flags")
	purego.RegisterLibFunc(&getModuleName, lib, "sv_get_module_name")
	purego.RegisterLibFunc(&getModuleCtlName, lib, "sv_get_module_ctl_name")
	purego.RegisterLibFunc(&getNumberOfModuleCtls, lib, "sv_get_number_of_module_ctls")
	purego.RegisterLibFunc(&getModuleCtlValue, lib, "sv_get_module_ctl_value")
	purego.RegisterLibFunc(&setModuleCtlValue, lib, "sv_set_module_ctl_value")
	purego.RegisterLibFunc(&getTicks, lib, "sv_get_ticks")
	purego.RegisterLibFunc(&getTicksPerSecond, lib, "sv_get_ticks_per_second")
	purego.RegisterLibFunc(&getModuleFinetuneRelativeNote, lib, "sv_get_module_finetune")
	purego.RegisterLibFunc(&setModuleFinetune, lib, "sv_set_module_finetune")
	purego.RegisterLibFunc(&setModuleRelativeNote, lib, "sv_set_module_relnote")

	extras := ""
	sampleRate := 0
	flags := uint32(0)

	if config != nil {
		extras = config.ExtraString
		sampleRate = config.SampleRate
		flags = config.Flags
	}

	if sampleRate <= 0 {
		sampleRate = 44100
	}

	ver := initEngine(extras, sampleRate, flags)
	if ver < 0 {
		e.Initialized = false
		return errors.New("error in initializing:" + strconv.Itoa(int(ver)))
	}

	// ver = 67846 // 0x010906 for v1.9.6.

	// major := ver >> 16
	// minor1 := ver &^ (major << 16) >> 8
	// minor2 := ver - (major << 16) - (minor1 << 8)

	major := (ver >> 16) & 255
	minor1 := (ver >> 8) & 255
	minor2 := (ver) & 255

	e.MajorVersion = int(major)
	e.MinorVersion = int(minor1)
	e.MinorVersion2 = int(minor2)

	e.Initialized = true

	return nil

}

// InitFromDirectory loads the SunvoxEngine using shared libraries in the base directory path given.
// The path is, by default, relative to the executable, in the current working directory.
// config is an InitConfig object that controls how the engine is initialized.
// The function automatically loads libraries using the OS and architecture folder hierarchy from the original
// DLL / library download.
func (e *SunvoxEngine) InitFromDirectory(libraryBaseDirectoryPath string, config *InitConfig) error {

	osFolder := ""

	switch runtime.GOOS {
	case "darwin":
		osFolder = "macos"
	case "linux":
		osFolder = "linux"
	case "windows":
		osFolder = "windows"
	}

	archFolder := ""

	switch runtime.GOARCH {
	case "386":
		archFolder = "lib_x86/"
	case "amd64":
		archFolder = "lib_x86_64/"
	case "arm":
		archFolder = "lib_arm/"
	case "arm64":
		archFolder = "lib_arm64/"
	}

	filename := ""

	switch runtime.GOOS {
	case "linux":
		filename = "sunvox.so"
	case "darwin":
		filename = "sunvox.dylib"
	case "windows":
		filename = "sunvox.dll"

	}

	dllPath := filepath.Join(libraryBaseDirectoryPath, osFolder, archFolder, filename)

	return e.Init(dllPath, config)
}

// Deinit deinitializes the Sunvox Engine.
// If for whatever reason that cannot be done, Deinit returns an error.
func (e *SunvoxEngine) Deinit() error {
	res := deinitEngine()

	if res != 0 {
		return errors.New(fmt.Sprintf("error deinitializing sunvox engine; error code %d", res))
	}

	return nil
}

// CreateChannel creates a SunvoxChannel and assigns it a custom ID to identify it.
// You may choose to assign unique IDs to each Channel.
// Note that a SunvoxEngine can only create 16 channels maximum.
func (e *SunvoxEngine) CreateChannel(id any) (*SunvoxChannel, error) {

	if !e.Initialized {
		return nil, errors.New("error: engine has not been initialized")
	}

	available := -1

	// 16 channels max
	for i := 0; i < 16; i++ {
		if _, exists := e.Channels[i]; !exists {
			available = i
			break
		}
	}

	if available < 0 {
		return nil, errors.New("error: a maximum of 16 channels have been created already; close an existing channel")
	}

	res := openSlot(available)

	if res != 0 {
		return nil, errors.New("error: error creating SunvoxChannel; error code" + strconv.Itoa(int(res)))
	}

	e.Channels[available] = &SunvoxChannel{
		ID:    id,
		Index: available,
	}

	return e.Channels[available], nil
}

// ChannelByID returns the first channel with the given ID. If onlyNotInUse is set to true, then only channels that are not playing
// any audio could be used.
// If no channel is found, ChannelByID returns nil.
func (e *SunvoxEngine) ChannelByID(id any, onlyNotInUse bool) *SunvoxChannel {
	for i := 0; i < 16; i++ {
		c, ok := e.Channels[i]
		if ok && c.ID == id {
			if onlyNotInUse && c.IsPlaying() {
				continue
			}
			return c
		}
	}
	return nil
}

// SampleRate returns the sample rate of the engine.
func (e *SunvoxEngine) SampleRate() (int, error) {
	sampleRate := getSampleRate()
	if sampleRate < 0 {
		return 0, errors.New(fmt.Sprintf("error retrieving engine sample rate: %d", sampleRate))
	}
	return int(sampleRate), nil
}

// Ticks returns the system ticks, used for setting the event timestamp.
func (s *SunvoxEngine) Ticks() uint32 {
	return getTicks()
}

// TicksPerSecond returns the system ticks, used for setting the event timestamp.
func (s *SunvoxEngine) TicksPerSecond() uint32 {
	return getTicksPerSecond()
}

// SunvoxChannel represents a channel of audio playback.
// Each channel can play, seek / rewind, load a .sunvox file, etc.
type SunvoxChannel struct {
	LoadedData bool
	Index      int
	ID         any
}

// LoadFileFromPath simply loads a file from the given filepath.
func (s *SunvoxChannel) LoadFileFromPath(filepath string) error {
	loaded := loadFile(s.Index, filepath)
	if loaded != 0 {
		s.LoadedData = false
		return errors.New(fmt.Sprintf("error loading sunvox file: %d", loaded))
	}
	s.LoadedData = true
	return nil
}

// LoadFile loads a slice of bytes obtained from reading a .sunvox file.
func (s *SunvoxChannel) LoadFileFromBytes(data []byte) error {
	loaded := loadFileFromMemory(s.Index, data, uint32(len(data)))
	if loaded != 0 {
		s.LoadedData = false
		return errors.New(fmt.Sprintf("error loading sunvox data: %d", loaded))
	}
	s.LoadedData = true
	return nil
}

// LoadFileFromFS loads a file of the provided filename from the given file system.
func (s *SunvoxChannel) LoadFileFromFS(fileSys fs.FS, filename string) error {

	data, err := fs.ReadFile(fileSys, filename)

	if err != nil {
		return err
	}

	return s.LoadFileFromBytes(data)

}

// ProjectName returns the name for the project loaded in the channel. If there is an issue getting the song name,
// the function will just return an empty string.
func (s *SunvoxChannel) ProjectName() string {
	return getSongName(s.Index)
}

// SetProjectName sets the name for the project loaded in the channel.
// If there is an issue getting the song name, the function will return an error.
func (s *SunvoxChannel) SetProjectName(name string) error {
	res := setSongName(s.Index, name)

	if res != 0 {
		return errors.New(fmt.Sprintf("error setting the project name for the project loaded in channel index %d; error code %d", s.Index, res))
	}

	return nil

}

// SetVolume sets the volume of the channel. Valid values range from 0 to 1. The fidelity is in 1/256 steps.
func (s *SunvoxChannel) SetVolume(volume float32) {
	if volume > 1 {
		volume = 1
	}
	if volume < 0 {
		volume = 0
	}
	setSlotVolume(s.Index, int(volume*256))
}

// Seek seeks the given line number for playback.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) Seek(lineNum int) error {

	res := rewind(s.Index, lineNum)

	if res != 0 {
		return errors.New(fmt.Sprintf("error seeking SunvoxChannel index %d; error code %d", s.Index, res))
	}

	return nil
}

// Play plays the song contained within the SunvoxChannel from wherever the playhead currently is.
func (s *SunvoxChannel) Play() error {
	res := play(s.Index)
	if res < 0 {
		return errors.New(fmt.Sprintf("error playing SunvoxChannel index %d; error code %d", s.Index, res))
	}
	return nil
}

// CurrentSignalLevel returns the current signal level of the engine, ranging from 0 to 1 for the left
// and right audio channels.
func (e *SunvoxChannel) CurrentSignalLevel() (float32, float32) {
	left := getCurrentSignalLevel(e.Index, 0)
	right := getCurrentSignalLevel(e.Index, 1)
	return float32(left) / 255, float32(right) / 255
}

// CurrentLine returns the current line of playback for the Sunvox project playing through the Channel.
func (e *SunvoxChannel) CurrentLine() int {
	return int(getCurrentLine(e.Index))
}

// LengthInFrames returns the length of the project in frames.
func (e *SunvoxChannel) LengthInFrames() int {
	return int(getLengthFrames(e.Index))
}

// LengthInFrames returns the length of the project in frames.
func (e *SunvoxChannel) LengthInLines() int {
	return int(getLengthLines(e.Index))
}

// Length returns the length of the project as a time.Duration.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (e *SunvoxChannel) Length() (time.Duration, error) {
	s, err := engine.SampleRate()
	if err != nil {
		return 0, errors.New("error calculating the length of the sunvox project in time due to a sample rate retrieval issue")
	}
	l := float32(e.LengthInFrames()) / float32(s)
	return time.Duration(l) * time.Second, nil
}

// PlayFromBeginning plays the song contained within the SunvoxChannel from the beginning.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) PlayFromBeginning() error {
	res := playFromBeginning(s.Index)
	if res < 0 {
		return errors.New(fmt.Sprintf("error playing SunvoxChannel index %d; error code %d", s.Index, res))
	}
	return nil
}

// Pause pauses audio playback through the SunvoxChannel.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) Pause() error {
	res := pause(s.Index)
	if res < 0 {
		return errors.New(fmt.Sprintf("error playing SunvoxChannel index %d; error code %d", s.Index, res))
	}
	return nil
}

// Resume resumes audio playback that was previously paused through the SunvoxChannel.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) Resume() error {
	res := resume(s.Index)
	if res < 0 {
		return errors.New(fmt.Sprintf("error playing SunvoxChannel index %d; error code %d", s.Index, res))
	}
	return nil
}

// Stop stops audio playback that is currently playing back through the SunvoxChannel.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) Stop() error {
	res := stop(s.Index)
	if res < 0 {
		return errors.New(fmt.Sprintf("error playing SunvoxChannel index %d; error code %d", s.Index, res))
	}
	return nil
}

// Looping returns if the SunvoxChannel is set to loop audio playback.
func (s *SunvoxChannel) Looping() bool {
	return getAutostop(s.Index) == 0
}

// SetLooping sets the SunvoxChannel to loop.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) SetLooping(loop bool) error {

	st := 1
	if loop {
		st = 0
	}

	res := setAutostop(s.Index, st)

	if res != 0 {
		return errors.New(fmt.Sprintf("error setting loop on SunvoxChannel index %d; error code %d", s.Index, res))
	}

	return nil
}

// Returns if the channel is currently playing back audio.
func (s *SunvoxChannel) IsPlaying() bool {
	return endOfSong(s.Index) == 0
}

// PatternCount returns the number of patterns in the channel, and an error if it was impossible to determine.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) PatternCount() (int, error) {

	// number of pattern slots, not number of patterns
	slotCount := getNumberOfPatternSlots(s.Index)

	if slotCount < 0 {
		return 0, errors.New(fmt.Sprintf("error getting pattern count for SunvoxChannel index %d; error code %d", s.Index, slotCount))
	}

	patternCount := 0

	for i := 0; i < int(slotCount); i++ {
		if getPatternLineCount(s.Index, i) > 0 {
			patternCount++
		}
	}

	return patternCount, nil

}

// PatternByName returns a Pattern with the specified name; if it doesn't exist, PatternByName will return nil.
func (s *SunvoxChannel) PatternByName(name string) *SunvoxPattern {
	patternID := findPattern(s.Index, name)
	if patternID >= 0 {
		return &SunvoxPattern{Channel: s, Index: int(patternID)}
	}
	return nil
}

// PatternByIndex returnss the specified numeric patternIndex argument.
// If patternIndex is outside of the range of patterns in the song, PatternByIndex will return nil.
func (s *SunvoxChannel) PatternByIndex(patternIndex int) *SunvoxPattern {

	patternCount, err := s.PatternCount()
	if err != nil {
		return nil
	}

	if patternIndex < 0 || patternIndex > patternCount {
		return nil
	}

	if getPatternLineCount(s.Index, patternIndex) <= 0 {
		return nil
	}

	return &SunvoxPattern{
		Channel: s,
		Index:   patternIndex,
	}

}

// ForEachPattern iterates through all patterns contained in the SunvoxChannel and executes the provided forEach
// function on each one. If the function returns false, the function stops iterating through the pattern set.
func (s *SunvoxChannel) ForEachPattern(forEach func(pattern *SunvoxPattern) bool) {
	count, err := s.PatternCount()
	if err != nil {
		return
	}
	for i := 0; i < count; i++ {
		p := &SunvoxPattern{
			Channel: s,
			Index:   i,
		}
		if !forEach(p) {
			break
		}
	}
}

// Locks the channel for simultaneous read/write from different threads / goroutines for the same channel.
// Some functions marked as "USE LOCK/UNLOCK" can't work without locking at all.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) Lock() error {
	res := lock(s.Index)
	if res != 0 {
		return errors.New(fmt.Sprintf("error locking channel %d", s.Index))
	}
	return nil
}

// Unlocks the channel for simultaneous read/write from different threads / goroutines for the same channel.
// Some functions marked as "USE LOCK/UNLOCK" can't work without locking at all.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) Unlock() error {
	res := unlock(s.Index)
	if res != 0 {
		return errors.New(fmt.Sprintf("error unlocking channel %d", s.Index))
	}
	return nil
}

// Close closes the channel and removes it from playback.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) Close() error {
	res := closeSlot(s.Index)
	if res != 0 {
		return errors.New(fmt.Sprintf("error closing channel %d", s.Index))
	}
	delete(engine.Channels, s.Index)
	return nil
}

// SetEventTimestamps sets the timestamp for sending events. The final timestamps is when the event
// can be heard from the speakers. If setTimestamp is false, then the event will be automatically set to
// the current time. Otherwise, the resulting time is the timestamp + sound latency * 2 (with timestamp
// being retrieved from GetTicks()).
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s *SunvoxChannel) SetEventTimestamp(setTimestamp bool, timestamp uint32) error {
	set := 0
	if setTimestamp {
		set = 1
	}
	res := setEventT(s.Index, set, timestamp)
	if res < 0 {
		return errors.New(fmt.Sprintf("error sending event to channel %d", s.Index))
	}
	return nil
}

func (s *SunvoxChannel) SendEvent(trackNum, note, velocity, module, ctrlEffect, parameterValue int) error {
	res := sendEvent(s.Index, trackNum, note, velocity, module, ctrlEffect, parameterValue)
	if res < 0 {
		return errors.New(fmt.Sprintf("error sending event to channel %d", s.Index))
	}
	return nil
}

// SunvoxPattern represents a pattern in a Sunvox song.
type SunvoxPattern struct {
	Channel *SunvoxChannel
	Index   int
}

// LineX returns the Line number (x-coordinate) of the pattern in Sunvox.
func (p *SunvoxPattern) LineX() int {
	return int(getPatternX(p.Channel.Index, p.Index))
}

// Y returns the Y coordinate of the pattern in Sunvox.
func (p *SunvoxPattern) Y() int {
	return int(getPatternY(p.Channel.Index, p.Index))
}

// Name returns the name of the given Pattern.
func (p *SunvoxPattern) Name() string {
	return getPatternName(p.Channel.Index, p.Index)
}

// SetMute sets the pattern to be muted (or not). It returns whether the channel was previously muted or not,
// and an error if muting could not be done for whatever reason.
// If the SunvoxPattern is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (p *SunvoxPattern) SetMute(muted bool) (bool, error) {
	m := 0
	if muted {
		m = 1
	}

	if err := p.Channel.Lock(); err != nil {
		return false, err
	}

	res := setPatternMute(int32(p.Channel.Index), int32(p.Index), int32(m))

	if err := p.Channel.Unlock(); err != nil {
		return false, err
	}

	if res < 0 {
		return false, errors.New(fmt.Sprintf("error muting pattern %d in channel %d; error code %d", p.Index, p.Channel.Index, res))
	}

	if res == 1 {
		return true, nil
	}
	return false, nil
}

// LineCount returns the number of lines in the pattern.
// If the SunvoxPattern is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (p *SunvoxPattern) LineCount() (int, error) {

	res := getPatternLineCount(p.Channel.Index, p.Index)
	if res < 0 {
		return int(res), errors.New(fmt.Sprintf("error getting pattern line count from channel %d and pattern %d", p.Channel.Index, p.Index))
	}
	return int(res), nil
}

// TrackCount returns the number of tracks in the pattern.
// If the SunvoxPattern is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (p *SunvoxPattern) TrackCount() (int, error) {

	res := getPatternTrackCount(p.Channel.Index, p.Index)
	if res < 0 {
		return int(res), errors.New(fmt.Sprintf("error getting pattern track count from channel %d and pattern %d", p.Channel.Index, p.Index))
	}
	return int(res), nil
}

// Data returns the data from the pattern for reading and modification.
// If the SunvoxPattern is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (p *SunvoxPattern) Data() (*SunvoxPatternData, error) {
	addr := getPatternData(p.Channel.Index, p.Index)

	lineCount, err := p.LineCount()
	if err != nil {
		return nil, err
	}

	trackCount, err := p.TrackCount()
	if err != nil {
		return nil, err
	}

	res := &SunvoxPatternData{
		trackCount: trackCount,
		Data:       (unsafe.Slice(addr, lineCount*trackCount)),
	}

	return res, nil
}

// ModuleByName returns the module by the specified moduleName.
// If a module with the specified name cannot be found, the function returns nil.
func (c *SunvoxChannel) ModuleByName(moduleName string) *SunvoxModule {
	id := findModule(c.Index, moduleName)
	if id < 0 {
		return nil
	}
	return &SunvoxModule{
		Channel: c,
		Index:   int(id),
	}
}

// ForEachModule iterates through all modules in the project to execute a given function (forEach()) for
// each module. If the function returns false, the function will stop iteration.
func (s *SunvoxChannel) ForEachModule(forEach func(module *SunvoxModule) bool) error {
	modCount := 0
	maxModCount, err := s.ModuleCount()
	if err != nil {
		return err
	}
	for i := 0; i < 1_000_000; i++ {
		mod := s.ModuleByIndex(i)
		if mod == nil {
			continue
		}
		if !forEach(mod) {
			break
		}
		modCount++

		if modCount >= maxModCount {
			break
		}
	}
	return nil
}

// ModuleCount returns the number of modules in the project.
// If the SunvoxChannel is unable to execute the function for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (c *SunvoxChannel) ModuleCount() (int, error) {

	// number of module slots, not number of modules, as modules take up slots when created and deleted
	slotCount := getNumberOfModuleSlots(c.Index)

	if slotCount < 0 {
		return 0, errors.New(fmt.Sprintf("error getting module count for SunvoxChannel index %d; error code %d", c.Index, slotCount))
	}

	moduleCount := 0

	for i := 0; i < int(slotCount); i++ {
		if flags := getModuleFlags(c.Index, i); flags >= 0 && (flags&ModuleFlagExists > 0) {
			moduleCount++
		}
	}

	return moduleCount, nil

}

// ModuleByIndex returns a module by the given index - this can be hexadecimal (e.g. 0x1a) to match
// the module's index in Sunvox.
func (c *SunvoxChannel) ModuleByIndex(moduleIndex int) *SunvoxModule {

	if moduleIndex < 0 {
		return nil
	}

	if flags := getModuleFlags(c.Index, moduleIndex); flags >= 0 && (flags&ModuleFlagExists > 0) {
		return &SunvoxModule{
			Channel: c,
			Index:   moduleIndex,
		}
	}

	return nil
}

// OutputModule returns the output module for the SunvoxChannel.
func (c *SunvoxChannel) OutputModule() *SunvoxModule {
	return &SunvoxModule{
		Channel: c,
		Index:   0, // Output is index 0
	}
}

// SetBPM sets the BPM for playback in the channel to the desired BPM (cast down to integers), with a minimum of 32 BPM.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (c *SunvoxChannel) SetBPM(bpm float32) error {
	if bpm < 0x0020 {
		bpm = 0x0020
	}
	return c.SendEvent(0, 0, 0, 0, 0x000f, int(bpm))
}

// BPM returns the beats per minute for the song in the channel as a float32 for easy speed multiplication.
func (c *SunvoxChannel) BPM() float32 {
	return float32(getSongBPM(c.Index))
}

// SetTPL sets the TPL (ticks per line) for the project. The maximum value is 1F (31).
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (c *SunvoxChannel) SetTPL(tpl int) error {
	if tpl > 0x001F {
		tpl = 0x001F
	}
	return c.SendEvent(0, 0, 0, 0, 0x000f, tpl)
}

// SunvoxModule represents a module connected to other modules in a Sunvox project.
type SunvoxModule struct {
	Channel *SunvoxChannel
	Index   int
}

// Name is the name of the module in the project.
func (m *SunvoxModule) Name() string {
	return getModuleName(m.Channel.Index, m.Index)
}

// Flags returns the flags set on the given module as a int32 flag set.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (m *SunvoxModule) Flags() (int32, error) {
	flags := getModuleFlags(m.Channel.Index, m.Index)
	if flags < 0 {
		return 0, errors.New(fmt.Sprintf("error retrieving flags for module %d of name %s in channel %d; error code %d", m.Index, m.Name(), m.Channel.Index))
	}
	return flags, nil
}

// Sets the bypass, solo, and mute values for the module. Note that this works only for instruments, not effects.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (m *SunvoxModule) SetBSM(bypass, solo, mute bool) error {
	bsm := 0
	if bypass {
		bsm += 256
	}
	if solo {
		bsm += 16
	}
	if mute {
		bsm += 1
	}
	return m.Channel.SendEvent(0, 0, 0, m.Index, 0x0013, bsm)
}

// ControllerValue returns the value associated with the control index - for hexadecimal, you can precede the value with "0x".
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (m *SunvoxModule) ControllerValue(ctrlNum int) (int, error) {
	if ctrlNum <= 0 {
		return 0, errors.New(fmt.Sprintf("error getting controller value; controllers 0 and below don't exist"))
	}
	if res := getModuleCtlValue(m.Channel.Index, m.Index, ctrlNum-1, 2); res < 0 {
		return 0, errors.New(fmt.Sprintf("error retrieving controller %d value; error code %d", ctrlNum, res))
	} else {
		return int(res), nil
	}
}

// ControllerName returns the name associated with the control index - for hexadecimal, you can precede the value with "0x".
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (m *SunvoxModule) ControllerName(ctrlNum int) (string, error) {
	if ctrlNum <= 0 {
		return "", errors.New(fmt.Sprintf("error getting controller name; controllers 0 and below don't exist"))
	}
	return getModuleCtlName(m.Channel.Index, m.Index, ctrlNum-1), nil
}

// ControllerMinimum returns the minimum value in the range associated with the control index -
// for hexadecimal, you can precede the value with "0x".
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (m *SunvoxModule) ControllerMinimum(ctrlNum int) (int, error) {
	if ctrlNum <= 0 {
		return 0, errors.New(fmt.Sprintf("error getting controller minimum value; controllers 0 and below don't exist"))
	}
	if res := getModuleCtlMin(m.Channel.Index, m.Index, ctrlNum-1, 2); res < 0 {
		return 0, errors.New(fmt.Sprintf("error retrieving controller %d minimum value; error code %d", ctrlNum, res))
	} else {
		return int(res), nil
	}
}

// ControllerMaximum returns the maximum value in the range associated with the control index -
// for hexadecimal, you can precede the value with "0x".
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (m *SunvoxModule) ControllerMaximum(ctrlNum int) (int, error) {
	if ctrlNum <= 0 {
		return 0, errors.New(fmt.Sprintf("error getting controller maximum value; controllers 0 and below don't exist"))
	}
	if res := getModuleCtlMin(m.Channel.Index, m.Index, ctrlNum-1, 2); res < 0 {
		return 0, errors.New(fmt.Sprintf("error retrieving controller %d maximum value; error code %d", ctrlNum, res))
	} else {
		return int(res), nil
	}
}

// SetControlValue sets the numbered controller of ctrlNum to the value indicated.
// ctrlNum is the number of the controller as seen in Sunvox, not the indexed value (i.e. the first controller
// is 1 in Sunvox, so you would use 1 here, 1C in Sunvox is 0x1C here, etc).
// If ctrlNum is less than or equal to zero, SetControlValue returns an error.
// The value should be the logical value from Sunvox for the controller specified, not 0 - 8000.
// Controller #3 for an Analog Generator, panning, ranges from -128 to 128; to set this to 50% right would be:
// channel.ModuleByName("Analog generator").SetControlValue(3, 64)
func (m *SunvoxModule) SetControllerValue(ctrlNum, value int) error {
	if ctrlNum <= 0 {
		return errors.New(fmt.Sprintf("error setting control; controllers 0 and below don't exist"))
	}
	if res := setModuleCtlValue(m.Channel.Index, m.Index, ctrlNum-1, value, 2); res < 0 {
		return errors.New(fmt.Sprintf("error setting controller %d to value %d; error code %d", ctrlNum, value, res))
	}
	return nil
}

// Connect connects a Module to a specified other Module.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (m *SunvoxModule) Connect(dest *SunvoxModule) error {

	if dest == nil {
		return errors.New(fmt.Sprintf("error connecting module %d (source) to destination module; it is nil", m.Index))
	}

	if err := m.Channel.Lock(); err != nil {
		return err
	}

	if res := connectModule(m.Channel.Index, m.Index, dest.Index); res < 0 {
		return errors.New(fmt.Sprintf("error connecting module %d (source) to module %d (dest); error code %d", m.Index, dest.Index, res))
	}

	if err := m.Channel.Unlock(); err != nil {
		return err
	}

	return nil
}

// Disconnect disconnects a Module from a specified other Module.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (m *SunvoxModule) Disconnect(dest *SunvoxModule) error {

	if dest == nil {
		return errors.New(fmt.Sprintf("error disconnecting module %d (source) from destination module; it is nil", m.Index))
	}

	if err := m.Channel.Lock(); err != nil {
		return err
	}

	if res := disconnectModule(m.Channel.Index, m.Index, dest.Index); res < 0 {
		return errors.New(fmt.Sprintf("error disconnecting module %d (source) to module %d (dest); error code %d", m.Index, dest.Index, res))
	}

	if err := m.Channel.Unlock(); err != nil {
		return err
	}

	return nil
}

// Finetune returns the finetune value of the Module.
func (m *SunvoxModule) Finetune() uint32 {
	f := getModuleFinetuneRelativeNote(m.Channel.Index, m.Index)
	finetune := f >> 16 & 0xFFFF
	if finetune&0x8000 > 0 {
		finetune -= 0x10000
	}
	return finetune
}

// RelativeNote returns the relative note value for the module.
func (m *SunvoxModule) RelativeNote() uint32 {
	f := getModuleFinetuneRelativeNote(m.Channel.Index, m.Index)
	relnote := f & 0xFFFF
	if relnote&0x8000 > 0 {
		relnote -= 0x10000
	}
	return relnote
}

// SetFinetune sets the finetune value for the module (with the default being 0).
// The value can range from -256 to 256.
// If the function is unable to execute for whatever reason, it will return an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (m *SunvoxModule) SetFinetune(finetune int) error {
	err := setModuleFinetune(m.Channel.Index, m.Index, finetune)
	if err > 0 {
		return errors.New(fmt.Sprintf("error setting finetune for module %d value %d; error code %d", m.Index, finetune, err))
	}
	return nil
}

// SetRelativeNote sets the relative note value for the module (with the default being 0).
// If the function is unable to execute for whatever reason, it will return an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (m *SunvoxModule) SetRelativeNote(relativeNote int) error {
	err := setModuleRelativeNote(m.Channel.Index, m.Index, relativeNote)
	if err > 0 {
		return errors.New(fmt.Sprintf("error setting finetune for module %d value %d; error code %d", m.Index, relativeNote, err))
	}
	return nil
}

// SunvoxPatternNoteData represents note data for one line for one track in a pattern's note data.
type SunvoxPatternNoteData struct {
	Note            uint8
	Velocity        uint8
	Module          uint16
	Controller      uint16
	ControllerValue uint16
}

// SunvoxPatternData represents note data for all lines for all tracks in a pattern's note data.
type SunvoxPatternData struct {
	trackCount int
	Data       []SunvoxPatternNoteData
}

// LineCount returns the number of lines in the pattern data.
func (s SunvoxPatternData) LineCount() int {
	return len(s.Data) / s.trackCount
}

// TrackCount returns the number of tracks in the pattern data.
func (s SunvoxPatternData) TrackCount() int {
	return s.trackCount
}

func (s SunvoxPatternData) noteData(trackNum, lineNum int) (*SunvoxPatternNoteData, error) {
	i := trackNum + (lineNum * s.trackCount)
	if i < 0 || i > len(s.Data) {
		return nil, errors.New("track number %d or line number %d outside of the range of the pattern")
	}
	return &s.Data[i], nil
}

// Note returns the note of the track and line given, from hexadecimal.
// C5 is 61.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s SunvoxPatternData) Note(trackNum, lineNum int) (uint8, error) {
	note, err := s.noteData(trackNum, lineNum)
	if err != nil {
		return 0, err
	}
	return note.Note, nil
}

// SetNote sets the note for the given note data to the value specified.
// You can use the NoteCommand constants for special note types.
// C5 is 61.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s SunvoxPatternData) SetNote(trackNum, lineNum int, noteValue uint8) error {
	note, err := s.noteData(trackNum, lineNum)
	if err != nil {
		return err
	}
	note.Note = noteValue
	return nil
}

// Velocity returns the velocity of the given note data, ranging from 0-129 (with 0 being the default volume level).
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s SunvoxPatternData) Velocity(trackNum, lineNum int) (uint8, error) {
	note, err := s.noteData(trackNum, lineNum)
	if err != nil {
		return 0, err
	}
	return note.Velocity, nil
}

// SetVelocity sets the velocity of the given note data, ranging from 0-129 (with 0 being the default volume level).
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s SunvoxPatternData) SetVelocity(trackNum, lineNum int, velocity uint8) error {
	note, err := s.noteData(trackNum, lineNum)
	if err != nil {
		return err
	}
	note.Velocity = velocity
	return nil
}

// Module returns the module number of the given note data for the track and line provided.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s SunvoxPatternData) Module(trackNum, lineNum int) (uint16, error) {
	note, err := s.noteData(trackNum, lineNum)
	if err != nil {
		return 0, err
	}
	return note.Module + 1, nil
}

// SetModule sets the module number of the given note data to the value given.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s SunvoxPatternData) SetModule(trackNum, lineNum int, moduleNumber uint16) error {
	note, err := s.noteData(trackNum, lineNum)
	if err != nil {
		return err
	}
	note.Module = moduleNumber + 1
	return nil
}

// Controller returns the specified controller index for the given note data for the track and line provided.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s SunvoxPatternData) Controller(trackNum, lineNum int) (uint16, error) {
	note, err := s.noteData(trackNum, lineNum)
	if err != nil {
		return 0, err
	}
	return note.Controller - 1, nil
}

// SetController sets the specified controller index for the given note data for the track and line provided.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s SunvoxPatternData) SetController(trackNum, lineNum int, controllerNumber uint16) error {
	note, err := s.noteData(trackNum, lineNum)
	if err != nil {
		return err
	}
	note.Controller = controllerNumber - 1
	return nil
}

// ControllerValue returns the controller value (XXYY) for the given note data for the track and line provided.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s SunvoxPatternData) ControllerValue(trackNum, lineNum int) (uint16, error) {
	note, err := s.noteData(trackNum, lineNum)
	if err != nil {
		return 0, err
	}
	return note.ControllerValue, nil
}

// SetControllerValue sets the controller value (XXYY) for the given note data for the track and line provided.
// If the function is unable to execute for whatever reason, the function returns an
// error code (and, if the SunvoxEngine is initialized in debug mode (which is the default), the Engine
// will print exactly what the error might be).
func (s SunvoxPatternData) SetControllerValue(trackNum, lineNum int, value uint16) error {
	// TODO: Add a hexadecimal converter, a XX vs YY variant, etc.
	note, err := s.noteData(trackNum, lineNum)
	if err != nil {
		return err
	}
	note.ControllerValue = value
	return nil
}
