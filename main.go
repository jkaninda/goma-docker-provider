/*
 *  MIT License
 *
 * Copyright (c) 2026 Jonas Kaninda
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in all
 *  copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 *  SOFTWARE.
 */

package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/jkaninda/goma-docker-provider/internal"
	"github.com/jkaninda/logger"
)

func main() {
	logger.Info("Starting Goma Docker provider...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	provider := internal.NewProvider()

	errCh := make(chan error, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		errCh <- provider.Start(ctx)
	}()

	select {
	case sig := <-sigCh:
		logger.Debug("Shutdown signal received", "signal", sig)
		logger.Info("Stopping Goma Docker provider")
		cancel()

		if err := provider.Stop(); err != nil {
			logger.Error("Provider stop failed", "error", err)
		}

	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("Provider exited with error", "error", err)
			os.Exit(1)
		}
	}
}
