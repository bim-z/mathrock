package system

import (
	"log"

	"github.com/bim-z/mathrock/main/system/box"
	"github.com/bim-z/mathrock/main/system/db"
)

// initialize all system requirments
func init() {
	if err := box.Setup(); err != nil {
		log.Fatal(err.Error())
	}

	if err := db.Setup(); err != nil {
		log.Fatal(err.Error())
	}
}
