package broadcast

type taggedObservation struct {
	sub *subObserver
	ob  interface{}
}

const (
	register = iota
	unregister
	purge
)

type taggedRegReq struct {
	sub     *subObserver
	ch      chan<- interface{}
	regType int
}

// A MuxObserver multiplexes several streams of observations onto a
// single delivery goroutine.
type MuxObserver struct {
	subs  map[*subObserver]map[chan<- interface{}]bool
	reg   chan taggedRegReq
	input chan taggedObservation
}

// NewMuxObserver constructs  a new MuxObserver.
//
// qlen is the size of the channel buffer for observations sent into
// the mux observer and reglen is the size of the channel buffer for
// registration/unregistration events.
func NewMuxObserver(qlen, reglen int) *MuxObserver {
	rv := &MuxObserver{
		subs:  map[*subObserver]map[chan<- interface{}]bool{},
		reg:   make(chan taggedRegReq, reglen),
		input: make(chan taggedObservation, qlen),
	}
	go rv.run()
	return rv
}

// Close shuts down this mux observer.
func (m *MuxObserver) Close() error {
	close(m.reg)
	return nil
}

func (m *MuxObserver) broadcast(to taggedObservation) {
	for ch := range m.subs[to.sub] {
		ch <- to.ob
	}
}

func (m *MuxObserver) doReg(tr taggedRegReq) {
	mm, exists := m.subs[tr.sub]
	if !exists {
		mm = map[chan<- interface{}]bool{}
		m.subs[tr.sub] = mm
	}
	mm[tr.ch] = true
}

func (m *MuxObserver) doUnreg(tr taggedRegReq) {
	mm, exists := m.subs[tr.sub]
	if exists {
		delete(mm, tr.ch)
		if len(mm) == 0 {
			delete(m.subs, tr.sub)
		}
	}
}

func (m *MuxObserver) handleReg(tr taggedRegReq) {
	switch tr.regType {
	case register:
		m.doReg(tr)
	case unregister:
		m.doUnreg(tr)
	case purge:
		delete(m.subs, tr.sub)
	}
}

func (m *MuxObserver) run() {
	for {
		select {
		case tr, ok := <-m.reg:
			if ok {
				m.handleReg(tr)
			} else {
				return
			}
		default:
			select {
			case to := <-m.input:
				m.broadcast(to)
			case tr, ok := <-m.reg:
				if ok {
					m.handleReg(tr)
				} else {
					return
				}
			}
		}
	}
}

// Sub creates a new sub-broadcaster from this MuxObserver.
func (m *MuxObserver) Sub() Broadcaster {
	return &subObserver{m}
}

type subObserver struct {
	mo *MuxObserver
}

func (s *subObserver) Register(ch chan<- interface{}) {
	s.mo.reg <- taggedRegReq{s, ch, register}
}

func (s *subObserver) Unregister(ch chan<- interface{}) {
	s.mo.reg <- taggedRegReq{s, ch, unregister}
}

func (s *subObserver) Close() error {
	s.mo.reg <- taggedRegReq{s, nil, purge}
	return nil
}

func (s *subObserver) Submit(ob interface{}) {
	s.mo.input <- taggedObservation{s, ob}
}
