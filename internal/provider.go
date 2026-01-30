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

package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/jkaninda/goma-docker-provider/internal/config"
	"github.com/jkaninda/logger"
)

type Provider struct {
	config       *config.Config
	dockerClient *client.Client
	lastHash     string
	isSwarmMode  bool
	ticker       *time.Ticker
}

func NewProvider() *Provider {
	return &Provider{config: config.New()}
}

func (p *Provider) Start(ctx context.Context) error {
	var err error

	p.dockerClient, err = client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() {
		if err := p.dockerClient.Close(); err != nil {
			logger.Error("failed to close docker client", "error", err)
		}
	}()

	info, err := p.dockerClient.Info(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Docker info: %w", err)
	}

	p.isSwarmMode = info.Swarm.LocalNodeState == swarm.LocalNodeStateActive
	if p.isSwarmMode {
		logger.Info("Docker Swarm mode detected")
	} else {
		logger.Info("Standalone Docker mode detected")
	}

	// Initial sync
	if err := p.syncConfiguration(ctx); err != nil {
		return fmt.Errorf("initial sync failed: %w", err)
	}

	p.ticker = time.NewTicker(p.config.PollInterval)
	defer p.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Provider context cancelled, stopping")
			return ctx.Err()

		case <-p.ticker.C:
			if err := p.syncConfiguration(ctx); err != nil {
				logger.Error("Failed to sync configuration", "error", err)
			}
		}
	}
}
func (p *Provider) Stop() error {
	return nil
}
