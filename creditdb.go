package creditdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"log"
	"time"

	"net/http"
)

var (
	ErrNotFound           = NewError("resource not found", CategoryNotFound)
	ErrBadRequest         = NewError("bad request", CategoryBadRequest)
	ErrInternalError      = NewError("internal server error", CategoryInternalError)
	ErrTimeout            = NewError("timeout", CategoryTimeout)
	ErrServiceUnavailable = NewError("service unavailable", CategoryServiceUnavailable)
)

type CreditDB struct {
	config config
	client *http.Client
}

type config struct {
	host        string
	currentPage uint
}

type Line struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Page struct {
	Status string `json:"status"`
	Page   uint   `json:"pagenumber"`
	Result []Line `json:"result"`
}

const defaultHost = "http://localhost:5622"
const defaultPage = 0

func NewClient() *CreditDB {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 100 * time.Millisecond
	b.MaxElapsedTime = 5 * time.Second
	b.MaxInterval = 30 * time.Second

	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 5 * time.Second,
		},
	}

	clonedClient := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 5 * time.Second,
		},
	}
	clone := &CreditDB{
		config: config{host: defaultHost, currentPage: defaultPage},
		client: clonedClient,
	}

	operation := func() error {
		return clone.Health(context.Background())
	}
	err := backoff.Retry(operation, b)
	if err != nil {
		log.Println("health check failed with error: ", err)
		return nil
	}
	return &CreditDB{
		config: config{host: defaultHost, currentPage: defaultPage},
		client: client,
	}
}

func (c *CreditDB) Close(ctx context.Context) error {
	c.client.CloseIdleConnections()
	return nil
}

func (c *CreditDB) WithHost(host string) *CreditDB {
	if host == "" {
		host = c.config.host
	}
	c.config.host = host
	return c
}

func (c *CreditDB) WithPage(page uint) *CreditDB {
	c.config.currentPage = page
	return c
}

func (c *CreditDB) SetLine(ctx context.Context, key, value string) error {
	if key == "" || value == "" {
		return ErrBadRequest
	}
	setURL := fmt.Sprintf("%s/set", c.config.host)
	line := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Page  uint   `json:"page"`
	}{
		Key:   key,
		Value: value,
		Page:  c.config.currentPage,
	}
	setJSON, err := json.Marshal(line)
	if err != nil {
		return ErrInternalError
	}
	req, err := http.NewRequestWithContext(ctx, "POST", setURL, bytes.NewBuffer(setJSON))
	if err != nil {
		return ErrInternalError
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return ErrInternalError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrBadRequest
	}
	return nil
}

func (c *CreditDB) GetLine(ctx context.Context, key string) (*Line, error) {
	if key == "" {
		return nil, ErrBadRequest
	}
	getURL := fmt.Sprintf("%s/get", c.config.host)
	getData := struct {
		Key  string `json:"key"`
		Page uint   `json:"page"`
	}{
		Key:  string(key),
		Page: c.config.currentPage,
	}
	getJSON, err := json.Marshal(getData)
	if err != nil {
		return nil, ErrInternalError
	}
	req, err := http.NewRequestWithContext(ctx, "GET", getURL, bytes.NewBuffer(getJSON))
	if err != nil {
		return nil, ErrInternalError
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, ErrInternalError
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrBadRequest
	}
	var data Line
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, ErrInternalError
	}
	return &data, nil
}

func (c *CreditDB) GetAllLines(ctx context.Context) ([]Line, error) {
	getAllURL := fmt.Sprintf("%s/getall", c.config.host)
	getAllData := struct {
		Page uint `json:"page"`
	}{
		Page: c.config.currentPage,
	}
	getAllJSON, err := json.Marshal(getAllData)
	if err != nil {
		return nil, ErrInternalError
	}
	req, err := http.NewRequestWithContext(ctx, "GET", getAllURL, bytes.NewBuffer(getAllJSON))
	if err != nil {
		return nil, ErrInternalError
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, ErrInternalError
	}
	defer resp.Body.Close()

	var response Page
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, ErrInternalError
	}
	if response.Status != "OK" {
		return nil, ErrBadRequest
	}
	return response.Result, nil
}

func (c *CreditDB) DeleteLine(ctx context.Context, key string) error {
	if key == "" {
		return ErrBadRequest
	}
	delURL := fmt.Sprintf("%s/delete", c.config.host)
	delData := struct {
		Page uint   `json:"page"`
		Key  string `json:"key"`
	}{
		Page: c.config.currentPage,
		Key:  key,
	}
	delJSON, err := json.Marshal(delData)
	if err != nil {
		return ErrInternalError
	}
	req, err := http.NewRequestWithContext(ctx, "DELETE", delURL, bytes.NewBuffer(delJSON))
	if err != nil {
		return ErrInternalError
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return ErrInternalError
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return ErrBadRequest
	}
	return nil
}

func (c *CreditDB) Flush(ctx context.Context) error {
	flushURL := fmt.Sprintf("%s/flush", c.config.host)
	flushData := struct {
		Page uint `json:"page"`
	}{
		Page: c.config.currentPage,
	}
	flushJSON, err := json.Marshal(flushData)
	if err != nil {
		return ErrInternalError
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", flushURL, bytes.NewBuffer(flushJSON))
	if err != nil {
		return ErrInternalError
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return ErrInternalError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrInternalError
	}
	return nil
}

func (c *CreditDB) Ping(ctx context.Context) (string, error) {
	pingURL := fmt.Sprintf("%s/ping", c.config.host)

	req, err := http.NewRequestWithContext(ctx, "GET", pingURL, nil)
	if err != nil {
		return "", ErrInternalError
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", ErrInternalError
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", ErrInternalError
	}
	pingValue, found := response["ping"].(string)
	if !found {
		return "", ErrInternalError
	}
	return pingValue, nil
}

func (c *CreditDB) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.host, nil)
	if err != nil {
		return ErrInternalError
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return ErrInternalError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrInternalError
	}
	return nil
}

func (c *CreditDB) GetCurrentPage() uint {
	return c.config.currentPage
}


func (c *CreditDB)Exists(ctx context.Context, key string)(bool, error){
	_, err := c.GetLine(ctx, key)
	if err != nil {
		if err == ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}