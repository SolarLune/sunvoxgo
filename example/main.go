package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/solarlune/sunvoxgo"
)

func main() {

	engine := sunvoxgo.Engine()

	// We could manually initialize using the library path exactly with SunvoxEngine.Init(), but
	// SunvoxEngine.InitFromDirectory dynamically loads the correct library for the current OS and architecture, so it's cross-platform-compatible.

	// For a Mac build, this should probably be relative to the path returned by os.Executable() as, as I recall, running an app through Finder
	// does some shenanigans to modify the working directory to be the home directory rather than the application's launching directory.
	absPath, err := filepath.Abs("./sunvox_lib-2.1.2b")
	if err != nil {
		panic(err)
	}

	err = engine.InitFromDirectory(absPath, nil)

	if err != nil {
		panic(err)
	}

	channel, err := engine.CreateChannel("music")
	if err != nil {
		panic(err)
	}

	channel.LoadFileFromFS(os.DirFS("./"), "assets/CityStatesOfGENOW.sunvox")

	channel.PlayFromBeginning()

	for channel.IsPlaying() {
		command := " "
		fmt.Println("")
		fmt.Println("Enter any of the following commands to change music playback:")
		fmt.Println("s+ : Speed Up | s- : Speed Up | q : Quit | p+ : Pitch Up | p- : Pitch Down")
		fmt.Scanln(&command)
		switch command {
		case "s+":
			channel.SetBPM(channel.BPM() * 1.2)
			fmt.Println("BPM sped up by 20%")
		case "s-":
			channel.SetBPM(channel.BPM() * 0.8)
			fmt.Println("BPM slowed down by 20%")
		case "q":
			channel.Stop()
			fmt.Println("Playback stopped")
		case "p+":
			channel.ForEachPattern(func(p *sunvoxgo.SunvoxPattern) bool {
				patternData, err := p.Data()
				if err != nil {
					panic(err) // Should never happen
				}
				// Skip drums that are above the main melody
				if p.Y() < 0 {
					return true
				}

				for line := range patternData.LineCount() {
					for track := range patternData.TrackCount() {
						noteValue, _ := patternData.Note(track, line)
						if noteValue < 128 {
							patternData.SetNote(track, line, noteValue+3)
						}
					}
				}
				return true
			})
			fmt.Println("Song pitched up by 3 semitones")
		case "p-":
			channel.ForEachPattern(func(p *sunvoxgo.SunvoxPattern) bool {
				patternData, err := p.Data()
				if err != nil {
					panic(err) // Should never happen
				}
				// Skip drums that are above the main melody
				if p.Y() < 0 {
					return true
				}

				for line := range patternData.LineCount() {
					for track := range patternData.TrackCount() {
						noteValue, _ := patternData.Note(track, line)
						if noteValue < 128 {
							patternData.SetNote(track, line, noteValue-3)
						}
					}
				}
				return true
			})
			fmt.Println("Song pitched down by 3 semitones")

		default:
			fmt.Println("Command '" + command + "' is not recognized")
		}
	}

	// time.Sleep(time.Second * 4)

	// channel.SetBPM(channel.BPM() * 1.2)

	// mod := channel.ModuleByName("Analog generator")

	// fmt.Println(mod.SetControllerValue(1, 128))

	// time.Sleep(time.Second * 100)

	// engine.Init(NewInitConfig().Device("alsa"), 44100, 0)

}
