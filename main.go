package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Timespan time.Duration

func (t Timespan) Format(format string) string {
	return time.Unix(0, 0).UTC().Add(time.Duration(t)).Format(format)
}

func getTitle(link string) (string, error) {
	cmdln := fmt.Sprintf("yt-dlp --dump-json %s | jq --raw-output \".title\"", link)

	cmd := exec.Command("bash", "-c", cmdln)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err := cmd.Run()
	if err != nil {
		newErr := errors.New(errb.String() + "\n" + err.Error())
		return " ", newErr
	}
	return outb.String(), nil
}

func getDuration(link string) (string, error) {
	cmdln := fmt.Sprintf("yt-dlp --dump-json %s  | jq --raw-output \".duration\" | awk '{printf(\"%%d:%%02d:%%02d\\n\",($1/60/60%%24),($1/60%%60),($1%%60))}'", link)

	cmd := exec.Command("bash", "-c", cmdln)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err := cmd.Run()
	if err != nil {
		newErr := errors.New(errb.String() + "\n" + err.Error())
		return " ", newErr
	}

	hms := strings.Split(outb.String(), ":")
	hms[2] = strings.Trim(hms[2], "\x0a")
	dur, err := time.ParseDuration(fmt.Sprintf("%sh%sm%ss", hms[0], hms[1], hms[2]))
	if err != nil {
		return "", err
	}
	dur = dur - (1 * time.Second)

	return Timespan(dur).Format("15:04:05"), nil
}

func getTimeStamps(link string) ([]string, error) {
	cmdln := fmt.Sprintf("yt-dlp --dump-json %s  | jq --raw-output \".chapters[].start_time\" | awk '{printf(\"%%d:%%02d:%%02d\\n\",($1/60/60%%24),($1/60%%60),($1%%60))}'", link)

	cmd := exec.Command("bash", "-c", cmdln)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err := cmd.Run()
	if err != nil {
		newErr := errors.New(errb.String() + "\n" + err.Error())
		return []string{}, newErr
	}

	output := []string{}

	scanner := bufio.NewScanner(&outb)
	for scanner.Scan() {
		output = append(output, scanner.Text())
	}

	duration, err := getDuration(link)
	if err != nil {
		return []string{}, err
	}

	output = append(output, duration)

	return output, nil
}

func getTrackNames(link string) ([]string, error) {
	cmdln := fmt.Sprintf("yt-dlp --dump-json %s | jq --raw-output \".chapters[].title\"", link)
	cmd := exec.Command("bash", "-c", cmdln)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err := cmd.Run()
	if err != nil {
		newErr := errors.New(errb.String() + "\n" + err.Error())
		return []string{}, newErr
	}
	output := []string{}

	scanner := bufio.NewScanner(&outb)
	for scanner.Scan() {
		output = append(output, scanner.Text())
	}
	return output, nil
}

func downloadVid(title, link string) (string, error) {
	app := "yt-dlp"
	args := []string{"-f", "ba/b", "-x", "--audio-format", "mp3", link, "-o", title + ".mp3"}

	cmd := exec.Command(app, args...)

	var errb bytes.Buffer

	cmd.Stderr = &errb
	log.Printf("Downloading: %s\n", title)
	err := cmd.Run()
	if err != nil {
		ret_err := errors.New(errb.String())
		return "", ret_err

	}

	return title, nil
}

func splitVid(filename string, link string) error {

	timestamps, err := getTimeStamps(link)
	if err != nil {
		return err
	}
	track_names, err := getTrackNames(link)
	if err != nil {
		return err
	}

	for i := 0; i < len(track_names); i++ {
		err := sectionAudio(timestamps[i], timestamps[i+1], track_names[i], filename)
		if err != nil {
			// in case file already exists just add "copy" at the end ... this while loop is baaaaad news
			if err.Error() == "exit status 1" {
				for err != nil {
					track_names[i] = track_names[i] + " copy"
					err = sectionAudio(timestamps[i], timestamps[i+1], track_names[i], filename)
				}
			}
		}
	}
	return nil
}

func sectionAudio(start, finish, track_name, filename string) error {
	name := fmt.Sprintf("%s/%s.mp3", filename, track_name)
	app := "ffmpeg"
	args := []string{"-i", filename + ".mp3", "-ss", start, "-to", finish, "-c", "copy", name}

	cmd := exec.Command(app, args...)
	var errb bytes.Buffer

	cmd.Stderr = &errb
	log.Printf("Saving: %s-%s %s", start, finish, track_name)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	playlist_file := flag.String("playlist", " ", " playlist file")
	flag.Parse()
	file, err := os.Open(*playlist_file)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		title, err := getTitle(scanner.Text())
		if err != nil {
			log.Fatal(err)
		}

		err = os.Mkdir(title, os.ModePerm)

		if err != nil {
			if !os.IsExist(err) {
				log.Fatal(err)
				return
			}
		}

		if os.IsExist(err) {
			log.Println("Folder exists, skiping...")
			continue
		} else {
			current_playlist, err := downloadVid(title, scanner.Text())
			if err != nil {
				log.Fatal(err)
				return
			}

			err = splitVid(current_playlist, scanner.Text())
			if err != nil {
				log.Fatal(err)
				return

			}

			err = os.Remove(fmt.Sprintf("%s.mp3", title))
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}
