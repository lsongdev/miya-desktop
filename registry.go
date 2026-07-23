package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lsongdev/miya-agents/mcp"
)

const (
	defaultSkillsRegistryURL = "https://raw.githubusercontent.com/lsongdev/skills/master/registry.json"
	defaultMCPRegistryURL    = "https://registry.modelcontextprotocol.io/v0.1/servers"
	registryResponseLimit    = 8 << 20
	registrySkillFileLimit   = 128
	registrySkillTotalLimit  = 32 << 20
)

var registryHTTPClient = &http.Client{Timeout: 15 * time.Second}

type RegistrySkillInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
	Registry    string `json:"registry"`
	Version     string `json:"version,omitempty"`
	Installed   bool   `json:"installed"`
}

type registrySkillFile struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

type registrySkill struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Source      string              `json:"source,omitempty"`
	Files       []registrySkillFile `json:"files"`
}

type skillRegistryManifest struct {
	Version int             `json:"version"`
	Skills  []registrySkill `json:"skills"`
}

type skillHubSearchResponse struct {
	Skills []skillHubSearchResult `json:"skills"`
}

type skillHubSearchResult struct {
	Slug             string `json:"slug"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	TotalInstalls    int    `json:"totalInstalls"`
	SourceIdentifier string `json:"sourceIdentifier"`
}

type skillHubDetailResponse struct {
	Skill struct {
		Slug             string `json:"slug"`
		Name             string `json:"name"`
		Description      string `json:"description"`
		SourceIdentifier string `json:"sourceIdentifier"`
		SkillPath        string `json:"skillPath"`
	} `json:"skill"`
	LatestVersion struct {
		Version   string `json:"version"`
		CommitSHA string `json:"commitSha"`
		Files     []struct {
			Path string `json:"path"`
			Size int64  `json:"size"`
		} `json:"fileManifest"`
	} `json:"latestVersion"`
}

type MCPRegistryServerInfo struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Description    string               `json:"description,omitempty"`
	Version        string               `json:"version,omitempty"`
	InstallLabel   string               `json:"installLabel"`
	RequiredInputs []string             `json:"requiredInputs,omitempty"`
	Config         *mcp.McpServerConfig `json:"config"`
}

type mcpRegistryResponse struct {
	Servers []mcpRegistryEntry `json:"servers"`
}

type mcpRegistryEntry struct {
	Server mcpRegistryServer `json:"server"`
	Meta   map[string]struct {
		IsLatest bool `json:"isLatest"`
	} `json:"_meta"`
}

type mcpRegistryServer struct {
	Name        string               `json:"name"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Version     string               `json:"version"`
	Packages    []mcpRegistryPackage `json:"packages"`
	Remotes     []mcpRegistryRemote  `json:"remotes"`
}

type mcpRegistryPackage struct {
	RegistryType         string                `json:"registryType"`
	Identifier           string                `json:"identifier"`
	Version              string                `json:"version"`
	RuntimeHint          string                `json:"runtimeHint"`
	RuntimeArguments     []mcpRegistryArgument `json:"runtimeArguments"`
	PackageArguments     []mcpRegistryArgument `json:"packageArguments"`
	EnvironmentVariables []mcpRegistryVariable `json:"environmentVariables"`
}

type mcpRegistryArgument struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	Default    string `json:"default"`
	Type       string `json:"type"`
	ValueHint  string `json:"valueHint"`
	IsRequired bool   `json:"isRequired"`
}

type mcpRegistryVariable struct {
	Name       string `json:"name"`
	Default    string `json:"default"`
	IsRequired bool   `json:"isRequired"`
}

type mcpRegistryRemote struct {
	Type    string              `json:"type"`
	URL     string              `json:"url"`
	Headers []mcpRegistryHeader `json:"headers"`
}

type mcpRegistryHeader struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	IsRequired bool   `json:"isRequired"`
}

func (a *App) ListSkillRegistry(query string) ([]RegistrySkillInfo, error) {
	manifest, err := loadSkillRegistry()
	if err != nil {
		return nil, err
	}
	installed, err := a.ListSkills()
	if err != nil {
		return nil, err
	}
	installedNames := make(map[string]struct{}, len(installed))
	for _, skill := range installed {
		installedNames[safeSkillName(skill.Name)] = struct{}{}
	}
	result := make([]RegistrySkillInfo, 0, len(manifest.Skills))
	for _, skill := range manifest.Skills {
		id := safeSkillName(skill.ID)
		if id == "" || len(skill.Files) == 0 {
			continue
		}
		_, isInstalled := installedNames[id]
		result = append(result, RegistrySkillInfo{
			ID: id, Name: skill.Name, Description: skill.Description, Source: skill.Source,
			Registry: "Miya", Installed: isInstalled,
		})
	}
	query = strings.TrimSpace(query)
	sort.Slice(result, func(i, j int) bool { return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name) })
	return result, nil
}

func (a *App) InstallRegistrySkill(id string) (SkillInfo, error) {
	manifest, err := loadSkillRegistry()
	if err != nil {
		return SkillInfo{}, err
	}
	id = safeSkillName(id)
	var selected *registrySkill
	for index := range manifest.Skills {
		if safeSkillName(manifest.Skills[index].ID) == id {
			selected = &manifest.Skills[index]
			break
		}
	}
	if selected == nil {
		return SkillInfo{}, fmt.Errorf("registry skill %q not found", id)
	}
	return a.installRegistrySkillFiles(id, selected.Description, selected.Files)
}

func (a *App) installRegistrySkillFiles(id, description string, files []registrySkillFile) (SkillInfo, error) {
	if len(files) == 0 || len(files) > registrySkillFileLimit {
		return SkillInfo{}, fmt.Errorf("skill %q has an invalid file count", id)
	}
	destination := filepath.Join(a.SkillsDirectory(), id)
	if _, err := os.Stat(destination); err == nil {
		return SkillInfo{}, fmt.Errorf("skill %q is already installed", id)
	} else if !os.IsNotExist(err) {
		return SkillInfo{}, err
	}
	temporary, err := os.MkdirTemp(a.SkillsDirectory(), ".skill-install-*")
	if err != nil {
		if mkErr := os.MkdirAll(a.SkillsDirectory(), 0755); mkErr != nil {
			return SkillInfo{}, fmt.Errorf("create skills directory: %w", mkErr)
		}
		temporary, err = os.MkdirTemp(a.SkillsDirectory(), ".skill-install-*")
	}
	if err != nil {
		return SkillInfo{}, fmt.Errorf("create skill staging directory: %w", err)
	}
	defer os.RemoveAll(temporary)
	var downloadedSize int64
	for _, file := range files {
		path, err := safeRegistryFilePath(temporary, file.Path)
		if err != nil {
			return SkillInfo{}, fmt.Errorf("install skill %q: %w", id, err)
		}
		data, err := fetchBytes(file.URL, registryResponseLimit)
		if err != nil {
			return SkillInfo{}, fmt.Errorf("download skill file %q: %w", file.Path, err)
		}
		downloadedSize += int64(len(data))
		if downloadedSize > registrySkillTotalLimit {
			return SkillInfo{}, fmt.Errorf("skill %q exceeds the installation size limit", id)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return SkillInfo{}, err
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return SkillInfo{}, err
		}
	}
	if _, err := os.Stat(filepath.Join(temporary, "SKILL.md")); err != nil {
		return SkillInfo{}, fmt.Errorf("registry skill %q does not contain SKILL.md", id)
	}
	if err := os.Rename(temporary, destination); err != nil {
		return SkillInfo{}, fmt.Errorf("activate skill %q: %w", id, err)
	}
	return SkillInfo{Name: id, Description: description, Path: filepath.Join(destination, "SKILL.md")}, nil
}

func githubRawFileURL(repository, commit, skillPath, filePath string) (string, error) {
	repositoryParts := strings.Split(repository, "/")
	if len(repositoryParts) != 2 || commit == "" {
		return "", fmt.Errorf("invalid GitHub skill source %q", repository)
	}
	pathParts := []string{repositoryParts[0], repositoryParts[1], commit}
	for index, path := range []string{skillPath, filePath} {
		if index == 0 && strings.TrimSpace(path) == "" {
			continue
		}
		clean := filepath.ToSlash(filepath.Clean(path))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
			return "", fmt.Errorf("invalid GitHub skill path %q", path)
		}
		pathParts = append(pathParts, strings.Split(clean, "/")...)
	}
	for index := range pathParts {
		pathParts[index] = url.PathEscape(pathParts[index])
	}
	return "https://raw.githubusercontent.com/" + strings.Join(pathParts, "/"), nil
}

func (a *App) ListMCPRegistry(query string) ([]MCPRegistryServerInfo, error) {
	requestURL, err := url.Parse(defaultMCPRegistryURL)
	if err != nil {
		return nil, err
	}
	params := requestURL.Query()
	params.Set("limit", "50")
	if query = strings.TrimSpace(query); query != "" {
		params.Set("search", query)
	}
	requestURL.RawQuery = params.Encode()
	var response mcpRegistryResponse
	if err := fetchJSON(requestURL.String(), &response); err != nil {
		return nil, fmt.Errorf("load MCP registry: %w", err)
	}
	result := make([]MCPRegistryServerInfo, 0, len(response.Servers))
	seen := make(map[string]struct{})
	for _, entry := range response.Servers {
		if !mcpRegistryEntryIsLatest(entry) {
			continue
		}
		candidate, ok := mcpInstallCandidate(entry.Server)
		if !ok {
			continue
		}
		if _, exists := seen[candidate.ID]; exists {
			continue
		}
		seen[candidate.ID] = struct{}{}
		result = append(result, candidate)
	}
	return result, nil
}

func loadSkillRegistry() (*skillRegistryManifest, error) {
	var manifest skillRegistryManifest
	if err := fetchJSON(defaultSkillsRegistryURL, &manifest); err != nil {
		return nil, fmt.Errorf("load skills registry: %w", err)
	}
	if manifest.Version != 1 {
		return nil, fmt.Errorf("unsupported skills registry version %d", manifest.Version)
	}
	return &manifest, nil
}

func fetchJSON(rawURL string, target any) error {
	data, err := fetchBytes(rawURL, registryResponseLimit)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func fetchBytes(rawURL string, limit int64) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json, text/plain;q=0.9")
	request.Header.Set("User-Agent", "miya-desktop/"+appVersion)
	response, err := registryHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("registry returned %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("registry response exceeds %d bytes", limit)
	}
	return data, nil
}

func safeRegistryFilePath(root, relative string) (string, error) {
	relative = filepath.FromSlash(strings.TrimSpace(relative))
	if relative == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("invalid registry file path %q", relative)
	}
	clean := filepath.Clean(relative)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("registry file path escapes skill directory: %q", relative)
	}
	return filepath.Join(root, clean), nil
}

func mcpRegistryEntryIsLatest(entry mcpRegistryEntry) bool {
	if len(entry.Meta) == 0 {
		return true
	}
	if metadata, ok := entry.Meta["io.modelcontextprotocol.registry/official"]; ok {
		return metadata.IsLatest
	}
	for key, metadata := range entry.Meta {
		if strings.HasSuffix(key, "/official") {
			return metadata.IsLatest
		}
	}
	return true
}

func mcpInstallCandidate(server mcpRegistryServer) (MCPRegistryServerInfo, bool) {
	id := safeSkillName(server.Title)
	if id == "" {
		parts := strings.FieldsFunc(server.Name, func(r rune) bool { return r == '/' || r == '.' })
		if len(parts) > 0 {
			id = safeSkillName(parts[len(parts)-1])
		}
	}
	if id == "" {
		return MCPRegistryServerInfo{}, false
	}
	name := strings.TrimSpace(server.Title)
	if name == "" {
		name = server.Name
	}
	base := MCPRegistryServerInfo{ID: id, Name: name, Description: server.Description, Version: server.Version}
	for _, remote := range server.Remotes {
		if strings.TrimSpace(remote.URL) == "" || len(remote.Headers) > 0 {
			continue
		}
		base.InstallLabel = "Remote"
		base.Config = remoteMCPConfig(remote, &base.RequiredInputs)
		return base, true
	}
	for _, registryType := range []string{"npm", "pypi"} {
		for _, pkg := range server.Packages {
			if strings.EqualFold(pkg.RegistryType, registryType) {
				config, label, required := packageMCPConfig(pkg)
				if config != nil {
					base.Config, base.InstallLabel, base.RequiredInputs = config, label, required
					return base, true
				}
			}
		}
	}
	for _, remote := range server.Remotes {
		if strings.TrimSpace(remote.URL) == "" {
			continue
		}
		base.InstallLabel = "Remote"
		base.Config = remoteMCPConfig(remote, &base.RequiredInputs)
		return base, true
	}
	return MCPRegistryServerInfo{}, false
}

func remoteMCPConfig(remote mcpRegistryRemote, required *[]string) *mcp.McpServerConfig {
	headers := make(map[string]string, len(remote.Headers))
	for _, header := range remote.Headers {
		value := header.Value
		if strings.Contains(value, "{") {
			value = ""
		}
		headers[header.Name] = value
		if header.IsRequired || value == "" {
			*required = append(*required, "Header: "+header.Name)
		}
	}
	typeName := strings.ToLower(strings.TrimSpace(remote.Type))
	if typeName == "streamable-http" {
		typeName = "streamablehttp"
	}
	return &mcp.McpServerConfig{Type: typeName, URL: remote.URL, Headers: headers}
}

func packageMCPConfig(pkg mcpRegistryPackage) (*mcp.McpServerConfig, string, []string) {
	command := strings.TrimSpace(pkg.RuntimeHint)
	label := command
	if command == "" {
		switch strings.ToLower(pkg.RegistryType) {
		case "npm":
			command, label = "npx", "npx"
		case "pypi":
			command, label = "uvx", "uvx"
		default:
			return nil, "", nil
		}
	}
	args := fixedRegistryArguments(pkg.RuntimeArguments)
	if command == "npx" && !containsString(args, "-y") {
		args = append([]string{"-y"}, args...)
	}
	identifier := pkg.Identifier
	if pkg.Version != "" {
		if command == "uvx" {
			identifier += "==" + pkg.Version
		} else {
			identifier += "@" + pkg.Version
		}
	}
	args = append(args, identifier)
	args = append(args, configurableRegistryArguments(pkg.PackageArguments)...)
	environment := make(map[string]string, len(pkg.EnvironmentVariables))
	required := requiredRegistryArguments(pkg.RuntimeArguments)
	required = append(required, requiredRegistryArguments(pkg.PackageArguments)...)
	for _, variable := range pkg.EnvironmentVariables {
		environment[variable.Name] = variable.Default
		if variable.IsRequired && variable.Default == "" {
			required = append(required, "Environment: "+variable.Name)
		}
	}
	return &mcp.McpServerConfig{Type: "stdio", Command: command, Args: args, Env: environment}, label, required
}

func requiredRegistryArguments(arguments []mcpRegistryArgument) []string {
	required := make([]string, 0)
	for _, argument := range arguments {
		if !argument.IsRequired || argument.Value != "" || argument.Default != "" {
			continue
		}
		name := strings.TrimSpace(argument.Name)
		if name == "" {
			name = "positional"
		}
		required = append(required, "Argument: "+name)
	}
	return required
}

func fixedRegistryArguments(arguments []mcpRegistryArgument) []string {
	result := make([]string, 0, len(arguments)*2)
	for _, argument := range arguments {
		if argument.Name != "" {
			result = append(result, argument.Name)
		}
		value := argument.Value
		if value == "" {
			value = argument.Default
		}
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func configurableRegistryArguments(arguments []mcpRegistryArgument) []string {
	result := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		value := argument.Value
		if value == "" {
			value = argument.Default
		}
		if strings.EqualFold(argument.Type, "named") || argument.Name != "" {
			if value != "" || argument.IsRequired {
				result = append(result, argument.Name+"="+value)
			}
			continue
		}
		if value != "" {
			result = append(result, value)
		} else if argument.IsRequired {
			hint := safeSkillName(argument.ValueHint)
			if hint == "" {
				hint = "value"
			}
			result = append(result, "<"+hint+">")
		}
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
