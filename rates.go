package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const cbuURL = "https://cbu.uz/uz/arkhiv-kursov-valyut/json/"

// Rate - bitta valyutaning 1 birligi uchun so'mdagi kursi.
type Rate struct {
	Code string
	UZS  float64 // 1 birlik uchun so'm
	Diff float64 // kechagiga nisbatan o'zgarish
	Date string
}

type cbuItem struct {
	Ccy     string `json:"Ccy"`
	Nominal string `json:"Nominal"`
	Rate    string `json:"Rate"`
	Diff    string `json:"Diff"`
	Date    string `json:"Date"`
}

// RateStore kursni xotirada 30 daqiqa saqlaydi (CBU'ni ortiqcha so'ramaslik uchun).
type RateStore struct {
	mu      sync.Mutex
	rates   map[string]Rate
	fetched time.Time
	ttl     time.Duration
	client  *http.Client
}

func NewRateStore() *RateStore {
	return &RateStore{
		ttl:    30 * time.Minute,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *RateStore) Get(ctx context.Context) (map[string]Rate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rates != nil && time.Since(s.fetched) < s.ttl {
		return s.rates, nil
	}
	rates, err := s.fetch(ctx)
	if err != nil {
		if s.rates != nil { // xatolik bo'lsa, eski kursni ishlataveramiz
			return s.rates, nil
		}
		return nil, err
	}
	s.rates, s.fetched = rates, time.Now()
	return rates, nil
}

func (s *RateStore) fetch(ctx context.Context) (map[string]Rate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cbuURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cbu: status %d", resp.StatusCode)
	}

	var items []cbuItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	out := make(map[string]Rate, len(items))
	for _, it := range items {
		nominal, err1 := strconv.ParseFloat(it.Nominal, 64)
		rate, err2 := strconv.ParseFloat(strings.ReplaceAll(it.Rate, ",", "."), 64)
		if err1 != nil || err2 != nil || nominal == 0 {
			continue
		}
		diff, _ := strconv.ParseFloat(strings.ReplaceAll(it.Diff, ",", "."), 64)
		out[it.Ccy] = Rate{
			Code: it.Ccy,
			UZS:  rate / nominal,
			Diff: diff / nominal,
			Date: it.Date,
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("cbu: bo'sh javob")
	}
	return out, nil
}