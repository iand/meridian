package conf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type Config struct {
	Peer              PeerConfig       `json:"peer"`
	Datastore         ModuleConfigList `json:"datastore"`
	DatastoreWrapper  ModuleConfigList `json:"datastore_wrapper"`
	Blockstore        ModuleConfigList `json:"blockstore"`
	BlockstoreWrapper ModuleConfigList `json:"blockstore_wrapper"`
	Bootstrapper      ModuleConfigList `json:"bootstrapper"`
	ConnectionManager ModuleConfigList `json:"connection_manager"`
}

type PeerConfig struct {
	ListenAddrs   []string `json:"listen_addrs"`
	AnnounceAddrs []string `json:"announce_addrs"`
	Libp2pKeyFile string

	ConnMgrLow   int
	ConnMgrHi    int
	ConnMgrGrace time.Duration
}

type ModuleConfig struct {
	ID     string
	Config json.RawMessage
}

func (mc *ModuleConfig) Configure(m Module) error {
	if err := json.Unmarshal(mc.Config, m); err != nil {
		return fmt.Errorf("unmarshal module config: %w", err)
	}
	return nil
}

type ModuleConfigList struct {
	Modules []ModuleConfig
}

func (m *ModuleConfigList) UnmarshalJSON(data []byte) error {
	r := bytes.NewReader(data)
	d := json.NewDecoder(r)

	t, err := d.Token()
	if err != nil {
		return fmt.Errorf("unmarshal ModuleConfigList: %w", err)
	} else if t != json.Delim('{') {
		return fmt.Errorf("unmarshal ModuleConfigList: expected object delimiter")
	}

	for d.More() {

		k, err := d.Token()
		if err != nil {
			return fmt.Errorf("unmarshal ModuleConfigList key: %w", err)
		}

		mc := ModuleConfig{
			ID: k.(string),
		}
		if err := d.Decode(&mc.Config); err != nil {
			return fmt.Errorf("unmarshal ModuleConfigList value: %w", err)
		}

		m.Modules = append(m.Modules, mc)
	}

	return nil
}
