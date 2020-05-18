---
tags: golang, logging
---
# Logging in Cobra
## Introduction
I recenlty wrote some CLI app based on Cobra. A nice logging framework is what I was looking for to support the structured logging in my CLI app. I have tried several including `klog`, `golang log`, `logrus`. But I end up with using `logrus` for my app.
## Klog try-out
https://github.com/kubernetes/klog
Klog seems not designed for CLI application logging. From my understanding, it is specifically for K8S logging, especially when you write a CRD controller, then klog might be the one you want to use.
### Configure klog in root.go
```
func init() {
	cobra.OnInitialize(initLoggingFlags)

	rootCmd.PersistentFlags().BoolVarP(&globalflags.LogQuietlyVar, globalflags.LogQuietlyFlag, "q", false, "Quiet (no output)")
	rootCmd.PersistentFlags().StringVarP(&globalflags.LogFileVar, globalflags.LogFileFlag, "", "", "log file")
	rootCmd.PersistentFlags().IntVarP(&globalflags.VerbosityVar, globalflags.VerbosityFlag, "v", 0, "verbosity")
}

func initLoggingFlags() {
	fmt.Println("quite:", globalflags.LogQuietlyVar)
	fmt.Println("logFile:", globalflags.LogFileVar)
	fmt.Println("verbosity:", globalflags.VerbosityVar)

	klog.InitFlags(nil)
	// https://github.com/kubernetes/klog/blob/master/klog.go
	// If true, avoid header prefixes in the log messages
	flag.Set("skip_headers", "false")
	// If true, avoid headers when opening log files
	flag.Set("skip_log_headers", "true")
	// If true, logs are written to standard error instead of to files.
	flag.Set("logtostderr", "false")
	// Fully qualified log file name
	flag.Set("log_file", parseLogFile(globalflags.LogFileVar))
	// If true, logs are written to standard error as well as to files.
	flag.Set("alsologtostderr", strconv.FormatBool(!globalflags.LogQuietlyVar))
	// Enable V-leveled logging at the specified level.
	flag.Set("v", strconv.Itoa(globalflags.VerbosityVar))

	flag.Parse()
}

func parseLogFile(logFileName string) string {
	if len(logFileName) != 0 {
		basePath := path.Dir(logFileName)
		if os.MkdirAll(basePath, 0777) != nil {
			fmt.Println("Unable to create log directory: " + basePath)
			// We should not proceed if there is IO error to create log directory
			os.Exit(1)
		}

		file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
		if err != nil {
			fmt.Println("Unable to create log file for write.", "Error: ", err)
			// We should not proceed if there is IO error to create log file
			os.Exit(1)
		}
		defer file.Close()

		return logFileName
	}
	// Use the same format as tkg cli
	return fmt.Sprintf("kmigrator-%s.log", time.Now().Format("20060102T150405"))
}
```

### Wrap the klog somewhere else
```
package logging

import (
	"log"

	"k8s.io/klog/v2"
)

type CLILogger interface {
	Info(args ...interface{})
	Warning(args ...interface{})
	Error(args ...interface{})
	V(level klog.Level) klog.Verbose
	SendStatusUpdate(args ...interface{})
	Flush()
}

var Logger = kLogger{}

type kLogger struct {
}

func (l kLogger) SendStatusUpdate(args ...interface{}) {
	log.Println(args)
}

func (l kLogger) Info(args ...interface{}) {
	klog.Info(args)
	klog.Flush()
}

func (l kLogger) Warning(args ...interface{}) {
	klog.Warning(args)
	klog.Flush()
}

func (l kLogger) Error(args ...interface{}) {
	klog.Warning(args)
	klog.Flush()
}

func (l kLogger) V(level klog.Level) klog.Verbose {
	return klog.V(level)
}

func (l kLogger) Flush() {
	klog.Flush()
}

```
### Use the klog in your go code under pkg
```
logging.Logger.Info("TEST")

Output:
I0428 19:58:31.497039   88647 logging.go:28] [TEST]
I0428 19:58:31.510630   88647 logging.go:28] [TEST1]
I0428 19:58:31.516909   88647 logging.go:28] [TEST2]
I0428 19:58:31.528549   88647 logging.go:28] [TEST3]
```
### Problems
* As you can see from the log output. No mater where you do the logging, it always shows `logging.go:28`.
* You definitely could not wrap the klog as the step two above. But it ends up with that NOT all go code picks up the same klog configuration. In another word, if you do `klog.Info` in `package A`, and `klog.Info` in `package B`, only one of them will pick up the klog config you have done in your `root.go`. However, you still can pass the logger reference around, but it is really ugly.

## logrus try-out
https://github.com/sirupsen/logrus
https://le-gall.bzh/post/go/integrating-logrus-with-cobra/
https://esc.sh/blog/golang-logging-using-logrus/
https://github.com/GoogleContainerTools/skaffold/blob/69776b15674898a87ac61b9431f93ee68cffa6fd/cmd/skaffold/app/cmd/cmd.go#L51-L53

The main reason I decided to use logrus is that `skaffold` also uses it. And it works perfect in my Cobra CLI app.
### Configure logrus in root.go
```
func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := setUpLogger(); err != nil {
			return err
		}
		return nil
	}

	rootCmd.PersistentFlags().StringVarP(&logFile, "log-file", "", "", "The log file where to store the log output")
	rootCmd.PersistentFlags().BoolVarP(&quite, "quite", "q", false, "Mute the log output from stdout/stderr")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "", logrus.InfoLevel.String(), "The log level")
}

// setUpLogs set the log output ans the log level
func setUpLogger() error {

	// Setup the logger output
	if len(logFile) == 0 {
		logFile = "someCLIApp-" + time.Now().Format("20060102T150405") + ".log"
	} else {
		basePath := path.Dir(logFile)
		if err := os.MkdirAll(basePath, 0777); err != nil {
			return err
		}
	}
	var f *os.File
	var err error

	if f, err = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
		// Using fmt to print to stdout since logger is not ready
		fmt.Println(err)
		return err
	}

	if quite {
		logger.SetOutput(f)
	} else {
		mw := io.MultiWriter(os.Stdout, f)
		logger.SetOutput(mw)
	}

	// Setup the logger level
	lvl, err := logger.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	logger.SetLevel(lvl)

	// Setup the logger format
	formatter := new(logger.TextFormatter)
	formatter.TimestampFormat = "02-01-2006 15:04:05"
	formatter.FullTimestamp = true
	logger.SetFormatter(formatter)

	return nil
}
```
### Use logrus in your code
```
import logger "github.com/sirupsen/logrus"
logger.Errorf("Unable to get namespaces for inspection. Error: %v", err)
```

That is it!!! All your code will pick up the same logrus config.