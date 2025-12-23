package box

import (
	"fmt"
	"os"

	"github.com/bim-z/mathrock/main/system/box/filesystem"
	gf "github.com/bim-z/mathrock/main/system/box/gridfs"
	"github.com/bim-z/mathrock/main/system/box/simple"
)

func Setup() (err error) {
	switch os.Getenv("STORAGE_ENGINE") {
	case "simple":
		if Box, err = simple.Setup(); err != nil {
			return
		}
	case "filesystem":
		Box = filesystem.Setup()
	case "gridfs":
		if Box, err = gf.Setup(); err != nil {
			return
		}
	}

	return fmt.Errorf("you must specify storage engine")
}
