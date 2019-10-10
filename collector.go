package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/caarlos0/env"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	BackupDir  string   `env:"BACKUP_DIR,required"`
	ServerHost string   `env:"SERVER_HOST" envDefault:"127.0.0.1"`
	ServerPort string   `env:"SERVER_PORT" envDefault:"9001"`
	Databases  []string `env:"DATABASES,required" envSeparator:","`
	Debug      bool     `env:"DEBUG" envDefault:"false"`
}

type ottMongoBackupCollector struct {
	config *config
	backup *prometheus.Desc
}

func (collector ottMongoBackupCollector) getSize(path string) (float64, error) {
	var dirSize int64 = 0
	readSize := func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !file.IsDir() {
			dirSize += file.Size()
		}
		return nil
	}
	err := filepath.Walk(path, readSize)
	if err != nil {
		return 0, err
	}
	var size float64
	size = float64(dirSize)
	return size, nil
}

func newOttMongoBackupCollector() *ottMongoBackupCollector {
	var cfg config
	if err := env.Parse(&cfg); err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("config init")
	}
	if strings.HasSuffix(cfg.BackupDir, "/") {
		log.Info("found trailing slash - deleting...")
		cfg.BackupDir = cfg.BackupDir[0 : len(cfg.BackupDir)-1]
	}
	backup := prometheus.NewDesc(
		"ott_mongodb_backup_size",
		"shows backup size in bytes",
		[]string{"database"}, nil)
	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}
	return &ottMongoBackupCollector{
		config: &cfg,
		backup: backup,
	}
}

func (collector *ottMongoBackupCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.backup
}

func (collector *ottMongoBackupCollector) Collect(ch chan<- prometheus.Metric) {
	for _, database := range collector.config.Databases {
		databaseSize, err := collector.getSize(fmt.Sprintf("%s/%s/%s", collector.config.BackupDir, time.Now().Format("2006-01-02"), database))
		if err != nil {
			log.WithFields(log.Fields{"error": err, "database": database}).Debug("Collect()")
			databaseSize = 0
		}
		ch <- prometheus.MustNewConstMetric(collector.backup, prometheus.GaugeValue, databaseSize, database)
	}
}

func (collector *ottMongoBackupCollector) Run() {
	http.Handle("/metrics", promhttp.Handler())
	log.Infof("ott mongodb backup exporter started on %s:%s", collector.config.ServerHost, collector.config.ServerPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", collector.config.ServerHost, collector.config.ServerPort), nil))
}
