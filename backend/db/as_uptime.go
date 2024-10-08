package db

import (
	"context"
	"fmt"
	"github.com/BaiMeow/NetworkMonitor/utils"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"log"
	"strconv"
	"time"
)

func BatchRecordASUp(ASNs []uint32, t time.Time) error {
	if !Enabled {
		return ErrDatabaseDisabled
	}
	if err := bgpWrite.WritePoint(context.Background(),
		utils.Map(ASNs, func(asn uint32) *write.Point {
			return influxdb2.NewPointWithMeasurement("bgp").
				AddField("up", 1).
				AddTag("asn", strconv.FormatUint(uint64(asn), 10)).
				SetTime(t)
		})...); err != nil {
		log.Printf("write record fail:%v", err)
		return ErrDatabase
	}
	return nil
}

func AllASRecordAfter(after time.Time) ([]uint32, error) {
	if !Enabled {
		return nil, ErrDatabaseDisabled
	}
	var asns []uint32
	res, err := bgpQuery.Query(context.Background(),
		fmt.Sprintf(`from(bucket: "bgp-uptime")
  				|> range(start: %d)    
				|> unique(column: "asn")`, after.Unix()))
	if err != nil {
		log.Printf("query fail:%v", err)
		return asns, ErrDatabase
	}

	for res.Next() {
		asn, ok := res.Record().ValueByKey("asn").(string)
		if !ok {
			log.Printf("convert fail:%v", res.Record().ValueByKey("asn"))
			return asns, ErrDatabase
		}
		asnNum, err := strconv.ParseUint(asn, 10, 32)
		if err != nil {
			log.Printf("convert fail:%v", err)
			continue
		} else {
			asns = append(asns, uint32(asnNum))
		}
	}

	return asns, nil
}

func BGPASNLast10Tickers(asn uint32, last time.Time) ([]time.Time, error) {
	if !Enabled {
		return nil, ErrDatabaseDisabled
	}
	var t []time.Time
	res, err := bgpQuery.Query(context.Background(),
		fmt.Sprintf(`from(bucket: "bgp-uptime")
  |> range(start: %d, stop: %d)
  |> filter(fn: (r) => r["_measurement"] == "bgp")
  |> filter(fn: (r) => r["asn"] == "%d")`, utils.TickOffset(last, -10).Unix(), last.Unix(), asn))
	if err != nil {
		log.Printf("query fail:%v", err)
		return t, ErrDatabase
	}
	for res.Next() {
		t = append(t, res.Record().Time())
	}
	return t, nil
}
