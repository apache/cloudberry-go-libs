package gplog_test

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"

	"github.com/cloudberrydb/gp-common-go-libs/gplog"
	"github.com/cloudberrydb/gp-common-go-libs/operating"
	"github.com/cloudberrydb/gp-common-go-libs/testhelper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pkg/errors"
)

func TestGpLog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gplog tests")
}

var _ = Describe("logger/log tests", func() {
	var (
		stdout   *gbytes.Buffer
		stderr   *gbytes.Buffer
		logfile  *gbytes.Buffer
		buffer   *gbytes.Buffer
		fakeInfo os.FileInfo
	)
	const defaultLogFile = "testDir/gpAdminLogs/testProgram_20170101.log"

	BeforeEach(func() {
		err := os.MkdirAll("/tmp/log_dir", 0755)
		Expect(err).ToNot(HaveOccurred())
		fakeInfo, err = os.Stat("/tmp/log_dir")
		Expect(err).ToNot(HaveOccurred())

		operating.System.CurrentUser = func() (*user.User, error) { return &user.User{Username: "testUser", HomeDir: "testDir"}, nil }
		operating.System.Getpid = func() int { return 0 }
		operating.System.Hostname = func() (string, error) { return "testHost", nil }
		operating.System.IsNotExist = func(err error) bool { return false }
		operating.System.Now = func() time.Time { return time.Date(2017, time.January, 1, 1, 1, 1, 1, time.Local) }
		operating.System.OpenFileWrite = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) { return buffer, nil }
		operating.System.Stat = func(name string) (os.FileInfo, error) { return fakeInfo, nil }
		stdout, stderr, logfile = testhelper.SetupTestLogger()
	})
	AfterEach(func() {
		operating.System = operating.InitializeSystemFunctions()
	})
	Describe("NewLogger", func() {
		Context("Setting logfile verbosity", func() {
			It("defaults to Debug if no argument is passed", func() {
				gplog.SetLogger(gplog.NewLogger(os.Stdout, os.Stderr, buffer, defaultLogFile,
					gplog.LOGINFO, "testProgram"))
				Expect(gplog.GetLogFileVerbosity()).To(Equal(gplog.LOGDEBUG))
			})
			It("defaults to Debug if too many arguments are passed", func() {
				gplog.SetLogger(gplog.NewLogger(os.Stdout, os.Stderr, buffer, defaultLogFile,
					gplog.LOGINFO, "testProgram", gplog.LOGINFO, gplog.LOGERROR))
				Expect(gplog.GetLogFileVerbosity()).To(Equal(gplog.LOGDEBUG))
			})
			It("defaults to Debug if an invalid argument is passed", func() {
				gplog.SetLogger(gplog.NewLogger(os.Stdout, os.Stderr, buffer, defaultLogFile,
					gplog.LOGINFO, "testProgram", 42))
				Expect(gplog.GetLogFileVerbosity()).To(Equal(gplog.LOGDEBUG))
			})
			It("sets the logfile verbosity if a valid argument is passed", func() {
				gplog.SetLogger(gplog.NewLogger(os.Stdout, os.Stderr, buffer, defaultLogFile,
					gplog.LOGINFO, "testProgram", gplog.LOGINFO))
				Expect(gplog.GetLogFileVerbosity()).To(Equal(gplog.LOGINFO))
			})
		})
	})
	Describe("InitializeLogging", func() {
		BeforeEach(func() {
			gplog.SetLogger(nil)
		})
		Context("Logger initialized with default log directory and Info log level", func() {
			It("creates a new logger writing to gpAdminLogs and sets utils.logger to this new logger", func() {
				gplog.InitializeLogging("testProgram", "")
				Expect(gplog.GetLogFilePath()).To(Equal("testDir/gpAdminLogs/testProgram_20170101.log"))
				Expect(gplog.GetVerbosity()).To(Equal(gplog.LOGINFO))
				Expect(gplog.GetLogFileVerbosity()).To(Equal(gplog.LOGDEBUG))
			})
		})
		Context("Logger initialized with a specified log directory and Info log level", func() {
			It("creates a new logger writing to the specified log directory and sets utils.logger to this new logger", func() {
				expectedFilePath := "/tmp/log_dir/testProgram_20170101.log"
				gplog.InitializeLogging("testProgram", "/tmp/log_dir")
				Expect(gplog.GetLogFilePath()).To(Equal(expectedFilePath))
				Expect(gplog.GetVerbosity()).To(Equal(gplog.LOGINFO))
				Expect(gplog.GetLogFileVerbosity()).To(Equal(gplog.LOGDEBUG))
			})
		})
		Context("Directory or log file does not exist or is not writable", func() {
			It("creates a log directory if given a nonexistent log directory", func() {
				calledWith := ""
				operating.System.IsNotExist = func(err error) bool { return true }
				operating.System.Stat = func(name string) (os.FileInfo, error) {
					calledWith = name
					return fakeInfo, errors.New("file does not exist")
				}
				gplog.InitializeLogging("testProgram", "/tmp/log_dir")
				Expect(calledWith).To(Equal("/tmp/log_dir"))
			})
			It("creates a log file if given a nonexistent log file", func() {
				calledWith := ""
				operating.System.OpenFileWrite = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
					calledWith = name
					return buffer, nil
				}
				operating.System.IsNotExist = func(err error) bool { return true }
				operating.System.Stat = func(name string) (os.FileInfo, error) { return fakeInfo, errors.New("file does not exist") }
				gplog.InitializeLogging("testProgram", "/tmp/log_dir")
				Expect(calledWith).To(Equal("/tmp/log_dir/testProgram_20170101.log"))
			})
			It("panics if given a non-writable log directory", func() {
				operating.System.Stat = func(name string) (os.FileInfo, error) { return fakeInfo, errors.New("permission denied") }
				defer testhelper.ShouldPanicWithMessage("permission denied")
				gplog.InitializeLogging("testProgram", "/tmp/log_dir")
			})
			It("panics if given a non-writable log file", func() {
				operating.System.OpenFileWrite = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
					return nil, errors.New("permission denied")
				}
				defer testhelper.ShouldPanicWithMessage("permission denied")
				gplog.InitializeLogging("testProgram", "/tmp/log_dir")
			})
		})
	})
	Describe("GetLogPrefix", func() {
		It("returns a prefix for the current time", func() {
			expectedMessage := "20170101:01:01:01 testProgram:testUser:testHost:000000-[INFO]:-"
			prefix := gplog.GetLogPrefix("INFO")
			Expect(expectedMessage).To(Equal(prefix))
		})
		It("returns a custom prefix", func() {
			expectedMessage := "20170101:01:01:01 testProgram:testUser:testHost:000000-[INFO]:- my extended info"

			myPrefixFunc := func(level string) string {
				logTimestamp := operating.System.Now().Format("20060102:15:04:05")
				return fmt.Sprintf("%s %s %s", logTimestamp, fmt.Sprintf(gplog.GetHeader("testProgram"), level), "my extended info")
			}
			gplog.SetLogPrefixFunc(myPrefixFunc)

			prefix := gplog.GetLogPrefix("INFO")
			Expect(expectedMessage).To(Equal(prefix))
			gplog.SetLogPrefixFunc(nil)
		})
	})
	Describe("GetShellLogPrefix", func() {
		It("returns a prefix for the current time", func() {
			expectedFormat := "20170101:01:01:01 testProgram:testUser:testHost:000000-[%s]:-"
			Expect(gplog.GetShellLogPrefix("DEBUG")).To(Equal(fmt.Sprintf(expectedFormat, "DEBUG")))
			Expect(gplog.GetShellLogPrefix("INFO")).To(Equal(fmt.Sprintf(expectedFormat, "INFO")))
			Expect(gplog.GetShellLogPrefix("WARNING")).To(Equal(fmt.Sprintf(expectedFormat, "WARNING")))
			Expect(gplog.GetShellLogPrefix("ERROR")).To(Equal(fmt.Sprintf(expectedFormat, "ERROR")))
			Expect(gplog.GetShellLogPrefix("CRITICAL")).To(Equal(fmt.Sprintf(expectedFormat, "CRITICAL")))
			Expect(gplog.GetShellLogPrefix("FOO")).To(Equal(fmt.Sprintf(expectedFormat, "FOO")))
			Expect(gplog.GetShellLogPrefix("")).To(Equal(fmt.Sprintf(expectedFormat, "")))
		})
		It("returns a default short log prefix", func() {
			gplog.SetShellLogPrefixFunc(gplog.DefaultShortLogPrefixFunc)
			Expect(gplog.GetShellLogPrefix("DEBUG")).To(Equal(""))
			Expect(gplog.GetShellLogPrefix("INFO")).To(Equal(""))
			Expect(gplog.GetShellLogPrefix("WARNING")).To(Equal("WARNING: "))
			Expect(gplog.GetShellLogPrefix("ERROR")).To(Equal("ERROR: "))
			Expect(gplog.GetShellLogPrefix("CRITICAL")).To(Equal("CRITICAL: "))
			Expect(gplog.GetShellLogPrefix("FOO")).To(Equal(""))
			Expect(gplog.GetShellLogPrefix("")).To(Equal(""))
			gplog.SetShellLogPrefixFunc(nil)
		})
		It("returns a custom log prefix", func() {
			myPrefixFunc := func(level string) string {
				logTimestamp := operating.System.Now().Format("20060102:15:04:05")
				return fmt.Sprintf("%s-custom-%s", logTimestamp, level)
			}
			gplog.SetShellLogPrefixFunc(myPrefixFunc)
			Expect(gplog.GetShellLogPrefix("DEBUG")).To(Equal("20170101:01:01:01-custom-DEBUG"))
			Expect(gplog.GetShellLogPrefix("INFO")).To(Equal("20170101:01:01:01-custom-INFO"))
			Expect(gplog.GetShellLogPrefix("WARNING")).To(Equal("20170101:01:01:01-custom-WARNING"))
			Expect(gplog.GetShellLogPrefix("ERROR")).To(Equal("20170101:01:01:01-custom-ERROR"))
			Expect(gplog.GetShellLogPrefix("CRITICAL")).To(Equal("20170101:01:01:01-custom-CRITICAL"))
			Expect(gplog.GetShellLogPrefix("FOO")).To(Equal("20170101:01:01:01-custom-FOO"))
			Expect(gplog.GetShellLogPrefix("")).To(Equal("20170101:01:01:01-custom-"))
			gplog.SetShellLogPrefixFunc(nil)
		})
	})
	Describe("Output function tests", func() {
		patternExpected := "20170101:01:01:01 testProgram:testUser:testHost:000000-[%s]:-"
		infoExpected := fmt.Sprintf(patternExpected, "INFO")
		warnExpected := fmt.Sprintf(patternExpected, "WARNING")
		verboseExpected := fmt.Sprintf(patternExpected, "DEBUG")
		debugExpected := fmt.Sprintf(patternExpected, "DEBUG")
		errorExpected := fmt.Sprintf(patternExpected, "ERROR")
		fatalExpected := fmt.Sprintf(patternExpected, "CRITICAL")

		Describe("FatalOnError", func() {
			It("Does not panic when err is nil", func() {
				gplog.FatalOnError(nil, "")
			})
			It("Logs fatally on error with no output", func() {
				defer testhelper.ShouldPanicWithMessage("this is an error")
				gplog.FatalOnError(errors.New("this is an error"))
			})
			It("Logs fatally on error with output", func() {
				defer testhelper.ShouldPanicWithMessage("this is an error: this is output")
				gplog.FatalOnError(errors.New("this is an error"), "this is output")
			})
		})
		Describe("Shell verbosity set to Error", func() {
			BeforeEach(func() {
				gplog.SetVerbosity(gplog.LOGERROR)
				gplog.SetLogFileVerbosity(gplog.LOGDEBUG)
			})

			Context("Info", func() {
				It("prints to the log file", func() {
					expectedMessage := "error info"
					gplog.Info(expectedMessage)
					testhelper.NotExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Success", func() {
				It("prints to the log file", func() {
					expectedMessage := "error info"
					gplog.Success(expectedMessage)
					testhelper.NotExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Warn", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "error warn"
					gplog.Warn(expectedMessage)
					testhelper.ExpectRegexp(stdout, warnExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, warnExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, warnExpected+expectedMessage)
				})
			})
			Context("Verbose", func() {
				It("prints to the log file", func() {
					expectedMessage := "error verbose"
					gplog.Verbose(expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Debug", func() {
				It("prints to the log file", func() {
					expectedMessage := "error debug"
					gplog.Debug(expectedMessage)
					testhelper.NotExpectRegexp(stdout, debugExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, debugExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, debugExpected+expectedMessage)
				})
			})
			Context("Error", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "error error"
					gplog.Error(expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
			Context("Fatal", func() {
				It("prints to the log file, then panics", func() {
					expectedMessage := "error fatal"
					defer func() {
						testhelper.NotExpectRegexp(stdout, fatalExpected+expectedMessage)
						testhelper.NotExpectRegexp(stderr, fatalExpected+expectedMessage)
						testhelper.ExpectRegexp(logfile, fatalExpected+expectedMessage)
					}()
					defer testhelper.ShouldPanicWithMessage(expectedMessage)
					gplog.Fatal(errors.New(expectedMessage), "")
				})
			})
			Context("Custom with shell as error and file as verbose", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "error custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGERROR, expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as verbose and file as verbose", func() {
				It("prints to the log file", func() {
					expectedMessage := "error custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGVERBOSE, expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as info and file as error", func() {
				It("prints to the log file", func() {
					expectedMessage := "error custom"
					gplog.Custom(gplog.LOGERROR, gplog.LOGINFO, expectedMessage)
					testhelper.NotExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
		})
		Describe("Shell verbosity set to Info", func() {
			BeforeEach(func() {
				gplog.SetVerbosity(gplog.LOGINFO)
				gplog.SetLogFileVerbosity(gplog.LOGDEBUG)
			})

			Context("Info", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "info info"
					gplog.Info(expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Success", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "info info"
					gplog.Success(expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Warn", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "info warn"
					gplog.Warn(expectedMessage)
					testhelper.ExpectRegexp(stdout, warnExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, warnExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, warnExpected+expectedMessage)
				})
			})
			Context("Verbose", func() {
				It("prints to the log file", func() {
					expectedMessage := "info verbose"
					gplog.Verbose(expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Debug", func() {
				It("prints to the log file", func() {
					expectedMessage := "info debug"
					gplog.Debug(expectedMessage)
					testhelper.NotExpectRegexp(stdout, debugExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, debugExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, debugExpected+expectedMessage)
				})
			})
			Context("Error", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "info error"
					gplog.Error(expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
			Context("Fatal", func() {
				It("prints to the log file, then panics", func() {
					expectedMessage := "info fatal"
					defer func() {
						testhelper.NotExpectRegexp(stdout, fatalExpected+expectedMessage)
						testhelper.NotExpectRegexp(stderr, fatalExpected+expectedMessage)
						testhelper.ExpectRegexp(logfile, fatalExpected+expectedMessage)
					}()
					defer testhelper.ShouldPanicWithMessage(expectedMessage)
					gplog.Fatal(errors.New(expectedMessage), "")
				})
			})
			Context("Custom with shell as error and file as verbose", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "info custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGERROR, expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as verbose and file as verbose", func() {
				It("prints to the log file", func() {
					expectedMessage := "info custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGVERBOSE, expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as info and file as error", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "info custom"
					gplog.Custom(gplog.LOGERROR, gplog.LOGINFO, expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
		})
		Describe("Shell verbosity set to Verbose", func() {
			BeforeEach(func() {
				gplog.SetVerbosity(gplog.LOGVERBOSE)
				gplog.SetLogFileVerbosity(gplog.LOGDEBUG)
			})

			Context("Info", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "verbose info"
					gplog.Info(expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Success", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "verbose info"
					gplog.Success(expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Warn", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "verbose warn"
					gplog.Warn(expectedMessage)
					testhelper.ExpectRegexp(stdout, warnExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, warnExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, warnExpected+expectedMessage)
				})
			})
			Context("Verbose", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "verbose verbose"
					gplog.Verbose(expectedMessage)
					testhelper.ExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Debug", func() {
				It("prints to the log file", func() {
					expectedMessage := "verbose debug"
					gplog.Debug(expectedMessage)
					testhelper.NotExpectRegexp(stdout, debugExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, debugExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, debugExpected+expectedMessage)
				})
			})
			Context("Error", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "verbose error"
					gplog.Error(expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
			Context("Fatal", func() {
				It("prints to the log file, then panics", func() {
					expectedMessage := "verbose fatal"
					defer func() {
						testhelper.NotExpectRegexp(stdout, fatalExpected+expectedMessage)
						testhelper.NotExpectRegexp(stderr, fatalExpected+expectedMessage)
						testhelper.ExpectRegexp(logfile, fatalExpected+expectedMessage)
						Expect(strings.Count(string(logfile.Contents()), expectedMessage)).To(Equal(1))
					}()
					defer testhelper.ShouldPanicWithMessage(expectedMessage)
					gplog.Fatal(errors.New(expectedMessage), "")
				})
			})
			Context("Custom with shell as error and file as verbose", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "verbose custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGERROR, expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as verbose and file as verbose", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "verbose custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGVERBOSE, expectedMessage)
					testhelper.ExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as info and file as error", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "verbose custom"
					gplog.Custom(gplog.LOGERROR, gplog.LOGINFO, expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
		})
		Describe("Shell verbosity set to Debug", func() {
			BeforeEach(func() {
				gplog.SetVerbosity(gplog.LOGDEBUG)
				gplog.SetLogFileVerbosity(gplog.LOGDEBUG)
			})

			Context("Info", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug info"
					gplog.Info(expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Success", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug info"
					gplog.Success(expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Warn", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug warn"
					gplog.Warn(expectedMessage)
					testhelper.ExpectRegexp(stdout, warnExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, warnExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, warnExpected+expectedMessage)
				})
			})
			Context("Verbose", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug verbose"
					gplog.Verbose(expectedMessage)
					testhelper.ExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Debug", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug debug"
					gplog.Debug(expectedMessage)
					testhelper.ExpectRegexp(stdout, debugExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, debugExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, debugExpected+expectedMessage)
				})
			})
			Context("Error", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "debug error"
					gplog.Error(expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
			Context("Fatal", func() {
				It("prints to the log file, then panics", func() {
					expectedMessage := "debug fatal"
					defer func() {
						testhelper.NotExpectRegexp(stdout, fatalExpected+expectedMessage)
						testhelper.NotExpectRegexp(stderr, fatalExpected+expectedMessage)
						testhelper.ExpectRegexp(logfile, fatalExpected+expectedMessage)
					}()
					defer testhelper.ShouldPanicWithMessage(expectedMessage)
					gplog.Fatal(errors.New(expectedMessage), "")
				})
			})
			Context("Custom with shell as error and file as verbose", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "debug custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGERROR, expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as verbose and file as verbose", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGVERBOSE, expectedMessage)
					testhelper.ExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as info and file as error", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug custom"
					gplog.Custom(gplog.LOGERROR, gplog.LOGINFO, expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
		})
		Describe("Shell verbosity set to Info, logfile verbosity set to Error", func() {
			BeforeEach(func() {
				gplog.SetVerbosity(gplog.LOGINFO)
				gplog.SetLogFileVerbosity(gplog.LOGERROR)
			})
			AfterEach(func() {
				gplog.SetLogFileVerbosity(gplog.LOGDEBUG)
			})

			Context("Info", func() {
				It("prints to stdout", func() {
					expectedMessage := "logfile error info"
					gplog.Info(expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Success", func() {
				It("prints to stdout", func() {
					expectedMessage := "logfile error info"
					gplog.Success(expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Warn", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "logfile error warn"
					gplog.Warn(expectedMessage)
					testhelper.ExpectRegexp(stdout, warnExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, warnExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, warnExpected+expectedMessage)
				})
			})
			Context("Verbose", func() {
				It("does not print", func() {
					expectedMessage := "logfile error verbose"
					gplog.Verbose(expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Debug", func() {
				It("does not print", func() {
					expectedMessage := "logfile error debug"
					gplog.Debug(expectedMessage)
					testhelper.NotExpectRegexp(stdout, debugExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, debugExpected+expectedMessage)
					testhelper.NotExpectRegexp(logfile, debugExpected+expectedMessage)
				})
			})
			Context("Error", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "logfile error error"
					gplog.Error(expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
			Context("Fatal", func() {
				It("prints to the log file, then panics", func() {
					expectedMessage := "logfile error fatal"
					defer func() {
						testhelper.NotExpectRegexp(stdout, fatalExpected+expectedMessage)
						testhelper.NotExpectRegexp(stderr, fatalExpected+expectedMessage)
						testhelper.ExpectRegexp(logfile, fatalExpected+expectedMessage)
					}()
					defer testhelper.ShouldPanicWithMessage(expectedMessage)
					gplog.Fatal(errors.New(expectedMessage), "")
				})
			})
			Context("FatalWithoutPanic", func() {
				It("prints to the log file, then exit(1)", func() {
					gplog.SetExitFunc(func() {})
					expectedMessage := "logfile error fatalwithoutpanic"
					gplog.FatalWithoutPanic(expectedMessage)
					testhelper.NotExpectRegexp(stdout, fatalExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, fatalExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, fatalExpected+expectedMessage)
				})
			})
			Context("Custom with shell as error and file as verbose", func() {
				It("prints to stderr", func() {
					expectedMessage := "logfile error custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGERROR, expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.NotExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as verbose and file as verbose", func() {
				It("does not print", func() {
					expectedMessage := "logfile error custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGVERBOSE, expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as info and file as error", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "info custom"
					gplog.Custom(gplog.LOGERROR, gplog.LOGINFO, expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
		})
		Describe("Shell verbosity set to Info, logfile verbosity set to Info", func() {
			BeforeEach(func() {
				gplog.SetVerbosity(gplog.LOGINFO)
				gplog.SetLogFileVerbosity(gplog.LOGINFO)
			})
			AfterEach(func() {
				gplog.SetLogFileVerbosity(gplog.LOGDEBUG)
			})

			Context("Info", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "logfile info info"
					gplog.Info(expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Success", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "logfile info info"
					gplog.Success(expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Warn", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "logfile info warn"
					gplog.Warn(expectedMessage)
					testhelper.ExpectRegexp(stdout, warnExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, warnExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, warnExpected+expectedMessage)
				})
			})
			Context("Verbose", func() {
				It("does not print", func() {
					expectedMessage := "logfile info verbose"
					gplog.Verbose(expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Debug", func() {
				It("does not print", func() {
					expectedMessage := "logfile info debug"
					gplog.Debug(expectedMessage)
					testhelper.NotExpectRegexp(stdout, debugExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, debugExpected+expectedMessage)
					testhelper.NotExpectRegexp(logfile, debugExpected+expectedMessage)
				})
			})
			Context("Error", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "logfile info error"
					gplog.Error(expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
			Context("Fatal", func() {
				It("prints to the log file, then panics", func() {
					expectedMessage := "logfile info fatal"
					defer func() {
						testhelper.NotExpectRegexp(stdout, fatalExpected+expectedMessage)
						testhelper.NotExpectRegexp(stderr, fatalExpected+expectedMessage)
						testhelper.ExpectRegexp(logfile, fatalExpected+expectedMessage)
					}()
					defer testhelper.ShouldPanicWithMessage(expectedMessage)
					gplog.Fatal(errors.New(expectedMessage), "")
				})
			})
			Context("FatalWithoutPanic", func() {
				It("prints to the log file, then exit(1)", func() {
					gplog.SetExitFunc(func() {})
					expectedMessage := "logfile info fatalwithoutpanic"
					gplog.FatalWithoutPanic(expectedMessage)
					testhelper.NotExpectRegexp(stdout, fatalExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, fatalExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, fatalExpected+expectedMessage)
				})
			})
			Context("Custom with shell as error and file as verbose", func() {
				It("prints to stderr", func() {
					expectedMessage := "logfile info custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGERROR, expectedMessage)
					testhelper.NotExpectRegexp(stdout, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.NotExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as verbose and file as verbose", func() {
				It("does not print", func() {
					expectedMessage := "logfile info custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGVERBOSE, expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as info and file as error", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "info custom"
					gplog.Custom(gplog.LOGERROR, gplog.LOGINFO, expectedMessage)
					testhelper.ExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, infoExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
		})
		Describe("Custom Shell Prefix used - shell verbosity set to Debug", func() {
			BeforeEach(func() {
				gplog.SetVerbosity(gplog.LOGDEBUG)
				gplog.SetLogFileVerbosity(gplog.LOGDEBUG)
				gplog.SetShellLogPrefixFunc(gplog.DefaultShortLogPrefixFunc)
			})
			AfterEach(func() {
				gplog.SetShellLogPrefixFunc(nil)
			})

			Context("Initialization", func() {
				It("returns colorization info", func() {
					Expect(gplog.GetColorize()).To(BeFalse())
				})
			})
			Context("Info", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug info"
					gplog.Info(expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Success", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug info"
					gplog.Success(expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Warn", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug warn"
					gplog.Warn(expectedMessage)
					testhelper.ExpectRegexp(stdout, "WARNING: "+expectedMessage)
					testhelper.NotExpectRegexp(stdout, warnExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, "WARNING: "+expectedMessage)
					testhelper.ExpectRegexp(logfile, warnExpected+expectedMessage)
				})
			})
			Context("Verbose", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug verbose"
					gplog.Verbose(expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Debug", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug debug"
					gplog.Debug(expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, debugExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, debugExpected+expectedMessage)
				})
			})
			Context("Error", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "debug error"
					gplog.Error(expectedMessage)
					testhelper.NotExpectRegexp(stdout, expectedMessage)
					testhelper.ExpectRegexp(stderr, "ERROR: "+expectedMessage)
					testhelper.NotExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
			Context("Fatal", func() {
				It("prints to the log file, then panics", func() {
					expectedMessage := "debug fatal"
					defer func() {
						testhelper.NotExpectRegexp(stdout, "CRITICAL: "+expectedMessage)
						testhelper.NotExpectRegexp(stdout, fatalExpected+expectedMessage)
						testhelper.NotExpectRegexp(stderr, "CRITICAL: "+expectedMessage)
						testhelper.NotExpectRegexp(stderr, fatalExpected+expectedMessage)
						testhelper.ExpectRegexp(logfile, fatalExpected+expectedMessage)
					}()
					defer testhelper.ShouldPanicWithMessage("CRITICAL: " + expectedMessage)
					gplog.Fatal(errors.New(expectedMessage), "")
				})
			})
			Context("Custom with shell as error and file as verbose", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "debug custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGERROR, expectedMessage)
					testhelper.NotExpectRegexp(stdout, expectedMessage)
					testhelper.ExpectRegexp(stderr, "ERROR: "+expectedMessage)
					testhelper.NotExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as verbose and file as verbose", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGVERBOSE, expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as info and file as error", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug custom"
					gplog.Custom(gplog.LOGERROR, gplog.LOGINFO, expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
		})
		Describe("Custom Shell Prefix used with colorization  - shell verbosity set to Debug", func() {
			BeforeEach(func() {
				gplog.SetVerbosity(gplog.LOGDEBUG)
				gplog.SetLogFileVerbosity(gplog.LOGDEBUG)
				gplog.SetShellLogPrefixFunc(gplog.DefaultShortLogPrefixFunc)
				gplog.SetColorize(true)
			})
			AfterEach(func() {
				gplog.SetShellLogPrefixFunc(nil)
				gplog.SetColorize(false)
			})

			Context("Initialization", func() {
				It("returns colorization info", func() {
					Expect(gplog.GetColorize()).To(BeTrue())
				})
			})
			Context("Info", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug info"
					gplog.Info(expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Success", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug success"
					expectedConsoleMessage := fmt.Sprintf("%[1]s[32mdebug success%[1]s[0m", "\x1b")
					gplog.Success(expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedConsoleMessage)
					testhelper.NotExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, infoExpected+expectedMessage)
				})
			})
			Context("Warn", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug warn"
					expectedConsoleMessage := fmt.Sprintf("%[1]s[33mWARNING: debug warn%[1]s[0m", "\x1b")
					gplog.Warn(expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedConsoleMessage)
					testhelper.NotExpectRegexp(stdout, warnExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, warnExpected+expectedMessage)
				})
			})
			Context("Verbose", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug verbose"
					gplog.Verbose(expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Debug", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug debug"
					gplog.Debug(expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, debugExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, debugExpected+expectedMessage)
				})
			})
			Context("Error", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "debug error"
					expectedConsoleMessage := fmt.Sprintf("%[1]s[31mERROR: debug error%[1]s[0m", "\x1b")
					gplog.Error(expectedMessage)
					testhelper.NotExpectRegexp(stdout, expectedMessage)
					testhelper.ExpectRegexp(stderr, expectedConsoleMessage)
					testhelper.NotExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
			Context("Fatal", func() {
				It("prints to the log file, then panics", func() {
					expectedMessage := "debug fatal"
					defer func() {
						testhelper.NotExpectRegexp(stdout, "CRITICAL: "+expectedMessage)
						testhelper.NotExpectRegexp(stdout, fatalExpected+expectedMessage)
						testhelper.NotExpectRegexp(stderr, "CRITICAL: "+expectedMessage)
						testhelper.NotExpectRegexp(stderr, fatalExpected+expectedMessage)
						testhelper.ExpectRegexp(logfile, fatalExpected+expectedMessage)
					}()
					defer testhelper.ShouldPanicWithMessage("CRITICAL: " + expectedMessage)
					gplog.Fatal(errors.New(expectedMessage), "")
				})
			})
			Context("Custom with shell as error and file as verbose", func() {
				It("prints to stderr and the log file", func() {
					expectedMessage := "debug custom"
					expectedConsoleMessage := fmt.Sprintf("%[1]s[31mERROR: debug custom%[1]s[0m", "\x1b")
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGERROR, expectedMessage)
					testhelper.NotExpectRegexp(stdout, expectedMessage)
					testhelper.ExpectRegexp(stderr, expectedConsoleMessage)
					testhelper.NotExpectRegexp(stderr, errorExpected+expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as verbose and file as verbose", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug custom"
					gplog.Custom(gplog.LOGVERBOSE, gplog.LOGVERBOSE, expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, verboseExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, verboseExpected+expectedMessage)
				})
			})
			Context("Custom with shell as info and file as error", func() {
				It("prints to stdout and the log file", func() {
					expectedMessage := "debug custom"
					gplog.Custom(gplog.LOGERROR, gplog.LOGINFO, expectedMessage)
					testhelper.ExpectRegexp(stdout, expectedMessage)
					testhelper.NotExpectRegexp(stdout, infoExpected+expectedMessage)
					testhelper.NotExpectRegexp(stderr, expectedMessage)
					testhelper.ExpectRegexp(logfile, errorExpected+expectedMessage)
				})
			})
		})
	})
})
