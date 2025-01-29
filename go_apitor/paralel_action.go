package go_apitor

import "sync"

type ParalelQuery struct {
	sync.Mutex
	w   sync.WaitGroup
	Err []error
}

func NewParalelAction() *ParalelQuery {
	return &ParalelQuery{
		Mutex: sync.Mutex{},
		Err:   []error{},
		w:     sync.WaitGroup{},
	}
}

func (p *ParalelQuery) Wait() error {
	p.w.Wait()

	if len(p.Err) == 0 {
		return nil
	}

	return p.Err[0]
}

func (p *ParalelQuery) SetError(err error) {
	if err == nil {
		return
	}

	p.Lock()
	defer p.Unlock()

	p.Err = append(p.Err, err)

}

func (p *ParalelQuery) Add(handle func() error) {
	p.w.Add(1)
	go func() {
		defer p.w.Done()
		err := handle()
		p.SetError(err)
	}()
}
