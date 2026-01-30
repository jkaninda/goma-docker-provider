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

type GomaConfig struct {
	Routes []Route `json:"routes" yaml:"routes"`
}
type (
	Route struct {
		// Name provides a descriptive name for the route.
		Name string `yaml:"name" json:"name"`
		// Path specifies the route's path.
		Path string `yaml:"path" json:"path"`
		// Rewrite rewrites the incoming request path to a desired path.
		// For example, `/cart` to `/` rewrites `/cart` to `/`.
		Rewrite string `yaml:"rewrite,omitempty" json:"rewrite,omitempty"`
		// Priority, Determines route matching order
		Priority int `yaml:"priority,omitempty" json:"priority,omitempty"`
		// Enabled specifies whether the route is enabled.
		Enabled bool `yaml:"enabled,omitempty" default:"true" json:"enabled,omitempty"`
		// Hosts lists domains or hosts for request routing.
		Hosts []string `yaml:"hosts,omitempty" json:"hosts,omitempty"`
		// Methods specifies the HTTP methods allowed for this route (e.g., GET, POST).
		Methods []string `yaml:"methods,omitempty" json:"methods,omitempty"`
		// Target defines the primary backend URL for this route.
		Target string `yaml:"target,omitempty" json:"target,omitempty"`
		// HealthCheck contains configuration for monitoring the health of backends.
		HealthCheck    RouteHealthCheck `yaml:"healthCheck,omitempty" json:"healthCheck,omitempty"`
		Security       Security         `yaml:"security,omitempty" json:"security,omitempty"`
		DisableMetrics bool             `yaml:"disableMetrics,omitempty" json:"disableMetrics,omitempty"`
		Middlewares    []string         `yaml:"middlewares,omitempty" json:"middlewares,omitempty"`
	}
)
type RouteHealthCheck struct {
	Path            string `yaml:"path,omitempty" json:"path,omitempty"`
	Interval        string `yaml:"interval,omitempty" json:"interval,omitempty"`
	Timeout         string `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	HealthyStatuses []int  `yaml:"healthyStatuses,omitempty" json:"healthyStatuses,omitempty"`
}
type Maintenance struct {
	Enabled    bool   `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	StatusCode int    `yaml:"statusCode,omitempty" json:"statusCode,omitempty" default:"503"` // default HTTP 503
	Message    string `yaml:"message,omitempty" json:"message,omitempty" default:"Service temporarily unavailable"`
}
type Security struct {
	ForwardHostHeaders      bool        `yaml:"forwardHostHeaders" json:"forwardHostHeaders" default:"true"`
	EnableExploitProtection bool        `yaml:"enableExploitProtection" json:"enableExploitProtection"`
	TLS                     SecurityTLS `yaml:"tls" json:"tls"`
}
type SecurityTLS struct {
	InsecureSkipVerify bool `yaml:"insecureSkipVerify,omitempty" json:"insecureSkipVerify,omitempty"`
}
