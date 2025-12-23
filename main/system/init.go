package system

import (
	"github.com/bim-z/mathrock/main/system/box"
	"github.com/bim-z/mathrock/main/system/db"
	zzz "github.com/charmbracelet/log"
)

// initialize all system requirments
func init() {
	if err := box.Setup(); err != nil {
		zzz.Fatal(err.Error())
	}

	if err := db.Setup(); err != nil {
		zzz.Fatal(err.Error())
	}
}
