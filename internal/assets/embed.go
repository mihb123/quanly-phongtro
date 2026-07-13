package assets

import _ "embed"

//go:embed Roboto-Bold.ttf
var RobotoBold []byte

//go:embed Roboto-Regular.ttf
var RobotoRegular []byte

//go:embed geoip/GeoLite2-City.mmdb
var GeoLite2City []byte
