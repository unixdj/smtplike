// Copyright 2012 Vadim Vygonets
// This program is free software. It comes without any warranty, to
// the extent permitted by applicable law. You can redistribute it
// and/or modify it under the terms of the Do What The Fuck You Want
// To Public License, Version 2, as published by Sam Hocevar. See
// the LICENSE file or http://sam.zoy.org/wtfpl/ for more details.

/*
Package smtplike implements the server side of an SMTP-like protocol.

For the purpose of this package, in an SMTP-like protocol the
client sends textual commands terminated by \n (possibly \r\n)
with arguments separated by whitespace, and the server replies
with \r\n-terminated lines, where each line is a three-digit
error code and a text message.  In the last line of a reply the
separator between the code and the message is a space character.
In case of a multi-line reply, in all lines except the last the
separator is a hyphen-minus, commonly known as dash ('-').

Example:

	S: 220 Hello
	C: HELO
	S: 250-Hi there
	S: 250 Nice to meet you
	C: quit
	S: 221 Bye
*/
package smtplike

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

/*
	1xx Positive Preliminary
	2xx Positive Completion
	3xx Positive Intermediate
	4xx Transient Negative Completion
	5xx Permanent Negative Completion
	6xx Protected

	x0x Syntax
	x1x Information
	x2x Connection
	x3x Authentication and accounting
	x4x Unspecified
	x5x mail system / mail server status (SMTP), file system (FTP)
*/

const (
	Hello       = 220 // traditional greet code for convenience
	Goodbye     = 221 // if a Handler returns this code, we're done
	Unavailable = 421 // if a Handler returns this code, we're also done
	UnknownCmd  = 500 // sent to client in case of an unknown command
)

var UnknownCmdMsg = "Unknown command" // the string to go along with UnknownCmd

/*
Proto defines the mapping between commands and hadlers.

Command strings should be lowercase.  The Command of the 0th
element may be an empty string, in which case its Handler is
called immediately after receiving a connection to greet the
client.

The Handler functions receive the arguments sent with the
Command and a context, and return the numeric code and message to
send to the client.  Multiline messages are possible with '\n'
as the line separator.  If a Handler returns Goodbye (221) or
Unavailable (421) as the code, the connection is terminated.

The handling of the protocol is described in more detail under
Run().
*/
type Proto []struct {
	Command string
	Handler func(args []string, ctx interface{}) (code int, msg string)
}

func out(c net.Conn, code int, msg string) error {
	lines := strings.Split(msg, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	s := ""
	for _, v := range lines[:len(lines)-1] {
		s += fmt.Sprintf("%03d-%s\r\n", code, v)
	}
	s += fmt.Sprintf("%03d %s\r\n", code, lines[len(lines)-1])
	_, err := c.Write([]byte(s))
	return err
}

/*
Run runs the server for the protocol described by p on the
connection c, passing application-dependent connection-specific
context ctx to Handlers.  It returns an error if reading from or
writing to c fails, or nil if the connection is terminated
successfully.

If p[0].Command is an empty string, Run calls p[0].Handler upon
entry to greet the client, with an empty array in args.  Its
return values are handled like those of any other Handler.  The
constant Hello (220) would be a good code to return.

Each time a line is received from the network connection,
it's broken by string.Fields() into command and arguments.  The
command is then converted to lower case and matched against the
Commands in the Proto array.

If a matching Command is found, its Handler is called with
the command's arguments in args and the context for the
particular connection (passed to Run()) in ctx.  The Handler is
expected to return code between 0 and 999 and msg consisting of
text lines separated by '\n'.

In case no matching Command is found, UnknownCmd (500) and
UnknownCmdMsg are used as code and msg.  Same happens if the
line received contains no command.

The msg is then broken into lines and sent to the client
prepended by the code and followed by '\r\n', in the normal
SMTP-like fashion.  If the code is equal to Goodbye (221) or
Unavailable (421), the connection is then terminated.
*/
func (p Proto) Run(c net.Conn, ctx interface{}) error {
	defer c.Close()
	in := bufio.NewReader(c)
	if len(p) != 0 && p[0].Command == "" {
		code, msg := p[0].Handler([]string{}, ctx)
		if err := out(c, code, msg); err != nil {
			return err
		}
		if code == Goodbye || code == Unavailable {
			return nil
		}
	}
	for {
		line, err := in.ReadString('\n')
		if err != nil {
			return err
		}
		f := strings.Fields(line)
		code, msg := UnknownCmd, UnknownCmdMsg
		if len(f) != 0 {
			cmd := strings.ToLower(f[0])
			for _, v := range p {
				if v.Command == cmd {
					code, msg = v.Handler(f[1:], ctx)
					break
				}
			}
		}
		if err = out(c, code, msg); err != nil {
			return err
		}
		if code == Goodbye || code == Unavailable {
			break
		}
	}
	return nil
}
