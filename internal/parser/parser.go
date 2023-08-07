package parser

import (
	"bufio"
	"strconv"
	"strings"
	"time"
	"yadro-intern/cmd/config"
	"yadro-intern/internal/apierror"
	"yadro-intern/internal/model"
)

type Parser interface {

	// ReadCoreData reads important data needed for further parsing.
	ReadCoreData() (*model.CoreData, error)

	// ReadEvents reads events from the file in a separate goroutine.
	// It returns a channel of events.
	// The channel is closed when all events are read, or when an error occurs.
	ReadEvents(maxTables int) <-chan model.WrappedIncomingEvent
}

type FileParser struct {
	scanner   *bufio.Scanner
	cfg       *config.Parser
	rowNumber int

	maxTables int
}

func NewFileParser(scanner *bufio.Scanner, cfg *config.Parser) *FileParser {
	return &FileParser{
		scanner: scanner,
		cfg:     cfg,
	}
}

func (p *FileParser) ReadCoreData() (*model.CoreData, error) {
	tablesCount, err := p.readTablesCount(apierror.MoreThenZero)
	if err != nil {
		return nil, err
	}

	workingTime, err := p.readWorkingTime()
	if err != nil {
		return nil, err
	}

	pricePerHour, err := p.readPricePerHour(apierror.MoreThenZero)
	if err != nil {
		return nil, err
	}

	return model.NewCoreData(tablesCount, pricePerHour, workingTime), nil
}

func (p *FileParser) ReadEvents(maxTables int) <-chan model.WrappedIncomingEvent {
	eventsChan := make(chan model.WrappedIncomingEvent, p.cfg.EventsChanSize)
	p.maxTables = maxTables

	go func() {
		defer close(eventsChan)

		for p.scanWithRowNumber() {
			event, err := p.readEvent()
			if err != nil {
				eventsChan <- model.WrappedIncomingEvent{Err: err}
				return
			}

			eventsChan <- model.WrappedIncomingEvent{Event: event}
		}
	}()

	return eventsChan
}

func (p *FileParser) scanWithRowNumber() bool {
	p.rowNumber++
	return p.scanner.Scan()
}

func (p *FileParser) readTablesCount(validate apierror.ValidationFn[int]) (int, error) {
	if !p.scanWithRowNumber() {
		return 0, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrTablesCountNotSpecified,
		}
	}

	n, err := strconv.Atoi(p.scanner.Text())
	if err != nil {
		return 0, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrTablesCountInvalidFormat,
			BaseErr:   err,
		}
	}

	if e := validate(n); e != nil {
		return 0, &apierror.ValidationError{
			RowNumber: p.rowNumber,
			UserMsg:   e.Error(),
		}
	}

	return n, nil
}

func (p *FileParser) readPricePerHour(validate apierror.ValidationFn[int]) (int, error) {
	if !p.scanWithRowNumber() {
		return 0, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrPricePerHourNotSpecified,
		}
	}

	n, err := strconv.Atoi(p.scanner.Text())
	if err != nil {
		return 0, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrPricePerHourInvalidFormat,
			BaseErr:   err,
		}
	}

	if e := validate(n); e != nil {
		return 0, &apierror.ValidationError{
			RowNumber: p.rowNumber,
			UserMsg:   e.Error(),
		}
	}

	return n, nil
}

func (p *FileParser) readWorkingTime() (*model.TimeInterval, error) {
	if !p.scanWithRowNumber() {
		return nil, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrWorkingTimeNotSpecified,
		}
	}

	workingHours := strings.Split(p.scanner.Text(), p.cfg.TimeSeparator)
	if len(workingHours) != 2 {
		return nil, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrWorkingTimeInvalidFormat,
		}
	}

	start, err := time.Parse(p.cfg.TimeFormat, workingHours[0])
	if err != nil {
		return nil, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrFailedToParseStartTime,
			BaseErr:   err,
		}
	}

	end, err := time.Parse(p.cfg.TimeFormat, workingHours[1])
	if err != nil {
		return nil, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrFailedToParseEndTime,
			BaseErr:   err,
		}
	}

	ti := model.NewTimeInterval(start, end)
	return ti, nil
}

func (p *FileParser) readEvent() (*model.IncomingEvent, error) {
	eventStrings := strings.SplitN(p.scanner.Text(), p.cfg.EventInfoSeparator, p.cfg.DistinctEventInfoCount)
	if len(eventStrings) != p.cfg.DistinctEventInfoCount {
		return nil, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrEventInvalidFormat,
		}
	}
	happensAtStr, eventTypeStr, clientDataStr := eventStrings[0], eventStrings[1], eventStrings[2]

	happensAt, err := time.Parse(p.cfg.TimeFormat, happensAtStr)
	if err != nil {
		return nil, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrFailedToParseEventTime,
			BaseErr:   err,
		}
	}

	eventType, err := p.parseIncomingEventType(eventTypeStr)
	if err != nil {
		return nil, err
	}

	clientData, err := p.parseClientData(eventType, clientDataStr)
	if err != nil {
		return nil, err
	}

	event := model.NewIncomingEvent(happensAt, eventType, clientData)
	return event, nil
}

func (p *FileParser) parseIncomingEventType(s string) (model.IncomingEventType, error) {
	eventType, err := strconv.Atoi(s)
	if err != nil {
		return 0, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrFailedToParseEventType,
			BaseErr:   err,
		}
	}

	switch eventType {
	case int(model.Arrives):
		return model.Arrives, nil
	case int(model.Sits):
		return model.Sits, nil
	case int(model.Waits):
		return model.Waits, nil
	case int(model.Leaves):
		return model.Leaves, nil
	}

	return 0, &apierror.ValidationError{
		RowNumber: p.rowNumber,
		UserMsg:   apierror.ErrUnknownEventType,
	}
}

func (p *FileParser) parseClientData(eventType model.IncomingEventType, s string) (model.ClientData, error) {
	content := strings.Split(s, p.cfg.EventInfoSeparator)
	switch len(content) {
	case 0:
		return nil, nil
	case model.GetValidClientDataSize(eventType):
		break
	default:
		return nil, &apierror.ParseError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrClientDataInvalidFormat,
		}
	}

	var clientData model.ClientData
	name := content[0]
	switch eventType {
	case model.Arrives:
		clientData = model.NewClientArrives(name)
	case model.Sits:
		table, err := strconv.Atoi(content[1])
		if err != nil {
			return nil, &apierror.ParseError{
				RowNumber: p.rowNumber,
				UserMsg:   apierror.ErrFailedToParseClientTableNumber,
				BaseErr:   err,
			}
		}

		clientData = model.NewClientSits(name, table, p.maxTables)
	case model.Waits:
		clientData = model.NewClientWaits(name)
	case model.Leaves:
		clientData = model.NewClientLeaves(name)
	default:
		return nil, &apierror.ValidationError{
			RowNumber: p.rowNumber,
			UserMsg:   apierror.ErrUnknownEventType,
		}
	}

	if e := clientData.Validate(); e != nil {
		return nil, &apierror.ValidationError{
			RowNumber: p.rowNumber,
			UserMsg:   e.Error(),
		}
	}

	return clientData, nil
}
