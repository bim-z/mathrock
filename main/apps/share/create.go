package share

import (
	"net/http"

	"github.com/bim-z/mathrock/main/system/auth"
	"github.com/bim-z/mathrock/main/system/db"
	"github.com/bim-z/mathrock/main/system/db/model/share"
	"github.com/bim-z/mathrock/main/system/valid"
	"github.com/labstack/echo/v4"
)

// create new space
func Create(c echo.Context) (err error) {
	var space share.Space

	if err = c.Bind(&space); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err = valid.Valid.Struct(space); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	space.UserID = auth.UserId(c)

	if err = db.DB.Create(space).Error; err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return
}
