# audio-extractor

CLI tool to extract audio from video and cut segments, using ffmpeg.

## Usage

### Extract audio from video
```bash
go run . extract -input video.mkv -output audio.mp3 [-track 0]
```
- `-input` — input video file (required)
- `-output` — output MP3 file (required)  
- `-track` — audio track number (default: 0)

Converts the first audio track to mono, 32 kbps bitrate.

### Cut audio segment
```bash
go run . cut -input audio.mp3 -output cut.mp3 -start 00:01:05 -end 00:46:00
```
- `-input` — input audio file (required)
- `-output` — output audio file (required)
- `-start` — start time in HH:MM:SS format (default: 00:00:00)
- `-end` — end time in HH:MM:SS format (required)

Cuts without re-encoding (lossless copy).

## Example

Extract and cut in one go:
```bash
go run . extract -input interview.mkv -output interview.mp3
go run . cut -input interview.mp3 -output interview_cut.mp3 -start 00:01:05 -end 00:46:00
```

## Requirements

- ffmpeg installed and in PATH
