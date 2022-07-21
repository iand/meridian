package conf

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Config struct {
	Peer              PeerConfig   `json:"peer"`
	Datastore         SystemConfig `json:"datastore"`
	DatastoreWrapper  SystemConfig `json:"datastore_wrapper"`
	Blockstore        SystemConfig `json:"blockstore"`
	BlockstoreWrapper SystemConfig `json:"blockstore_wrapper"`
	Bootstrapper      SystemConfig `json:"bootstrapper"`
	ConnectionManager SystemConfig `json:"connection_manager"`
	Routing           SystemConfig `json:"routing"`
	RoutingComposer   SystemConfig `json:"routing_composer"`
}

type PeerConfig struct {
	Name          string   `json:"name"`         // Name of peer to use when reporting metrics
	ListenAddrs   []string `json:"listen_addrs"` // Multiaddresses IPFS node should listen on.
	AnnounceAddrs []string `json:"announce_addrs"`
	KeyFile       string   `json:"key_file"` // Path to libp2p key file.
	Offline       bool     `json:"offline"`
}

type ModuleConfig struct {
	ID     string
	Config json.RawMessage
}

func (mc *ModuleConfig) Apply(m Module) error {
	if err := json.Unmarshal(mc.Config, m); err != nil {
		return fmt.Errorf("unmarshal module config: %w", err)
	}
	return nil
}

type SystemConfig struct {
	Modules []ModuleConfig
}

func (m *SystemConfig) UnmarshalJSON(data []byte) error {
	r := bytes.NewReader(data)
	d := json.NewDecoder(r)

	t, err := d.Token()
	if err != nil {
		return fmt.Errorf("unmarshal SystemConfig: %w", err)
	} else if t != json.Delim('{') {
		return fmt.Errorf("unmarshal SystemConfig: expected object delimiter")
	}

	for d.More() {
		k, err := d.Token()
		if err != nil {
			return fmt.Errorf("unmarshal ModuleConfig key: %w", err)
		}

		mc := ModuleConfig{
			ID: k.(string),
		}
		if err := d.Decode(&mc.Config); err != nil {
			return fmt.Errorf("unmarshal ModuleConfig value: %w", err)
		}

		m.Modules = append(m.Modules, mc)
	}

	return nil
}

func (c *Config) SystemConfig(category string) (SystemConfig, bool) {
	switch category {
	case ModuleCategoryDatastore:
		return c.Datastore, true
	case ModuleCategoryDatastoreWrapper:
		return c.DatastoreWrapper, true
	case ModuleCategoryBlockstore:
		return c.Blockstore, true
	case ModuleCategoryBlockstoreWrapper:
		return c.BlockstoreWrapper, true
	case ModuleCategoryBootstrapper:
		return c.Bootstrapper, true
	case ModuleCategoryConnManager:
		return c.ConnectionManager, true
	case ModuleCategoryRouting:
		return c.Routing, true
	case ModuleCategoryRoutingComposer:
		return c.RoutingComposer, true
	default:
		return SystemConfig{}, false
	}
}

func (c *Config) LoadSingletonModule(category string, defaultModuleID string) (Module, error) {
	syscfg, ok := c.SystemConfig(category)
	if !ok {
		return nil, fmt.Errorf("cannot load modules for unknown category %s", category)
	}

	if len(syscfg.Modules) > 1 {
		return nil, fmt.Errorf("too many %s modules specified in configuration, only one is allowed", category)
	}

	modules := ModulesForCategory(category)

	var module Module
	if len(syscfg.Modules) == 0 {
		var ok bool
		module, ok = modules[defaultModuleID]
		if !ok {
			return nil, fmt.Errorf("default %s %q is not a loaded module", category, defaultModuleID)
		}
	} else {
		moduleconf := syscfg.Modules[0]
		var ok bool
		module, ok = modules[moduleconf.ID]
		if !ok {
			return nil, fmt.Errorf("%s %q specified in configuratiion is not a loaded module", category, moduleconf.ID)
		}
		if err := moduleconf.Apply(module); err != nil {
			return nil, fmt.Errorf("unmarshal %s %q config: %w", category, moduleconf.ID, err)
		}

		if err := module.ValidateModule(); err != nil {
			return nil, fmt.Errorf("validate %s %q config: %w", category, moduleconf.ID, err)
		}
	}

	return module, nil
}

func (c *Config) LoadModules(category string) ([]Module, error) {
	syscfg, ok := c.SystemConfig(category)
	if !ok {
		return nil, fmt.Errorf("cannot load modules for unknown category %s", category)
	}

	modules := ModulesForCategory(category)

	mods := []Module{}
	for _, moduleconf := range syscfg.Modules {
		module, ok := modules[moduleconf.ID]
		if !ok {
			return nil, fmt.Errorf("%s %q specified in configuration is not a loaded module", category, moduleconf.ID)
		}
		if err := moduleconf.Apply(module); err != nil {
			return nil, fmt.Errorf("unmarshal %s %q config: %w", category, moduleconf.ID, err)
		}

		if err := module.ValidateModule(); err != nil {
			return nil, fmt.Errorf("validate %s %q config: %w", category, moduleconf.ID, err)
		}

		mods = append(mods, module)
	}

	return mods, nil
}
