package processor

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"time"
	"yadro-intern/cmd/config"
	"yadro-intern/internal/apierror"
	"yadro-intern/internal/ifces"
	"yadro-intern/internal/model"
	"yadro-intern/internal/storage"
)

type EventProcessor interface {

	// ProcessEvents processes events from the channel
	// and stores the result in some storage, for example, in a file.
	//
	// It returns an error if an error occurs while processing events.
	ProcessEvents(<-chan model.WrappedIncomingEvent) error

	// ShowRevenue displays the final result of the program.
	//
	// Used, when all events are processed.
	ShowRevenue()
}

type EventProcessorImpl struct {
	out      io.Writer
	cfg      *config.Processor
	coreData *model.CoreData

	// tables is mapper from table number to sit event.
	// we need to know, when and who sat at the table.
	tables storage.Storage[int, *model.IncomingEvent]

	// clients used to say, that client is entered in the computer club.
	// when client is entered, he doesn't have a table yet, but we need to know,
	// that he is in the club.
	clients storage.Storage[string, int]

	// revenue is mapper from table number to revenue stats.
	// we need to know, how much money we earned from each table.
	//
	// called, when client has changed his table or left the club.
	revenue storage.Storage[int, *model.RevenueStats]

	waitingQueue storage.Queue[model.ClientData]
}

func NewEventProcessor(
	out io.Writer,
	cfg *config.Processor,
	coreData *model.CoreData,
	tablesStorage storage.Storage[int, *model.IncomingEvent],
	revenueStorage storage.Storage[int, *model.RevenueStats],
	clientsStorage storage.Storage[string, int],
	clientsQueue storage.Queue[model.ClientData],
) *EventProcessorImpl {
	return &EventProcessorImpl{
		out:          out,
		coreData:     coreData,
		cfg:          cfg,
		tables:       tablesStorage,
		clients:      clientsStorage,
		revenue:      revenueStorage,
		waitingQueue: clientsQueue,
	}
}

func (p *EventProcessorImpl) ProcessEvents(events <-chan model.WrappedIncomingEvent) error {
	_, _ = io.WriteString(p.out, p.coreData.WorkingTime.Start.Format(p.cfg.TimeFormat)+"\n")

	for wrapped := range events {
		if wrapped.Err != nil {
			return wrapped.Err
		}

		p.writeOutEvent(wrapped.Event)
		p.processEvent(wrapped.Event)
	}

	p.leaveClients()

	_, _ = io.WriteString(p.out, p.coreData.WorkingTime.End.Format(p.cfg.TimeFormat)+"\n")
	return nil
}

func (p *EventProcessorImpl) ShowRevenue() {
	for i := 1; i <= p.coreData.TablesCount; i++ {
		stats, ok := p.revenue.Get(i)
		if !ok {
			continue
		}

		_, _ = fmt.Fprintf(p.out, "%d %s\n", i, stats)
	}
}

func (p *EventProcessorImpl) leaveClients() {
	var clients = make([]string, 0, p.clients.Len()+p.waitingQueue.Len())
	for _, pair := range p.clients.GetAll() {
		clients = append(clients, pair.Key)
	}

	for p.waitingQueue.Len() > 0 {
		client, _ := p.waitingQueue.Pop()
		clients = append(clients, client.GetName())
	}

	sort.Strings(clients)

	for _, clientName := range clients {
		leaveEvent := model.NewIncomingEvent(p.coreData.WorkingTime.End, model.Leaves, model.NewClientLeaves(clientName))

		if _, ok := p.clients.Get(clientName); !ok {
			p.writeOutEvent(leaveEvent)
			continue
		}

		p.processLeaves(leaveEvent, true)
	}
}

func (p *EventProcessorImpl) processEvent(event *model.IncomingEvent) {
	switch event.Type {
	case model.Arrives:
		p.processArrives(event)
	case model.Sits:
		p.processSits(event, false)
	case model.Waits:
		p.processWaits(event)
	case model.Leaves:
		p.processLeaves(event, false)
	}
}

func (p *EventProcessorImpl) writeOutEvent(event ifces.TimeFormatter) {
	if p.out == nil {
		return
	}

	_, _ = io.WriteString(p.out, event.String(p.cfg.TimeFormat)+"\n")
}

func (p *EventProcessorImpl) processArrives(event *model.IncomingEvent) {
	if !p.coreData.WorkingTime.In(event.HappensAt) {
		notOpenYet := model.NewErrorEvent(event.HappensAt, errors.New(apierror.ErrNotOpenYet))
		p.writeOutEvent(notOpenYet)
		return
	}

	if _, ok := p.clients.Get(event.Client.GetName()); ok {
		alreadyIn := model.NewErrorEvent(event.HappensAt, errors.New(apierror.ErrYouShallNotPass))
		p.writeOutEvent(alreadyIn)
		return
	}

	p.clients.Set(event.Client.GetName(), -1)
}

func (p *EventProcessorImpl) processSits(event *model.IncomingEvent, generateSatEvent bool) {
	if _, ok := p.clients.Get(event.Client.GetName()); !ok {
		unknownClient := model.NewErrorEvent(event.HappensAt, errors.New(apierror.ErrClientUnknown))
		p.writeOutEvent(unknownClient)
		return
	}

	clientSits := event.Client.(*model.ClientSits)
	if _, ok := p.tables.Get(clientSits.GetTable()); ok {
		alreadyTaken := model.NewErrorEvent(event.HappensAt, errors.New(apierror.ErrTableIsBusy))
		p.writeOutEvent(alreadyTaken)
		return
	}

	prevSitTable, ok := p.clients.Get(event.Client.GetName())
	if ok && prevSitTable != -1 {
		p.updateRevenue(prevSitTable, event.HappensAt)
		p.clients.Delete(event.Client.GetName())
		p.tables.Delete(prevSitTable)
	}

	p.tables.Set(clientSits.GetTable(), event)
	p.clients.Set(event.Client.GetName(), clientSits.GetTable())

	if generateSatEvent {
		p.writeOutEvent(model.NewClientSatEvent(event.HappensAt, clientSits))
	}
}

func (p *EventProcessorImpl) processWaits(event *model.IncomingEvent) {
	if p.tables.Len() < p.coreData.TablesCount {
		haveFreeTables := model.NewErrorEvent(event.HappensAt, errors.New(apierror.ErrCantWaitLonger))
		p.writeOutEvent(haveFreeTables)
		return
	}

	if p.waitingQueue.Len() >= p.coreData.TablesCount {
		queueIsFull := model.NewClientLeftEvent(event.HappensAt, event.Client)
		p.writeOutEvent(queueIsFull)
		return
	}

	p.waitingQueue.Push(event.Client)
}

func (p *EventProcessorImpl) processLeaves(event *model.IncomingEvent, generateLeftEvent bool) {
	busyTable, ok := p.clients.Get(event.Client.GetName())
	if !ok {
		unknownClient := model.NewErrorEvent(event.HappensAt, errors.New(apierror.ErrClientUnknown))
		p.writeOutEvent(unknownClient)
		return
	}

	p.clients.Delete(event.Client.GetName())
	if generateLeftEvent {
		p.writeOutEvent(model.NewClientLeftEvent(event.HappensAt, event.Client))
	}

	if busyTable == -1 {
		return
	}

	p.updateRevenue(busyTable, event.HappensAt)
	p.tables.Delete(busyTable)

	if p.waitingQueue.Len() > 0 {
		client, _ := p.waitingQueue.Pop()
		sitClientData := model.NewClientSits(client.GetName(), busyTable, p.coreData.TablesCount)
		sitEvent := model.NewIncomingEvent(event.HappensAt, model.Sits, sitClientData)
		p.clients.Set(client.GetName(), -1)
		p.processSits(sitEvent, true)
	}
}

func (p *EventProcessorImpl) updateRevenue(busyTable int, releaseTime time.Time) {
	if busyTable == -1 {
		return
	}

	sittingEvent, ok := p.tables.Get(busyTable)
	if !ok {
		return
	}

	sittingTime := int(math.Ceil(releaseTime.Sub(sittingEvent.HappensAt).Hours()))
	prevRevenue, ok := p.revenue.Get(busyTable)
	if !ok {
		prevRevenue = &model.RevenueStats{
			Income:    0,
			UsageTime: time.Duration(0),
		}
	}

	p.revenue.Set(busyTable, &model.RevenueStats{
		Income:    prevRevenue.Income + p.coreData.PricePerHour*sittingTime,
		UsageTime: prevRevenue.UsageTime + releaseTime.Sub(sittingEvent.HappensAt),
	})
}
