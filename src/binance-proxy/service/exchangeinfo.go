package service

import (
	"context"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"binance-proxy/tool"

	log "github.com/sirupsen/logrus"
)

type ExchangeInfoSrv struct {
	rw sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	initCtx  context.Context
	initDone context.CancelFunc

	refreshDur   time.Duration
	si           *symbolInterval
	exchangeInfo []byte
}

func NewExchangeInfoSrv(ctx context.Context, si *symbolInterval) *ExchangeInfoSrv {
	s := &ExchangeInfoSrv{
		si:         si,
		refreshDur: 3600 * time.Second,
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.initCtx, s.initDone = context.WithCancel(context.Background())

	return s
}

func (s *ExchangeInfoSrv) Start() {
	s.reTryRefreshExchangeInfo()

	go func() {
		rTimer := time.NewTimer(s.refreshDur)
		for {
			rTimer.Reset(s.refreshDur)
			select {
			case <-s.ctx.Done():
				rTimer.Stop()
				return
			case <-rTimer.C:
			}

			s.reTryRefreshExchangeInfo()
		}
	}()
}

// Nothing to do
func (s *ExchangeInfoSrv) Stop() {}

func (s *ExchangeInfoSrv) GetExchangeInfo() []byte {
	<-s.initCtx.Done()
	s.rw.RLock()
	defer s.rw.RUnlock()

	return s.exchangeInfo
}

func (s *ExchangeInfoSrv) reTryRefreshExchangeInfo() {
	for d := tool.NewDelayIterator(); ; d.Delay() {
		if s.refreshExchangeInfo() == nil {
			break
		}
	}
}

func (s *ExchangeInfoSrv) refreshExchangeInfo() error {
	var url string
	if s.si.Class == SPOT {
		url = "https://api.binance.com/api/v3/exchangeInfo"
		RateWait(s.ctx, s.si.Class, http.MethodGet, "/api/v3/exchangeInfo", nil)
	} else {
		url = "https://fapi.binance.com/fapi/v1/exchangeInfo"
		RateWait(s.ctx, s.si.Class, http.MethodGet, "/fapi/v1/exchangeInfo", nil)
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Errorf("%s exchangeInfo init error!Error:%s", s.si, err)
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	s.rw.Lock()
	defer s.rw.Unlock()

	if s.exchangeInfo == nil {
		defer s.initDone()
	}

	s.exchangeInfo = data

	log.Debugf("%s exchangeInfo refresh success!", s.si)

	return nil
}
