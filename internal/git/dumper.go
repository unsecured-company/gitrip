package git

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	gourl "net/url"
	"os"
	"regexp"
	"sync"
	"unsecured.company/gitrip/internal/application"
	"unsecured.company/gitrip/internal/network"
	"unsecured.company/gitrip/internal/utils"
)

const (
	ChanFetchSize         = 100
	LogDebugRunningForUrl = "(%s) running"
	LogErrSavingFile      = "[%s]: %v"
	LogFailedRunUrl       = "(%s): %v"
	LogAddDomain          = "(%s) added"
)

type Dumper struct {
	app        *application.App
	fetcher    *network.Fetcher
	chanSave   chan *Item
	cntFailed  int
	cntSuccess int
	domains    *utils.SafeMapStrings // TODO remove too, it will be chan
	hashRegexp *regexp.Regexp
	wgSaver    *sync.WaitGroup
	wgWorker   sync.WaitGroup
}

func NewDumper(app *application.App) *Dumper {
	gf := Dumper{
		app:        app,
		fetcher:    network.NewFetcher(app),
		chanSave:   make(chan *Item, ChanFetchSize),
		domains:    utils.NewSafeMapStrings(),
		hashRegexp: regexp.MustCompile(HashRegexp),
		wgSaver:    &sync.WaitGroup{},
		wgWorker:   sync.WaitGroup{},
	}

	gf.wgSaver.Add(1)

	go gf.runnerSave()

	return &gf
}

func (d *Dumper) Run() (err error) {
	if d.app.Cfg.URL != "" {
		err = d.runForUrl(d.app.Cfg.URL)
	} else if d.app.Cfg.BatchFile != "" {
		err = d.runForFile()
	} else {
		err = errors.New("URL or BatchFile is required") // Should be caught by the config validation
	}

	return
}

func (d *Dumper) Close() {
	d.app.Out.Debug("Closing save channel")
	close(d.chanSave)

	d.wgSaver.Wait()
}

func (d *Dumper) runForUrl(urlStr string) (err error) {
	urlP, err := utils.ParseUrlOrDomain(urlStr)

	if err != nil {
		return
	}

	if urlP.Scheme == "" {
		urlP.Scheme = "https"
	}

	rp := NewRepo(d, urlP)
	err = rp.Run()

	return
}

func (d *Dumper) runnerSave() {
	for it := range d.chanSave {
		err := it.save()

		if err != nil {
			d.app.Out.Logf(LogErrSavingFile, it.fileName, err)
		}
	}

	return
}

func (d *Dumper) runForFile() (err error) {
	d.app.Out.Logf("Running for batch file [%s]", d.app.Cfg.BatchFile)
	err = d.readFile(d.app.Cfg.BatchFile)

	if err != nil {
		err = fmt.Errorf("error reading file: %w", err)

		return
	}

	d.wgWorker.Add(2)
	go d.worker()
	go d.worker()
	d.wgWorker.Wait()

	d.app.Out.Logf("Finished file [%s]", d.app.Cfg.BatchFile)

	return
}

func (d *Dumper) worker() {
	var err error

	for {
		_, url := d.domains.PullRand()

		if url == "" {
			break
		}

		d.app.Out.Debugf(LogDebugRunningForUrl, url)
		err = d.runForUrl(url)

		if err != nil {
			d.app.Out.Logf(LogFailedRunUrl, url, err)
		}

	}

	d.wgWorker.Done()
}

func (d *Dumper) readFile(filePath string) (err error) {
	d.app.Out.Logf("Input file [%s]", filePath)

	//fileInfo, err := os.Stat(filePath)
	//_ = fileInfo.Size() > 1*1024*1024

	file, err := os.Open(filePath)

	if err != nil {
		return
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		urlP, err := d.parseLine(line)

		if err == nil && urlP.IsAbs() {
			d.cntSuccess++
			d.domains.Add(urlP.String())
			d.app.Out.Debugf(LogAddDomain, urlP.String())
		} else {
			d.cntFailed++
			d.app.Out.Logf("Invalid URL [%s]", line)
		}
	}

	err = scanner.Err()

	if err != nil {
		return
	}

	pValid := Percentage(d.cntSuccess, d.cntSuccess+d.cntFailed)
	pFailed := Percentage(d.cntFailed, d.cntSuccess+d.cntFailed)

	d.app.Out.Logf("Valid: %d (%s), Failed: %d (%s)", d.cntSuccess, pValid, d.cntFailed, pFailed)

	return
}

func Percentage(value, total int) string {
	if total == 0 {
		return "0%"
	}

	percent := math.Round(float64(value) / float64(total) * 100)

	return fmt.Sprintf("%d%%", int(percent))
}

func (d *Dumper) parseLine(line string) (urlP *gourl.URL, err error) {
	urlP, err = gourl.Parse(line)

	if err != nil {
		return
	}

	if urlP.Scheme == "" {
		urlP.Scheme = "https"
		line = urlP.String()
		urlP, err = gourl.Parse(line)
	}

	return
}
