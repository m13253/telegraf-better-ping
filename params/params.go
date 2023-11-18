package params

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type PingParams struct {
	Destinations []DestinationParams
}

type DestinationParams struct {
	Comment     string
	Source      string
	Destination string
	HostTag     string
	Interval    time.Duration
	Protocol    string
	Size        uint16
}

type Argument struct {
	Option   string
	HasValue bool
	Value    string
}

func ParseParams(args []string) PingParams {
	var params PingParams

	waitNextDest := false
	nextDest := DestinationParams{
		Interval: time.Second,
		Protocol: "ip",
		Size:     56,
	}

	needValue := map[string]struct{}{
		"":           {},
		"--comment":  {},
		"--dest":     {},
		"--host-tag": {},
		"-I":         {},
		"-i":         {},
		"-s":         {},
	}
	var arg0 string
	for i, arg := range parseCommandLine(args, needValue) {
		if i == 0 {
			arg0 = arg.Value
			continue
		}
		if _, ok := needValue[arg.Option]; ok {
			if !arg.HasValue {
				printShortHelp(arg0, fmt.Sprintf("option %s requires an argument", arg.Option))
			}
		} else if arg.HasValue {
			printShortHelp(arg0, fmt.Sprintf("option %s requires no argument", arg.Option))
		}
		switch arg.Option {
		case "", "--dest":
			nextDest.Destination = arg.Value
			params.Destinations = append(params.Destinations, nextDest)
			waitNextDest = false
			nextDest.Comment = ""
			nextDest.Destination = ""
		case "--prefer-ipv6":
			waitNextDest = true
			nextDest.Protocol = "ip"
		case "--comment":
			waitNextDest = true
			nextDest.Comment = arg.Value
		case "--help":
			printHelp(arg0)
		case "--host-tag":
			waitNextDest = true
			nextDest.HostTag = arg.Value
		case "-4":
			waitNextDest = true
			nextDest.Protocol = "ip4"
		case "-6":
			waitNextDest = true
			nextDest.Protocol = "ip6"
		case "-I":
			waitNextDest = true
			nextDest.Source = arg.Value
		case "-i":
			waitNextDest = true
			if interval, err := strconv.ParseFloat(arg.Value, 64); err == nil && interval >= 0.002 {
				nextDest.Interval = time.Duration(math.Ceil(interval * float64(time.Second)))
			} else {
				printShortHelp(arg0, fmt.Sprintf("invalid interval for option -i: %q", arg.Value))
			}
		case "-s":
			waitNextDest = true
			if size, err := strconv.ParseUint(arg.Value, 10, 16); err == nil && size >= 40 && size <= 65528 {
				nextDest.Size = uint16(size)
			} else {
				printShortHelp(arg0, fmt.Sprintf("invalid interval for option -s: %s", arg.Value))
			}
		default:
			printShortHelp(arg0, fmt.Sprintf("invalid option: %q", arg.Option))
		}
	}

	if waitNextDest {
		printShortHelp(arg0, "the last command line argument must be a destination.")
	}
	if len(params.Destinations) == 0 {
		printShortHelp(arg0, "you must specify at least one destination.")
	}
	return params
}

func printShortHelp(arg0 string, message string) {
	fmt.Fprintf(os.Stderr, `Usage:
  %s {[OPTIONS] [--dest] DESTINATION} [[OPTIONS] [--dest] DESTINATION]...

Error: %s
Use "%s --help" for detailed information.
`, arg0, message, arg0)
	os.Exit(1)
}

func printHelp(arg0 string) {
	fmt.Printf(`Usage:
  %s {[OPTIONS] [--dest] DESTINATION} [[OPTIONS] [--dest] DESTINATION]...

Options:
  --comment=COMMENT     Comment of the following destination.
  [--dest=]DESTINATION  The destination address to send packets to.
                        The text "--dest=" can be omitted.
  --host-tag TAG        Add an extra "host" tag to the InfluxDB entries.
  --prefer-ipv6         Prefer IPv6 / ICMPv6 protocol,
                        fallback to IPv4 / ICMP. The default mode.
  -4                    Use IPv4 / ICMP protocol.
  -6                    Use IPv6 / ICMPv6 protocol.
  -I SOURCE             The source address to send packets from.
  -i INTERVAL           Wait INTERVAL seconds between sending each packet.
                        Must be greater or equal to 0.002 seconds.
  -s SIZE               The number of data bytes to be sent. The default is 56.
                        Must be between 40 and 65528.

Notes:
  All options, except for --comment, only affect the destinations followed by.
  The option --comment only affects the single destination followed by.
  The last command line argument must be a destination.
`, arg0)
	os.Exit(0)
}

func parseCommandLine(args []string, needValue map[string]struct{}) []Argument {
	parsed := make([]Argument, 0, len(args))
	var lastOption string

	const (
		stateStart = iota
		stateWaitOption
		stateWaitValue
		stateEndOptions
	)
	state := stateStart

	for _, arg := range args {
		switch state {
		case stateStart, stateEndOptions:
			parsed = append(parsed, Argument{
				HasValue: true,
				Value:    arg,
			})
			state = stateWaitOption
		case stateWaitOption:
			if strings.HasPrefix(arg, "--") {
				if len(arg) <= 2 {
					state = stateEndOptions
					continue
				}
				idxEqual := strings.IndexByte(arg[2:], '=')
				if idxEqual >= 0 {
					parsed = append(parsed, Argument{
						Option:   arg[:idxEqual+2],
						HasValue: true,
						Value:    arg[idxEqual+3:],
					})
				} else if _, ok := needValue[arg]; ok {
					lastOption = arg
					state = stateWaitValue
				} else {
					parsed = append(parsed, Argument{
						Option: arg,
					})
				}
			} else if strings.HasPrefix(arg, "-") {
				for i, w := 1, 0; i < len(arg); i += w {
					_, w = utf8.DecodeRuneInString(arg[i:])
					if w <= 0 {
						log.Fatalf("invalid string: %q\n", arg)
					}
					lastOption = "-" + arg[i:i+w]
					if _, ok := needValue[lastOption]; ok {
						if len(arg) > i+w {
							parsed = append(parsed, Argument{
								Option:   lastOption,
								HasValue: true,
								Value:    arg[i+w:],
							})
						} else {
							state = stateWaitValue
						}
						break
					}
					parsed = append(parsed, Argument{
						Option: lastOption,
					})
				}
			} else {
				parsed = append(parsed, Argument{
					HasValue: true,
					Value:    arg,
				})
			}
		case stateWaitValue:
			parsed = append(parsed, Argument{
				Option:   lastOption,
				HasValue: true,
				Value:    arg,
			})
			state = stateWaitOption
		}
	}
	if state == stateWaitValue {
		parsed = append(parsed, Argument{
			Option: lastOption,
		})
	}
	return parsed
}
