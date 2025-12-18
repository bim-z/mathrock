package share

type Space struct {
	ID     uint
	Name   string
	UserID string
}

func (Space) TableName() string {
	return "share.space"
}
