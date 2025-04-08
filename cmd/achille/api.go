package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/VaiTon/Agamennone/pkg/agamennone"
	"github.com/VaiTon/Agamennone/pkg/flag"
)

type AgamennoneApi struct {
	Url string

	client *http.Client
	config agamennone.ClientConfig
}

func NewAgamennoneApi(hostname string) *AgamennoneApi {
	api := &AgamennoneApi{
		Url:    hostname,
		client: &http.Client{},
	}
	return api
}

func (a *AgamennoneApi) Connect() error {
	// store config from /config
	config, err := a.GetConfig()
	if err != nil {
		return fmt.Errorf("error getting config: %w", err)
	}

	a.config = config
	return nil
}

func (a *AgamennoneApi) GetConfig() (agamennone.ClientConfig, error) {
	resp, err := a.client.Get(a.Url + "/api/config")
	if err != nil {
		return agamennone.ClientConfig{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return agamennone.ClientConfig{}, fmt.Errorf("error getting config: %s", resp.Status)
	}

	var config agamennone.ClientConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return agamennone.ClientConfig{}, err
	}

	a.config = config
	return config, nil
}

func (a *AgamennoneApi) SubmitFlags(flags []flag.Flag) error {
	data, err := json.Marshal(flags)
	if err != nil {
		return fmt.Errorf("error marshalling flags: %w", err)
	}

	resp, err := a.client.Post(a.Url+"/api/flags", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error submitting flags: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error submitting flags: %s", resp.Status)
	}

	defer resp.Body.Close()
	return nil
}
