module github.com/julienlevasseur/profiler

go 1.16

require (
	github.com/aws/aws-sdk-go v1.25.41
	github.com/hashicorp/consul/api v1.12.0
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/miekg/dns v1.1.41 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-testing-interface v1.14.0 // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/spf13/cobra v1.3.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.10.1
	gopkg.in/yaml.v3 v3.0.0
)

replace github.com/tencentcloud/tencentcloud-sdk-go v3.0.83+incompatible => github.com/tencentcloud/tencentcloud-sdk-go v1.0.308
