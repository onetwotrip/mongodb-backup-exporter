package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	mongodbBackupExporter := newOttMongoBackupCollector()
	prometheus.MustRegister(mongodbBackupExporter)
	mongodbBackupExporter.Run()
}
