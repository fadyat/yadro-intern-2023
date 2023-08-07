package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"yadro-intern/cmd/config"
	"yadro-intern/internal/model"
	"yadro-intern/internal/parser"
	"yadro-intern/internal/processor"
	"yadro-intern/internal/storage"
)

func parseArgs() (string, error) {
	if len(os.Args) < 2 {
		return "", fmt.Errorf("usage: ./yadro-intern <filename>")
	}

	return os.Args[1], nil
}

func openFile(filename string) (*os.File, error) {
	f, err := os.Open(filepath.Clean(filename))
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return nil, fmt.Errorf("file does not exist: %s", filename)
	case errors.Is(err, os.ErrPermission):
		return nil, fmt.Errorf("not enough permissions to open file: %s", filename)
	default:
		return nil, fmt.Errorf("could not open file: %s", err)
	}

	return f, nil
}

func main() {
	log.SetFlags(0)

	parserConfig, err := config.NewParserConfig()
	if err != nil {
		log.Println("checkout configuration:", err)
		return
	}

	processorConfig, err := config.NewProcessorConfig()
	if err != nil {
		log.Println("checkout configuration:", err)
		return
	}

	filename, err := parseArgs()
	if err != nil {
		log.Println(err)
		return
	}

	f, err := openFile(filename)
	if err != nil {
		log.Println(err)
		return
	}

	defer func() {
		if err = f.Close(); err != nil {
			log.Println("failed to close file:", err)
		}
	}()

	// reading core file data in synchronous way
	// because all others operations depend on it
	//
	// can use fan-out pattern here, but still fine in sync way
	fp := parser.NewFileParser(bufio.NewScanner(f), parserConfig)
	coreData, err := fp.ReadCoreData()
	if err != nil {
		log.Println(err)
		return
	}

	// reading events are done in a separate goroutine
	// made for making performance better?
	// (probably not, in case of printing errors first)
	eventsChan := fp.ReadEvents(coreData.TablesCount)

	// in case of parallel processing:
	//	by the task, we should print error, if it occurs
	//  without any additional information, for that purpose
	//  we can use buffer to store all successfully parsed events here
	var temporaryBuffer = bytes.NewBuffer(nil)

	p := processor.NewEventProcessor(
		temporaryBuffer,
		processorConfig,
		coreData,
		storage.NewInMemoryStorage[int, *model.IncomingEvent](),
		storage.NewInMemoryStorage[int, *model.RevenueStats](),
		storage.NewInMemoryStorage[string, int](),
		storage.NewInMemoryQueue[model.ClientData](nil),
	)

	done := make(chan error)
	defer close(done)

	go func() {
		e := p.ProcessEvents(eventsChan)
		if e != nil {
			done <- e
		} else {
			p.ShowRevenue()
			done <- nil
		}
	}()

	if err = <-done; err != nil {
		log.Println(err)
		return
	}

	// no errors mean that all events are successfully processed,
	// so we can print all successfully parsed events
	for scanner := bufio.NewScanner(temporaryBuffer); scanner.Scan(); {
		log.Println(scanner.Text())
	}
}
