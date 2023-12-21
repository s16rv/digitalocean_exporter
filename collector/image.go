package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/digitalocean/godo"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// ImageCollector collects metrics about all images created by the user.
type ImageCollector struct {
	logger  log.Logger
	errors  *prometheus.CounterVec
	client  *godo.Client
	timeout time.Duration

	MinDiskSize *prometheus.Desc
}

// NewImageCollector returns a new ImageCollector.
func NewImageCollector(logger log.Logger, errors *prometheus.CounterVec, client *godo.Client, timeout time.Duration) *ImageCollector {
	errors.WithLabelValues("image").Add(0)

	labels := []string{"id", "name", "region", "type", "distribution"}
	return &ImageCollector{
		logger:  logger,
		errors:  errors,
		client:  client,
		timeout: timeout,

		MinDiskSize: prometheus.NewDesc(
			"digitalocean_image_min_disk_size_bytes",
			"Minimum disk size for a droplet to run this image on in bytes",
			labels, nil,
		),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector.
func (c *ImageCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.MinDiskSize
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *ImageCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	images, _, err := c.client.Images.ListUser(ctx, nil)
	if err != nil {
		c.errors.WithLabelValues("image").Add(1)
		level.Warn(c.logger).Log(
			"msg", "can't list images",
			"err", err,
		)
		return
	}

	for _, img := range images {
		if len(img.Regions) == 0 {
			return
		}

		ch <- prometheus.MustNewConstMetric(
			c.MinDiskSize,
			prometheus.GaugeValue,
			float64(img.MinDiskSize*1024*1024*1024),
			fmt.Sprintf("%d", img.ID), img.Name, img.Regions[0], img.Type, img.Distribution,
		)
	}
}
