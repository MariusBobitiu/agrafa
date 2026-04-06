package heartbeat

import (
	"os"
	"runtime"

	"github.com/MariusBobitiu/agrafa-agent/src/types"
	"github.com/MariusBobitiu/agrafa-agent/src/utils"
)

func BuildRequest(nodeID int64, source string) types.HeartbeatRequest {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return types.HeartbeatRequest{
		NodeID:     nodeID,
		ObservedAt: utils.NowUTC(),
		Source:     source,
		Payload: map[string]any{
			"hostname": hostname,
			"platform": runtime.GOOS,
			"arch":     runtime.GOARCH,
		},
	}
}
