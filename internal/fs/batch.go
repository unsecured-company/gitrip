package fs

import (
	"bufio"
	"fmt"
	gourl "net/url"
	"os"
	"unsecured.company/gitrip/internal/application"
	"unsecured.company/gitrip/internal/utils"
)

type Batch struct {
	app                   *application.App
	file                  string
	UrlChan               chan *gourl.URL
	CntValidWithScheme    int
	CntValidWithoutScheme int
	CntInvalid            int
}

func NewBatch(app *application.App, file string, urlChan chan *gourl.URL) *Batch {
	bat := Batch{
		app:     app,
		file:    file,
		UrlChan: urlChan,
	}

	return &bat
}

func (bat *Batch) Run() (err error) {
	err = bat.analyzeFile()

	if err != nil {
		return err
	}

	cntAll := bat.CntValidWithScheme + bat.CntValidWithoutScheme + bat.CntInvalid

	msg := "BatchFile <%s> contains %d invalid | %d / %d valid with/without scheme | %d in summary."
	bat.app.Out.Logf(msg, bat.file, bat.CntInvalid, bat.CntValidWithScheme, bat.CntValidWithoutScheme, cntAll)

	err = bat.readFile()

	return
}

func (bat *Batch) analyzeFile() (err error) {
	file, err := os.Open(bat.file)

	if err != nil {
		return fmt.Errorf("failed to open file [%s]: %w", bat.file, err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		url := scanner.Text()
		parsedURL, err := utils.ParseUrlOrDomain(url)
		isValid := err == nil && parsedURL.Host != ""
		hasScheme := isValid && parsedURL.Scheme == ""

		if !isValid {
			bat.CntInvalid++

			continue
		}

		if hasScheme {
			bat.CntValidWithScheme++
		} else {
			bat.CntValidWithoutScheme++
		}
	}

	if err := scanner.Err(); err != nil {
		err = fmt.Errorf("error reading file: %v", err)
	}

	return
}

func (bat *Batch) readFile() (err error) {
	bat.app.Out.Logf("Reading file %s", bat.file)

	file, err := os.Open(bat.file)

	if err != nil {
		return
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		err = bat.processLine(line)

		if err != nil {
			bat.app.Out.Log(err.Error())
		} else {
			bat.app.Out.Debugf("URL added to channel <%s>", line)
		}
	}

	err = scanner.Err()
	close(bat.UrlChan)

	return
}

func (bat *Batch) processLine(line string) (err error) {
	urls, err := utils.GetUrls(line)

	if err == nil {
		for _, url := range urls {
			bat.UrlChan <- url
		}
	}

	return
}
