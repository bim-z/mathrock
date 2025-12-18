package share

type File struct {
	ID   uint
	Name string
	Hash string
}

func (File) TableName() string {
	return "share.files"
}
