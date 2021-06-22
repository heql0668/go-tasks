package utils

import "time"

var Timezone = time.FixedZone("CST", 8*3600)

func CurrentTimestamp() int {
    return int(time.Now().In(Timezone).Unix())
}
