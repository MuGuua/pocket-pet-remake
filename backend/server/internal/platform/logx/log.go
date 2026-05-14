package logx

import (
	"log"
	"os"
)

func New() *log.Logger {
	return log.New(os.Stdout, "[pocketpet] ", log.LstdFlags|log.LUTC)
}
