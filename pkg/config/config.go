package pkg_config

import (
  "encoding/json"
  "log"
  "os"
)

// --------------------------------------------------------- //

type ConfigServerHttp struct {
  Listener struct {
    Address string `json:"address"`
    Port int16 `json:"port"`
  } `json:"listener"`
  Limiter struct {
    MaxRequestPerIp int `json:"max_request_per_ip"`
    MaxRequestInterval int `json:"max_request_interval"`
    CleanupOldRequestInterval int `json:"cleanup_old_request_interval"`
  } `json:"limiter"`
  Server struct {
    IdleTimeout int `json:"idle_timeout"`
    ReadTimeout int `json:"read_timeout"`
    WriteTimeout int `json:"write_timeout"`
  } `json:"server"`
}

func ConfigServerHttpLoad(fp string) (ConfigServerHttp, error) {
  var cfg ConfigServerHttp

  content, err := os.ReadFile(fp); if err != nil {
    log.Fatalf("error: %v\nconfig file is required\n", err)
    return cfg, err
  }

  err = json.Unmarshal(content, &cfg); if err != nil {
    log.Fatalf("error: %v\n", err)
    return cfg, err
  }

  return cfg, nil
}

// --------------------------------------------------------- //

type ConfigServerGrpc struct {
  Listener struct {
    Address string `json:"address"`
    Port int16 `json:"port"`
  } `json:"listener"`
  Limiter struct {
    MaxRequestPerIp int `json:"max_request_per_ip"`
    MaxRequestInterval int `json:"max_request_interval"`
    CleanupOldRequestInterval int `json:"cleanup_old_request_interval"`
  } `json:"limiter"`
}

func ConfigServerGrpcLoad(fp string) (ConfigServerGrpc, error) {
  var cfg ConfigServerGrpc

  content, err := os.ReadFile(fp); if err != nil {
    log.Fatalf("error: %v\nconfig file is required\n", err)
    return cfg, err
  }

  err = json.Unmarshal(content, &cfg); if err != nil {
    log.Fatalf("error: %v\n", err)
    return cfg, err
  }

  return cfg, nil
}
