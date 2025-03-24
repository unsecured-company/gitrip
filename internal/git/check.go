package git

import (
	"fmt"
	"net/url"
	"sync"
	"unsecured.company/gitrip/internal/application"
	"unsecured.company/gitrip/internal/fs"
	"unsecured.company/gitrip/internal/network"
	"unsecured.company/gitrip/internal/utils"
)

const UrlsChanSize = 1000

type Checker struct {
	UrlChan chan *url.URL

	app        *application.App
	cntFailed  int
	cntSuccess int
	wgProcess  *sync.WaitGroup
	batch      *fs.Batch
	cntYes     int
	cntNo      int
}

func NewChecker(app *application.App) (ch *Checker) {
	ch = &Checker{
		app:       app,
		wgProcess: &sync.WaitGroup{},
		UrlChan:   make(chan *url.URL, UrlsChanSize),
	}

	if app.Cfg.BatchFile != "" {
		ch.batch = fs.NewBatch(app, app.Cfg.BatchFile, ch.UrlChan)
	}

	return
}

func (ch *Checker) Run() (err error) {
	ch.wgProcess.Add(application.DefaultCntDownThreads)

	for i := 0; i < application.DefaultCntDownThreads; i++ {
		go ch.processor()
	}

	if ch.app.Cfg.URL != "" {
		err = ch.runForUrl()
	} else if ch.app.Cfg.BatchFile != "" {
		err = ch.batch.Run()
	} else {
		err = fmt.Errorf("no url or batch file provided")
	}

	ch.wgProcess.Wait()

	return
}

func (ch *Checker) runForUrl() (err error) {
	defer close(ch.UrlChan)
	urls, err := utils.GetUrls(ch.app.Cfg.URL)

	if err != nil {
		return
	}

	for _, uri := range urls {
		ch.UrlChan <- uri
	}

	return
}

func (ch *Checker) processor() {
	fetcher := network.NewFetcher(ch.app)

	for uri := range ch.UrlChan {
		urlRoot := utils.GetNewSuffixedUrl(uri, PathRoot)

		ch.check(fetcher, urlRoot)
	}

	ch.wgProcess.Done()
}

func (ch *Checker) check(fetcher *network.Fetcher, urlRoot *url.URL) {
	urlIndex := utils.GetNewSuffixedUrl(urlRoot, PathIndex)
	data, code, err := fetcher.Fetch(ch.app.Ctx, urlIndex.String())

	if err != nil {
		ch.app.Out.Logf("%s failed, error: %v", urlIndex, err)

		return
	}

	if code != 200 {
		ch.app.Out.Logf("%s failed, code: %d", urlIndex, code)

		return
	}

	index, err := NewIndexFromBytes(data)

	if err != nil {
		ch.app.Out.Logf("%s failed for Index file, error: %v", urlIndex, err)

		return
	}

	ch.app.Out.Println(urlRoot.String())

	ch.app.Out.Logf("%s\tOK, files: %d", urlIndex, len(index.Index.Entries))
}
