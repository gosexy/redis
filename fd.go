// This file is based on src/pkg/net/fd.go.
//
// I removed most of the networking stuff only to leave a bare bones file
// descriptor watcher that could make use of already existing multi-OS code,
// such as fd_linux.go, fd_darwin.go, etc.
//
//                                             <jose.carlos@menteslibres.net>
//
// Original net/fd.go's notice:
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin freebsd linux netbsd openbsd

package redis

import (
	"sync"
	"syscall"
	"time"
)

// File descriptor.
type fileFD struct {
	sysfd int

	rdeadline int64
	wdeadline int64
}

var fds = map[int]*fileFD{}

type pollServer struct {
	poll       *pollster // low-level OS hooks
	sync.Mutex           // controls pending and deadline
	pending    map[int]*fileFD
	deadline   int64 // next deadline (nsec since 1970)
}

func (s *pollServer) AddFD(fd *fileFD, mode int) error {

	intfd := fd.sysfd

	if intfd < 0 {
		return nil
	}

	var t int64
	key := intfd << 1
	if mode == 'r' {
		t = fd.rdeadline
	} else {
		key++
		t = fd.wdeadline
	}
	s.pending[key] = fd

	if t > 0 && (s.deadline == 0 || t < s.deadline) {
		s.deadline = t
	}

	_, err := s.poll.AddFD(intfd, mode, false)

	if err != nil {
		s.EvictWrite(intfd)
		s.EvictRead(intfd)
		return nil
	}

	return nil
}

func (s *pollServer) EvictWrite(fd int) {
	if ffd, ok := s.pending[fd<<1|1]; ok == true {
		s.poll.DelFD(ffd.sysfd, 'w')
		delete(s.pending, ffd.sysfd<<1|1)
	}
}

func (s *pollServer) EvictRead(fd int) {
	if ffd, ok := s.pending[fd<<1]; ok == true {
		s.poll.DelFD(ffd.sysfd, 'r')
		delete(s.pending, ffd.sysfd<<1)
	}
}

func (s *pollServer) Now() int64 {
	return time.Now().UnixNano()
}

func (s *pollServer) Run() {

	s.Lock()

	defer s.Unlock()

	for {

		var t = s.deadline

		if t > 0 {
			t = t - s.Now()
			if t <= 0 {
				continue
			}
		}

		fd, m, err := s.poll.WaitFD(s, t)

		if err != nil {
			print("pollServer WaitFD: ", err.Error(), "\n")
			return
		}

		if fd < 0 {
			continue
		}

		if fdev[m] != nil {
			if fn, ok := fdev[m][fd]; ok == true {
				fn()
			}
		}

	}
}

func (s *pollServer) WaitRead(fd int) error {
	var err error

	/*
		var ok bool
		if _, ok = s.pending[fd<<1]; ok == true {
			s.EvictRead(fd)
		}
	*/

	ffd, err := newFD(fd)
	if err != nil {
		return err
	}
	err = s.AddFD(ffd, 'r')
	return err
}

func (s *pollServer) WaitWrite(fd int) error {
	var err error

	/*
		var ok bool
		if _, ok = s.pending[fd<<1|1]; ok == true {
			s.EvictWrite(fd)
		}
	*/

	ffd, err := newFD(fd)
	if err != nil {
		return err
	}
	err = s.AddFD(ffd, 'w')
	return err
}

// Network FD methods.
// All the network FDs use a single pollServer.

var pollserver *pollServer

func startServer() {
	p, err := newPollServer()
	if err != nil {
		print("Start pollServer: ", err.Error(), "\n")
	}
	pollserver = p
}

func newFD(fd int) (*fileFD, error) {
	if err := syscall.SetNonblock(fd, true); err != nil {
		return nil, err
	}
	ffd := &fileFD{
		sysfd: fd,
	}
	return ffd, nil
}
