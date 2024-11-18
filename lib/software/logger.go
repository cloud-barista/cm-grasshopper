package software

import (
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/jollaman999/utils/fileutil"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	logger *log.Logger
	fpLog  *os.File
}

func initLoggerWithUUID(uuid string) (*Logger, error) {
	var migrationLogger = &Logger{}

	logPath := filepath.Join(
		config.CMGrasshopperConfig.CMGrasshopper.Software.LogFolder,
		uuid,
	)

	return migrationLogger.Init(logPath, "migration.log")
}

const (
	// INFO : Informational messages printed with green.
	INFO = "INFO"
	// DEBUG : Debugging messages printed with teal.
	DEBUG = "DEBUG"
	// WARN : Warning messages printed with yellow.
	WARN = "WARN"
	// ERROR : Error messages printed with red.
	ERROR = "ERROR"
	// CRITICAL : Critical messages printed with magenta.
	CRITICAL = "CRITICAL"
	// NONE : Print normal messages with none of color and date, prefix.
	NONE = "NONE"
)

var date string
var _time string

func getDateAndTime() {
	now := time.Now()

	year := fmt.Sprintf("%04d", now.Year())
	month := fmt.Sprintf("%02d", now.Month())
	day := fmt.Sprintf("%02d", now.Day())

	hour := fmt.Sprintf("%02d", now.Hour())
	minute := fmt.Sprintf("%02d", now.Minute())
	second := fmt.Sprintf("%02d", now.Second())

	date = year + "/" + month + "/" + day
	_time = hour + ":" + minute + ":" + second
}

func (l *Logger) getPrefix(logLevel string) string {
	getDateAndTime()

	switch logLevel {
	case INFO:
		return "[ INFO ] "
	case DEBUG:
		return "[ DEBUG ] "
	case WARN:
		return "[ WARN ] "
	case ERROR:
		return "[ ERROR ] "
	case CRITICAL:
		return "[ CRITICAL ] "
	case NONE:
		fallthrough
	default:
		return ""
	}
}

// Print : Print the log with a colored level without new line
func (l *Logger) Print(logLevel string, msg ...interface{}) {
	if l == nil {
		return
	}

	if logLevel == NONE {
		_, _ = fmt.Fprint(io.Writer(l.fpLog), l.getPrefix(logLevel)+fmt.Sprint(msg...))
	} else {
		getDateAndTime()
		_, _ = fmt.Fprint(io.Writer(l.fpLog), date+" "+_time+" [ "+logLevel+" ] "+fmt.Sprint(msg...))
	}
}

// Println : Print the log with a colored level with new line
func (l *Logger) Println(logLevel string, msg ...interface{}) {
	if l.logger == nil {
		return
	}
	if logLevel == NONE {
		_, _ = fmt.Fprintln(io.Writer(l.fpLog), l.getPrefix(logLevel)+fmt.Sprint(msg...))
	} else {
		l.logger.Println(l.getPrefix(logLevel) + fmt.Sprint(msg...))
	}
}

// Printf : Print the formatted log with a colored level
func (l *Logger) Printf(logLevel string, format string, a ...any) {
	if l.logger == nil {
		return
	}
	if logLevel == NONE {
		_, _ = fmt.Fprintf(io.Writer(l.fpLog), l.getPrefix(logLevel)+format, a...)
	} else {
		l.logger.Printf(l.getPrefix(logLevel)+format, a...)
	}
}

func (l *Logger) closeLogFile() {
	if l.fpLog != nil {
		_ = l.fpLog.Close()
	}
}

// Fatal : Print the log with a colored level then exit with return value 1
func (l *Logger) Fatal(logLevel string, exitCode int, msg ...interface{}) {
	l.Print(logLevel, msg...)
	l.closeLogFile()
	os.Exit(exitCode)
}

// Fatalln : Print the log with a colored level with new line then exit with return value 1
func (l *Logger) Fatalln(logLevel string, exitCode int, msg ...interface{}) {
	l.Println(logLevel, msg...)
	l.closeLogFile()
	os.Exit(exitCode)
}

// Fatalf : Print the formatted log with a colored level then exit with return value 1
func (l *Logger) Fatalf(logLevel string, exitCode int, format string, a ...any) {
	l.Printf(logLevel, format, a...)
	l.closeLogFile()
	os.Exit(exitCode)
}

// Panic : Print the log with a colored level then call panic()
func (l *Logger) Panic(logLevel string, msg ...interface{}) {
	l.Print(logLevel, msg...)
	l.closeLogFile()
	panic(fmt.Sprint(msg...))
}

// Panicln : Print the log with a colored level with new line then call panic()
func (l *Logger) Panicln(logLevel string, msg ...interface{}) {
	l.Println(logLevel, msg...)
	l.closeLogFile()
	panic(fmt.Sprintln(msg...))
}

// Panicf : Print the formatted log with a colored level then call panic()
func (l *Logger) Panicf(logLevel string, format string, a ...any) {
	l.Printf(logLevel, format, a...)
	l.closeLogFile()
	panic(fmt.Sprintf(format, a...))
}

// Init : Initialize log file
func (l *Logger) Init(logPath, logFileName string) (*Logger, error) {
	var err error

	if _, err = os.Stat(logPath); os.IsNotExist(err) {
		err = fileutil.CreateDirIfNotExist(logPath)
		if err != nil {
			return nil, err
		}
	}

	l.fpLog, err = os.OpenFile(filepath.Join(logPath, logFileName), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		l.logger = log.New(io.Writer(os.Stdout), "", log.Ldate|log.Ltime)
		return nil, err
	}

	l.logger = log.New(io.Writer(l.fpLog), "", log.Ldate|log.Ltime)

	return l, nil
}

// Close : Close log file
func (l *Logger) Close() {
	if l.logger != nil {
		l.closeLogFile()
	}
}
