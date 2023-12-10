package view

import (
	"fmt"
	"htmxtodo/gen/htmxtodo_dev/public/model"
)

type Card struct {
	EditingName bool
	List        model.List
}

func (c Card) DeleteUrl() string {
	return fmt.Sprintf("/lists/%d", c.List.ID)
}

func (c Card) PatchUrl() string {
	return fmt.Sprintf("/lists/%d", c.List.ID)
}

func (c Card) EditUrl() string {
	return fmt.Sprintf("/lists/%d/edit", c.List.ID)
}

func (c Card) Id() string {
	return fmt.Sprintf("card-%d", c.List.ID)
}

func (c Card) Selector() string {
	return fmt.Sprintf("#card-%d", c.List.ID)
}
