package main

import "log"

type FlexRenderer interface {
	Render(models []*FlexModel)
}

type fakeRenderer struct{}

func (f *fakeRenderer) Render(models []*FlexModel) {
	log.Printf("Fake rendering %d models.\n %+v", len(models), models)
}
