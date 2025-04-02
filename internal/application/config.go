package application

import "fmt"

const (
	Version               = "1.1.0-250402"
	DefaultFetchDir       = "dumps"
	DefaultFetchWorkers   = 4
	DefaultTimeout        = 10
	DefaultCntDownThreads = 10
	LimitHashes           = 2000 // Max hashes to read by regex from files other than /objects.
	DebugPrintEveryFetch  = false
	RetryAfterXSeconds    = 5

	FetchClientRetryMax           = 4
	FetchClientRetryWaitMinSec    = 1
	FetchClientRetryWaitMaxSec    = 30
	FetchClientMaxIdleCnt         = 100
	FetchClientMaxConnPerHost     = 10
	FetchClientIdleConnTimeoutSec = 90

	// SkipExisting TODO check what we downloaded, and skip these. Will work for /objects, as they are verified by their hash name
	SkipExisting                = true
	IgnoreInvalidObjectChecksum = true
)

const (
	CmdCheck = "check"
	CmdFetch = "fetch"
	CmdIndex = "index"
	CmdHelp  = "help"
	FlagFile = "file"
	FlagUrl  = "url"
)

type Config struct {
	BatchFile  string
	Command    string
	DwnDir     string
	DwnThreads int
	IndexFile  string
	OutputDir  string
	Raw        bool
	Timeout    int
	Tree       bool
	URL        string
	Update     bool
	UserAgent  string
	Verbose    bool
}

func NewConfig(args []string, out *Output) (cfg *Config, mErr *MultiErr) {
	var err error
	args = args[1:]
	cfg, mErr = getConfigCli(args)

	if mErr.HasErrors() {
		return
	}

	// TODO validate url

	cfg.DwnThreads = DefaultFetchWorkers
	out.SetVerbose(cfg.Verbose)

	cfg.DwnDir, err = getAbsolutePath(DefaultFetchDir)
	mErr.Add(err)

	if cfg.OutputDir != "" {
		cfg.OutputDir, err = GetOutputFilePrefix(cfg.OutputDir, cfg.URL)
		if err == nil {
			err = out.SetOutputDir(cfg.OutputDir)
		}

		mErr.Add(err)
	}

	return
}

func (cfg *Config) PrintConfigurationText(out *Output) {
	msgCommon := "Timeout " + fmt.Sprintf("%d", cfg.Timeout) + " seconds."

	if cfg.Update {
		msgCommon += " Will update existing repository."
	}

	cfgArr := []string{
		"Downloading into [" + cfg.DwnDir + "]",
	}

	if cfg.Verbose {
		msgCommon += " Verbose mode."
	}

	if cfg.UserAgent != "" {
		cfgArr = append(cfgArr, "User Agent "+cfg.UserAgent)
	}

	if cfg.OutputDir != "" {
		cfgArr = append(cfgArr, "Logging into "+cfg.OutputDir+".*")
	}

	cfgArr = append(cfgArr, msgCommon)

	for _, v := range cfgArr {
		out.Log(v)
	}
}
