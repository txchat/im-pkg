package log

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

//The kinds of log will be printed, enum:
//trace
//debug
//info
//warn
//error
//fatal
//panic
//benchmark

//The kinds of log format will be displayed
const (
	DefaultDisplay Display = "json"
	JsonDisplay    Display = "json"
	ConsoleDisplay Display = "console"
)

const (
	FileMode    Mode = "file"
	ConsoleMode Mode = "console"
)

type Level string

func (l Level) SetLogLevel() error {
	switch s := strings.ToLower(string(l)); s {
	case "benchmark":
		zerolog.SetGlobalLevel(zerolog.Disabled)
	case "":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	default:
		level, err := zerolog.ParseLevel(s)
		if err != nil {
			return err
		}
		zerolog.SetGlobalLevel(level)
	}
	return nil
}

type Display string

type Mode string

type Config struct {
	Level   Level //zerolog.Level
	Mode    Mode
	Path    string
	Display Display
}

func Init(cfg Config) (zerolog.Logger, error) {
	var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	switch cfg.Display {
	case ConsoleDisplay:
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	default:
	}

	switch cfg.Mode {
	case ConsoleMode:
	case FileMode:
		fp, err := createLogFile(cfg.Path)
		if err != nil {
			return logger, err
		}
		logger = logger.Output(fp)
	default:
	}

	//Set Level
	err := cfg.Level.SetLogLevel()
	if err != nil {
		return logger, err
	}
	return logger, nil
}

var fds = make([]*os.File, 0)

func createLogFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		return nil, err
	}
	//defer f.Close()
	fds = append(fds, f)
	return f, nil
}

func Close() {
	for _, fd := range fds {
		fd.Close()
	}
}
