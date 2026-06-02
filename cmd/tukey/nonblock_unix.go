// Copyright (c) 2025 Boone Studios
// SPDX-License-Identifier: MIT

//go:build !windows

package main

import "syscall"

func setNonblock(fd int, nonblocking bool) error {
	return syscall.SetNonblock(fd, nonblocking)
}
