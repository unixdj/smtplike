// Copyright 2012 Vadim Vygonets
// This program is free software. It comes without any warranty, to
// the extent permitted by applicable law. You can redistribute it
// and/or modify it under the terms of the Do What The Fuck You Want
// To Public License, Version 2, as published by Sam Hocevar. See
// the LICENSE file or http://sam.zoy.org/wtfpl/ for more details.

/*
Example server for smtplike package.  Listens on port 1234.
"Help" is a good command to start with, but don't try to speak
SMTP to it, it gets offended.
*/
package main

import (
	"fmt"
	"github.com/unixdj/smtplike"
	"net"
)

func greet(args []string, ctx interface{}) (code int, msg string) {
	return smtplike.Hello, "may i help you?"
}

func help(args []string, ctx interface{}) (code int, msg string) {
	return 214, `commands:
help
helo
how are you
how is [someone]
quit`
}

func hello(args []string, ctx interface{}) (code int, msg string) {
	*ctx.(*bool) = true
	return 250, "oh, hi!"
}

func how(args []string, ctx interface{}) (code int, msg string) {
	if !*ctx.(*bool) {
		return 503, "say helo first"
	}
	code, msg = 501, "usage:\n    how are you\n    how is [name]"
	if len(args) != 2 {
		return
	}
	switch args[0] {
	case "are":
		if args[1] != "you" {
			return
		}
		return 200, "fine, thanks"
	case "is":
		return 201, args[1] + " is ok"
	}
	return
}

func smtp(args []string, ctx interface{}) (code int, msg string) {
	return smtplike.Unavailable, "what is it, ESMTP?  service unavailable!"
}

func quit(args []string, ctx interface{}) (code int, msg string) {
	return smtplike.Goodbye, "bye"
}

// the protocol
var proto = smtplike.Proto{
	{"", greet},
	{"help", help},
	{"helo", hello},
	{"how", how},
	{"quit", quit},
	{"mail", smtp},
	{"rcpt", smtp},
	{"data", smtp},
}

func main() {
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return
		}
		go func(c net.Conn) {
			var ctx bool
			if err := proto.Run(c, &ctx); err != nil {
				fmt.Println(err)
			}
		}(c)
	}
}
