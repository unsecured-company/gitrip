package application

import (
	"fmt"
	"log"
)

type Output struct {
	Verbose bool
	Files   *OutputFiles
}

type OutputFiles struct {
	TxtFile *File
	LogFile *File
}

func (f OutputFiles) Close() {
	if f.TxtFile != nil {
		_ = f.TxtFile.Close()
	}

	if f.LogFile != nil {
		_ = f.LogFile.Close()
	}
}

func NewOutput() (out *Output) {
	return &Output{}
}

func NewOutputFiles(fileAbs string) (files *OutputFiles, err error) {
	files = &OutputFiles{}
	files.TxtFile, err = NewFile(fileAbs+SuffixTxt, BuffSize)

	if err == nil {
		files.LogFile, err = NewFile(fileAbs+SuffixLog, 0)
	}

	return
}

func (out *Output) Println(msg string) {
	fmt.Println(msg)
}

func (out *Output) Printf(msg string, v ...interface{}) {
	fmt.Printf(msg, v...)
}

func (out *Output) Debug(msg string) {
	if out.Verbose {
		out.log(msg)
	}
}

func (out *Output) Debugf(msg string, v ...interface{}) {
	if out.Verbose {
		out.log(msg, v...)
	}
}

func (out *Output) Log(msg string) {
	out.log(msg)
}

func (out *Output) Logf(msg string, v ...interface{}) {
	out.log(msg, v...)
}

func (out *Output) ErrorIf(err error, msg string) {
	if err != nil {
		out.log(fmt.Sprintf(msg+": %v", err))
	}
}

func (out *Output) SetVerbose(verbose bool) {
	out.Verbose = verbose
}

func (out *Output) SetOutputDir(outDir string) (err error) {
	out.Files, err = NewOutputFiles(outDir)

	return
}

func (out *Output) log(msg string, v ...interface{}) {
	if len(v) == 0 {
		log.Println(msg)
	} else {
		log.Printf(msg+"\n", v...)
	}
}

func (out *Output) txt(msg string, v ...interface{}) {
	if len(v) == 0 {
		log.Println(msg)
	} else {
		log.Printf(msg+"\n", v...)
	}
}
