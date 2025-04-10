package application

import "errors"

type MultiErr struct {
	Errs []error
}

func (m *MultiErr) Add(err error) {
	if err != nil {
		m.Errs = append(m.Errs, err)
	}
}

func (m *MultiErr) Dump(out *Output) {
	for _, err := range m.Errs {
		out.Log(err.Error())
	}
}

func (m *MultiErr) Err() (err error) {
	for _, errN := range m.Errs {
		errors.Join(err, errN)
	}

	return
}

func (m *MultiErr) HasErrors() bool {
	return len(m.Errs) > 0
}

func (m *MultiErr) Join(me *MultiErr) {
	for _, e := range me.Errs {
		m.Add(e)
	}
}
