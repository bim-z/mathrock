package share

import (
	"fmt"
	"net/http"

	"github.com/bim-z/mathrock/main/system/valid"
	"github.com/labstack/echo/v4"
)

func push(c echo.Context) (err error) {
	req := struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}{}

	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err = valid.Valid.Struct(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	form, err := c.MultipartForm()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	files := form.File["files"]

	// must be the same
	if len(files) != req.Count {
		return fmt.Errorf("")
	}

	for _, file := range files {
	}
}
