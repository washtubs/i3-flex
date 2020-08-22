package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"go.i3wm.org/i3/v4"
)

type i3FlexRenderer struct{}

type ByCurrent []*FlexItem

func (b ByCurrent) Len() int               { return len(b) }
func (b ByCurrent) Less(i int, j int) bool { return b[i].current < b[j].current }
func (b ByCurrent) Swap(i int, j int)      { b[i], b[j] = b[j], b[i] }

func (r *i3FlexRenderer) Render(models []*FlexModel) {
	// Batch all resize commands
	sb := strings.Builder{}
	for _, model := range models {
		r.render(model, &sb)
	}

	//log.Printf("i3-msg %s", sb.String())
	//_, err := i3.RunCommand(sb.String())
	//if err != nil {
	//log.Printf("Error rendering: %s", err.Error())
	//}
}

func (r *i3FlexRenderer) render(model *FlexModel, cmd *strings.Builder) {
	resizeDirection := "height"
	if model.direction == Horizontal {
		resizeDirection = "width"
	}

	for _, v := range model.items {
		log.Printf("item current %d", v.current)
	}
	byCurrent := make(ByCurrent, len(model.items))
	copy(byCurrent, model.items)
	sort.Sort(sort.Reverse(byCurrent))

	// Rescale from normal to percents (100 scale)
	scaled := make([]*int, len(byCurrent))
	for i, item := range byCurrent {
		scaled[i] = new(int)
		*scaled[i] = int(item.current)
	}
	checkScale(scaled, normal)
	rescale(scaled, normal, 100)

	retryCmds := make([]string, 0)
	for i, item := range byCurrent {
		//cmd.WriteString(fmt.Sprintf("[con_id=%d] resize set %s %d ppt; ", item.id, resizeDirection, *scaled[i]))
		cmd := fmt.Sprintf("[con_id=%d] resize set %s %d ppt; ", item.id, resizeDirection, *scaled[i])
		log.Printf("i3-msg %s", cmd)
		_, err := i3.RunCommand(cmd)
		if err != nil {
			retryCmds = append(retryCmds, cmd)
		}
	}
	passes := 3
	for len(retryCmds) > 0 && passes > 0 {
		passes = passes - 1
		retryCmdsTmp := make([]string, 0)
		for _, cmd := range retryCmds {
			_, err := i3.RunCommand(cmd)
			if err != nil {
				retryCmdsTmp = append(retryCmdsTmp, cmd)
				log.Printf("Error rendering: %s", err.Error())
			}
		}
		retryCmds = retryCmdsTmp
	}
}
