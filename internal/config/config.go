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

package config

import (
	"time"

	goutils "github.com/jkaninda/go-utils"
	"github.com/jkaninda/logger"
	"github.com/joho/godotenv"
)

type Config struct {
	OutputDir    string
	PollInterval time.Duration
	DockerHost   string
	EnableSwarm  bool
	SwarmNetwork string
}

func init() {
	_ = godotenv.Load()
}
func New() *Config {
	interval, err := time.ParseDuration(goutils.Env("GOMA_POLL_INTERVAL", "10s"))
	if err != nil {
		logger.Error("Failed to parse poll interval", "error", err)
		interval = 10 * time.Second
	}
	return &Config{
		OutputDir:    goutils.Env("GOMA_OUTPUT_DIR", "/etc/goma/routes.d"),
		PollInterval: interval,
		EnableSwarm:  goutils.EnvBool("GOMA_ENABLE_SWARM", false),
		SwarmNetwork: goutils.Env("GOMA_SWARM_NETWORK", "goma-net"),
	}

}
