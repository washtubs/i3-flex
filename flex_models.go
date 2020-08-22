package main

import (
	"log"

	"go.i3wm.org/i3/v4"
)

// The in-memory store for all flex models
type FlexModels struct {
	models   map[i3.NodeID]*FlexModel
	renderer FlexRenderer
}

func (f *FlexModels) RegisterRenderer(renderer FlexRenderer) { f.renderer = renderer }

func (f *FlexModels) Updates(updates []FlexUpdate, full bool) {
	markForPrune := make(map[i3.NodeID]bool)
	// If full, prune all by default. Otherwise none
	for k, _ := range f.models {
		markForPrune[k] = full
	}
	// Prune depending on whether the update invalidates the model
	for _, update := range updates {
		model, ok := f.models[update.ExternalId]
		if ok {
			markForPrune[update.ExternalId] = f.isInvalidated(update, model)
		}
	}
	// Prune everything that's been marked
	for k, v := range markForPrune {
		if v {
			delete(f.models, k)
		}
	}

	for _, update := range updates {
		f.update(update)
	}
}

func (f *FlexModels) isInvalidated(update FlexUpdate, model *FlexModel) bool {
	if update.ExternalId != model.id {
		panic("ExternalId must be the same as id")
	}
	if update.Direction != model.direction {
		// Invalidate it if it changed directions somehow
		return true
	}
	if len(update.Items) != len(model.items) {
		return true
	}
	dupeCheck := make(map[i3.NodeID]bool)
	for _, item := range model.items {
		dupeCheck[item.id] = true
	}
	for _, item := range update.Items {
		_, ok := dupeCheck[item.ExternalId]
		if !ok {
			// Unique item
			return true
		}
	}

	return false
}

func (f *FlexModels) update(update FlexUpdate) {
	scaled := make([]*int, 0, len(update.Items))
	for _, item := range update.Items {
		sizeCopy := item.Size
		scaled = append(scaled, &sizeCopy)
	}
	rescale(scaled, 0, normal) // infer the scale from the total length
	checkScale(scaled, normal)

	model, ok := f.models[update.ExternalId]
	if ok {
		// Update existing sizes
		events := make([]FlexEvent, 0)
		for i, itemUpdate := range update.Items {
			for _, item := range model.items {
				// TODO: O(n^2), I know who cares, but it could be bad
				if itemUpdate.ExternalId == item.id {
					increase := Size(*scaled[i]) - item.current
					if increase > 0 {
						log.Printf("increase %d", increase)
						events = append(events, FlexEvent{
							item.id,
							increase,
						})
					}
					break
				}
			}
		}
		if len(events) > 0 {
			model.OnUpdate(events)
		}
	} else {
		items := make([]*FlexItem, 0, len(update.Items))
		for i, itemUpdate := range update.Items {
			items = append(items, &FlexItem{
				id:      itemUpdate.ExternalId,
				current: Size(*scaled[i]),
			})
		}
		model := &FlexModel{
			id:          update.ExternalId,
			direction:   update.Direction,
			globals:     globals,
			items:       items,
			constraints: make([]MinItemConstraint, 0),
		}
		f.models[update.ExternalId] = model
	}
}

func (f *FlexModels) OnFocus(id i3.NodeID) {
	found := false
	toRender := make([]*FlexModel, 0)
	var firstModel *FlexModel = nil
	// Find the id as an item in one of the models
	for _, model := range f.models {
		for i, item := range model.items {
			if item.id == id {
				found = true
				log.Printf("Flexing [%s] model [%d]-->[%d]", model.direction, model.id, id)
				rerender := model.Flex(i)
				if rerender {
					toRender = append(toRender, model)
				}
				firstModel = model
				break
			}
		}
		if found {
			break
		}
	}

	// Search for another model with this firstModel as a child
	// and the direction being different
	found = false
	for _, model := range f.models {
		for i, item := range model.items {
			if item.id == firstModel.id && model.direction != firstModel.direction {
				found = true
				log.Printf("Flexing [%s] parent model [%d]-->[%d]", model.direction, model.id, id)
				rerender := model.Flex(i)
				if rerender {
					toRender = append(toRender, model)
				}
				break
			}
		}
		if found {
			break
		}
	}

	if len(toRender) > 0 {
		f.renderer.Render(toRender)
	}
}

func initFlexModels() *FlexModels {
	return &FlexModels{
		models:   make(map[i3.NodeID]*FlexModel),
		renderer: &fakeRenderer{},
	}
}
