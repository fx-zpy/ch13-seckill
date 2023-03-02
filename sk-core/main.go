package main

import (
	"github.com/fx-zpy/SecKillGolang/sk-core/setup"
)

func main() {

	setup.InitZk()
	setup.InitRedis()
	setup.RunService()

}
