module go.etcd.io/etcd/v3

require (
	github.com/bgentry/speakeasy v0.1.0
	github.com/cockroachdb/datadriven v0.0.0-20190531201743-edce55837238
	github.com/coreos/go-semver v0.2.0
	github.com/coreos/go-systemd v0.0.0-20180511133405-39ca1b05acc7
	github.com/coreos/pkg v0.0.0-20160727233714-3ac0863d7acf
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dustin/go-humanize v0.0.0-20171111073723-bb3d318650d4
	github.com/fatih/color v1.7.0 // indirect
	github.com/gogo/protobuf v1.1.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/golang/groupcache v0.0.0-20160516000752-02826c3e7903
	github.com/golang/protobuf v1.3.1
	github.com/google/btree v0.0.0-20180124185431-e89373fe6b4a
	github.com/google/uuid v1.0.0
	github.com/gorilla/websocket v0.0.0-20170926233335-4201258b820c // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.1-0.20190118093823-f849b5445de4
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.4.1
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jonboulle/clockwork v0.1.0
	github.com/json-iterator/go v1.1.5
	github.com/kr/pty v1.1.1
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.7 // indirect
	github.com/mattn/go-runewidth v0.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1
	github.com/olekukonko/tablewriter v0.0.0-20170122224234-a0225b3f23b5
	github.com/pkg/errors v0.8.1 // indirect
	github.com/prometheus/client_golang v0.9.3
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.1
	github.com/stretchr/testify v1.3.0 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20170815181823-89b8d40f7ca8
	github.com/urfave/cli v1.20.0
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2
	go.etcd.io/bbolt v1.3.2
	go.uber.org/atomic v1.3.2 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20190404164418-38d8ce5564a5
	golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3
	golang.org/x/time v0.0.0-20180412165947-fbb02b2291d2
	google.golang.org/genproto v0.0.0-20180608181217-32ee49c4dd80 // indirect
	google.golang.org/grpc v1.14.0
	gopkg.in/cheggaaa/pb.v1 v1.0.25
	gopkg.in/yaml.v2 v2.2.2
	sigs.k8s.io/yaml v1.1.0
)

// go-resty is imported under its old import path by grpc-gateway.
// This works around that problem until we pick up a version of
// grpc-gateway that has the imports fixed.
//
// See: https://github.com/go-resty/resty/issues/230
replace github.com/go-resty/resty => gopkg.in/resty.v1 v1.11.0
