module github.com/zxfonline/misc

go 1.13

require (
	github.com/levenlabs/golib v0.0.0-20180911183212-0f8974794783
	github.com/oschwald/geoip2-golang v1.3.0 // indirect
	github.com/oschwald/maxminddb-golang v1.5.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	github.com/yuin/gopher-lua v0.0.0-20190206043414-8bfc7677f583
	golang.org/x/net v0.0.0-20191009170851-d66e71096ffb
	golang.org/x/sys v0.0.0-20191009170203-06d7bd2c5f4f // indirect
	gopkg.in/yaml.v2 v2.2.7
	layeh.com/gopher-json v0.0.0-20190114024228-97fed8db8427
	layeh.com/gopher-luar v0.0.0-00010101000000-000000000000

)

replace layeh.com/gopher-luar => github.com/zxfonline/gopher-luar v1.0.7

replace layeh.com/gopher-json => github.com/zxfonline/gopher-json v1.0.0

replace github.com/yuin/gopher-lua => github.com/zxfonline/gopher-lua v1.0.0
