// Configure updates native MyGo package metadata without a JavaScript runtime.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	version := flag.String("version", "", "set the package version")
	flag.Parse()
	if err := configure("mygo.json", *version, os.Getenv("MYGO_UPDATER_PUBLIC_KEY"), os.Getenv("MYGO_UPDATER_PRIVATE_KEY") != ""); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func configure(path, version, publicKey string, hasPrivateKey bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("read MyGo config: %w", err)
	}
	changed := false
	if version != "" {
		parts := strings.Split(version, ".")
		if len(parts) != 3 {
			return fmt.Errorf("version must contain three nonnegative integers")
		}
		for _, part := range parts {
			n, err := strconv.Atoi(part)
			if err != nil || n < 0 || strconv.Itoa(n) != part {
				return fmt.Errorf("invalid version %q", version)
			}
		}
		cfg["version"], _ = json.Marshal(version)
		changed = true
	}
	publicKey = strings.TrimSpace(publicKey)
	if publicKey != "" {
		if !hasPrivateKey {
			return fmt.Errorf("signed builds require MYGO_UPDATER_PRIVATE_KEY")
		}
		cfg["updates"], _ = json.Marshal(map[string]string{"github": "lsongdev/miya-desktop", "publicKey": publicKey})
		changed = true
	}
	var updates struct {
		PublicKey string `json:"publicKey"`
	}
	if raw := cfg["updates"]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &updates); err != nil {
			return err
		}
		if updates.PublicKey != "" && !hasPrivateKey {
			return fmt.Errorf("signed builds require MYGO_UPDATER_PRIVATE_KEY")
		}
	}
	if hasPrivateKey && updates.PublicKey == "" {
		return fmt.Errorf("set MYGO_UPDATER_PUBLIC_KEY before signing a build")
	}
	if !changed {
		return nil
	}
	encoded, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".mygo-config-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(append(encoded, '\n')); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(0644); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}
