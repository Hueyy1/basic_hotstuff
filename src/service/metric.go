package service

import (
	"encoding/csv"
	"fmt"
	"hxy352/src/log"
	"hxy352/src/model"
	"hxy352/src/types"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type MetricService struct {
	conf   *model.ReplicaConf
	gConf  *model.Config
	Chan   chan model.MetricChanInfo
	tmpMap map[types.Hash]model.MetricChanInfo
	index  int
	writer *csv.Writer
}

func NewMetricService(conf *model.ReplicaConf, gConf *model.Config) *MetricService {
	m := &MetricService{
		conf:   conf,
		gConf:  gConf,
		Chan:   make(chan model.MetricChanInfo, 1000),
		tmpMap: make(map[types.Hash]model.MetricChanInfo),
	}

	filename := filepath.Join(gConf.FilePath["files"], fmt.Sprintf("metric_%d.csv", conf.Id))
	m.InitCsv(filename)
	return m
}

func (m *MetricService) Put(info model.MetricChanInfo) {
	m.Chan <- info
}

func (m *MetricService) Handle() {

	for {
		select {
		case info := <-m.Chan:

			if v, ok := m.tmpMap[info.Hash]; ok {
				// update time
				v.CommitTime = info.CommitTime

				// write to csv
				// todo: temp comment
				// m.WriteToCSV(v)

				delete(m.tmpMap, info.Hash)

			} else {
				m.tmpMap[info.Hash] = info
			}
		}
	}
}

// InitCsv 初始化 CSV 文件（只需调用一次）
func (m *MetricService) InitCsv(filename string) {
	var err error

	rawFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	m.writer = csv.NewWriter(rawFile)

	// 写 CSV 表头
	err = m.writer.Write([]string{"index", "hash", "view", "propose_time", "commit_time", "latency_ms"})
	if err != nil {
		panic(err)
	}
	m.writer.Flush()
}

func (m *MetricService) WriteToCSV(info model.MetricChanInfo) {

	latency := info.CommitTime.Sub(*info.ProposeTime).Seconds() * 1000

	record := []string{
		strconv.Itoa(m.index),
		info.Hash.String()[0:8],
		strconv.FormatUint(uint64(info.View), 10),
		info.ProposeTime.Format(time.RFC3339),
		info.CommitTime.Format(time.RFC3339),
		fmt.Sprintf("%.3f", latency),
	}

	if err := m.writer.Write(record); err != nil {
		log.Errorf("WriteToCSV: error writing record to csv: %v", err)
		return
	}
	m.writer.Flush()

	m.index += 1
}
