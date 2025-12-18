package share

type Space struct {
	ID     uint
	Name   string `json:"name"`
	UserID string
}

func (Space) TableName() string {
	return "share.spaces"
}
