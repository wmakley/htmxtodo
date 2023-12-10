package lists

import (
	"fmt"
	"htmxtodo/gen/htmxtodo_dev/public/model"
)

type CardProps struct {
	model.List
	EditingName bool
}

func (c CardProps) ListUrl() string {
	return fmt.Sprintf("/lists/%d", c.List.ID)
}

func (c CardProps) EditListUrl() string {
	return fmt.Sprintf("/lists/%d/edit", c.List.ID)
}

func (c CardProps) Id() string {
	return fmt.Sprintf("card-%d", c.List.ID)
}

func (c CardProps) Selector() string {
	return fmt.Sprintf("#card-%d", c.List.ID)
}
