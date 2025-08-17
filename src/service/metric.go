package service

import (
	"encoding/csv"
	"fmt"
	"hxy352/src/log"
	"hxy352/src/model"
	"os"
	"path/filepath"
	"strconv"
)

type MetricService struct {
	//conf   *model.ReplicaConf
	gConf  *model.Config
	Chan   chan model.MetricChanInfo
	tmpMap map[string]model.MetricChanInfo
	//index  int
	writer *csv.Writer
}

func NewMetricService(gConf *model.Config) *MetricService {
	m := &MetricService{
		gConf:  gConf,
		Chan:   make(chan model.MetricChanInfo, 1000),
		tmpMap: make(map[string]model.MetricChanInfo),
	}

	// create files
	_, err := os.Stat(gConf.FilePath["files"])
	if os.IsNotExist(err) {
		err = os.MkdirAll(gConf.FilePath["files"], 0755)
		if err != nil {
			panic(err)
		}
	}

	name := fmt.Sprintf("metric_with_total_%d_fault_%d", gConf.TotalNumber, gConf.FaultNumber)
	if gConf.PacemakerLoaded {
		name = name + "_with_cogsworth.csv"
	} else {
		name = name + "_without_cogsworth.csv"
	}
	filename := filepath.Join(gConf.FilePath["files"], name)
	m.InitCsv(filename)

	go m.Handle()

	return m
}

func (m *MetricService) Put(info model.MetricChanInfo) {
	m.Chan <- info
}

func (m *MetricService) Handle() {

	for {
		select {
		case info := <-m.Chan:

			log.Debugf("get metric info: %v", info)

			m.WriteToCSV(info)

		}
	}
}

// InitCsv 初始化 CSV 文件（只需调用一次）
func (m *MetricService) InitCsv(filename string) {
	var err error
	var rawFile *os.File

	if fileExists(filename) {
		log.Debug("文件存在")

		rawFile, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			panic(err)
		}

		m.writer = csv.NewWriter(rawFile)

	} else {
		log.Debug("文件不存在")

		rawFile, err = os.Create(filename)

		if err != nil {
			panic(err)
		}

		m.writer = csv.NewWriter(rawFile)

		// 写 CSV 表头

		err = m.writer.Write([]string{"payload", "latency_ms"})
		if err != nil {
			panic(err)
		}
		m.writer.Flush()
	}

}

func (m *MetricService) WriteToCSV(info model.MetricChanInfo) {

	latency := info.Duration.Seconds() * 1000

	record := []string{
		//info.TraceId,
		strconv.Itoa(info.PayloadSize),
		fmt.Sprintf("%.3f", latency),
	}

	if err := m.writer.Write(record); err != nil {
		log.Errorf("WriteToCSV: error writing record to csv: %v", err)
		return
	}
	m.writer.Flush()

	//m.index += 1
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}
