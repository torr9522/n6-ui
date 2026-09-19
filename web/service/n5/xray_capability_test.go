package n5

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestXrayRuntimeCapabilities(t *testing.T) {
	binaryPath := os.Getenv("N5_XRAY_TEST_BINARY")
	if binaryPath == "" {
		t.Skip("N5_XRAY_TEST_BINARY is not set")
	}
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skipf("xray test binary unavailable: %v", err)
	}

	cases := []struct {
		name   string
		config string
	}{
		{
			name: "outbound_tag",
			config: `{
				"log":{"loglevel":"warning"},
				"outbounds":[
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000001"},
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000002"}
				]
			}`,
		},
		{
			name: "balancer_selector",
			config: `{
				"log":{"loglevel":"warning"},
				"outbounds":[
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000001"},
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000002"}
				],
				"routing":{
					"rules":[
						{"type":"field","domain":["full:selector.example"],"balancerTag":"n5-pool-0000000001"}
					],
					"balancers":[
						{"tag":"n5-pool-0000000001","selector":["n5-egress-0000000001","n5-egress-0000000002"]}
					]
				}
			}`,
		},
		{
			name: "balancer_strategy",
			config: `{
				"log":{"loglevel":"warning"},
				"outbounds":[
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000001"},
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000002"}
				],
				"routing":{
					"rules":[
						{"type":"field","domain":["full:strategy.example"],"balancerTag":"n5-pool-0000000001"}
					],
					"balancers":[
						{"tag":"n5-pool-0000000001","selector":["n5-egress-0000000001","n5-egress-0000000002"],"strategy":{"type":"random"}}
					]
				}
			}`,
		},
		{
			name: "balancer_fallback",
			config: `{
				"log":{"loglevel":"warning"},
				"outbounds":[
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000001"},
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000002"},
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000003"}
				],
				"observatory":{
					"subjectSelector":["n5-egress-0000000001","n5-egress-0000000002"],
					"probeUrl":"https://www.google.com/generate_204"
				},
				"routing":{
					"rules":[
						{"type":"field","domain":["full:fallback.example"],"balancerTag":"n5-pool-0000000001"}
					],
					"balancers":[
						{"tag":"n5-pool-0000000001","selector":["n5-egress-0000000001","n5-egress-0000000002"],"fallbackTag":"n5-egress-0000000003"}
					]
				}
			}`,
		},
		{
			name: "balancer_strategy_fallback_combined",
			config: `{
				"log":{"loglevel":"warning"},
				"outbounds":[
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000001"},
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000002"},
					{"protocol":"freedom","settings":{},"tag":"n5-egress-0000000003"}
				],
				"observatory":{
					"subjectSelector":["n5-egress-0000000001","n5-egress-0000000002"],
					"probeUrl":"https://www.google.com/generate_204"
				},
				"routing":{
					"rules":[
						{"type":"field","domain":["full:combo.example"],"balancerTag":"n5-pool-0000000001"}
					],
					"balancers":[
						{"tag":"n5-pool-0000000001","selector":["n5-egress-0000000001","n5-egress-0000000002"],"strategy":{"type":"random"},"fallbackTag":"n5-egress-0000000003"}
					]
				}
			}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, tc.name+".json")
			if err := os.WriteFile(configPath, []byte(tc.config), 0644); err != nil {
				t.Fatalf("write config failed: %v", err)
			}

			cmd := exec.Command(binaryPath, "run", "-test", "-c", configPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("xray capability test failed: %v, output: %s", err, string(output))
			}
		})
	}
}
