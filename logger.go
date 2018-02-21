package pageicon

import "fmt"

// Logger logs information.
type Logger func(string)

var infoLogger Logger

// SetInfoLogger to the given Logger.
func SetInfoLogger(l Logger) {
	infoLogger = l
}

func infofln(format string, values ...interface{}) {
	if infoLogger != nil {
		infoLogger(fmt.Sprintf(fmt.Sprintf("%s\n", format), values...))
	}
}
