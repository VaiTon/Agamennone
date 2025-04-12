package achille

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"

	"github.com/VaiTon/Agamennone/pkg/agamennone"
	"github.com/VaiTon/Agamennone/pkg/flag"
)

type AgamennoneApi struct {
	url    string
	client *http.Client
}

func NewAgamennoneApi(hostname string) *AgamennoneApi {
	return &AgamennoneApi{url: hostname, client: &http.Client{}}
}

func (a *AgamennoneApi) GetConfig() (*agamennone.ClientConfig, error) {
	resp, err := a.client.Get(a.url + "/api/config")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status not OK: %s", resp.Status)
	}

	var config *agamennone.ClientConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}

	err = resp.Body.Close()
	if err != nil {
		// do not return error, just log it
		log.Error("can't close response body", "err", err)
	}

	return config, nil
}

func (a *AgamennoneApi) SubmitFlags(flags []flag.Flag) error {
	data, err := json.Marshal(flags)
	if err != nil {
		return fmt.Errorf("error marshalling flags: %w", err)
	}

	resp, err := a.client.Post(a.url+"/api/flags", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error submitting flags: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error submitting flags: got status %s", resp.Status)
	}

	return resp.Body.Close()
}
