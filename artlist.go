package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var activeRoutines int32
var maxRoutines int
var downloadSongs bool
var exitRequested int32 = 0
var allPagesDone int32 = 0
var totalLoaded int64 = 0

var songsChan = make(chan Song, 100)
var wg sync.WaitGroup

var appStart = time.Now()

type Resp struct {
	Songs []Song `json:"songs"`
}

type Song struct {
	ArtistName string `json:"artistName"`
	SongName string `json:"songName"`
	MP3FilePath string `json:"MP3FilePath"`
}

func main() {
	minPage := flag.Int("min", 0, "Min page")
	maxPage := flag.Int("max", 9999, "Max page")
	flag.IntVar(&maxRoutines, "conns", 4, "Concurrency")
	flag.BoolVar(&downloadSongs, "download", false, "Download found songs")

	flag.Parse()

	// Catch Ctrl+C
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		<-c
		atomic.StoreInt32(&exitRequested, 1)
	}()

	go writeCsv()

	if downloadSongs {
		var err error
		err = os.MkdirAll("Downloads", 0777)
		if err != nil { log.Fatal(err) }
		err = os.Chdir("Downloads")
		if err != nil { log.Fatal(err) }
	}

	for i := *minPage; i < *maxPage; i++ {
		if atomic.LoadInt32(&exitRequested) != 0 {
			break
		}
		if atomic.LoadInt32(&allPagesDone) > 100 {
			break
		}

		startRoutine()

		wg.Add(1)
		go loadPage(i)
	}

	wg.Wait()
	close(songsChan)
}

func loadPage(i int) {
	defer wg.Done()
	defer stopRoutine()

	u := `https://artlist.io/api/Song/List` +
		`?searchTerm=` +
		`&categoryIDs=` +
		`&songSortID=1` +
		`&durationMin=0` +
		`&durationMax=0` +
		`&onlyVocal=` +
		`&page=` +
		strconv.FormatInt(int64(i), 10)

	res, err := http.Get(u)
	if err != nil {
		log.Printf("Failed to get page %d: %s", i, err)
		return
	}
	defer res.Body.Close()

	var resp Resp

	jr := json.NewDecoder(res.Body)
	err = jr.Decode(&resp)
	if err != nil {
		log.Printf("Failed to get page %d: %s", i, err)
		return
	}

	if len(resp.Songs) == 0 {
		atomic.AddInt32(&allPagesDone, 1)
		return
	} else {
		atomic.StoreInt32(&allPagesDone, 0)
	}

	wg.Add(len(resp.Songs))
	for _, song := range resp.Songs {
		songsChan <- song
		go loadSong(song)
	}

	log.Printf("Got page #%d: %d songs", i, len(resp.Songs))
}

func loadSong(s Song) {
	defer wg.Done()

	startRoutine()
	defer stopRoutine()

	fName := fmt.Sprintf("%s - %s.mp3", s.ArtistName, s.SongName)

	start := time.Now()
	n, err := _loadSong(fName, s.MP3FilePath)
	totalMB := float64(atomic.AddInt64(&totalLoaded, n)) / (1<<20)
	dur := time.Since(start)
	totalDur := time.Since(appStart).Seconds()

	if err != nil {
		log.Printf("Error downloading %s: %s", fName, err)
	} else {
		log.Printf("Downloaded %s: (%.2f MiB/s) %.2f MiB in %.2fs",
			fName, totalMB / totalDur, float32(n) / (1<<20), dur.Seconds())
	}
}

func _loadSong(fName, u string) (int64, error) {
	f, err := os.OpenFile(fName, os.O_CREATE | os.O_EXCL | os.O_WRONLY, 0666)
	if err != nil { return 0, err }
	defer f.Close()

	res, err := http.Get(u)
	if err != nil { return 0, err }
	defer res.Body.Close()

	return io.Copy(f, res.Body)
}

func startRoutine() {
	for int(atomic.LoadInt32(&activeRoutines)) > maxRoutines {
		time.Sleep(200 * time.Millisecond)
	}
	atomic.AddInt32(&activeRoutines, 1)
}

func stopRoutine() {
	atomic.AddInt32(&activeRoutines, -1)
}

func writeCsv() {
	f, err := os.OpenFile("songs.csv", os.O_CREATE | os.O_WRONLY | os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error writing songs.csv: %s", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	for song := range songsChan {
		err := w.Write([]string{ song.ArtistName, song.SongName, song.MP3FilePath })
		if err != nil {
			log.Fatalf("Error writing songs.csv: %s", err)
		}
	}
}
