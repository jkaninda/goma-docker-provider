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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/jkaninda/logger"
	"gopkg.in/yaml.v3"
)

// Label patterns
var (
	// goma.routes.{routeName}.{field}
	namedRoutePattern = regexp.MustCompile(`^goma\.routes\.([^.]+)\.(.+)$`)
	outputFile        = "goma-docker-provider.yaml"
)

func (p *Provider) syncConfiguration(ctx context.Context) error {
	routes := make([]Route, 0)

	// Get routes from Swarm services
	if p.config.EnableSwarm && p.isSwarmMode {
		swarmRoutes, err := p.getSwarmRoutes(ctx)
		if err != nil {
			logger.Error("Failed to get Swarm routes", "error", err)
			return err
		} else {
			routes = append(routes, swarmRoutes...)
		}
	} else {
		// Get routes from containers
		containerRoutes, err := p.getContainerRoutes(ctx)
		if err != nil {
			logger.Error("Failed to get container routes", "error", err)
			return err
		} else {
			routes = append(routes, containerRoutes...)
		}
	}

	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Name < routes[j].Name
	})

	config := GomaConfig{Routes: routes}

	// Generate hash
	currentHash := p.calculateHash(config)
	if currentHash == p.lastHash {
		return nil
	}

	// Write configuration
	if err := p.writeConfiguration(config); err != nil {
		return err
	}

	p.lastHash = currentHash
	logger.Info("Goma Gateway routes configuration updated", "count", len(routes), "file", outputFile)
	return nil
}

func (p *Provider) getContainerRoutes(ctx context.Context) ([]Route, error) {
	containers, err := p.dockerClient.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "goma.enable=true"),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	routes := make([]Route, 0)
	for _, container := range containers {
		containerRoutes := p.parseContainerLabels(container)
		if containerRoutes != nil {
			routes = append(routes, containerRoutes...)
		}
	}

	return routes, nil
}

func (p *Provider) getSwarmRoutes(ctx context.Context) ([]Route, error) {
	services, err := p.dockerClient.ServiceList(ctx, swarm.ServiceListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "goma.enable=true"),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	routes := make([]Route, 0)
	for _, service := range services {
		serviceRoutes := p.parseServiceLabels(service)
		if serviceRoutes != nil {
			routes = append(routes, serviceRoutes...)
		}
	}

	return routes, nil
}

func (p *Provider) parseContainerLabels(container container.Summary) []Route {
	labels := container.Labels

	if labels["goma.enable"] != "true" {
		return nil
	}

	containerName := container.Names[0][1:]

	// Extract route names
	routeNames := p.extractRouteNames(labels)

	if len(routeNames) == 0 {
		// single route mode
		if route := p.parseSingleContainerRoute(labels, containerName); route != nil {
			return []Route{*route}
		}
		return nil
	}

	// Parse named routes
	routes := make([]Route, 0, len(routeNames))
	for _, routeName := range routeNames {
		if route := p.parseNamedContainerRoute(labels, routeName, containerName); route != nil {
			routes = append(routes, *route)
		}
	}

	return routes
}

func (p *Provider) parseServiceLabels(service swarm.Service) []Route {
	labels := service.Spec.Labels

	if labels["goma.enable"] != "true" {
		return nil
	}

	serviceName := service.Spec.Name

	// Extract route names
	routeNames := p.extractRouteNames(labels)

	if len(routeNames) == 0 {
		// single route mode
		if route := p.parseSingleServiceRoute(service, labels, serviceName); route != nil {
			return []Route{*route}
		}
		return nil
	}

	// Parse named routes
	routes := make([]Route, 0, len(routeNames))
	for _, routeName := range routeNames {
		if route := p.parseNamedServiceRoute(service, labels, routeName, serviceName); route != nil {
			routes = append(routes, *route)
		}
	}

	return routes
}

func (p *Provider) extractRouteNames(labels map[string]string) []string {
	routeMap := make(map[string]bool)

	for key := range labels {
		if matches := namedRoutePattern.FindStringSubmatch(key); matches != nil {
			routeName := matches[1]
			routeMap[routeName] = true
		}
	}

	routes := make([]string, 0, len(routeMap))
	for route := range routeMap {
		routes = append(routes, route)
	}

	// Sort
	sort.Strings(routes)

	return routes
}

func (p *Provider) parseNamedContainerRoute(labels map[string]string, routeName, containerName string) *Route {
	prefix := fmt.Sprintf("goma.routes.%s.", routeName)

	// Extract route specific labels
	routeLabels := make(map[string]string)
	for key, value := range labels {
		if strings.HasPrefix(key, prefix) {
			field := strings.TrimPrefix(key, prefix)
			routeLabels[field] = value
		}
	}

	// Path is required
	path, exists := routeLabels["path"]
	if !exists || path == "" {
		path = "/"
	}

	route := &Route{
		Name:    getRouteLabel(routeLabels, "name", fmt.Sprintf("%s-%s", containerName, routeName)),
		Path:    path,
		Enabled: parseBoolFromMap(routeLabels, "enabled", true),
	}

	// Build target URL
	port := getRouteLabel(routeLabels, "port", "80")
	scheme := getLabel(labels, "goma.scheme", "http")
	if scheme != "http" && scheme != "https" {
		scheme = "http"
	}
	route.Target = fmt.Sprintf("%s://%s:%s", scheme, containerName, port)

	// Parse all route fields
	p.parseRouteFields(route, routeLabels)

	return route
}

func (p *Provider) parseNamedServiceRoute(service swarm.Service, labels map[string]string, routeName, serviceName string) *Route {
	prefix := fmt.Sprintf("goma.routes.%s.", routeName)

	// Extract route specific labels
	routeLabels := make(map[string]string)
	for key, value := range labels {
		if strings.HasPrefix(key, prefix) {
			field := strings.TrimPrefix(key, prefix)
			routeLabels[field] = value
		}
	}

	// Path is required
	path, exists := routeLabels["path"]
	if !exists || path == "" {
		path = "/"
	}

	route := &Route{
		Name:    getRouteLabel(routeLabels, "name", fmt.Sprintf("%s-%s", serviceName, routeName)),
		Path:    path,
		Enabled: parseBoolFromMap(routeLabels, "enabled", true),
	}

	// Get service port
	port := routeLabels["port"]
	if port == "" && len(service.Spec.EndpointSpec.Ports) > 0 {
		port = fmt.Sprintf("%d", service.Spec.EndpointSpec.Ports[0].TargetPort)
	}
	if port == "" {
		port = "80"
	}

	// Swarm mode, use service name as DNS
	scheme := getLabel(labels, "goma.scheme", "http")
	if scheme != "http" && scheme != "https" {
		scheme = "http"
	}
	route.Target = fmt.Sprintf("%s://%s:%s", scheme, serviceName, port)

	// Parse all route fields
	p.parseRouteFields(route, routeLabels)

	return route
}

func (p *Provider) parseSingleContainerRoute(labels map[string]string, containerName string) *Route {
	path := labels["goma.path"]
	if path == "" {
		path = "/"
	}

	route := &Route{
		Name:    getLabel(labels, "goma.name", containerName),
		Path:    path,
		Enabled: true,
	}

	port := getLabel(labels, "goma.port", "80")
	scheme := getLabel(labels, "goma.scheme", "http")
	if scheme != "http" && scheme != "https" {
		scheme = "http"
	}
	route.Target = fmt.Sprintf("%s://%s:%s", scheme, containerName, port)

	p.parseRouteLabels(route, labels)

	return route
}

func (p *Provider) parseSingleServiceRoute(service swarm.Service, labels map[string]string, serviceName string) *Route {
	path := labels["goma.path"]
	if path == "" {
		path = "/"
	}

	route := &Route{
		Name:    getLabel(labels, "goma.name", serviceName),
		Path:    path,
		Enabled: true,
	}

	// Get service port
	port := labels["goma.port"]
	if port == "" && len(service.Spec.EndpointSpec.Ports) > 0 {
		port = fmt.Sprintf("%d", service.Spec.EndpointSpec.Ports[0].TargetPort)
	}
	if port == "" {
		port = "80"
	}
	scheme := getLabel(labels, "goma.scheme", "http")
	if scheme != "http" && scheme != "https" {
		scheme = "http"
	}
	route.Target = fmt.Sprintf("%s://%s:%s", scheme, serviceName, port)

	// Parse labels
	p.parseRouteLabels(route, labels)

	return route
}

func (p *Provider) parseRouteFields(route *Route, labels map[string]string) {
	// Basic fields
	if rewrite := labels["rewrite"]; rewrite != "" {
		route.Rewrite = rewrite
	}

	if priority := labels["priority"]; priority != "" {
		if val, err := strconv.Atoi(priority); err == nil {
			route.Priority = val
		}
	}

	// Hosts
	if hosts := labels["hosts"]; hosts != "" {
		route.Hosts = parseList(hosts)
	}

	// Methods
	if methods := labels["methods"]; methods != "" {
		route.Methods = parseList(methods)
	}

	// Health Check
	if hcPath := labels["health_check.path"]; hcPath != "" {
		route.HealthCheck = RouteHealthCheck{
			Path:     hcPath,
			Interval: getRouteLabel(labels, "health_check.interval", "30s"),
			Timeout:  getRouteLabel(labels, "health_check.timeout", "5s"),
		}

		if statuses := labels["health_check.healthy_statuses"]; statuses != "" {
			route.HealthCheck.HealthyStatuses = parseIntList(statuses)
		}
	}

	// Security
	route.Security = Security{
		ForwardHostHeaders:      parseBoolFromMap(labels, "security.forward_host_headers", true),
		EnableExploitProtection: parseBoolFromMap(labels, "security.enable_exploit_protection", false),
		TLS: SecurityTLS{
			InsecureSkipVerify: parseBoolFromMap(labels, "security.tls.insecure_skip_verify", false),
		},
	}

	// Metrics
	route.DisableMetrics = parseBoolFromMap(labels, "disable_metrics", false)

	// Middlewares
	if middlewares := labels["middlewares"]; middlewares != "" {
		route.Middlewares = parseList(middlewares)
	}
}

func (p *Provider) parseRouteLabels(route *Route, labels map[string]string) {
	// Basic fields
	if rewrite := labels["goma.rewrite"]; rewrite != "" {
		route.Rewrite = rewrite
	}

	if priority := labels["goma.priority"]; priority != "" {
		if val, err := strconv.Atoi(priority); err == nil {
			route.Priority = val
		}
	}

	// Hosts
	if hosts := labels["goma.hosts"]; hosts != "" {
		route.Hosts = parseList(hosts)
	}

	// Methods
	if methods := labels["goma.methods"]; methods != "" {
		route.Methods = parseList(methods)
	}

	// Health Check
	if hcPath := labels["goma.health_check.path"]; hcPath != "" {
		route.HealthCheck = RouteHealthCheck{
			Path:     hcPath,
			Interval: getLabel(labels, "goma.health_check.interval", "30s"),
			Timeout:  getLabel(labels, "goma.health_check.timeout", "5s"),
		}

		// Parse healthy statuses
		if statuses := labels["goma.health_check.healthy_statuses"]; statuses != "" {
			route.HealthCheck.HealthyStatuses = parseIntList(statuses)
		}
	}

	// Security
	route.Security = Security{
		ForwardHostHeaders:      parseBool(labels, "goma.security.forward_host_headers", true),
		EnableExploitProtection: parseBool(labels, "goma.security.enable_exploit_protection", false),
		TLS: SecurityTLS{
			InsecureSkipVerify: parseBool(labels, "goma.security.tls.insecure_skip_verify", false),
		},
	}

	// Metrics
	route.DisableMetrics = parseBool(labels, "goma.disable_metrics", false)

	// Middlewares
	if middlewares := labels["goma.middlewares"]; middlewares != "" {
		route.Middlewares = parseList(middlewares)
	}
}

func (p *Provider) writeConfiguration(config GomaConfig) error {
	if err := os.MkdirAll(p.config.OutputDir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(p.config.OutputDir, outputFile)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	header := []byte(`# Generated by Goma Gateway Docker provider
# DO NOT EDIT MANUALLY

`)

	data = append(header, data...)

	return os.WriteFile(filename, data, 0644)
}

func (p *Provider) calculateHash(config GomaConfig) string {
	data, _ := json.Marshal(config)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
