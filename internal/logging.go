package logging

import (
	"fmt"
	"log"
)

func LogErrorf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	log.Println(err)
	return err
}
