package main

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"

"strings"
)

func ProcessFiles(dir string) {
	command := fmt.Sprintf("for file in %s/*.mkv; do if [[ -f '$file' ]]; then base='${file%%.*}'; ffmpeg -i '$file' -map 0:v -map 0:a -map 0:s:0 -c copy '${base}_sub.mkv'; fi; done", dir)
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error executing command: %v", err)
	}

	// Print the output
	fmt.Println(string(output))
}

func CountSubtitleTracks(filePath string) (int, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-select_streams", "s", "-show_entries", "stream=index", "-of", "csv=p=0", filePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("error running ffprobe: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0, nil
	}

	return len(lines), nil
}

func FindEnglishSubtitleTrack(filePath string) (int, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-select_streams", "s", "-show_entries", "stream=index:stream_tags=language", "-of", "csv=p=0", filePath)
	output, err := cmd.Output()
	if err != nil {
		return -1, fmt.Errorf("error running ffprobe: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			lang := strings.ToLower(strings.TrimSpace(parts[1]))
			if lang == "eng" || lang == "en" || lang == "english" {
				return 0, nil // Return relative subtitle index (0 for first subtitle track)
			}
		}
	}
	
	return -1, fmt.Errorf("no English subtitle track found")
}

func ProcessSingleFile(filePath string) error {
	base := filePath[:len(filePath)-len(filepath.Ext(filePath))]
	outputFile := base + "_sub.mkv"
	
	engSubIndex, err := FindEnglishSubtitleTrack(filePath)
	if err != nil {
		fmt.Printf("Warning: %v, using first subtitle track\n", err)
		engSubIndex = 0
	}
	
	subtitleMap := fmt.Sprintf("0:s:%d", engSubIndex)
	cmd := exec.Command("ffmpeg", "-i", filePath, "-map", "0:v", "-map", "0:a", "-map", subtitleMap, "-c", "copy", outputFile)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error executing ffmpeg: %v", err)
	}
	
	fmt.Println(string(output))
	return nil
}

func ProcessMKVIfMultipleSubtitles(filePath string) error {
	subtitleCount, err := CountSubtitleTracks(filePath)
	if err != nil {
		return err
	}

	fmt.Printf("File %s has %d subtitle tracks\n", filePath, subtitleCount)

	if subtitleCount > 2 {
		return ProcessSingleFile(filePath)
	}

	return nil
}
