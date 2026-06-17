package main

import (
	"github.com/huypham67/bookmark-common/pkg/common"
	"github.com/huypham67/bookmark-worker/internal/bootstrap"
)

func main() {
	app, err := bootstrap.NewApp()
	common.ExitOnError(err, "Failed to create application")

	common.ExitOnError(app.Run(), "Failed to run application")
}
