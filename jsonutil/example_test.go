package jsonutil_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/drblury/apiweaver/jsonutil"
)

func Example() {
	type serviceInfo struct {
		Name        string `json:"name"`
		Build       int    `json:"build"`
		Environment string `json:"environment"`
	}

	info := serviceInfo{
		Name:        "payment",
		Build:       42,
		Environment: "staging",
	}

	data, _ := jsonutil.Marshal(info)
	fmt.Println(string(data))

	var decoded serviceInfo
	_ = jsonutil.Unmarshal(data, &decoded)
	fmt.Println(decoded.Build)

	buf := &bytes.Buffer{}
	_ = jsonutil.Encode(buf, info)

	var streamed serviceInfo
	_ = jsonutil.Decode(buf, &streamed)
	fmt.Println(streamed.Environment)

	// Output:
	// {"name":"payment","build":42,"environment":"staging"}
	// 42
	// staging
}

func ExampleMarshalIndent() {
	type release struct {
		Service string   `json:"service"`
		Tags    []string `json:"tags"`
		Version string   `json:"version"`
	}

	payload := release{
		Service: "billing",
		Tags:    []string{"stable", "edge"},
		Version: "1.4.0",
	}

	data, err := jsonutil.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Println("marshal error:", err)
		return
	}

	fmt.Println(strings.TrimSpace(string(data)))

	var decoded release
	if err := jsonutil.Unmarshal(data, &decoded); err != nil {
		fmt.Println("unmarshal error:", err)
		return
	}
	fmt.Println(decoded.Version)

	// Output:
	// {
	//   "service": "billing",
	//   "tags": [
	//     "stable",
	//     "edge"
	//   ],
	//   "version": "1.4.0"
	// }
	// 1.4.0
}

func ExampleEncode_stream() {
	type metrics struct {
		Service  string `json:"service"`
		Requests int    `json:"requests"`
	}

	buf := &bytes.Buffer{}
	payload := metrics{Service: "ledger", Requests: 512}

	if err := jsonutil.Encode(buf, payload); err != nil {
		fmt.Println("encode error:", err)
		return
	}
	fmt.Println(strings.TrimSpace(buf.String()))

	var decoded metrics
	if err := jsonutil.Decode(bytes.NewReader(buf.Bytes()), &decoded); err != nil {
		fmt.Println("decode error:", err)
		return
	}
	fmt.Printf("%s %d\n", decoded.Service, decoded.Requests)

	// Output:
	// {"service":"ledger","requests":512}
	// ledger 512
}
