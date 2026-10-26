package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: audio-extractor <extract|cut> [options]")
		fmt.Println("\nExtract audio from video:")
		fmt.Println("  audio-extractor extract -input video.mkv -output audio.mp3 [-track 0]")
		fmt.Println("\nCut audio segment:")
		fmt.Println("  audio-extractor cut -input audio.mp3 -output cut.mp3 -start 00:01:05 -end 00:46:00")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "extract":
		extractCmd := flag.NewFlagSet("extract", flag.ExitOnError)
		input := extractCmd.String("input", "", "Input video file (required)")
		output := extractCmd.String("output", "", "Output MP3 file (required)")
		track := extractCmd.Int("track", 0, "Audio track number")
		extractCmd.Parse(os.Args[2:])

		if err := extract(*input, *output, *track); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "cut":
		cutCmd := flag.NewFlagSet("cut", flag.ExitOnError)
		input := cutCmd.String("input", "", "Input audio file (required)")
		output := cutCmd.String("output", "", "Output audio file (required)")
		start := cutCmd.String("start", "00:00:00", "Start time (HH:MM:SS)")
		end := cutCmd.String("end", "", "End time (HH:MM:SS, required)")
		cutCmd.Parse(os.Args[2:])

		if err := cut(*input, *output, *start, *end); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func extract(input, output string, track int) error {
	if input == "" || output == "" {
		return fmt.Errorf("input and output are required")
	}

	cmd := exec.Command("ffmpeg",
		"-i", input,
		"-map", fmt.Sprintf("0:a:%d", track),
		"-ac", "1",
		"-b:a", "32k",
		output,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func cut(input, output, start, end string) error {
	if input == "" || output == "" || end == "" {
		return fmt.Errorf("input, output and end are required")
	}

	cmd := exec.Command("ffmpeg",
		"-i", input,
		"-ss", start,
		"-to", end,
		"-c", "copy",
		output,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
