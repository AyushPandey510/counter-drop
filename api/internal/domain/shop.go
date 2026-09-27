package domain

type Shop struct {
	ID           string         `json:"id"`
	Slug         string         `json:"slug"`
	Name         string         `json:"name"`
	IntakePaused bool           `json:"intakePaused"`
	Prices       map[string]int `json:"prices"`
	WaitMinutes  int            `json:"waitMinutes"`
}
