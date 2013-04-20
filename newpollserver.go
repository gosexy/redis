// This file is based on src/pkg/net/newpollserver.go.
//
// I removed most of the networking stuff only to leave a bare bones file
// descriptor watcher that could make use of already existing multi-OS code,
// such as fd_linux.go, fd_darwin.go, etc.
//
//                                             <jose.carlos@menteslibres.net>
//
// Original net/newpollserver.go's notice:
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin freebsd linux netbsd openbsd

package redis

import (
	"errors"
)

var ErrFailedPollServer = errors.New(`Failed to initialize poll server.`)

func newPollServer() (s *pollServer, err error) {
	s = new(pollServer)
	if s.poll, err = newpollster(); err != nil {
		return nil, ErrFailedPollServer
	}
	s.pending = make(map[int]*fileFD)
	go s.Run()
	return s, nil
}
