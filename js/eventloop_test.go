/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package js

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBasicEventLoop(t *testing.T) {
	t.Parallel()
	loop := newEventLoop()
	var ran int
	f := func() error { ran++; return nil } //nolint:unparam
	require.NoError(t, loop.start(f))
	require.Equal(t, 1, ran)
	require.NoError(t, loop.start(f))
	require.Equal(t, 2, ran)
	require.Error(t, loop.start(func() error {
		if err := f(); err != nil {
			return err
		}
		loop.reserve()(f)
		return errors.New("somethjing")
	}))
	require.Equal(t, 3, ran)
}

func TestEventLoopReserve(t *testing.T) {
	t.Parallel()
	loop := newEventLoop()
	var ran int
	start := time.Now()
	require.NoError(t, loop.start(func() error {
		ran++
		r := loop.reserve()
		go func() {
			time.Sleep(time.Second)
			r(func() error {
				ran++
				return nil
			})
		}()
		return nil
	}))
	took := time.Since(start)
	require.Equal(t, 2, ran)
	require.Less(t, time.Second, took)
	require.Greater(t, time.Second+time.Millisecond*100, took)
}

func TestEventLoopReserveStopBetweenStarts(t *testing.T) {
	t.Parallel()
	loop := newEventLoop()
	var ran int
	require.Error(t, loop.start(func() error {
		ran++
		r := loop.reserve()
		go func() {
			time.Sleep(time.Second)
			r(func() error {
				ran++
				return nil
			})
		}()
		return errors.New("something")
	}))
	require.Equal(t, 1, ran)

	require.NoError(t, loop.start(func() error {
		ran++
		r := loop.reserve()
		go func() {
			time.Sleep(time.Second)
			r(func() error {
				ran++
				return nil
			})
		}()
		return nil
	}))
	require.Equal(t, 3, ran)
}
